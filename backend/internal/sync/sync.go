package sync

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"

	"tsunagu/backend/internal/db/sqlcgen"
	"tsunagu/backend/internal/repository"
	"tsunagu/backend/internal/sandbox"
)

type Syncer struct {
	db       *sql.DB
	q        *sqlcgen.Queries
	cacheDir string

	installMu sync.Mutex
	locks     map[string]*sync.Mutex
}

func New(db *sql.DB, q *sqlcgen.Queries, cacheDir string) *Syncer {
	return &Syncer{db: db, q: q, cacheDir: cacheDir, locks: make(map[string]*sync.Mutex)}
}

func (s *Syncer) lockFor(packageName string) *sync.Mutex {
	s.installMu.Lock()
	defer s.installMu.Unlock()
	l, ok := s.locks[packageName]
	if !ok {
		l = &sync.Mutex{}
		s.locks[packageName] = l
	}
	return l
}

func (s *Syncer) AddRepository(ctx context.Context, indexURL string, name string) (sqlcgen.Repository, error) {
	parsed, err := repository.FetchIndex(indexURL)
	if err != nil {
		return sqlcgen.Repository{}, err
	}

	repoContentType := "manga"
	if len(parsed) > 0 && parsed[0].ContentType != "" {
		repoContentType = parsed[0].ContentType
	}

	if name == "" {
		name = repository.DeriveRepoName(indexURL)
	}

	repo, err := s.q.GetRepositoryByURL(ctx, indexURL)
	if err != nil {
		repo, err = s.q.CreateRepository(ctx, sqlcgen.CreateRepositoryParams{
			IndexUrl:    indexURL,
			Name:        nullString(name),
			ContentType: repoContentType,
		})
		if err != nil {
			return sqlcgen.Repository{}, fmt.Errorf("create repository: %w", err)
		}
	}

	for _, ext := range parsed {
		contentType := ext.ContentType
		if contentType == "" {
			contentType = repository.ClassifyContentType(ext.PackageName)
		}
		if _, err := s.q.UpsertExtension(ctx, sqlcgen.UpsertExtensionParams{
			RepositoryID: repo.ID,
			PackageName:  ext.PackageName,
			Name:         ext.Name,
			Version:      ext.VersionName,
			ContentType:  contentType,
			Lang:         ext.Lang,
			IconUrl:      nullString(ext.IconURL),
			ApkUrl:       ext.ApkURL,
			JarUrl:       nullString(ext.JarURL),
		}); err != nil {
			return sqlcgen.Repository{}, fmt.Errorf("upsert extension %s: %w", ext.PackageName, err)
		}
	}

	if err := s.q.TouchRepositorySync(ctx, repo.ID); err != nil {
		return sqlcgen.Repository{}, fmt.Errorf("touch repository sync: %w", err)
	}
	return s.q.GetRepository(ctx, repo.ID)
}

func (s *Syncer) ListRepositories(ctx context.Context) ([]sqlcgen.Repository, error) {
	return s.q.ListRepositories(ctx)
}

func (s *Syncer) RenameRepository(ctx context.Context, id int64, name string) (sqlcgen.Repository, error) {
	return s.q.UpdateRepositoryName(ctx, sqlcgen.UpdateRepositoryNameParams{
		Name: nullString(name),
		ID:   id,
	})
}

func (s *Syncer) DeleteRepository(ctx context.Context, id int64) error {
	exts, err := s.q.ListExtensionsByRepository(ctx, id)
	if err != nil {
		return fmt.Errorf("list extensions for repository %d: %w", id, err)
	}

	for _, ext := range exts {
		if err := s.q.MarkLibraryEntriesExtensionRemoved(ctx, sql.NullInt64{Int64: ext.ID, Valid: true}); err != nil {
			return fmt.Errorf("mark library entries for extension %s: %w", ext.PackageName, err)
		}
	}
	return s.q.DeleteRepository(ctx, id)
}

func (s *Syncer) ListAvailableExtensions(ctx context.Context, repositoryID int64) ([]sqlcgen.Extension, error) {
	return s.q.ListExtensionsByRepository(ctx, repositoryID)
}

func extensionDownloadTarget(ext sqlcgen.Extension) (url string, fileExt string, err error) {
	if ext.ContentType == "novel" {
		if ext.JarUrl.Valid && ext.JarUrl.String != "" {
			return ext.JarUrl.String, "js", nil
		}
		return "", "", fmt.Errorf("novel extension %s has no source url", ext.PackageName)
	}
	if ext.JarUrl.Valid && ext.JarUrl.String != "" {
		return ext.JarUrl.String, "jar", nil
	}
	if ext.ApkUrl != "" {
		return ext.ApkUrl, "apk", nil
	}
	return "", "", fmt.Errorf("extension %s has no jar_url or apk_url", ext.PackageName)
}

