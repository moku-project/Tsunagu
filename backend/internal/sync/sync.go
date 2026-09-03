package sync

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"tsunagu/backend/internal/chapternum"
	"tsunagu/backend/internal/db/sqlcgen"
	"tsunagu/backend/internal/image"
	"tsunagu/backend/internal/repository"
	"tsunagu/backend/internal/sandbox"
	sandboxv1 "tsunagu/backend/internal/sandbox/gen/sandbox/v1"
)

type Syncer struct {
	db       *sql.DB
	q        *sqlcgen.Queries
	cacheDir string
	mediaDir string

	installMu sync.Mutex
	locks     map[string]*sync.Mutex

	updateMu   sync.Mutex
	updateProg LibraryUpdateProgress

	enricher  Enricher
	recompute Recomputer
}

type Enricher interface {
	AutoEnrich(ctx context.Context, mediaID int64) error
}

type Recomputer interface {
	RecomputeMedia(ctx context.Context, mediaID int64) error
}

func (s *Syncer) SetEnricher(e Enricher)     { s.enricher = e }
func (s *Syncer) SetRecomputer(r Recomputer) { s.recompute = r }

func New(db *sql.DB, q *sqlcgen.Queries, cacheDir, mediaDir string) *Syncer {
	return &Syncer{db: db, q: q, cacheDir: cacheDir, mediaDir: mediaDir, locks: make(map[string]*sync.Mutex)}
}

type LibraryUpdateProgress struct {
	Running      bool
	Total        int
	Done         int
	CurrentTitle string
	NewChapters  int
	FailedTitles []string
	StartedAt    time.Time
	FinishedAt   time.Time
}

func (s *Syncer) LibraryUpdateStatus() LibraryUpdateProgress {
	s.updateMu.Lock()
	defer s.updateMu.Unlock()
	p := s.updateProg
	p.FailedTitles = append([]string(nil), s.updateProg.FailedTitles...)
	return p
}

func (s *Syncer) StartLibraryUpdate(sc *sandbox.SupervisedClient, folderID *int64) (bool, error) {
	s.updateMu.Lock()
	if s.updateProg.Running {
		s.updateMu.Unlock()
		return false, nil
	}

	var ids []int64
	var err error
	if folderID != nil {
		var rows []sqlcgen.Medium
		rows, err = s.q.ListMediaInFolder(context.Background(), *folderID)
		for _, m := range rows {
			ids = append(ids, m.ID)
		}
	} else {
		ids, err = s.q.ListUpdateTargetMediaIDs(context.Background())
	}
	if err != nil {
		s.updateMu.Unlock()
		return false, err
	}

	s.updateProg = LibraryUpdateProgress{
		Running:   true,
		Total:     len(ids),
		StartedAt: time.Now(),
	}
	s.updateMu.Unlock()

	go s.runLibraryUpdate(sc, ids)
	return true, nil
}

func (s *Syncer) runLibraryUpdate(sc *sandbox.SupervisedClient, ids []int64) {
	ctx := context.Background()
	defer func() {
		s.updateMu.Lock()
		s.updateProg.Running = false
		s.updateProg.CurrentTitle = ""
		s.updateProg.FinishedAt = time.Now()
		s.updateMu.Unlock()
	}()

	c, err := sc.Ensure(ctx)
	if err != nil {
		s.updateMu.Lock()
		s.updateProg.FailedTitles = append(s.updateProg.FailedTitles, "sandbox unavailable: "+err.Error())
		s.updateMu.Unlock()
		return
	}

	for _, id := range ids {
		m, err := s.q.GetMedia(ctx, id)
		if err != nil {
			continue
		}

		s.updateMu.Lock()
		s.updateProg.CurrentTitle = m.Title
		s.updateMu.Unlock()

		before, _ := s.q.ListChapterExternalIDsByMedia(ctx, id)
		known := make(map[string]struct{}, len(before))
		for _, e := range before {
			known[e] = struct{}{}
		}

		after, err := s.SyncChapters(ctx, c, id)
		newCount := 0
		if err != nil {
			s.updateMu.Lock()
			s.updateProg.FailedTitles = append(s.updateProg.FailedTitles, m.Title)
			s.updateProg.Done++
			s.updateMu.Unlock()
			continue
		}
		for _, ch := range after {
			if _, ok := known[ch.ExternalID]; !ok {
				newCount++
			}
		}

		s.updateMu.Lock()
		s.updateProg.NewChapters += newCount
		s.updateProg.Done++
		s.updateMu.Unlock()
	}
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
	if len(parsed) > 0 {
		if parsed[0].ContentType != "" {
			repoContentType = parsed[0].ContentType
		} else {
			repoContentType = repository.ClassifyContentType(parsed[0].PackageName)
		}
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

	if err := s.applyRepoIndex(ctx, repo.ID, parsed); err != nil {
		return sqlcgen.Repository{}, err
	}
	return s.q.GetRepository(ctx, repo.ID)
}

func (s *Syncer) applyRepoIndex(ctx context.Context, repoID int64, parsed []repository.ParsedExtension) error {
	for _, ext := range parsed {
		contentType := ext.ContentType
		if contentType == "" {
			contentType = repository.ClassifyContentType(ext.PackageName)
		}
		if _, err := s.q.UpsertExtension(ctx, sqlcgen.UpsertExtensionParams{
			RepositoryID: repoID,
			PackageName:  ext.PackageName,
			Name:         ext.Name,
			Version:      ext.VersionName,
			ContentType:  contentType,
			Lang:         ext.Lang,
			IconUrl:      nullString(ext.IconURL),
			ApkUrl:       ext.ApkURL,
			JarUrl:       nullString(ext.JarURL),
			IsNsfw:       ext.IsNsfw,
		}); err != nil {
			return fmt.Errorf("upsert extension %s: %w", ext.PackageName, err)
		}
	}
	if err := s.q.TouchRepositorySync(ctx, repoID); err != nil {
		return fmt.Errorf("touch repository sync: %w", err)
	}
	return nil
}

func (s *Syncer) ListRepositories(ctx context.Context) ([]sqlcgen.Repository, error) {
	return s.q.ListRepositories(ctx)
}

func (s *Syncer) SyncRepository(ctx context.Context, id int64) (sqlcgen.Repository, error) {
	repo, err := s.q.GetRepository(ctx, id)
	if err != nil {
		return sqlcgen.Repository{}, fmt.Errorf("get repository %d: %w", id, err)
	}
	if strings.HasPrefix(repo.IndexUrl, "local:") {
		return repo, nil
	}
	parsed, err := repository.FetchIndex(repo.IndexUrl)
	if err != nil {
		return sqlcgen.Repository{}, err
	}
	if err := s.applyRepoIndex(ctx, repo.ID, parsed); err != nil {
		return sqlcgen.Repository{}, err
	}
	return s.q.GetRepository(ctx, repo.ID)
}

func (s *Syncer) SyncAllRepositories(ctx context.Context) ([]sqlcgen.Repository, error) {
	repos, err := s.q.ListRepositories(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]sqlcgen.Repository, 0, len(repos))
	for _, repo := range repos {
		if strings.HasPrefix(repo.IndexUrl, "local:") {
			out = append(out, repo)
			continue
		}
		updated, err := s.SyncRepository(ctx, repo.ID)
		if err != nil {
			log.Printf("sync repository %d (%s): %v", repo.ID, repo.IndexUrl, err)
			out = append(out, repo)
			continue
		}
		out = append(out, updated)
	}
	return out, nil
}

