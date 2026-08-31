package graph

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"tsunagu/backend/internal/aniskip"
	"tsunagu/backend/internal/api/graph/model"
	"tsunagu/backend/internal/db/sqlcgen"
	"tsunagu/backend/internal/image"
	"tsunagu/backend/internal/metadata"
	"tsunagu/backend/internal/sandbox"
	sandboxv1 "tsunagu/backend/internal/sandbox/gen/sandbox/v1"
	syncpkg "tsunagu/backend/internal/sync"
)

func (r *chapterResolver) ReadingProgress(ctx context.Context, obj *model.Chapter) (*model.ReadingProgress, error) {
	chID, err := parseID(obj.ID)
	if err != nil {
		return nil, err
	}
	p, err := LoadersFromContext(ctx).ProgressByChapter.Load(ctx, chID)
	if err != nil || p == nil {
		return nil, err
	}
	return toReadingProgress(*p), nil
}

func (r *chapterResolver) Completed(ctx context.Context, obj *model.Chapter) (bool, error) {
	chID, err := parseID(obj.ID)
	if err != nil {
		return false, err
	}
	p, err := LoadersFromContext(ctx).ProgressByChapter.Load(ctx, chID)
	if err != nil {
		return false, err
	}
	return p != nil && p.Completed, nil
}

func (r *chapterResolver) Downloaded(ctx context.Context, obj *model.Chapter) (bool, error) {
	chID, err := parseID(obj.ID)
	if err != nil {
		return false, err
	}
	return LoadersFromContext(ctx).DownloadedByChapter.Load(ctx, chID)
}

func (r *chapterResolver) Download(ctx context.Context, obj *model.Chapter) (*model.Download, error) {
	chID, err := parseID(obj.ID)
	if err != nil {
		return nil, err
	}
	d, err := LoadersFromContext(ctx).LatestDownloadByChapter.Load(ctx, chID)
	if err != nil || d == nil {
		return nil, err
	}
	mID, _ := parseID(obj.MediaID)
	return toDownloadFields(d.ID, d.ChapterID, mID, d.Status, d.Progress, d.DownloadedBytes, d.BytesPerSec, d.Error, d.CreatedAt, d.CompletedAt), nil
}

func (r *chapterResolver) Pages(ctx context.Context, obj *model.Chapter) ([]string, error) {
	chID, err := parseID(obj.ID)
	if err != nil {
		return nil, err
	}
	if n, err := LoadersFromContext(ctx).DownloadedPagesByChapter.Load(ctx, chID); err == nil && n > 0 {
		return contentPageURLs(obj.MediaID, obj.ID, int(n)), nil
	}

	dctx, err := r.Q.GetChapterDownloadContext(ctx, chID)
	if err != nil {
		return nil, err
	}
	if dctx.ContentType != "manga" {
		return nil, nil
	}
	client, err := r.Sc.Ensure(ctx)
	if err != nil {
		return nil, fmt.Errorf("sandbox unavailable: %w", err)
	}
	pages, err := client.GetPages(ctx, dctx.ExtensionPackageName, dctx.SourceEntryID, dctx.SourceChapterID)
	if err != nil {
		return nil, fmt.Errorf("fetching page list: %w", err)
	}
	return contentPageURLs(obj.MediaID, obj.ID, len(pages.GetPageUrls())), nil
}

func (r *chapterResolver) PageCount(ctx context.Context, obj *model.Chapter) (*int32, error) {
	chID, err := parseID(obj.ID)
	if err != nil {
		return nil, err
	}
	n, err := LoadersFromContext(ctx).DownloadedPagesByChapter.Load(ctx, chID)
	if err != nil || n == 0 {
		return nil, err
	}
	return &n, nil
}

func (r *chapterResolver) VideoURL(ctx context.Context, obj *model.Chapter) (*string, error) {

	url := fmt.Sprintf("/content/%s/%s/video", obj.MediaID, obj.ID)
	return &url, nil
}

func (r *chapterResolver) VideoStream(ctx context.Context, obj *model.Chapter) (*model.VideoStream, error) {
	chID, err := parseID(obj.ID)
	if err != nil {
		return nil, err
	}
	dctx, err := r.Q.GetChapterDownloadContext(ctx, chID)
	if err != nil || dctx.ContentType != "anime" {
		return nil, nil
	}

	if row, err := r.Q.GetAnimeEpisodeStream(ctx, chID); err == nil && row.LocalPath.Valid && row.LocalPath.String != "" {
		return toVideoStream(&sandboxv1.StreamInfo{}, obj.MediaID, obj.ID), nil
	}
	info, err := r.Sr.Resolve(ctx, dctx.ExtensionPackageName, dctx.SourceEntryID, dctx.SourceChapterID)
	if err != nil {
		return nil, nil
	}
	return toVideoStream(info, obj.MediaID, obj.ID), nil
}

func (r *downloadResolver) Chapter(ctx context.Context, obj *model.Download) (*model.Chapter, error) {
	id, err := parseID(obj.ChapterID)
	if err != nil {
		return nil, err
	}
	ch, err := r.Q.GetChapter(ctx, id)
	if err != nil {
		return nil, err
	}
	return toChapter(ch), nil
}

func (r *mediaResolver) Chapters(ctx context.Context, obj *model.Media) ([]*model.Chapter, error) {
	id, err := parseID(obj.ID)
	if err != nil {
		return nil, err
	}
	rows, err := r.Sy.EnsureChapters(ctx, r.Sc, id)
	if err != nil {
		return nil, err
	}
	out := make([]*model.Chapter, 0, len(rows))
	for _, ch := range rows {
		out = append(out, toChapter(ch))
	}
	return out, nil
}

func (r *mediaResolver) ChapterCount(ctx context.Context, obj *model.Media) (int32, error) {
	id, err := parseID(obj.ID)
	if err != nil {
		return 0, err
	}
	return LoadersFromContext(ctx).ChapterCountByMedia.Load(ctx, id)
}

func (r *mediaResolver) UnreadCount(ctx context.Context, obj *model.Media) (int32, error) {
	id, err := parseID(obj.ID)
	if err != nil {
		return 0, err
	}
	return LoadersFromContext(ctx).UnreadCountByMedia.Load(ctx, id)
}

func (r *mediaResolver) DownloadedCount(ctx context.Context, obj *model.Media) (int32, error) {
	id, err := parseID(obj.ID)
	if err != nil {
		return 0, err
	}
	return LoadersFromContext(ctx).DownloadedCountByMedia.Load(ctx, id)
}

func (r *mediaResolver) NextUnreadChapter(ctx context.Context, obj *model.Media) (*model.Chapter, error) {
	id, err := parseID(obj.ID)
	if err != nil {
		return nil, err
	}
	c, err := LoadersFromContext(ctx).NextUnreadByMedia.Load(ctx, id)
	if err != nil || c == nil {
		return nil, err
	}
	return toChapter(*c), nil
}

