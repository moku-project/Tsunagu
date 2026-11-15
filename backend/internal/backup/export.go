package backup

import (
	"context"
	"strconv"
	"strings"

	"tsunagu/backend/internal/backup/mihonpb"
	"tsunagu/backend/internal/db/sqlcgen"
)

var mangaStatusToInt = map[string]int32{
	"ongoing":             1,
	"completed":           2,
	"licensed":            3,
	"publishing finished": 4,
	"cancelled":           5,
	"on hiatus":           6,
}

var mangaStatusFromInt = map[int32]string{
	1: "Ongoing",
	2: "Completed",
	3: "Licensed",
	4: "Publishing Finished",
	5: "Cancelled",
	6: "On Hiatus",
}

var trackerKeyToSyncID = map[string]int32{
	"mal":     1,
	"anilist": 2,
}

var trackerSyncIDToKey = map[int32]string{
	1: "mal",
	2: "anilist",
}

func statusToInt(status string) int32 {
	return mangaStatusToInt[strings.ToLower(strings.TrimSpace(status))]
}

func Export(ctx context.Context, q *sqlcgen.Queries) (*mihonpb.Backup, error) {
	mediaList, err := q.ListLibraryMediaForExport(ctx)
	if err != nil {
		return nil, err
	}

	accounts, err := q.ListTrackerAccounts(ctx)
	if err != nil {
		return nil, err
	}
	accountTypeByID := make(map[int64]string, len(accounts))
	for _, a := range accounts {
		accountTypeByID[a.ID] = a.TrackerType
	}

	mediaIDs := make([]int64, len(mediaList))
	extIDSet := make(map[int64]struct{})
	for i, m := range mediaList {
		mediaIDs[i] = m.ID
		if m.ExtensionID.Valid {
			extIDSet[m.ExtensionID.Int64] = struct{}{}
		}
	}
	extIDs := make([]int64, 0, len(extIDSet))
	for id := range extIDSet {
		extIDs = append(extIDs, id)
	}
	extensions, err := q.GetExtensionsByIDs(ctx, extIDs)
	if err != nil {
		return nil, err
	}
	extByID := make(map[int64]sqlcgen.Extension, len(extensions))
	sourceIDSeen := make(map[int64]string)
	for _, e := range extensions {
		extByID[e.ID] = e
		if e.SourceID != 0 {
			sourceIDSeen[e.SourceID] = e.Name
		}
	}

	folderRows, err := q.ListFoldersByMediaIDs(ctx, mediaIDs)
	if err != nil {
		return nil, err
	}
	categoryIndexByFolderID := make(map[int64]int64)
	var categories []*mihonpb.BackupCategory
	foldersByMedia := make(map[int64][]int64)
	for _, row := range folderRows {
		if row.Kind != "custom" {
			continue
		}
		idx, ok := categoryIndexByFolderID[row.ID]
		if !ok {
			idx = int64(len(categories))
			categoryIndexByFolderID[row.ID] = idx
			categories = append(categories, &mihonpb.BackupCategory{
				Name:  row.Name,
				Order: idx,
				Id:    idx,
			})
		}
		foldersByMedia[row.MediaID] = append(foldersByMedia[row.MediaID], idx)
	}

	genreRows, err := q.ListGenresByMediaIDs(ctx, mediaIDs)
	if err != nil {
		return nil, err
	}
	genresByMedia := make(map[int64][]string)
	for _, row := range genreRows {
		genresByMedia[row.MediaID] = append(genresByMedia[row.MediaID], row.Name)
	}

	trackerLinks, err := q.ListTrackerLinksByMediaIDs(ctx, mediaIDs)
	if err != nil {
		return nil, err
	}
	trackingByMedia := make(map[int64][]sqlcgen.TrackerLink)
	for _, l := range trackerLinks {
		trackingByMedia[l.MediaID] = append(trackingByMedia[l.MediaID], l)
	}

	backupManga := make([]*mihonpb.BackupManga, 0, len(mediaList))
	for _, m := range mediaList {
		var sourceID int64
		if m.ExtensionID.Valid {
			sourceID = extByID[m.ExtensionID.Int64].SourceID
		}

		chapters, err := q.ListChaptersByMedia(ctx, m.ID)
		if err != nil {
			return nil, err
		}
		progress, err := q.ListReadingProgressByMedia(ctx, m.ID)
		if err != nil {
			return nil, err
		}
		progressByChapter := make(map[int64]sqlcgen.ReadingProgress, len(progress))
		for _, p := range progress {
			progressByChapter[p.ChapterID] = p
		}

		bkChapters := make([]*mihonpb.BackupChapter, 0, len(chapters))
		for _, c := range chapters {
			bc := &mihonpb.BackupChapter{
				Url:           c.ExternalID,
				Name:          c.Title.String,
				ChapterNumber: float32(c.Number.Float64),
				SourceOrder:   c.SourceOrder.Int64,
				DateUpload:    c.UploadedAt.Int64 * 1000,
			}
			if c.Scanlator != "" {
				bc.Scanlator = &c.Scanlator
			}
			if p, ok := progressByChapter[c.ID]; ok {
				bc.Read = p.Completed
			}
			bc.Bookmark = c.Bookmarked
			bkChapters = append(bkChapters, bc)
		}

		var tracking []*mihonpb.BackupTracking
		for _, l := range trackingByMedia[m.ID] {
			trackerType := accountTypeByID[l.TrackerAccountID]
			syncID, ok := trackerKeyToSyncID[trackerType]
			if !ok {
				continue
			}
			remoteID, _ := strconv.ParseInt(l.ExternalTrackerID, 10, 64)
			libID, _ := strconv.ParseInt(l.LibraryID.String, 10, 64)
			tracking = append(tracking, &mihonpb.BackupTracking{
				SyncId:          syncID,
				MediaId:         remoteID,
				LibraryId:       libID,
				TrackingUrl:     l.RemoteUrl,
				Title:           l.TrackerTitle,
				LastChapterRead: float32(l.LastChapterRead),
				TotalChapters:   int32(l.TotalChapters),
				Score:           float32(l.Score),
				Status:          int32(l.Status),
				Private:         l.Private != 0,
			})
		}

		bm := &mihonpb.BackupManga{
			Source:      sourceID,
			Url:         m.ExternalID,
			Title:       m.Title,
			Genre:       genresByMedia[m.ID],
			Status:      statusToInt(m.Status.String),
			Chapters:    bkChapters,
			Categories:  foldersByMedia[m.ID],
			Tracking:    tracking,
			Favorite:    true,
			Initialized: m.DetailsFetchedAt.Valid,
		}
		if m.Artist.String != "" {
			bm.Artist = &m.Artist.String
		}
		if m.Author.String != "" {
			bm.Author = &m.Author.String
		}
		if m.Description.String != "" {
			bm.Description = &m.Description.String
		}
		cover := m.CoverPath.String
		if cover != "" {
			bm.ThumbnailUrl = &cover
		}
		if m.AddedAt.Valid {
			bm.DateAdded = m.AddedAt.Time.UnixMilli()
		}
		backupManga = append(backupManga, bm)
	}

	sources := make([]*mihonpb.BackupSource, 0, len(sourceIDSeen))
	for id, name := range sourceIDSeen {
		if id == 0 {
			continue
		}
		sources = append(sources, &mihonpb.BackupSource{Name: name, SourceId: id})
	}

	return &mihonpb.Backup{
		Manga:      backupManga,
		Categories: categories,
		Sources:    sources,
	}, nil
}