const sideloadRepoURL = "local:sideload"

func (s *Syncer) getOrCreateSideloadRepo(ctx context.Context) (sqlcgen.Repository, error) {
	repo, err := s.q.GetRepositoryByURL(ctx, sideloadRepoURL)
	if err == nil {
		return repo, nil
	}
	return s.q.CreateRepository(ctx, sqlcgen.CreateRepositoryParams{
		IndexUrl:    sideloadRepoURL,
		Name:        sql.NullString{String: "Sideloaded", Valid: true},
		ContentType: "manga",
	})
}

func (s *Syncer) InstallExternalExtension(ctx context.Context, c *sandbox.Client, url string) (sqlcgen.Extension, error) {
	ext := "apk"
	switch {
	case strings.HasSuffix(url, ".jar"):
		ext = "jar"
	case strings.HasSuffix(url, ".js"):
		ext = "js"
	}

	tempDir := filepath.Join(s.cacheDir, "sideload-tmp")
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return sqlcgen.Extension{}, fmt.Errorf("creating temp dir: %w", err)
	}
	tempPath := filepath.Join(tempDir, fmt.Sprintf("sideload-%d.%s", time.Now().UnixNano(), ext))

	if err := downloadFile(url, tempPath); err != nil {
		return sqlcgen.Extension{}, fmt.Errorf("downloading %s: %w", url, err)
	}
	defer os.Remove(tempPath)

	meta, err := c.PeekExtension(ctx, tempPath)
	if err != nil {
		return sqlcgen.Extension{}, fmt.Errorf("inspecting extension: %w", err)
	}

	repo, err := s.getOrCreateSideloadRepo(ctx)
	if err != nil {
		return sqlcgen.Extension{}, fmt.Errorf("getting sideload repository: %w", err)
	}

	contentType := strings.ToLower(meta.ContentType.String())

	saved, err := s.q.UpsertExtension(ctx, sqlcgen.UpsertExtensionParams{
		RepositoryID: repo.ID,
		PackageName:  meta.PackageName,
		Name:         meta.Name,

		Version:     "unknown",
		ContentType: contentType,
		Lang:        meta.Lang,
	})
	if err != nil {
		return sqlcgen.Extension{}, fmt.Errorf("upsert extension: %w", err)
	}

	return s.q.MarkExtensionInstalled(ctx, sqlcgen.MarkExtensionInstalledParams{
		JarPath: sql.NullString{String: "", Valid: false},
		ID:      saved.ID,
	})
}

func downloadFile(url, destPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
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
		if err := s.q.MarkMediaExtensionRemoved(ctx, sql.NullInt64{Int64: ext.ID, Valid: true}); err != nil {
			return fmt.Errorf("mark media for extension %s: %w", ext.PackageName, err)
		}
	}
	return s.q.DeleteRepository(ctx, id)
}

func (s *Syncer) ListAvailableExtensions(ctx context.Context, repositoryID int64) ([]sqlcgen.Extension, error) {
	return s.q.ListExtensionsByRepository(ctx, repositoryID)
}

type ExtensionQuery struct {
	RepositoryID *int64
	ContentType  string
	Lang         string
	Search       string
	Installed    *bool
	Limit        int64
	Offset       int64
}