func (r *mediaResolver) LatestChapter(ctx context.Context, obj *model.Media) (*model.Chapter, error) {
	id, err := parseID(obj.ID)
	if err != nil {
		return nil, err
	}
	c, err := LoadersFromContext(ctx).LatestChapterByMedia.Load(ctx, id)
	if err != nil || c == nil {
		return nil, err
	}
	return toChapter(*c), nil
}

func (r *mediaResolver) ReadingProgress(ctx context.Context, obj *model.Media) ([]*model.ReadingProgress, error) {
	id, err := parseID(obj.ID)
	if err != nil {
		return nil, err
	}
	rows, err := LoadersFromContext(ctx).ProgressByMedia.Load(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]*model.ReadingProgress, 0, len(rows))
	for _, p := range rows {
		out = append(out, toReadingProgress(p))
	}
	return out, nil
}

func (r *mediaResolver) Tags(ctx context.Context, obj *model.Media) ([]string, error) {
	id, err := parseID(obj.ID)
	if err != nil {
		return nil, err
	}
	tags, err := LoadersFromContext(ctx).TagsByMedia.Load(ctx, id)
	if err != nil {
		return nil, err
	}
	if tags == nil {
		return []string{}, nil
	}
	return tags, nil
}

func (r *mediaResolver) Genres(ctx context.Context, obj *model.Media) ([]string, error) {
	id, err := parseID(obj.ID)
	if err != nil {
		return nil, err
	}
	genres, err := LoadersFromContext(ctx).GenresByMedia.Load(ctx, id)
	if err != nil {
		return nil, err
	}
	if genres == nil {
		return []string{}, nil
	}
	return genres, nil
}

func (r *mediaResolver) Folders(ctx context.Context, obj *model.Media) ([]*model.Folder, error) {
	id, err := parseID(obj.ID)
	if err != nil {
		return nil, err
	}
	rows, err := LoadersFromContext(ctx).FoldersByMedia.Load(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]*model.Folder, 0, len(rows))
	for _, f := range rows {
		out = append(out, toFolder(f))
	}
	return out, nil
}

func (r *mediaResolver) Source(ctx context.Context, obj *model.Media) (*model.Extension, error) {
	if obj.ExtensionID == nil {
		return nil, nil
	}
	extID, err := parseID(*obj.ExtensionID)
	if err != nil {
		return nil, err
	}
	ext, err := LoadersFromContext(ctx).ExtensionByID.Load(ctx, extID)
	if err != nil || ext == nil {
		return nil, err
	}
	return toExtension(*ext, r.MediaDir), nil
}

func (r *mutationResolver) CreateFolder(ctx context.Context, name string, parentFolderID *string) (*model.Folder, error) {
	var parentID sql.NullInt64
	if parentFolderID != nil {
		pid, err := parseID(*parentFolderID)
		if err != nil {
			return nil, err
		}
		parentID = sql.NullInt64{Int64: pid, Valid: true}
	}
	row, err := r.Q.CreateFolder(ctx, sqlcgen.CreateFolderParams{Name: name, ParentFolderID: parentID})
	if err != nil {
		return nil, err
	}
	return toFolder(row), nil
}

func (r *mutationResolver) RenameFolder(ctx context.Context, folderID string, name string) (*model.Folder, error) {
	fid, err := parseID(folderID)
	if err != nil {
		return nil, err
	}
	row, err := r.Q.RenameFolder(ctx, sqlcgen.RenameFolderParams{Name: name, ID: fid})
	if err != nil {
		return nil, err
	}
	return toFolder(row), nil
}

func (r *mutationResolver) DeleteFolder(ctx context.Context, folderID string) (bool, error) {
	fid, err := parseID(folderID)
	if err != nil {
		return false, err
	}
	return true, r.Q.DeleteFolder(ctx, fid)
}

func (r *mutationResolver) AddMediaToFolder(ctx context.Context, mediaID string, folderID string) (bool, error) {
	mid, err := parseID(mediaID)
	if err != nil {
		return false, err
	}
	fid, err := parseID(folderID)
	if err != nil {
		return false, err
	}
	return true, r.Q.AddMediaToFolder(ctx, sqlcgen.AddMediaToFolderParams{MediaID: mid, FolderID: fid})
}

func (r *mutationResolver) RemoveMediaFromFolder(ctx context.Context, mediaID string, folderID string) (bool, error) {
	mid, err := parseID(mediaID)
	if err != nil {
		return false, err
	}
	fid, err := parseID(folderID)
	if err != nil {
		return false, err
	}
	return true, r.Q.RemoveMediaFromFolder(ctx, sqlcgen.RemoveMediaFromFolderParams{MediaID: mid, FolderID: fid})
}

func (r *mutationResolver) MarkChaptersRead(ctx context.Context, mediaID string, chapterIds []string, read bool) ([]*model.ReadingProgress, error) {
	mID, err := parseID(mediaID)
	if err != nil {
		return nil, err
	}
	progress := 1.0
	if !read {
		progress = 0.0
	}
	results := make([]*model.ReadingProgress, 0, len(chapterIds))
	for _, chapterID := range chapterIds {
		chID, err := parseID(chapterID)
		if err != nil {
			return nil, err
		}
		row, err := r.Q.UpsertReadingProgress(ctx, sqlcgen.UpsertReadingProgressParams{
			MediaID:   mID,
			ChapterID: chID,
			Completed: read,
			Progress:  progress,
		})
		if err != nil {
			return nil, err
		}
		results = append(results, toReadingProgress(row))
	}
	if read {
		go r.Tk.SyncMediaProgress(context.Background(), mID)
	}
	return results, nil
}

func (r *mutationResolver) DequeueDownload(ctx context.Context, mediaID string, chapterID string) (bool, error) {
	chID, err := parseID(chapterID)
	if err != nil {
		return false, err
	}
	if err := r.Dm.Cancel(ctx, chID); err != nil {
		return false, err
	}
	return true, r.Q.DeleteDownloadByChapter(ctx, chID)
}

func (r *mutationResolver) AddRepository(ctx context.Context, indexURL string, name *string) (*model.Repository, error) {
	n := ""
	if name != nil {
		n = *name
	}
	repo, err := r.Sy.AddRepository(ctx, indexURL, n)
	if err != nil {
		return nil, err
	}
	return toRepository(repo), nil
}

func (r *mutationResolver) RenameRepository(ctx context.Context, repositoryID string, name string) (*model.Repository, error) {
	id, err := parseID(repositoryID)
	if err != nil {
		return nil, err
	}
	repo, err := r.Sy.RenameRepository(ctx, id, name)
	if err != nil {
		return nil, err
	}
	return toRepository(repo), nil
}

