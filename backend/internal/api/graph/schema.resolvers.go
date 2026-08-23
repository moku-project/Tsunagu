package graph

import (
	"context"
	"database/sql"
	"fmt"
	"tsunagu/backend/internal/api/graph/model"
	"tsunagu/backend/internal/sandbox"
	sandboxv1 "tsunagu/backend/internal/sandbox/gen/sandbox/v1"
)

func (r *chapterResolver) Download(ctx context.Context, obj *model.Chapter) (*model.Download, error) {
	chapterID, err := parseID(obj.ID)
	if err != nil {
		return nil, err
	}
	downloads, err := r.Q.ListDownloadsByStatus(ctx, "done")
	if err != nil {
		return nil, err
	}
	for _, d := range downloads {
		if d.ChapterID == chapterID {
			return toDownload(d), nil
		}
	}

	for _, status := range []string{"queued", "downloading", "failed"} {
		rows, err := r.Q.ListDownloadsByStatus(ctx, status)
		if err != nil {
			return nil, err
		}
		for _, d := range rows {
			if d.ChapterID == chapterID {
				return toDownload(d), nil
			}
		}
	}
	return nil, nil
}

func (r *libraryEntryResolver) Chapters(ctx context.Context, obj *model.LibraryEntry) ([]*model.Chapter, error) {
	id, err := parseID(obj.ID)
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

func (r *libraryEntryResolver) ReadingProgress(ctx context.Context, obj *model.LibraryEntry) ([]*model.ReadingProgress, error) {
	id, err := parseID(obj.ID)
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
	if err := r.Sy.DeleteRepository(ctx, id); err != nil {
		return false, err
	}
	return true, nil
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
	if _, err := c.LoadExtensions(ctx, []*sandboxv1.ExtensionToLoad{{
		ExtensionId: ext.PackageName,
		JarPath:     ext.JarPath.String,
		ContentType: sandbox.ContentTypeToProto(ext.ContentType),
		Lang:        ext.Lang,
	}}); err != nil {
		return nil, err
	}
	return toExtension(ext), nil
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
	return toExtension(ext), nil
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
	if _, err := c.LoadExtensions(ctx, []*sandboxv1.ExtensionToLoad{{
		ExtensionId: ext.PackageName,
		JarPath:     ext.JarPath.String,
		ContentType: sandbox.ContentTypeToProto(ext.ContentType),
		Lang:        ext.Lang,
	}}); err != nil {
		return nil, err
	}
	return toExtension(ext), nil
}

func (r *mutationResolver) AddToLibrary(ctx context.Context, extensionID string, sourceEntryID string) (*model.LibraryEntry, error) {
	c, err := r.Sc.Ensure(ctx)
	if err != nil {
		return nil, err
	}
	entry, err := r.Sy.AddToLibrary(ctx, c, extensionID, sourceEntryID)
	if err != nil {
		return nil, err
	}
	return toLibraryEntry(entry), nil
}

func (r *mutationResolver) SyncChapters(ctx context.Context, libraryEntryID string) ([]*model.Chapter, error) {
	id, err := parseID(libraryEntryID)
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

func (r *mutationResolver) UpdateReadingProgress(ctx context.Context, libraryEntryID string, chapterID string, progress float64, completed *bool, positionSeconds *float64, durationSeconds *float64) (*model.ReadingProgress, error) {
	entryID, err := parseID(libraryEntryID)
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
	prog, err := r.Sy.RecordProgress(ctx, entryID, chID, progress, comp, positionSeconds, durationSeconds)
	if err != nil {
		return nil, err
	}
	return toReadingProgress(prog), nil
}

func (r *mutationResolver) MarkChapterRead(ctx context.Context, libraryEntryID string, chapterID string) (*model.ReadingProgress, error) {
	entryID, err := parseID(libraryEntryID)
	if err != nil {
		return nil, err
	}
	chID, err := parseID(chapterID)
	if err != nil {
		return nil, err
	}
	prog, err := r.Sy.MarkChapterRead(ctx, entryID, chID)
	if err != nil {
		return nil, err
	}
	return toReadingProgress(prog), nil
}

func (r *mutationResolver) EnqueueDownload(ctx context.Context, chapterID string) (*model.Download, error) {
	id, err := parseID(chapterID)
	if err != nil {
		return nil, err
	}
	d, err := r.Q.EnqueueDownload(ctx, id)
	if err != nil {
		return nil, err
	}
	r.Dm.Wake()
	return toDownload(d), nil
}

func (r *mutationResolver) RetryDownload(ctx context.Context, chapterID string) (*model.Download, error) {
	id, err := parseID(chapterID)
	if err != nil {
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
	return toDownload(d), nil
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
		out = append(out, toExtension(ext))
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
		out = append(out, toExtension(ext))
	}
	return out, nil
}

func (r *queryResolver) Library(ctx context.Context, contentType *model.ContentType) ([]*model.LibraryEntry, error) {
	entries, err := r.Sy.ListLibraryEntries(ctx, contentTypeToString(contentType))
	if err != nil {
		return nil, err
	}
	out := make([]*model.LibraryEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, toLibraryEntry(e))
	}
	return out, nil
}

func (r *queryResolver) LibraryEntry(ctx context.Context, id string) (*model.LibraryEntry, error) {
	entryID, err := parseID(id)
	if err != nil {
		return nil, err
	}
	entry, err := r.Sy.GetLibraryEntry(ctx, entryID)
	if err != nil {
		return nil, err
	}
	return toLibraryEntry(entry), nil
}

func (r *queryResolver) ReadingProgress(ctx context.Context, libraryEntryID string) ([]*model.ReadingProgress, error) {
	id, err := parseID(libraryEntryID)
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

func (r *queryResolver) Search(ctx context.Context, extensionID string, query string, page *int32) (*model.SearchResponse, error) {
	p := int32(1)
	if page != nil {
		p = *page
	}
	c, err := r.Sc.Ensure(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := c.Search(ctx, extensionID, query, p)
	if err != nil {
		return nil, err
	}
	results := make([]*model.SearchResult, 0, len(resp.Results))
	for _, res := range resp.Results {
		var cover *string
		if res.CoverUrl != "" {
			cover = &res.CoverUrl
		}
		results = append(results, &model.SearchResult{
			SourceEntryID: res.SourceEntryId,
			Title:         res.Title,
			CoverURL:      cover,
		})
	}
	return &model.SearchResponse{
		Results:     results,
		HasNextPage: resp.HasNextPage,
	}, nil
}

func (r *queryResolver) SourceDetails(ctx context.Context, extensionID string, sourceEntryID string) (*model.SourceDetails, error) {
	c, err := r.Sc.Ensure(ctx)
	if err != nil {
		return nil, err
	}
	d, err := c.GetDetails(ctx, extensionID, sourceEntryID)
	if err != nil {
		return nil, err
	}
	var desc, cover, status *string
	if d.Description != "" {
		desc = &d.Description
	}
	if d.CoverUrl != "" {
		cover = &d.CoverUrl
	}
	if d.Status != "" {
		status = &d.Status
	}
	return &model.SourceDetails{
		SourceEntryID: sourceEntryID,
		Title:         d.Title,
		Description:   desc,
		CoverURL:      cover,
		Status:        status,
	}, nil
}

func (r *queryResolver) DownloadStatus(ctx context.Context, chapterID string) (*model.Download, error) {
	id, err := parseID(chapterID)
	if err != nil {
		return nil, err
	}
	for _, status := range []string{"queued", "downloading", "done", "failed"} {
		rows, err := r.Q.ListDownloadsByStatus(ctx, status)
		if err != nil {
			return nil, err
		}
		for _, d := range rows {
			if d.ChapterID == id {
				return toDownload(d), nil
			}
		}
	}
	return nil, nil
}

func (r *Resolver) Chapter() ChapterResolver { return &chapterResolver{r} }

func (r *Resolver) LibraryEntry() LibraryEntryResolver { return &libraryEntryResolver{r} }

func (r *Resolver) Mutation() MutationResolver { return &mutationResolver{r} }

func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

type (
	chapterResolver      struct{ *Resolver }
	libraryEntryResolver struct{ *Resolver }
	mutationResolver     struct{ *Resolver }
	queryResolver        struct{ *Resolver }
)