func (s *Syncer) QueryExtensions(ctx context.Context, q ExtensionQuery) ([]sqlcgen.Extension, int64, error) {
	if q.Limit <= 0 || q.Limit > 200 {
		q.Limit = 60
	}
	if q.Offset < 0 {
		q.Offset = 0
	}
	repoID := sql.NullInt64{}
	if q.RepositoryID != nil {
		repoID = sql.NullInt64{Int64: *q.RepositoryID, Valid: true}
	}
	ct := sql.NullString{}
	if q.ContentType != "" {
		ct = sql.NullString{String: q.ContentType, Valid: true}
	}
	lang := sql.NullString{}
	if q.Lang != "" {
		lang = sql.NullString{String: q.Lang, Valid: true}
	}
	installed := sql.NullBool{}
	if q.Installed != nil {
		installed = sql.NullBool{Bool: *q.Installed, Valid: true}
	}
	search := sql.NullString{}
	if s := strings.TrimSpace(q.Search); s != "" {
		search = sql.NullString{String: s, Valid: true}
	}

	total, err := s.q.CountExtensions(ctx, sqlcgen.CountExtensionsParams{
		RepositoryID: repoID,
		ContentType:  ct,
		Lang:         lang,
		Installed:    installed,
		Search:       search,
	})
	if err != nil {
		return nil, 0, err
	}
	items, err := s.q.QueryExtensions(ctx, sqlcgen.QueryExtensionsParams{
		RepositoryID: repoID,
		ContentType:  ct,
		Lang:         lang,
		Installed:    installed,
		Search:       search,
		Lim:          q.Limit,
		Off:          q.Offset,
	})
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *Syncer) ExtensionLanguages(ctx context.Context, q ExtensionQuery) ([]string, error) {
	repoID := sql.NullInt64{}
	if q.RepositoryID != nil {
		repoID = sql.NullInt64{Int64: *q.RepositoryID, Valid: true}
	}
	ct := sql.NullString{}
	if q.ContentType != "" {
		ct = sql.NullString{String: q.ContentType, Valid: true}
	}
	installed := sql.NullBool{}
	if q.Installed != nil {
		installed = sql.NullBool{Bool: *q.Installed, Valid: true}
	}
	search := sql.NullString{}
	if s := strings.TrimSpace(q.Search); s != "" {
		search = sql.NullString{String: s, Valid: true}
	}
	return s.q.ListExtensionLanguages(ctx, sqlcgen.ListExtensionLanguagesParams{
		RepositoryID: repoID,
		ContentType:  ct,
		Installed:    installed,
		Search:       search,
	})
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
	s.removeExtensionFiles(ext)
	return updated, nil
}

func (s *Syncer) removeExtensionFiles(ext sqlcgen.Extension) {
	if ext.JarPath.Valid && ext.JarPath.String != "" {
		if err := os.Remove(ext.JarPath.String); err != nil && !os.IsNotExist(err) {
			log.Printf("uninstall: removing %s: %v", ext.JarPath.String, err)
		}
	}
	for _, e := range []string{"jar", "apk", "js"} {
		matches, _ := filepath.Glob(filepath.Join(s.cacheDir, ext.PackageName+"-*."+e))
		also, _ := filepath.Glob(filepath.Join(s.cacheDir, ext.PackageName+"."+e))
		for _, p := range append(matches, also...) {
			if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
				log.Printf("uninstall: removing %s: %v", p, err)
			} else if err == nil {
				log.Printf("uninstall: removed cached %s", filepath.Base(p))
			}
		}
	}
}