func (r *mutationResolver) DeleteRepository(ctx context.Context, repositoryID string) (bool, error) {
	id, err := parseID(repositoryID)
	if err != nil {
		return false, err
	}
	return true, r.Sy.DeleteRepository(ctx, id)
}

func (r *mutationResolver) InstallExtension(ctx context.Context, packageName string) (*model.Extension, error) {
	ext, err := r.Sy.InstallExtension(ctx, packageName)
	if err != nil {
		return nil, err
	}
	c, err := r.Sc.Ensure(ctx)
	if err != nil {
		return nil, err
	}
	loaded, err := c.LoadExtensions(ctx, []*sandboxv1.ExtensionToLoad{{
		ExtensionId: ext.PackageName,
		JarPath:     ext.JarPath.String,
		ContentType: sandbox.ContentTypeToProto(ext.ContentType),
		Lang:        ext.Lang,
	}})
	if err != nil {
		return nil, err
	}
	ext = r.persistSupportsLatest(ctx, ext, loaded)
	return toExtension(ext, r.MediaDir), nil
}

func (r *mutationResolver) InstallExternalExtension(ctx context.Context, url string) (*model.Extension, error) {
	c, err := r.Sc.Ensure(ctx)
	if err != nil {
		return nil, err
	}
	ext, err := r.Sy.InstallExternalExtension(ctx, c, url)
	if err != nil {
		return nil, err
	}
	return toExtension(ext, r.MediaDir), nil
}

