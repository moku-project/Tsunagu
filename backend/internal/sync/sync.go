package sync

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"

	"tsunagu/backend/internal/db/sqlcgen"
	"tsunagu/backend/internal/repository"
)

type Syncer struct {
	q        *sqlcgen.Queries
	cacheDir string

	installMu sync.Mutex
	locks     map[string]*sync.Mutex
}

func New(q *sqlcgen.Queries, cacheDir string) *Syncer {
	return &Syncer{q: q, cacheDir: cacheDir, locks: make(map[string]*sync.Mutex)}
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