func (s *Syncer) ListInstalledExtensions(ctx context.Context) ([]sqlcgen.Extension, error) {
	return s.q.ListInstalledExtensions(ctx)
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

	if err := qtx.RemoveMediaFromFoldersByKind(ctx, sqlcgen.RemoveMediaFromFoldersByKindParams{
		MediaID: libraryEntryID,
		Kind:    "reading_status",
	}); err != nil {
		return fmt.Errorf("remove existing reading status: %w", err)
	}

	if err := qtx.AddMediaToFolder(ctx, sqlcgen.AddMediaToFolderParams{
		MediaID:  libraryEntryID,
		FolderID: folder.ID,
	}); err != nil {
		return fmt.Errorf("add to folder %q: %w", systemKey, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit reading status tx: %w", err)
	}
	return nil
}

func (s *Syncer) ResolveMedia(ctx context.Context, c *sandbox.Client, packageName, sourceEntryID string, syncChapters bool) (sqlcgen.Medium, error) {
	ext, err := s.q.GetExtensionByPackageName(ctx, packageName)
	if err != nil {
		return sqlcgen.Medium{}, fmt.Errorf("lookup extension %s: %w", packageName, err)
	}

	existing, err := s.q.GetMediaByExtensionAndExternalID(ctx, sqlcgen.GetMediaByExtensionAndExternalIDParams{
		ExtensionID: sql.NullInt64{Int64: ext.ID, Valid: true},
		ExternalID:  sourceEntryID,
	})
	if err == nil && existing.DetailsFetchedAt.Valid {

		if syncChapters {
			if _, err := s.SyncChapters(ctx, c, existing.ID); err != nil {
				return sqlcgen.Medium{}, fmt.Errorf("sync chapters for media %d: %w", existing.ID, err)
			}
		}
		_ = s.q.TouchMediaViewed(ctx, existing.ID)
		return s.q.GetMedia(ctx, existing.ID)
	}

	details, err := c.GetDetails(ctx, packageName, sourceEntryID)
	if err != nil {
		return sqlcgen.Medium{}, fmt.Errorf("get details for %s/%s: %w", packageName, sourceEntryID, err)
	}

	entry, err := s.upsertEntryFromDetails(ctx, c, ext, details)
	if err != nil {
		return sqlcgen.Medium{}, err
	}

	if syncChapters {
		if _, err := s.SyncChapters(ctx, c, entry.ID); err != nil {
			return sqlcgen.Medium{}, fmt.Errorf("sync chapters for media %d: %w", entry.ID, err)
		}
	}
	_ = s.q.TouchMediaViewed(ctx, entry.ID)

	return s.q.GetMedia(ctx, entry.ID)
}

func (s *Syncer) SetInLibrary(ctx context.Context, c *sandbox.Client, mediaID int64, inLibrary bool) (sqlcgen.Medium, error) {
	m, err := s.q.GetMedia(ctx, mediaID)
	if err != nil {
		return sqlcgen.Medium{}, fmt.Errorf("get media %d: %w", mediaID, err)
	}

	if inLibrary && !m.DetailsFetchedAt.Valid && m.ExtensionID.Valid {
		ext, err := s.q.GetExtension(ctx, m.ExtensionID.Int64)
		if err != nil {
			return sqlcgen.Medium{}, fmt.Errorf("get extension %d: %w", m.ExtensionID.Int64, err)
		}
		if _, err := s.ResolveMedia(ctx, c, ext.PackageName, m.ExternalID, true); err != nil {
			return sqlcgen.Medium{}, err
		}
	}

	if inLibrary {
		return s.q.AddMediaToLibrary(ctx, mediaID)
	}

	return s.q.RemoveMediaFromLibrary(ctx, mediaID)
}

func (s *Syncer) MigrateMedia(ctx context.Context, c *sandbox.Client, fromMediaID int64, toPackageName, toExternalID string) (sqlcgen.Medium, error) {
	if _, err := s.q.GetMedia(ctx, fromMediaID); err != nil {
		return sqlcgen.Medium{}, fmt.Errorf("get source media %d: %w", fromMediaID, err)
	}

	newMedia, err := s.ResolveMedia(ctx, c, toPackageName, toExternalID, true)
	if err != nil {
		return sqlcgen.Medium{}, fmt.Errorf("adopt target %s/%s: %w", toPackageName, toExternalID, err)
	}
	if newMedia.ID == fromMediaID {
		return sqlcgen.Medium{}, fmt.Errorf("migration target resolves to the source entry")
	}

	oldChapters, err := s.q.ListChaptersByMedia(ctx, fromMediaID)
	if err != nil {
		return sqlcgen.Medium{}, fmt.Errorf("list source chapters: %w", err)
	}
	newChapters, err := s.q.ListChaptersByMedia(ctx, newMedia.ID)
	if err != nil {
		return sqlcgen.Medium{}, fmt.Errorf("list target chapters: %w", err)
	}
	progress, err := s.q.ListReadingProgressByMedia(ctx, fromMediaID)
	if err != nil {
		return sqlcgen.Medium{}, fmt.Errorf("list source progress: %w", err)
	}
	folders, err := s.q.ListFoldersByMediaIDs(ctx, []int64{fromMediaID})
	if err != nil {
		return sqlcgen.Medium{}, fmt.Errorf("list source folders: %w", err)
	}

	oldByID := make(map[int64]sqlcgen.Chapter, len(oldChapters))
	for _, ch := range oldChapters {
		oldByID[ch.ID] = ch
	}
	match := chapterMatcher(newChapters)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return sqlcgen.Medium{}, err
	}
	defer func() { _ = tx.Rollback() }()
	qtx := s.q.WithTx(tx)

	for _, p := range progress {
		oldCh, ok := oldByID[p.ChapterID]
		if !ok {
			continue
		}
		newCh, ok := match(oldCh)
		if !ok {
			continue
		}
		if _, err := qtx.UpsertReadingProgress(ctx, sqlcgen.UpsertReadingProgressParams{
			MediaID:         newMedia.ID,
			ChapterID:       newCh.ID,
			Progress:        p.Progress,
			Completed:       p.Completed,
			PositionSeconds: p.PositionSeconds,
			DurationSeconds: p.DurationSeconds,
		}); err != nil {
			return sqlcgen.Medium{}, fmt.Errorf("carry progress for chapter %d: %w", p.ChapterID, err)
		}
	}

	for _, f := range folders {
		if err := qtx.AddMediaToFolder(ctx, sqlcgen.AddMediaToFolderParams{
			MediaID:  newMedia.ID,
			FolderID: f.ID,
		}); err != nil {
			return sqlcgen.Medium{}, fmt.Errorf("re-add to folder %d: %w", f.ID, err)
		}
	}

	if _, err := qtx.AddMediaToLibrary(ctx, newMedia.ID); err != nil {
		return sqlcgen.Medium{}, fmt.Errorf("add target to library: %w", err)
	}
	if _, err := qtx.RemoveMediaFromLibrary(ctx, fromMediaID); err != nil {
		return sqlcgen.Medium{}, fmt.Errorf("remove source from library: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return sqlcgen.Medium{}, err
	}

	return s.q.GetMedia(ctx, newMedia.ID)
}

func chapterMatcher(newChapters []sqlcgen.Chapter) func(sqlcgen.Chapter) (sqlcgen.Chapter, bool) {
	type numScan struct {
		num  float64
		scan string
	}
	byNumScan := make(map[numScan]sqlcgen.Chapter, len(newChapters))
	byNum := make(map[float64]sqlcgen.Chapter, len(newChapters))
	for _, ch := range newChapters {
		n := chapterNumber(ch)
		if n <= 0 {
			continue
		}
		if _, ok := byNum[n]; !ok {
			byNum[n] = ch
		}
		byNumScan[numScan{n, ch.Scanlator}] = ch
	}
	return func(old sqlcgen.Chapter) (sqlcgen.Chapter, bool) {
		n := chapterNumber(old)
		if n <= 0 {
			return sqlcgen.Chapter{}, false
		}
		if ch, ok := byNumScan[numScan{n, old.Scanlator}]; ok {
			return ch, true
		}
		if ch, ok := byNum[n]; ok {
			return ch, true
		}
		return sqlcgen.Chapter{}, false
	}
}

func chapterNumber(ch sqlcgen.Chapter) float64 {
	if ch.Number.Valid && ch.Number.Float64 > 0 {
		return ch.Number.Float64
	}
	if ch.Title.Valid {
		return chapternum.FromTitle(ch.Title.String)
	}
	return 0
}

func (s *Syncer) RefreshMetadata(ctx context.Context, c *sandbox.Client, libraryEntryID int64, syncChapters bool) (sqlcgen.Medium, error) {
	entry, err := s.q.GetMedia(ctx, libraryEntryID)
	if err != nil {
		return sqlcgen.Medium{}, fmt.Errorf("get media %d: %w", libraryEntryID, err)
	}
	if !entry.ExtensionID.Valid {
		return sqlcgen.Medium{}, fmt.Errorf("library entry %d has no extension (extension was removed)", libraryEntryID)
	}
	ext, err := s.q.GetExtension(ctx, entry.ExtensionID.Int64)
	if err != nil {
		return sqlcgen.Medium{}, fmt.Errorf("get extension %d: %w", entry.ExtensionID.Int64, err)
	}

	details, err := c.GetDetails(ctx, ext.PackageName, entry.ExternalID)
	if err != nil {
		return sqlcgen.Medium{}, fmt.Errorf("get details for %s/%s: %w", ext.PackageName, entry.ExternalID, err)
	}

	medium, err := s.upsertEntryFromDetails(ctx, c, ext, details)
	if err != nil {
		return sqlcgen.Medium{}, err
	}

	if syncChapters {
		if _, err := s.SyncChapters(ctx, c, libraryEntryID); err != nil {
			return sqlcgen.Medium{}, fmt.Errorf("sync chapters for %d: %w", libraryEntryID, err)
		}
	}

	return medium, nil
}

func (s *Syncer) upsertEntryFromDetails(ctx context.Context, c *sandbox.Client, ext sqlcgen.Extension, details *sandboxv1.EntryDetails) (sqlcgen.Medium, error) {
	author := strings.Join(details.GetAuthors(), ", ")

	entry, err := s.q.UpsertMediaDetails(ctx, sqlcgen.UpsertMediaDetailsParams{
		ExtensionID:   sql.NullInt64{Int64: ext.ID, Valid: true},
		ExtensionName: ext.Name,
		ExternalID:    details.SourceEntryId,
		ContentType:   ext.ContentType,
		Title:         details.Title,
		CoverPath:     sql.NullString{String: details.CoverUrl, Valid: details.CoverUrl != ""},
		Description:   sql.NullString{String: details.Description, Valid: details.Description != ""},
		Status:        sql.NullString{String: details.Status, Valid: details.Status != ""},
		Author:        sql.NullString{String: author, Valid: author != ""},
	})
	if err != nil {
		return sqlcgen.Medium{}, fmt.Errorf("upsert media details: %w", err)
	}

	coverURL := details.CoverUrl
	if coverURL == "" && entry.CoverPath.Valid {
		coverURL = entry.CoverPath.String
	}
	if coverURL != "" && !entry.CoverLocalPath.Valid {
		img, err := c.GetImageBytes(ctx, ext.PackageName, coverURL)
		if err != nil {
			log.Printf("sync: caching cover for entry %d failed: %v", entry.ID, err)
		} else {
			coverPath, err := image.SaveBytesToFile(img.GetData(), img.GetContentType(), filepath.Join(s.mediaDir, "covers"), strconv.FormatInt(entry.ID, 10))
			if err != nil {
				log.Printf("sync: saving cover for entry %d failed: %v", entry.ID, err)
			} else if err := s.q.UpdateMediaCoverLocalPath(ctx, sqlcgen.UpdateMediaCoverLocalPathParams{
				CoverLocalPath: sql.NullString{String: coverPath, Valid: true},
				ID:             entry.ID,
			}); err != nil {
				log.Printf("sync: recording cover path for entry %d failed: %v", entry.ID, err)
			}
		}
	}

	if genres := nonEmpty(details.GetGenres()); len(genres) > 0 {
		if err := s.q.ClearGenresForMedia(ctx, entry.ID); err != nil {
			return sqlcgen.Medium{}, fmt.Errorf("clear genres for media %d: %w", entry.ID, err)
		}
		for _, name := range genres {
			genre, err := s.q.CreateGenre(ctx, name)
			if err != nil {
				return sqlcgen.Medium{}, fmt.Errorf("create genre %q: %w", name, err)
			}
			if err := s.q.AddGenreToMedia(ctx, sqlcgen.AddGenreToMediaParams{
				MediaID: entry.ID,
				GenreID: genre.ID,
			}); err != nil {
				return sqlcgen.Medium{}, fmt.Errorf("add genre %q to media %d: %w", name, entry.ID, err)
			}
		}
	}

	s.maybeEnrich(ctx, entry.ID)
	if s.recompute != nil {
		id := entry.ID
		go func() { _ = s.recompute.RecomputeMedia(context.Background(), id) }()
	}
	if refreshed, err := s.q.GetMedia(ctx, entry.ID); err == nil {
		entry = refreshed
	}
	return entry, nil
}

func (s *Syncer) maybeEnrich(ctx context.Context, mediaID int64) {
	if s.enricher == nil {
		return
	}
	ec, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := s.enricher.AutoEnrich(ec, mediaID); err != nil {
		log.Printf("sync: metadata auto-enrich for media %d: %v", mediaID, err)
	}
}

func nonEmpty(s []string) []string {
	out := s[:0:0]
	for _, v := range s {
		if strings.TrimSpace(v) != "" {
			out = append(out, v)
		}
	}
	return out
}

func (s *Syncer) GetMedia(ctx context.Context, id int64) (sqlcgen.Medium, error) {
	return s.q.GetMedia(ctx, id)
}

func (s *Syncer) EnsureHydrated(ctx context.Context, sc *sandbox.SupervisedClient, id int64) (sqlcgen.Medium, error) {
	m, err := s.q.GetMedia(ctx, id)
	if err != nil {
		return sqlcgen.Medium{}, err
	}
	if m.DetailsFetchedAt.Valid || !m.ExtensionID.Valid {
		return m, nil
	}

	lk := s.lockFor(fmt.Sprintf("hydrate:%d", id))
	lk.Lock()
	defer lk.Unlock()

	m, err = s.q.GetMedia(ctx, id)
	if err != nil {
		return sqlcgen.Medium{}, err
	}
	if m.DetailsFetchedAt.Valid {
		return m, nil
	}

	ext, err := s.q.GetExtension(ctx, m.ExtensionID.Int64)
	if err != nil {
		return sqlcgen.Medium{}, fmt.Errorf("get extension %d: %w", m.ExtensionID.Int64, err)
	}
	c, err := sc.Ensure(ctx)
	if err != nil {
		return sqlcgen.Medium{}, err
	}
	details, err := c.GetDetails(ctx, ext.PackageName, m.ExternalID)
	if err != nil {
		return sqlcgen.Medium{}, fmt.Errorf("get details for %s/%s: %w", ext.PackageName, m.ExternalID, err)
	}
	return s.upsertEntryFromDetails(ctx, c, ext, details)
}

func (s *Syncer) EnsureChapters(ctx context.Context, sc *sandbox.SupervisedClient, id int64) ([]sqlcgen.Chapter, error) {
	m, err := s.q.GetMedia(ctx, id)
	if err != nil {
		return nil, err
	}
	if m.ChaptersSyncedAt.Valid || !m.ExtensionID.Valid {
		return s.q.ListChaptersByMedia(ctx, id)
	}

	lk := s.lockFor(fmt.Sprintf("chapters:%d", id))
	lk.Lock()
	defer lk.Unlock()

	m, err = s.q.GetMedia(ctx, id)
	if err != nil {
		return nil, err
	}
	if m.ChaptersSyncedAt.Valid {
		return s.q.ListChaptersByMedia(ctx, id)
	}
	c, err := sc.Ensure(ctx)
	if err != nil {
		return nil, err
	}
	return s.SyncChapters(ctx, c, id)
}

type LibraryQuery struct {
	ContentType    string
	InLibrary      *bool
	UnreadOnly     bool
	DownloadedOnly bool
	TagIDs         []int64
	GenreIDs       []int64
	FolderID       *int64
	Search         string
	SortBy         string
	Ascending      bool
	Limit          int
	Offset         int

	// ContentFilterRank: 0 = unrestricted (no filtering), 1 = moderate, 2 = strict.
	ContentFilterRank int
}

func (s *Syncer) QueryLibrary(ctx context.Context, q LibraryQuery) ([]sqlcgen.Medium, int, error) {
	var where []string
	var args []interface{}

	switch {
	case q.InLibrary != nil && *q.InLibrary:
		where = append(where, "m.added_at IS NOT NULL")
	case q.InLibrary != nil && !*q.InLibrary:
		where = append(where, "m.added_at IS NULL")
	}
	if q.ContentType != "" {
		where = append(where, "m.content_type = ?")
		args = append(args, q.ContentType)
	}
	if q.Search != "" {
		where = append(where, "m.title LIKE '%' || ? || '%'")
		args = append(args, q.Search)
	}
	if q.FolderID != nil {
		where = append(where, "m.id IN (SELECT media_id FROM media_folders WHERE folder_id = ?)")
		args = append(args, *q.FolderID)
	}
	if len(q.TagIDs) > 0 {
		ph := strings.TrimSuffix(strings.Repeat("?,", len(q.TagIDs)), ",")
		where = append(where, "m.id IN (SELECT media_id FROM media_tags WHERE tag_id IN ("+ph+
			") GROUP BY media_id HAVING COUNT(DISTINCT tag_id) = ?)")
		for _, id := range q.TagIDs {
			args = append(args, id)
		}
		args = append(args, len(q.TagIDs))
	}
	if len(q.GenreIDs) > 0 {
		ph := strings.TrimSuffix(strings.Repeat("?,", len(q.GenreIDs)), ",")
		where = append(where, "m.id IN (SELECT media_id FROM media_genres WHERE genre_id IN ("+ph+
			") GROUP BY media_id HAVING COUNT(DISTINCT genre_id) = ?)")
		for _, id := range q.GenreIDs {
			args = append(args, id)
		}
		args = append(args, len(q.GenreIDs))
	}
	if q.ContentFilterRank > 0 {
		where = append(where, "(m.content_block_rank IS NULL OR m.content_block_rank > ?)")
		args = append(args, q.ContentFilterRank)
	}
	if q.UnreadOnly {
		where = append(where, `EXISTS (
			SELECT 1 FROM chapters c
			LEFT JOIN reading_progress rp ON rp.chapter_id = c.id AND rp.media_id = c.media_id
			WHERE c.media_id = m.id AND (rp.completed IS NULL OR rp.completed = 0))`)
	}
	if q.DownloadedOnly {
		where = append(where, `EXISTS (
			SELECT 1 FROM downloads d JOIN chapters c ON c.id = d.chapter_id
			WHERE c.media_id = m.id AND d.status = 'done')`)
	}

	whereSQL := ""
	if len(where) > 0 {
		whereSQL = " WHERE " + strings.Join(where, " AND ")
	}

	dir := "DESC"
	if q.Ascending {
		dir = "ASC"
	}
	var orderJoin, orderSQL string
	switch q.SortBy {
	case "title":
		orderSQL = "ORDER BY m.title COLLATE NOCASE " + dir
	case "last_read_at":
		orderJoin = "LEFT JOIN (SELECT media_id, MAX(updated_at) v FROM reading_progress GROUP BY media_id) rp ON rp.media_id = m.id"
		orderSQL = "ORDER BY rp.v " + dir
	case "latest_chapter_at":
		orderJoin = "LEFT JOIN (SELECT media_id, MAX(uploaded_at) v FROM chapters GROUP BY media_id) ch ON ch.media_id = m.id"
		orderSQL = "ORDER BY ch.v " + dir
	case "unread_count":
		orderJoin = `LEFT JOIN (
			SELECT c.media_id, COUNT(*) v FROM chapters c
			LEFT JOIN reading_progress rp2 ON rp2.chapter_id = c.id AND rp2.media_id = c.media_id
			WHERE rp2.completed IS NULL OR rp2.completed = 0
			GROUP BY c.media_id) uc ON uc.media_id = m.id`
		orderSQL = "ORDER BY uc.v " + dir
	default:
		orderSQL = "ORDER BY m.added_at " + dir
	}

	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM media m"+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count library: %w", err)
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	pageArgs := append(append([]interface{}{}, args...), limit, q.Offset)
	idSQL := "SELECT m.id FROM media m " + orderJoin + whereSQL + " " + orderSQL + " LIMIT ? OFFSET ?"
	rows, err := s.db.QueryContext(ctx, idSQL, pageArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("query library: %w", err)
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, 0, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if len(ids) == 0 {
		return nil, total, nil
	}

	fetched, err := s.q.ListMediaByIDs(ctx, ids)
	if err != nil {
		return nil, 0, fmt.Errorf("hydrate library page: %w", err)
	}
	byID := make(map[int64]sqlcgen.Medium, len(fetched))
	for _, m := range fetched {
		byID[m.ID] = m
	}
	ordered := make([]sqlcgen.Medium, 0, len(ids))
	for _, id := range ids {
		if m, ok := byID[id]; ok {
			ordered = append(ordered, m)
		}
	}
	return ordered, total, nil
}

type chapterSummary struct {
	SourceID string
	Name     string
	Number   float64
	UploadTS int64
}

func (s *Syncer) BackfillChapterNumbers(ctx context.Context) (int, error) {
	rows, err := s.q.ListChaptersMissingNumber(ctx)
	if err != nil {
		return 0, err
	}
	fixed := 0
	for _, r := range rows {
		if !r.Title.Valid {
			continue
		}
		n := chapternum.FromTitle(r.Title.String)
		if n <= 0 {
			continue
		}
		if err := s.q.SetChapterNumber(ctx, sqlcgen.SetChapterNumberParams{
			Number: sql.NullFloat64{Float64: n, Valid: true},
			ID:     r.ID,
		}); err != nil {
			return fixed, err
		}
		fixed++
	}
	return fixed, nil
}

func (s *Syncer) fetchChapterSummaries(ctx context.Context, c *sandbox.Client, packageName, contentType, sourceEntryID string) ([]chapterSummary, error) {
	var summaries []chapterSummary
	switch contentType {
	case "anime":
		list, err := c.GetEpisodes(ctx, packageName, sourceEntryID)
		if err != nil {
			return nil, fmt.Errorf("get episodes for %s/%s: %w", packageName, sourceEntryID, err)
		}
		for _, e := range list.Episodes {
			summaries = append(summaries, chapterSummary{e.SourceEpisodeId, e.Name, e.Number, e.UploadTimestamp})
		}
	default:
		list, err := c.GetChapters(ctx, packageName, sourceEntryID)
		if err != nil {
			return nil, fmt.Errorf("get chapters for %s/%s: %w", packageName, sourceEntryID, err)
		}
		for _, ch := range list.Chapters {
			summaries = append(summaries, chapterSummary{ch.SourceChapterId, ch.Name, ch.Number, ch.UploadTimestamp})
		}
	}

	if chaptersAreNewestFirst(summaries) {
		for i, j := 0, len(summaries)-1; i < j; i, j = i+1, j-1 {
			summaries[i], summaries[j] = summaries[j], summaries[i]
		}
	}
	return summaries, nil
}

func chaptersAreNewestFirst(s []chapterSummary) bool {
	if len(s) < 2 {
		return false
	}
	asc, desc := 0, 0
	prev, havePrev := 0.0, false
	for _, c := range s {
		if c.Number <= 0 {
			continue
		}
		if havePrev {
			switch {
			case c.Number > prev:
				asc++
			case c.Number < prev:
				desc++
			}
		}
		prev, havePrev = c.Number, true
	}
	if asc != desc {
		return desc > asc
	}
	if a, b := s[0].UploadTS, s[len(s)-1].UploadTS; a > 0 && b > 0 && a != b {
		return a > b
	}
	return true
}

var wsRun = regexp.MustCompile(`\s+`)

func deriveScanlators(summaries []chapterSummary) []string {
	out := make([]string, len(summaries))
	groups := map[string][]int{}
	for i, s := range summaries {
		k := chapterGroupKey(s)
		groups[k] = append(groups[k], i)
	}
	for _, idxs := range groups {
		if len(idxs) < 2 {
			continue
		}
		for pos, i := range idxs {
			out[i] = strconv.Itoa(pos + 1)
		}
	}
	return out
}

func chapterGroupKey(s chapterSummary) string {
	if s.Number > 0 {
		return "n:" + strconv.FormatFloat(s.Number, 'f', 4, 64)
	}
	base := strings.TrimSpace(s.Name)
	if i := strings.IndexAny(base, ":|"); i >= 0 {
		base = strings.TrimSpace(base[:i])
	}
	if i := strings.IndexAny(base, "[("); i >= 0 {
		base = strings.TrimSpace(base[:i])
	}
	base = wsRun.ReplaceAllString(strings.ToLower(base), " ")
	if base == "" {
		return "i:" + s.SourceID
	}
	return "t:" + base
}

func (s *Syncer) SyncChapters(ctx context.Context, c *sandbox.Client, libraryEntryID int64) ([]sqlcgen.Chapter, error) {
	entry, err := s.q.GetMedia(ctx, libraryEntryID)
	if err != nil {
		return nil, fmt.Errorf("get media %d: %w", libraryEntryID, err)
	}
	if !entry.ExtensionID.Valid {
		return nil, fmt.Errorf("media %d has no extension (extension was removed)", libraryEntryID)
	}
	ext, err := s.q.GetExtension(ctx, entry.ExtensionID.Int64)
	if err != nil {
		return nil, fmt.Errorf("get extension %d: %w", entry.ExtensionID.Int64, err)
	}

	summaries, err := s.fetchChapterSummaries(ctx, c, ext.PackageName, ext.ContentType, entry.ExternalID)
	if err != nil {
		return nil, err
	}

	for i := range summaries {
		summaries[i].Number = chapternum.Resolve(summaries[i].Number, summaries[i].Name)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin chapter sync tx: %w", err)
	}
	defer tx.Rollback()
	qtx := s.q.WithTx(tx)

	scanlators := deriveScanlators(summaries)

	chapters := make([]sqlcgen.Chapter, 0, len(summaries))
	for idx, sm := range summaries {
		var uploadedAt sql.NullInt64
		if sm.UploadTS > 0 {
			uploadedAt = sql.NullInt64{Int64: sm.UploadTS / 1000, Valid: true}
		}
		chapter, err := qtx.CreateChapter(ctx, sqlcgen.CreateChapterParams{
			MediaID:     libraryEntryID,
			ExternalID:  sm.SourceID,
			Title:       sql.NullString{String: sm.Name, Valid: sm.Name != ""},
			Number:      sql.NullFloat64{Float64: sm.Number, Valid: true},
			UploadedAt:  uploadedAt,
			SourceOrder: sql.NullInt64{Int64: int64(idx), Valid: true},
			Scanlator:   scanlators[idx],
		})
		if err != nil {
			return nil, fmt.Errorf("upsert chapter %s: %w", sm.SourceID, err)
		}
		chapters = append(chapters, chapter)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit chapter sync tx: %w", err)
	}

	if err := s.q.MarkChaptersSynced(ctx, libraryEntryID); err != nil {
		log.Printf("sync: marking chapters synced for media %d failed: %v", libraryEntryID, err)
	}

	return chapters, nil
}

func (s *Syncer) RecordProgress(ctx context.Context, libraryEntryID, chapterID int64, progress float64, completed bool, positionSeconds, durationSeconds *float64) (sqlcgen.ReadingProgress, error) {
	params := sqlcgen.UpsertReadingProgressParams{
		MediaID:   libraryEntryID,
		ChapterID: chapterID,
		Progress:  progress,
		Completed: completed,
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
	return s.q.ListReadingProgressByMedia(ctx, libraryEntryID)
}