func (r *mutationResolver) UninstallExtension(ctx context.Context, packageName string) (*model.Extension, error) {
	c, err := r.Sc.Ensure(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := c.UnloadExtension(ctx, packageName); err != nil {
		return nil, err
	}
	ext, err := r.Sy.UninstallExtension(ctx, packageName)
	if err != nil {
		return nil, err
	}
	return toExtension(ext, r.MediaDir), nil
}

func (r *mutationResolver) UpdateExtension(ctx context.Context, packageName string) (*model.Extension, error) {
	c, err := r.Sc.Ensure(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := c.UnloadExtension(ctx, packageName); err != nil {
		return nil, err
	}
	ext, err := r.Sy.UpdateExtension(ctx, packageName)
	if err != nil {
		return nil, err
	}
	loaded, err := c.LoadExtensions(ctx, []*sandboxv1.ExtensionToLoad{{
		ExtensionId: ext.PackageName,
		JarPath:     ext.JarPath.String,
		ContentType: sandbox.ContentTypeToProto(ext.ContentType),
		Lang:        ext.Lang,
	}})
	if err != nil {
		return nil, err
	}
	ext = r.persistSupportsLatest(ctx, ext, loaded)
	return toExtension(ext, r.MediaDir), nil
}

func (r *mutationResolver) SetInLibrary(ctx context.Context, mediaID string, inLibrary bool) (*model.Media, error) {
	id, err := parseID(mediaID)
	if err != nil {
		return nil, err
	}
	var c *sandbox.Client
	if inLibrary {
		c, err = r.Sc.Ensure(ctx)
		if err != nil {
			return nil, err
		}
	}
	m, err := r.Sy.SetInLibrary(ctx, c, id, inLibrary)
	if err != nil {
		return nil, err
	}
	return toMedia(m, r.MediaDir), nil
}

func (r *mutationResolver) MigrateMedia(ctx context.Context, fromMediaID string, toExtensionID string, toExternalID string) (*model.Media, error) {
	fromID, err := parseID(fromMediaID)
	if err != nil {
		return nil, err
	}
	ext, err := r.resolveExtension(ctx, toExtensionID)
	if err != nil {
		return nil, err
	}
	c, err := r.Sc.Ensure(ctx)
	if err != nil {
		return nil, err
	}

	oldLinks, _ := r.Tk.LinksByMedia(ctx, fromID)

	m, err := r.Sy.MigrateMedia(ctx, c, fromID, ext.PackageName, toExternalID)
	if err != nil {
		return nil, err
	}

	for _, l := range oldLinks {
		key := r.Tk.KeyForAccountID(ctx, l.TrackerAccountID)
		if key == "" {
			continue
		}
		_, _ = r.Tk.Bind(ctx, key, m.ID, l.ExternalTrackerID)
	}

	if r.Md != nil {
		go r.Md.AutoEnrich(context.Background(), m.ID)
	}
	go r.Tk.SyncMediaProgress(context.Background(), m.ID)

	return toMedia(m, r.MediaDir), nil
}

func (r *mutationResolver) SyncChapters(ctx context.Context, mediaID string) ([]*model.Chapter, error) {
	id, err := parseID(mediaID)
	if err != nil {
		return nil, err
	}
	c, err := r.Sc.Ensure(ctx)
	if err != nil {
		return nil, err
	}
	chapters, err := r.Sy.SyncChapters(ctx, c, id)
	if err != nil {
		return nil, err
	}
	out := make([]*model.Chapter, 0, len(chapters))
	for _, ch := range chapters {
		out = append(out, toChapter(ch))
	}
	return out, nil
}

func (r *mutationResolver) UpdateReadingProgress(ctx context.Context, mediaID string, chapterID string, progress float64, completed *bool, positionSeconds *float64, durationSeconds *float64) (*model.ReadingProgress, error) {
	mID, err := parseID(mediaID)
	if err != nil {
		return nil, err
	}
	chID, err := parseID(chapterID)
	if err != nil {
		return nil, err
	}
	comp := false
	if completed != nil {
		comp = *completed
	}
	prog, err := r.Sy.RecordProgress(ctx, mID, chID, progress, comp, positionSeconds, durationSeconds)
	if err != nil {
		return nil, err
	}
	if comp {
		go r.Tk.SyncMediaProgress(context.Background(), mID)
	}
	return toReadingProgress(prog), nil
}

func (r *mutationResolver) MarkChapterRead(ctx context.Context, mediaID string, chapterID string) (*model.ReadingProgress, error) {
	mID, err := parseID(mediaID)
	if err != nil {
		return nil, err
	}
	chID, err := parseID(chapterID)
	if err != nil {
		return nil, err
	}
	prog, err := r.Sy.MarkChapterRead(ctx, mID, chID)
	if err != nil {
		return nil, err
	}
	go r.Tk.SyncMediaProgress(context.Background(), mID)
	return toReadingProgress(prog), nil
}

func (r *mutationResolver) EnqueueDownload(ctx context.Context, mediaID string, chapterIds []string) ([]*model.Download, error) {
	mID, err := parseID(mediaID)
	if err != nil {
		return nil, err
	}
	downloads := make([]*model.Download, 0, len(chapterIds))
	for _, cid := range chapterIds {
		id, err := parseID(cid)
		if err != nil {
			return nil, err
		}
		if _, err := r.validateChapterMedia(ctx, mID, id); err != nil {
			return nil, err
		}
		d, err := r.Q.EnqueueDownload(ctx, id)
		if err != nil {
			return nil, err
		}
		downloads = append(downloads, toDownloadFields(d.ID, d.ChapterID, mID, d.Status, d.Progress, d.DownloadedBytes, d.BytesPerSec, d.Error, d.CreatedAt, d.CompletedAt))
	}
	r.Dm.Wake()
	return downloads, nil
}

func (r *mutationResolver) RetryDownload(ctx context.Context, mediaID string, chapterID string) (*model.Download, error) {
	mID, err := parseID(mediaID)
	if err != nil {
		return nil, err
	}
	id, err := parseID(chapterID)
	if err != nil {
		return nil, err
	}
	if _, err := r.validateChapterMedia(ctx, mID, id); err != nil {
		return nil, err
	}
	latest, err := r.Q.GetLatestDownloadForChapter(ctx, id)
	if err != nil {
		return nil, err
	}
	d, err := r.Q.RetryDownload(ctx, latest.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("cannot retry download in status %q: only failed downloads can be retried", latest.Status)
		}
		return nil, err
	}
	r.Dm.Wake()
	return toDownloadFields(d.ID, d.ChapterID, mID, d.Status, d.Progress, d.DownloadedBytes, d.BytesPerSec, d.Error, d.CreatedAt, d.CompletedAt), nil
}

func (r *mutationResolver) DeleteDownload(ctx context.Context, mediaID string, chapterIds []string) (bool, error) {
	mID, err := parseID(mediaID)
	if err != nil {
		return false, fmt.Errorf("invalid media id: %w", err)
	}
	ids := make([]int64, 0, len(chapterIds))
	for _, cid := range chapterIds {
		id, err := parseID(cid)
		if err != nil {
			return false, fmt.Errorf("invalid chapter id %q: %w", cid, err)
		}
		if _, err := r.validateChapterMedia(ctx, mID, id); err != nil {
			return false, err
		}
		ids = append(ids, id)
	}
	for _, id := range ids {
		if err := r.Dm.Cancel(ctx, id); err != nil {
			return false, fmt.Errorf("cancelling in-flight download %d: %w", id, err)
		}
		if err := r.Dm.DeleteChapterFiles(ctx, id); err != nil {
			return false, fmt.Errorf("deleting chapter files %d: %w", id, err)
		}
	}
	if err := r.Q.DeleteDownloadsByChapters(ctx, ids); err != nil {
		return false, fmt.Errorf("deleting download rows: %w", err)
	}
	return true, nil
}

func (r *mutationResolver) ReorderDownload(ctx context.Context, mediaID string, chapterID string, position int32) (bool, error) {
	mID, err := parseID(mediaID)
	if err != nil {
		return false, fmt.Errorf("invalid media id: %w", err)
	}
	id, err := parseID(chapterID)
	if err != nil {
		return false, fmt.Errorf("invalid chapter id: %w", err)
	}
	if _, err := r.validateChapterMedia(ctx, mID, id); err != nil {
		return false, err
	}
	return true, r.Dm.Reorder(ctx, id, int64(position))
}

func (r *mutationResolver) ClearDownloads(ctx context.Context, status []model.DownloadStatus) (bool, error) {
	if len(status) == 0 {
		return true, r.Dm.ClearQueue(ctx)
	}
	statuses := make([]string, 0, len(status))
	for _, st := range status {
		statuses = append(statuses, strings.ToLower(string(st)))
	}
	if err := r.Q.ClearDownloadsByStatus(ctx, statuses); err != nil {
		return false, fmt.Errorf("clearing downloads by status: %w", err)
	}
	return true, nil
}

func (r *mutationResolver) StartDownloader(ctx context.Context) (bool, error) {
	r.Dm.Resume()
	return true, nil
}

func (r *mutationResolver) StopDownloader(ctx context.Context) (bool, error) {
	r.Dm.Pause()
	return true, nil
}

func (r *mutationResolver) RefreshMetadata(ctx context.Context, mediaID string, syncChapters *bool) (*model.Media, error) {
	id, err := parseID(mediaID)
	if err != nil {
		return nil, err
	}
	c, err := r.Sc.Ensure(ctx)
	if err != nil {
		return nil, err
	}
	doSync := true
	if syncChapters != nil {
		doSync = *syncChapters
	}
	entry, err := r.refreshMediaFull(ctx, c, id, doSync)
	if err != nil {
		return nil, err
	}
	return toMedia(entry, r.MediaDir), nil
}

func (r *mutationResolver) refreshMediaFull(ctx context.Context, c *sandbox.Client, id int64, syncChapters bool) (sqlcgen.Medium, error) {
	entry, err := r.Sy.RefreshMetadata(ctx, c, id, syncChapters)
	if err != nil {
		return sqlcgen.Medium{}, err
	}
	if enriched, mErr := r.Md.Refresh(ctx, id); mErr == nil {
		entry = enriched
	}
	r.Tk.SyncMediaProgress(ctx, id)
	if m, mErr := r.Q.GetMedia(ctx, id); mErr == nil {
		entry = m
	}
	return entry, nil
}

func (r *mutationResolver) RefreshFolder(ctx context.Context, folderID string) ([]*model.Media, error) {
	fid, err := parseID(folderID)
	if err != nil {
		return nil, err
	}
	entries, err := r.Q.ListMediaInFolder(ctx, fid)
	if err != nil {
		return nil, fmt.Errorf("listing media in folder: %w", err)
	}
	c, err := r.Sc.Ensure(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*model.Media, 0, len(entries))
	for _, e := range entries {
		refreshed, err := r.refreshMediaFull(ctx, c, e.ID, true)
		if err != nil {
			continue
		}
		out = append(out, toMedia(refreshed, r.MediaDir))
	}
	return out, nil
}

func (r *mutationResolver) ReorderFolder(ctx context.Context, folderID string, sortOrder int32) (*model.Folder, error) {
	fid, err := parseID(folderID)
	if err != nil {
		return nil, err
	}
	f, err := r.Q.SetFolderSortOrder(ctx, sqlcgen.SetFolderSortOrderParams{SortOrder: int64(sortOrder), ID: fid})
	if err != nil {
		return nil, err
	}
	return toFolder(f), nil
}

func (r *mutationResolver) UpdateFolderFlags(ctx context.Context, folderID string, includeInUpdate *bool, includeInDownload *bool) (*model.Folder, error) {
	fid, err := parseID(folderID)
	if err != nil {
		return nil, err
	}
	current, err := r.Q.GetFolder(ctx, fid)
	if err != nil {
		return nil, fmt.Errorf("get folder %d: %w", fid, err)
	}
	updateVal := current.IncludeInUpdate
	if includeInUpdate != nil {
		updateVal = boolToInt64(*includeInUpdate)
	}
	downloadVal := current.IncludeInDownload
	if includeInDownload != nil {
		downloadVal = boolToInt64(*includeInDownload)
	}
	f, err := r.Q.UpdateFolderFlags(ctx, sqlcgen.UpdateFolderFlagsParams{
		IncludeInUpdate:   updateVal,
		IncludeInDownload: downloadVal,
		ID:                fid,
	})
	if err != nil {
		return nil, err
	}
	return toFolder(f), nil
}

func (r *mutationResolver) ClearImageCache(ctx context.Context) (bool, error) {
	if err := image.ClearDir(filepath.Join(r.MediaDir, "icons")); err != nil {
		return false, fmt.Errorf("clearing icon cache: %w", err)
	}
	if err := image.ClearDir(filepath.Join(r.MediaDir, "covers")); err != nil {
		return false, fmt.Errorf("clearing cover cache: %w", err)
	}
	if err := r.Q.ClearExtensionIconPaths(ctx); err != nil {
		return false, fmt.Errorf("clearing icon path records: %w", err)
	}
	if err := r.Q.ClearMediaCoverPaths(ctx); err != nil {
		return false, fmt.Errorf("clearing cover path records: %w", err)
	}
	return true, nil
}

func (r *mutationResolver) RescanLocalMedia(ctx context.Context) ([]*model.Media, error) {
	if _, err := r.Ls.Scan(ctx); err != nil {
		return nil, fmt.Errorf("rescanning local media: %w", err)
	}
	rows, err := r.Q.ListLocalMedia(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*model.Media, 0, len(rows))
	for _, row := range rows {
		out = append(out, toMedia(row, r.MediaDir))
	}
	return out, nil
}

func (r *mutationResolver) StartLibraryUpdate(ctx context.Context, folderID *string) (bool, error) {
	var fid *int64
	if folderID != nil {
		id, err := parseID(*folderID)
		if err != nil {
			return false, err
		}
		fid = &id
	}
	return r.Sy.StartLibraryUpdate(r.Sc, fid)
}

func (r *mutationResolver) SetMediaCover(ctx context.Context, mediaID string, url *string) (*model.Media, error) {
	mid, err := parseID(mediaID)
	if err != nil {
		return nil, err
	}
	var arg sql.NullString
	if url != nil && *url != "" {
		arg = sql.NullString{String: *url, Valid: true}
	}
	m, err := r.Q.SetMediaCoverOverride(ctx, sqlcgen.SetMediaCoverOverrideParams{CoverOverride: arg, ID: mid})
	if err != nil {
		return nil, err
	}
	return toMedia(m, r.MediaDir), nil
}

func (r *queryResolver) About(ctx context.Context) (*model.AboutServer, error) {
	return &model.AboutServer{Name: r.Name, Version: r.Version, BuildTime: r.BuildTime}, nil
}

func (r *queryResolver) Folders(ctx context.Context) ([]*model.Folder, error) {
	rows, err := r.Q.ListFolders(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*model.Folder, 0, len(rows))
	for _, row := range rows {
		out = append(out, toFolder(row))
	}
	return out, nil
}

func (r *queryResolver) Folder(ctx context.Context, id string) (*model.Folder, error) {
	fid, err := parseID(id)
	if err != nil {
		return nil, err
	}
	row, err := r.Q.GetFolder(ctx, fid)
	if err != nil {
		return nil, nil
	}
	return toFolder(row), nil
}

func (r *queryResolver) MediaInFolder(ctx context.Context, folderID string) ([]*model.Media, error) {
	fid, err := parseID(folderID)
	if err != nil {
		return nil, err
	}
	rows, err := r.Q.ListMediaInFolder(ctx, fid)
	if err != nil {
		return nil, err
	}
	out := make([]*model.Media, 0, len(rows))
	for _, row := range rows {
		out = append(out, toMedia(row, r.MediaDir))
	}
	return out, nil
}

func (r *queryResolver) Repositories(ctx context.Context) ([]*model.Repository, error) {
	repos, err := r.Sy.ListRepositories(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*model.Repository, 0, len(repos))
	for _, repo := range repos {
		out = append(out, toRepository(repo))
	}
	return out, nil
}

func (r *queryResolver) AvailableExtensions(ctx context.Context, repositoryID string) ([]*model.Extension, error) {
	id, err := parseID(repositoryID)
	if err != nil {
		return nil, err
	}
	exts, err := r.Sy.ListAvailableExtensions(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]*model.Extension, 0, len(exts))
	for _, ext := range exts {
		out = append(out, toExtension(ext, r.MediaDir))
	}
	return out, nil
}