func (s *Syncer) InstallExtension(ctx context.Context, packageName string) (sqlcgen.Extension, error) {
	if !repository.IsValidPackageName(packageName) {
		return sqlcgen.Extension{}, fmt.Errorf("invalid package name: %q", packageName)
	}

	lock := s.lockFor(packageName)
	lock.Lock()
	defer lock.Unlock()

	ext, err := s.q.GetExtensionByPackageName(ctx, packageName)
	if err != nil {
		return sqlcgen.Extension{}, fmt.Errorf("lookup extension %s: %w", packageName, err)
	}

	downloadURL, fileExt, err := extensionDownloadTarget(ext)
	if err != nil {
		return sqlcgen.Extension{}, err
	}

	path, err := repository.DownloadExtensionFile(s.cacheDir, ext.PackageName, ext.Version, downloadURL, fileExt)
	if err != nil {
		return sqlcgen.Extension{}, err
	}

	return s.q.MarkExtensionInstalled(ctx, sqlcgen.MarkExtensionInstalledParams{
		JarPath: nullString(path),
		ID:      ext.ID,
	})
}

func (s *Syncer) UninstallExtension(ctx context.Context, packageName string) (sqlcgen.Extension, error) {
	if !repository.IsValidPackageName(packageName) {
		return sqlcgen.Extension{}, fmt.Errorf("invalid package name: %q", packageName)
	}

	lock := s.lockFor(packageName)
	lock.Lock()
	defer lock.Unlock()

	ext, err := s.q.GetExtensionByPackageName(ctx, packageName)
	if err != nil {
		return sqlcgen.Extension{}, fmt.Errorf("lookup extension %s: %w", packageName, err)
	}
	updated, err := s.q.MarkExtensionUninstalled(ctx, ext.ID)
	if err != nil {
		return sqlcgen.Extension{}, err
	}
	if ext.JarPath.Valid {
		_ = os.Remove(ext.JarPath.String)
	}
	return updated, nil
}

func (s *Syncer) ListInstalledExtensions(ctx context.Context) ([]sqlcgen.Extension, error) {
	return s.q.ListInstalledExtensions(ctx)
}

func (s *Syncer) CheckForUpdates(ctx context.Context) ([]sqlcgen.Extension, error) {
	return s.q.ListExtensionsNeedingUpdate(ctx)
}

