package backup

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"tsunagu/backend/internal/backup/mihonpb"
	"tsunagu/backend/internal/chapternum"
	"tsunagu/backend/internal/db/sqlcgen"
	"tsunagu/backend/internal/localsource"
)

// Mihon's reserved source ID for local manga (LocalSource.ID = 0L).
const localSourceID = 0

type ImportResult struct {
	MangaImported      int
	MangaSkipped       int
	CategoriesImported int
	TrackingImported   int
	Warnings           []string
}

func derefOr(s *string, fallback string) string {
	if s == nil {
		return fallback
	}
	return *s
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func b2i(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func Import(ctx context.Context, q *sqlcgen.Queries, b *mihonpb.Backup) (ImportResult, error) {
	var res ImportResult

	extensions, err := q.ListInstalledExtensionsWithSourceID(ctx)
	if err != nil {
		return res, err
	}
	extBySourceID := make(map[int64]sqlcgen.Extension, len(extensions))
	for _, e := range extensions {
		extBySourceID[e.SourceID] = e
	}

	accounts, err := q.ListTrackerAccounts(ctx)
	if err != nil {
		return res, err
	}
	accountByType := make(map[string]sqlcgen.TrackerAccount, len(accounts))
	for _, a := range accounts {
		accountByType[a.TrackerType] = a
	}

	catFolderID := make(map[int64]int64, len(b.Categories))
	for i, c := range b.Categories {
		existing, err := q.GetCustomFolderByName(ctx, c.Name)
		if err == nil {
			catFolderID[int64(i)] = existing.ID
			continue
		}
		if err != sql.ErrNoRows {
			return res, err
		}
		created, err := q.CreateFolder(ctx, sqlcgen.CreateFolderParams{Name: c.Name})
		if err != nil {
			return res, err
		}
		catFolderID[int64(i)] = created.ID
		res.CategoriesImported++
	}

	for _, bm := range b.Manga {
		var media sqlcgen.Medium

		if bm.Source == localSourceID {
			ct, ok := localsource.ParseLocalExternalID(bm.Url)
			if !ok {
				res.MangaSkipped++
				res.Warnings = append(res.Warnings, fmt.Sprintf("skipped %q: local-source entry with an unrecognized URL", bm.Title))
				continue
			}
			m, err := upsertLocalMedia(ctx, q, ct, bm)
			if err != nil {
				return res, err
			}
			media = m
		} else {
			ext, ok := extBySourceID[bm.Source]
			if !ok {
				res.MangaSkipped++
				res.Warnings = append(res.Warnings, fmt.Sprintf("skipped %q: matching source not installed", bm.Title))
				continue
			}

			m, err := q.UpsertMediaDetails(ctx, sqlcgen.UpsertMediaDetailsParams{
				ExtensionID:   sql.NullInt64{Int64: ext.ID, Valid: true},
				ExtensionName: ext.Name,
				ExternalID:    bm.Url,
				ContentType:   ext.ContentType,
				Title:         bm.Title,
				CoverPath:     nullString(derefOr(bm.ThumbnailUrl, "")),
				Description:   nullString(derefOr(bm.Description, "")),
				Status:        nullString(mangaStatusFromInt[bm.Status]),
				Author:        nullString(derefOr(bm.Author, "")),
				Artist:        nullString(derefOr(bm.Artist, "")),
			})
			if err != nil {
				return res, err
			}
			media = m
		}
		if _, err := q.AddMediaToLibrary(ctx, media.ID); err != nil {
			return res, err
		}

		for _, catIdx := range bm.Categories {
			folderID, ok := catFolderID[catIdx]
			if !ok {
				continue
			}
			if err := q.AddMediaToFolder(ctx, sqlcgen.AddMediaToFolderParams{MediaID: media.ID, FolderID: folderID}); err != nil {
				return res, err
			}
		}

		for _, bc := range bm.Chapters {
			ch, err := q.CreateChapter(ctx, sqlcgen.CreateChapterParams{
				MediaID:     media.ID,
				ExternalID:  bc.Url,
				Title:       nullString(bc.Name),
				Number:      sql.NullFloat64{Float64: chapternum.Round(float64(bc.ChapterNumber)), Valid: true},
				UploadedAt:  sql.NullInt64{Int64: bc.DateUpload / 1000, Valid: bc.DateUpload > 0},
				SourceOrder: sql.NullInt64{Int64: bc.SourceOrder, Valid: true},
				Scanlator:   derefOr(bc.Scanlator, ""),
			})
			if err != nil {
				return res, err
			}
			if bc.Read {
				if _, err := q.UpsertReadingProgress(ctx, sqlcgen.UpsertReadingProgressParams{
					MediaID:   media.ID,
					ChapterID: ch.ID,
					Progress:  1,
					Completed: true,
				}); err != nil {
					return res, err
				}
			}
		}

		for _, bt := range bm.Tracking {
			trackerType, ok := trackerSyncIDToKey[bt.SyncId]
			if !ok {
				continue
			}
			acct, ok := accountByType[trackerType]
			if !ok {
				res.Warnings = append(res.Warnings, fmt.Sprintf("%q: not logged into %s, tracking skipped", bm.Title, trackerType))
				continue
			}
			remoteID := ""
			switch {
			case bt.MediaId != 0:
				remoteID = strconv.FormatInt(bt.MediaId, 10)
			case bt.MediaIdInt != 0:
				remoteID = strconv.FormatInt(int64(bt.MediaIdInt), 10)
			}
			if remoteID == "" {
				continue
			}
			if _, err := q.UpsertTrackerLink(ctx, sqlcgen.UpsertTrackerLinkParams{
				MediaID:           media.ID,
				TrackerAccountID:  acct.ID,
				ExternalTrackerID: remoteID,
				LibraryID:         nullString(strconv.FormatInt(bt.LibraryId, 10)),
				TrackerTitle:      bt.Title,
				RemoteUrl:         bt.TrackingUrl,
				Status:            int64(bt.Status),
				LastChapterRead:   float64(bt.LastChapterRead),
				TotalChapters:     int64(bt.TotalChapters),
				Score:             float64(bt.Score),
				Private:           b2i(bt.Private),
				SyncProgress:      true,
			}); err != nil {
				return res, err
			}
			res.TrackingImported++
		}

		res.MangaImported++
	}

	return res, nil
}

// Looks up by external_id — NULL extension_id never conflicts in SQLite, so
// UpsertMediaDetails's ON CONFLICT can't dedupe local rows.
func upsertLocalMedia(ctx context.Context, q *sqlcgen.Queries, contentType string, bm *mihonpb.BackupManga) (sqlcgen.Medium, error) {
	existing, err := q.GetLocalMediaByExternalID(ctx, bm.Url)
	if err == nil {
		return existing, nil
	}
	if err != sql.ErrNoRows {
		return sqlcgen.Medium{}, err
	}
	return q.CreateLocalMedia(ctx, sqlcgen.CreateLocalMediaParams{
		ExternalID:  bm.Url,
		ContentType: contentType,
		Title:       bm.Title,
	})
}