func (r *queryResolver) InstalledExtensions(ctx context.Context) ([]*model.Extension, error) {
	exts, err := r.Sy.ListInstalledExtensions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*model.Extension, 0, len(exts))
	for _, ext := range exts {
		out = append(out, toExtension(ext, r.MediaDir))
	}
	return out, nil
}

func (r *queryResolver) Library(ctx context.Context, filter *model.LibraryFilter, sortInput *model.LibrarySortInput, limit *int32, offset *int32) (*model.MediaPage, error) {
	q := syncpkg.LibraryQuery{Limit: 100}
	inLib := true
	q.InLibrary = &inLib
	if filter != nil {
		if filter.ContentType != nil {
			q.ContentType = contentTypeToString(filter.ContentType)
		}
		if filter.InLibrary != nil {
			q.InLibrary = filter.InLibrary
		}
		if filter.UnreadOnly != nil {
			q.UnreadOnly = *filter.UnreadOnly
		}
		if filter.DownloadedOnly != nil {
			q.DownloadedOnly = *filter.DownloadedOnly
		}
		if filter.Query != nil {
			q.Search = *filter.Query
		}
		if filter.FolderID != nil {
			fid, err := parseID(*filter.FolderID)
			if err != nil {
				return nil, err
			}
			q.FolderID = &fid
		}
		for _, t := range filter.TagIds {
			tid, err := parseID(t)
			if err != nil {
				return nil, err
			}
			q.TagIDs = append(q.TagIDs, tid)
		}
	}
	if sortInput != nil {
		q.Ascending = sortInput.Ascending
		switch sortInput.By {
		case model.LibrarySortTitle:
			q.SortBy = "title"
		case model.LibrarySortLastReadAt:
			q.SortBy = "last_read_at"
		case model.LibrarySortLatestChapterAt:
			q.SortBy = "latest_chapter_at"
		case model.LibrarySortUnreadCount:
			q.SortBy = "unread_count"
		default:
			q.SortBy = "added_at"
		}
	}
	if limit != nil {
		q.Limit = int(*limit)
	}
	if offset != nil {
		q.Offset = int(*offset)
	}

	rows, total, err := r.Sy.QueryLibrary(ctx, q)
	if err != nil {
		return nil, err
	}
	items := make([]*model.Media, 0, len(rows))
	for _, m := range rows {
		items = append(items, toMedia(m, r.MediaDir))
	}
	return &model.MediaPage{
		Items:   items,
		Total:   int32(total),
		HasMore: q.Offset+len(items) < total,
	}, nil
}