func (s *Syncer) UpdateExtension(ctx context.Context, packageName string) (sqlcgen.Extension, error) {
	if !repository.IsValidPackageName(packageName) {
		return sqlcgen.Extension{}, fmt.Errorf("invalid package name: %q", packageName)
	}

	lock := s.lockFor(packageName)
	lock.Lock()
	defer lock.Unlock()

	ext, err := s.q.GetExtensionByPackageName(ctx, packageName)
	if err != nil {
		return sqlcgen.Extension{}, fmt.Errorf("lookup extension %s: %w", packageName, err)
	}

	downloadURL, fileExt, err := extensionDownloadTarget(ext)
	if err != nil {
		return sqlcgen.Extension{}, err
	}

	oldPath := ext.JarPath
	path, err := repository.DownloadExtensionFile(s.cacheDir, ext.PackageName, ext.Version, downloadURL, fileExt)
	if err != nil {
		return sqlcgen.Extension{}, err
	}

	updated, err := s.q.MarkExtensionUpdated(ctx, sqlcgen.MarkExtensionUpdatedParams{
		JarPath: nullString(path),
		ID:      ext.ID,
	})
	if err != nil {
		return sqlcgen.Extension{}, err
	}

	if oldPath.Valid && oldPath.String != path {
		_ = os.Remove(oldPath.String)
	}
	return updated, nil
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func (s *Syncer) SetReadingStatus(ctx context.Context, libraryEntryID int64, systemKey string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin reading status tx: %w", err)
	}
	defer tx.Rollback()

	qtx := s.q.WithTx(tx)

	folder, err := qtx.GetFolderBySystemKey(ctx, sql.NullString{String: systemKey, Valid: true})
	if err != nil {
		return fmt.Errorf("get folder %q: %w", systemKey, err)
	}
	if folder.Kind != "reading_status" {
		return fmt.Errorf("folder %q is not a reading-status folder (kind=%s)", systemKey, folder.Kind)
	}

	if err := qtx.RemoveEntryFromFoldersByKind(ctx, sqlcgen.RemoveEntryFromFoldersByKindParams{
		LibraryEntryID: libraryEntryID,
		Kind:           "reading_status",
	}); err != nil {
		return fmt.Errorf("remove existing reading status: %w", err)
	}

	if err := qtx.AddEntryToFolder(ctx, sqlcgen.AddEntryToFolderParams{
		LibraryEntryID: libraryEntryID,
		FolderID:       folder.ID,
	}); err != nil {
		return fmt.Errorf("add to folder %q: %w", systemKey, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit reading status tx: %w", err)
	}
	return nil
}

func (s *Syncer) AddToLibrary(ctx context.Context, c *sandbox.Client, packageName, sourceEntryID string) (sqlcgen.LibraryEntry, error) {
	ext, err := s.q.GetExtensionByPackageName(ctx, packageName)
	if err != nil {
		return sqlcgen.LibraryEntry{}, fmt.Errorf("lookup extension %s: %w", packageName, err)
	}

	details, err := c.GetDetails(ctx, packageName, sourceEntryID)
	if err != nil {
		return sqlcgen.LibraryEntry{}, fmt.Errorf("get details for %s/%s: %w", packageName, sourceEntryID, err)
	}

	entry, err := s.q.CreateLibraryEntry(ctx, sqlcgen.CreateLibraryEntryParams{
		ExtensionID:   sql.NullInt64{Int64: ext.ID, Valid: true},
		ExtensionName: ext.Name,
		ExternalID:    details.SourceEntryId,
		ContentType:   ext.ContentType,
		Title:         details.Title,
		CoverPath:     sql.NullString{String: details.CoverUrl, Valid: details.CoverUrl != ""},
		Description:   sql.NullString{String: details.Description, Valid: details.Description != ""},
		Status:        sql.NullString{String: details.Status, Valid: details.Status != ""},
	})
	if err != nil {
		return sqlcgen.LibraryEntry{}, fmt.Errorf("create library entry: %w", err)
	}

	return entry, nil
}

func (s *Syncer) ListLibraryEntries(ctx context.Context, contentType string) ([]sqlcgen.LibraryEntry, error) {
	if contentType == "" {
		return s.q.ListLibraryEntries(ctx)
	}
	return s.q.ListLibraryEntriesByContentType(ctx, contentType)
}

func (s *Syncer) SyncChapters(ctx context.Context, c *sandbox.Client, libraryEntryID int64) ([]sqlcgen.Chapter, error) {
	entry, err := s.q.GetLibraryEntry(ctx, libraryEntryID)
	if err != nil {
		return nil, fmt.Errorf("get library entry %d: %w", libraryEntryID, err)
	}
	if !entry.ExtensionID.Valid {
		return nil, fmt.Errorf("library entry %d has no extension (extension was removed)", libraryEntryID)
	}
	ext, err := s.q.GetExtension(ctx, entry.ExtensionID.Int64)
	if err != nil {
		return nil, fmt.Errorf("get extension %d: %w", entry.ExtensionID.Int64, err)
	}

	type summary struct {
		sourceID string
		name     string
		number   float64
		uploadTS int64
	}
	var summaries []summary

	switch ext.ContentType {
	case "anime":
		list, err := c.GetEpisodes(ctx, ext.PackageName, entry.ExternalID)
		if err != nil {
			return nil, fmt.Errorf("get episodes for %s/%s: %w", ext.PackageName, entry.ExternalID, err)
		}
		for _, e := range list.Episodes {
			summaries = append(summaries, summary{e.SourceEpisodeId, e.Name, e.Number, e.UploadTimestamp})
		}
	default: // manga, novel
		list, err := c.GetChapters(ctx, ext.PackageName, entry.ExternalID)
		if err != nil {
			return nil, fmt.Errorf("get chapters for %s/%s: %w", ext.PackageName, entry.ExternalID, err)
		}
		for _, ch := range list.Chapters {
			summaries = append(summaries, summary{ch.SourceChapterId, ch.Name, ch.Number, ch.UploadTimestamp})
		}
	}

	chapters := make([]sqlcgen.Chapter, 0, len(summaries))
	for _, sm := range summaries {
		var uploadedAt sql.NullInt64
		if sm.uploadTS > 0 {
			uploadedAt = sql.NullInt64{Int64: sm.uploadTS / 1000, Valid: true}
		}
		chapter, err := s.q.CreateChapter(ctx, sqlcgen.CreateChapterParams{
			LibraryEntryID: libraryEntryID,
			ExternalID:     sm.sourceID,
			Title:          sql.NullString{String: sm.name, Valid: sm.name != ""},
			Number:         sql.NullFloat64{Float64: sm.number, Valid: true},
			UploadedAt:     uploadedAt,
		})
		if err != nil {
			return nil, fmt.Errorf("upsert chapter %s: %w", sm.sourceID, err)
		}
		chapters = append(chapters, chapter)
	}

	return chapters, nil
}

func (s *Syncer) RecordProgress(ctx context.Context, libraryEntryID, chapterID int64, progress float64, completed bool, positionSeconds, durationSeconds *float64) (sqlcgen.ReadingProgress, error) {
	params := sqlcgen.UpsertReadingProgressParams{
		LibraryEntryID: libraryEntryID,
		ChapterID:      chapterID,
		Progress:       progress,
		Completed:      completed,
	}
	if positionSeconds != nil {
		params.PositionSeconds = sql.NullFloat64{Float64: *positionSeconds, Valid: true}
	}
	if durationSeconds != nil {
		params.DurationSeconds = sql.NullFloat64{Float64: *durationSeconds, Valid: true}
	}
	return s.q.UpsertReadingProgress(ctx, params)
}

func (s *Syncer) MarkChapterRead(ctx context.Context, libraryEntryID, chapterID int64) (sqlcgen.ReadingProgress, error) {
	return s.RecordProgress(ctx, libraryEntryID, chapterID, 1.0, true, nil, nil)
}

func (s *Syncer) ListReadingProgress(ctx context.Context, libraryEntryID int64) ([]sqlcgen.ReadingProgress, error) {
	return s.q.ListReadingProgressByLibraryEntry(ctx, libraryEntryID)
}