func (r *queryResolver) Media(ctx context.Context, id string) (*model.Media, error) {
	mid, err := parseID(id)
	if err != nil {
		return nil, err
	}
	m, err := r.Sy.EnsureHydrated(ctx, r.Sc, mid)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	_ = r.Q.TouchMediaViewed(ctx, mid)
	return toMedia(m, r.MediaDir), nil
}

func (r *queryResolver) Chapter(ctx context.Context, id string) (*model.Chapter, error) {
	cid, err := parseID(id)
	if err != nil {
		return nil, err
	}
	ch, err := r.Q.GetChapter(ctx, cid)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return toChapter(ch), nil
}

func (r *queryResolver) ResolveMedia(ctx context.Context, extensionID string, externalID string, syncChapters *bool) (*model.Media, error) {
	ext, err := r.resolveExtension(ctx, extensionID)
	if err != nil {
		return nil, err
	}
	c, err := r.Sc.Ensure(ctx)
	if err != nil {
		return nil, err
	}
	doSync := true
	if syncChapters != nil {
		doSync = *syncChapters
	}
	m, err := r.Sy.ResolveMedia(ctx, c, ext.PackageName, externalID, doSync)
	if err != nil {
		return nil, fmt.Errorf("resolve media %s/%s: %w", ext.PackageName, externalID, err)
	}
	return toMedia(m, r.MediaDir), nil
}

func (r *queryResolver) ReadingProgress(ctx context.Context, mediaID string) ([]*model.ReadingProgress, error) {
	id, err := parseID(mediaID)
	if err != nil {
		return nil, err
	}
	list, err := r.Sy.ListReadingProgress(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]*model.ReadingProgress, 0, len(list))
	for _, p := range list {
		out = append(out, toReadingProgress(p))
	}
	return out, nil
}

func (r *queryResolver) Search(ctx context.Context, extensionID string, query string, page *int32, filters []*model.FilterInput) (*model.SearchResponse, error) {
	p := int32(1)
	if page != nil {
		p = *page
	}
	ext, err := r.resolveExtension(ctx, extensionID)
	if err != nil {
		return nil, err
	}
	c, err := r.Sc.Ensure(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := c.Search(ctx, ext.PackageName, query, p, toProtoFilterNodes(filters))
	if err != nil {
		return nil, err
	}
	return r.toSearchResponse(ctx, ext, resp)
}

func (r *queryResolver) FilterOptions(ctx context.Context, extensionID string) ([]model.FilterNode, error) {
	ext, err := r.resolveExtension(ctx, extensionID)
	if err != nil {
		return nil, err
	}
	c, err := r.Sc.Ensure(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := c.GetFilterList(ctx, ext.PackageName)
	if err != nil {
		return nil, err
	}
	return toFilterNodes(resp.Filters), nil
}

func (r *queryResolver) PopularManga(ctx context.Context, extensionID string, page *int32) (*model.SearchResponse, error) {
	p := int32(1)
	if page != nil && *page >= 1 {
		p = *page
	}
	ext, err := r.resolveExtension(ctx, extensionID)
	if err != nil {
		return nil, err
	}
	c, err := r.Sc.Ensure(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := c.GetPopularManga(ctx, ext.PackageName, p)
	if err != nil {
		return nil, err
	}
	return r.toSearchResponse(ctx, ext, resp)
}

func (r *queryResolver) LatestUpdates(ctx context.Context, extensionID string, page *int32) (*model.SearchResponse, error) {
	p := int32(1)
	if page != nil && *page >= 1 {
		p = *page
	}
	ext, err := r.resolveExtension(ctx, extensionID)
	if err != nil {
		return nil, err
	}
	c, err := r.Sc.Ensure(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := c.GetLatestUpdates(ctx, ext.PackageName, p)
	if err != nil {
		return nil, err
	}
	return r.toSearchResponse(ctx, ext, resp)
}

func (r *queryResolver) DownloadStatus(ctx context.Context, mediaID string, chapterID string) (*model.Download, error) {
	mID, err := parseID(mediaID)
	if err != nil {
		return nil, err
	}
	id, err := parseID(chapterID)
	if err != nil {
		return nil, err
	}
	if _, err := r.validateChapterMedia(ctx, mID, id); err != nil {
		return nil, err
	}
	d, err := r.Q.GetLatestDownloadForChapter(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return toDownloadFields(d.ID, d.ChapterID, mID, d.Status, d.Progress, d.DownloadedBytes, d.BytesPerSec, d.Error, d.CreatedAt, d.CompletedAt), nil
}

func (r *queryResolver) DownloadQueue(ctx context.Context) ([]*model.Download, error) {
	rows, err := r.Q.ListAllDownloads(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*model.Download, 0, len(rows))
	for _, d := range rows {
		out = append(out, toDownloadFields(d.ID, d.ChapterID, d.MediaID, d.Status, d.Progress, d.DownloadedBytes, d.BytesPerSec, d.Error, d.CreatedAt, d.CompletedAt))
	}
	return out, nil
}

func (r *queryResolver) DownloaderStatus(ctx context.Context) (*model.DownloaderStatus, error) {
	rows, err := r.Q.CountDownloadsByStatus(ctx)
	if err != nil {
		return nil, err
	}
	var queued, downloading, failed int32
	for _, row := range rows {
		switch row.Status {
		case "queued":
			queued = int32(row.Count)
		case "downloading":
			downloading = int32(row.Count)
		case "failed":
			failed = int32(row.Count)
		}
	}
	return &model.DownloaderStatus{
		IsRunning:        !r.Dm.IsPaused(),
		QueuedCount:      queued,
		DownloadingCount: downloading,
		FailedCount:      failed,
	}, nil
}

func (r *queryResolver) RecentChapters(ctx context.Context, since *time.Time, limit *int32) ([]*model.RecentChapter, error) {
	var sinceParam interface{}
	if since != nil {
		sinceParam = since.Unix()
	}
	lim := int64(50)
	if limit != nil {
		lim = int64(*limit)
	}
	rows, err := r.Q.ListRecentChapters(ctx, sqlcgen.ListRecentChaptersParams{Column1: sinceParam, Limit: lim})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []*model.RecentChapter{}, nil
	}

	mediaIDs := make([]int64, 0, len(rows))
	seen := map[int64]bool{}
	for _, row := range rows {
		if !seen[row.MediaID] {
			seen[row.MediaID] = true
			mediaIDs = append(mediaIDs, row.MediaID)
		}
	}
	mediaRows, err := r.Q.ListMediaByIDs(ctx, mediaIDs)
	if err != nil {
		return nil, err
	}
	mediaByID := make(map[int64]sqlcgen.Medium, len(mediaRows))
	for _, m := range mediaRows {
		mediaByID[m.ID] = m
	}

	out := make([]*model.RecentChapter, 0, len(rows))
	for _, row := range rows {
		m, ok := mediaByID[row.MediaID]
		if !ok {
			continue
		}
		mediaID := strconv.FormatInt(row.MediaID, 10)
		out = append(out, &model.RecentChapter{
			Chapter: &model.Chapter{
				ID:          strconv.FormatInt(row.ID, 10),
				MediaID:     mediaID,
				ExternalID:  row.ExternalID,
				Title:       nullStringPtr(row.Title),
				Number:      nullFloat64Ptr(row.Number),
				SourceOrder: nullInt64Int32Ptr(row.SourceOrder),
				UploadedAt:  epochToTimePtr(row.UploadedAt),
			},
			Media: toMedia(m, r.MediaDir),
		})
	}
	return out, nil
}

func (r *queryResolver) ChapterUpdates(ctx context.Context, since *time.Time, limit *int32) ([]*model.RecentChapter, error) {
	var sinceParam interface{}
	if since != nil {

		sinceParam = since.UTC().Format("2006-01-02 15:04:05")
	}
	lim := int64(50)
	if limit != nil {
		lim = int64(*limit)
	}
	rows, err := r.Q.ListChapterUpdates(ctx, sqlcgen.ListChapterUpdatesParams{Column1: sinceParam, Limit: lim})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []*model.RecentChapter{}, nil
	}

	mediaIDs := make([]int64, 0, len(rows))
	seen := map[int64]bool{}
	for _, row := range rows {
		if !seen[row.MediaID] {
			seen[row.MediaID] = true
			mediaIDs = append(mediaIDs, row.MediaID)
		}
	}
	mediaRows, err := r.Q.ListMediaByIDs(ctx, mediaIDs)
	if err != nil {
		return nil, err
	}
	mediaByID := make(map[int64]sqlcgen.Medium, len(mediaRows))
	for _, m := range mediaRows {
		mediaByID[m.ID] = m
	}

	out := make([]*model.RecentChapter, 0, len(rows))
	for _, row := range rows {
		m, ok := mediaByID[row.MediaID]
		if !ok {
			continue
		}
		out = append(out, &model.RecentChapter{
			Chapter: &model.Chapter{
				ID:          strconv.FormatInt(row.ID, 10),
				MediaID:     strconv.FormatInt(row.MediaID, 10),
				ExternalID:  row.ExternalID,
				Title:       nullStringPtr(row.Title),
				Number:      nullFloat64Ptr(row.Number),
				SourceOrder: nullInt64Int32Ptr(row.SourceOrder),
				UploadedAt:  epochToTimePtr(row.UploadedAt),
			},
			Media: toMedia(m, r.MediaDir),
		})
	}
	return out, nil
}

func (r *queryResolver) LibraryUpdateStatus(ctx context.Context) (*model.LibraryUpdateStatus, error) {
	p := r.Sy.LibraryUpdateStatus()
	out := &model.LibraryUpdateStatus{
		Running:         p.Running,
		Total:           int32(p.Total),
		Done:            int32(p.Done),
		NewChapterCount: int32(p.NewChapters),
		FailedTitles:    p.FailedTitles,
	}
	if out.FailedTitles == nil {
		out.FailedTitles = []string{}
	}
	if p.CurrentTitle != "" {
		out.CurrentTitle = &p.CurrentTitle
	}
	if !p.StartedAt.IsZero() {
		t := p.StartedAt
		out.StartedAt = &t
	}
	if !p.FinishedAt.IsZero() {
		t := p.FinishedAt
		out.FinishedAt = &t
	}
	return out, nil
}

func (r *queryResolver) StorageInfo(ctx context.Context) (*model.StorageInfo, error) {
	used, err := dirSize(r.MediaDir)
	if err != nil {
		return nil, fmt.Errorf("computing media dir size: %w", err)
	}
	total, free, err := diskStats(r.MediaDir)
	if err != nil {
		return nil, fmt.Errorf("computing disk stats: %w", err)
	}
	return &model.StorageInfo{
		UsedBytes:  float64(used),
		TotalBytes: float64(total),
		FreeBytes:  float64(free),
	}, nil
}

func (r *mediaResolver) TrackLinks(ctx context.Context, obj *model.Media) ([]*model.TrackLink, error) {
	mid, err := parseID(obj.ID)
	if err != nil {
		return nil, err
	}
	var links []sqlcgen.TrackerLink
	if l := LoadersFromContext(ctx); l != nil {
		links, err = l.TrackLinksByMedia.Load(ctx, mid)
	} else {
		links, err = r.Tk.LinksByMedia(ctx, mid)
	}
	if err != nil {
		return nil, err
	}
	out := make([]*model.TrackLink, 0, len(links))
	for _, l := range links {
		out = append(out, toTrackLink(l, r.Tk.KeyForAccountID(ctx, l.TrackerAccountID)))
	}
	return out, nil
}

func (r *queryResolver) Trackers(ctx context.Context) ([]*model.Tracker, error) {
	infos, err := r.Tk.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*model.Tracker, 0, len(infos))
	for _, i := range infos {
		out = append(out, toTracker(i))
	}
	return out, nil
}

func (r *queryResolver) TrackSearch(ctx context.Context, trackerKey string, query string, contentType *model.ContentType) ([]*model.TrackSearchResult, error) {
	res, err := r.Tk.Search(ctx, trackerKey, query, contentTypeToString(contentType))
	if err != nil {
		return nil, err
	}
	out := make([]*model.TrackSearchResult, 0, len(res))
	for _, s := range res {
		out = append(out, toTrackSearchResult(s))
	}
	return out, nil
}

func (r *mutationResolver) TrackerLogin(ctx context.Context, trackerKey string, token string) (*model.Tracker, error) {
	info, err := r.Tk.Login(ctx, trackerKey, token)
	if err != nil {
		return nil, err
	}
	return toTracker(info), nil
}

func (r *mutationResolver) TrackerLogout(ctx context.Context, trackerKey string) (bool, error) {
	if err := r.Tk.Logout(ctx, trackerKey); err != nil {
		return false, err
	}
	return true, nil
}

func (r *mutationResolver) BindTrack(ctx context.Context, mediaID string, trackerKey string, remoteID string) (*model.TrackLink, error) {
	mid, err := parseID(mediaID)
	if err != nil {
		return nil, err
	}
	link, err := r.Tk.Bind(ctx, trackerKey, mid, remoteID)
	if err != nil {
		return nil, err
	}
	return toTrackLink(link, trackerKey), nil
}

func (r *mutationResolver) UpdateTrack(ctx context.Context, linkID string, status *int32, score *float64, lastChapterRead *float64) (*model.TrackLink, error) {
	lid, err := parseID(linkID)
	if err != nil {
		return nil, err
	}
	var statusInt *int
	if status != nil {
		v := int(*status)
		statusInt = &v
	}
	link, err := r.Tk.Update(ctx, lid, statusInt, score, lastChapterRead)
	if err != nil {
		return nil, err
	}
	return toTrackLink(link, r.Tk.KeyForAccountID(ctx, link.TrackerAccountID)), nil
}

func (r *mutationResolver) UnbindTrack(ctx context.Context, linkID string) (bool, error) {
	lid, err := parseID(linkID)
	if err != nil {
		return false, err
	}
	if err := r.Tk.Unbind(ctx, lid); err != nil {
		return false, err
	}
	return true, nil
}

func (r *mutationResolver) ResyncTrack(ctx context.Context, linkID string) (*model.TrackLink, error) {
	lid, err := parseID(linkID)
	if err != nil {
		return nil, err
	}
	link, err := r.Tk.GetLink(ctx, lid)
	if err != nil {
		return nil, err
	}
	r.Tk.SyncMediaProgress(ctx, link.MediaID)
	fresh, err := r.Tk.GetLink(ctx, lid)
	if err != nil {
		return nil, err
	}
	return toTrackLink(fresh, r.Tk.KeyForAccountID(ctx, fresh.TrackerAccountID)), nil
}

func (r *mutationResolver) PullTracker(ctx context.Context, mediaID string) ([]*model.TrackLink, error) {
	mid, err := parseID(mediaID)
	if err != nil {
		return nil, err
	}
	r.Tk.SyncMediaProgress(ctx, mid)
	links, err := r.Tk.LinksByMedia(ctx, mid)
	if err != nil {
		return nil, err
	}
	out := make([]*model.TrackLink, 0, len(links))
	for _, l := range links {
		out = append(out, toTrackLink(l, r.Tk.KeyForAccountID(ctx, l.TrackerAccountID)))
	}
	return out, nil
}

func (r *queryResolver) SkipTimestamps(ctx context.Context, chapterID string, episodeLengthMs *int32) ([]*model.SkipMarker, error) {
	cid, err := parseID(chapterID)
	if err != nil {
		return nil, err
	}
	ch, err := r.Q.GetChapter(ctx, cid)
	if err != nil {
		return nil, err
	}
	media, err := r.Q.GetMedia(ctx, ch.MediaID)
	if err != nil || media.ContentType != "anime" || !ch.Number.Valid || ch.Number.Float64 <= 0 {
		return []*model.SkipMarker{}, nil
	}
	malID, err := r.Md.MalID(ctx, ch.MediaID)
	if err != nil || malID <= 0 {
		return []*model.SkipMarker{}, nil
	}
	var lengthSec float64
	if episodeLengthMs != nil && *episodeLengthMs > 0 {
		lengthSec = float64(*episodeLengthMs) / 1000
	}
	markers, err := aniskip.Fetch(ctx, malID, int(ch.Number.Float64), lengthSec)
	if err != nil {
		return []*model.SkipMarker{}, nil
	}
	out := make([]*model.SkipMarker, 0, len(markers))
	for _, m := range markers {
		out = append(out, &model.SkipMarker{
			Type:    m.Type,
			Name:    m.Name,
			StartMs: int32(m.StartMs),
			EndMs:   int32(m.EndMs),
		})
	}
	return out, nil
}

func (r *mediaResolver) Metadata(ctx context.Context, obj *model.Media) (*model.MetadataMatch, error) {
	id, err := parseID(obj.ID)
	if err != nil {
		return nil, err
	}
	var link *sqlcgen.MetadataLink
	if l := LoadersFromContext(ctx); l != nil {
		link, err = l.MetadataLinkByMedia.Load(ctx, id)
	} else {
		link, err = r.Md.Link(ctx, id)
	}
	if err != nil || link == nil {
		return nil, err
	}
	return toMetadataMatch(*link), nil
}

func (r *queryResolver) SearchMetadata(ctx context.Context, query string, contentType model.ContentType, provider *string) ([]*model.MetadataCandidate, error) {
	prov := metadata.DefaultProvider
	if provider != nil && *provider != "" {
		prov = *provider
	}
	ct := metadata.ContentType(contentTypeToString(&contentType))
	cands, err := r.Md.Search(ctx, prov, query, ct)
	if err != nil {
		return nil, err
	}
	out := make([]*model.MetadataCandidate, 0, len(cands))
	for _, c := range cands {
		out = append(out, toMetadataCandidate(c))
	}
	return out, nil
}

func (r *mutationResolver) ApplyMetadataMatch(ctx context.Context, mediaID string, providerID string, provider *string) (*model.Media, error) {
	id, err := parseID(mediaID)
	if err != nil {
		return nil, err
	}
	prov := metadata.DefaultProvider
	if provider != nil && *provider != "" {
		prov = *provider
	}
	updated, err := r.Md.Apply(ctx, id, prov, providerID)
	if err != nil {
		return nil, err
	}
	return toMedia(updated, r.MediaDir), nil
}

func (r *mutationResolver) UnlinkMetadata(ctx context.Context, mediaID string) (bool, error) {
	id, err := parseID(mediaID)
	if err != nil {
		return false, err
	}
	if err := r.Md.Unlink(ctx, id); err != nil {
		return false, err
	}
	return true, nil
}

func (r *mutationResolver) RefreshMetadataMatch(ctx context.Context, mediaID string) (*model.Media, error) {
	id, err := parseID(mediaID)
	if err != nil {
		return nil, err
	}
	updated, err := r.Md.Refresh(ctx, id)
	if err != nil {
		return nil, err
	}

	r.Tk.SyncMediaProgress(ctx, id)
	if m, mErr := r.Q.GetMedia(ctx, id); mErr == nil {
		updated = m
	}
	return toMedia(updated, r.MediaDir), nil
}

func (r *Resolver) Chapter() ChapterResolver   { return &chapterResolver{r} }
func (r *Resolver) Download() DownloadResolver { return &downloadResolver{r} }
func (r *Resolver) Media() MediaResolver       { return &mediaResolver{r} }
func (r *Resolver) Mutation() MutationResolver { return &mutationResolver{r} }
func (r *Resolver) Query() QueryResolver       { return &queryResolver{r} }

type (
	chapterResolver  struct{ *Resolver }
	downloadResolver struct{ *Resolver }
	mediaResolver    struct{ *Resolver }
	mutationResolver struct{ *Resolver }
	queryResolver    struct{ *Resolver }
)
