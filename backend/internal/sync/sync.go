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
	"strconv"
	"strings"
	"sync"
	"time"

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
}

func New(db *sql.DB, q *sqlcgen.Queries, cacheDir, mediaDir string) *Syncer {
	return &Syncer{db: db, q: q, cacheDir: cacheDir, mediaDir: mediaDir, locks: make(map[string]*sync.Mutex)}
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

	for _, ext := range parsed {
		contentType := ext.ContentType
		if contentType == "" {
			contentType = repository.ClassifyContentType(ext.PackageName)
		}
		_, err = s.q.UpsertExtension(ctx, sqlcgen.UpsertExtensionParams{
			RepositoryID: repo.ID,
			PackageName:  ext.PackageName,
			Name:         ext.Name,
			Version:      ext.VersionName,
			ContentType:  contentType,
			Lang:         ext.Lang,
			IconUrl:      nullString(ext.IconURL),
			ApkUrl:       ext.ApkURL,
			JarUrl:       nullString(ext.JarURL),
			IsNsfw:       ext.IsNsfw,
		})
		if err != nil {
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

const sideloadRepoURL = "local:sideload"

// getOrCreateSideloadRepo returns the singleton pseudo-repository used to
// satisfy extensions.repository_id's NOT NULL constraint for extensions
// installed by direct URL rather than from a real repository index.
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

// InstallExternalExtension downloads an extension package (.apk/.jar/.js)
// from an arbitrary URL, has the sandbox load it to discover its real
// package name/content type/language, and registers it in the DB under a
// singleton "Sideloaded" pseudo-repository. Unlike InstallExtension, no
// prior repository index entry is required.
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
		// Version metadata is not available for sideloaded packages (jars
		// have no version field, and apk versionName is discarded by the
		// sandbox's loader); this is a known limitation, not a bug.
		Version:     "unknown",
		ContentType: contentType,
		Lang:        meta.Lang,
	})
	if err != nil {
		return sqlcgen.Extension{}, fmt.Errorf("upsert extension: %w", err)
	}

	// The sandbox's PeekExtension call already copies the file into the
	// extensions dir and loads it (via registry.install), so the extension
	// is already installed/running at this point. Just record that fact.
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
		MediaID: libraryEntryID,
		Kind:    "reading_status",
	}); err != nil {
		return fmt.Errorf("remove existing reading status: %w", err)
	}

	if err := qtx.AddEntryToFolder(ctx, sqlcgen.AddEntryToFolderParams{
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

// EnsureMediaShadow finds-or-creates a media row for a source entry the
// user is interacting with (reading, downloading, marking read) but has
// not explicitly added to the library. This is the same upsert
// AddToLibrary uses, just without the SetMediaAddedAt step -- the row's
// added_at stays NULL, so it remains invisible to library()/inLibrary
// checks, exactly like RefreshMetadata's upsert. Chapters are synced so
// every downstream resolver (Pages, Completed, Download, etc.) can treat
// this exactly like any other library entry -- no more separate live/DB
// branching for "not yet in library" content.
func (s *Syncer) EnsureMediaShadow(ctx context.Context, c *sandbox.Client, packageName, sourceEntryID string) (sqlcgen.Medium, error) {
	ext, err := s.q.GetExtensionByPackageName(ctx, packageName)
	if err != nil {
		return sqlcgen.Medium{}, fmt.Errorf("lookup extension %s: %w", packageName, err)
	}

	if existing, err := s.q.GetMediaByExtensionIDAndExternalID(ctx, sqlcgen.GetMediaByExtensionIDAndExternalIDParams{
		ExtensionID: sql.NullInt64{Int64: ext.ID, Valid: true},
		ExternalID:  sourceEntryID,
	}); err == nil {
		// Row already exists (shadow or real library entry) -- reuse it.
		// Chapters are synced unconditionally here; SyncChapters/CreateChapter
		// upsert, so repeat views just refresh rather than duplicate.
		if _, err := s.SyncChapters(ctx, c, existing.ID); err != nil {
			return sqlcgen.Medium{}, fmt.Errorf("sync chapters for shadow media %d: %w", existing.ID, err)
		}
		return s.q.GetLibraryEntry(ctx, existing.ID)
	}

	details, err := c.GetDetails(ctx, packageName, sourceEntryID)
	if err != nil {
		return sqlcgen.Medium{}, fmt.Errorf("get details for %s/%s: %w", packageName, sourceEntryID, err)
	}

	entry, err := s.upsertEntryFromDetails(ctx, c, ext, details)
	if err != nil {
		return sqlcgen.Medium{}, err
	}

	if _, err := s.SyncChapters(ctx, c, entry.ID); err != nil {
		return sqlcgen.Medium{}, fmt.Errorf("sync chapters for new shadow media %d: %w", entry.ID, err)
	}

	return s.q.GetLibraryEntry(ctx, entry.ID)
}

func (s *Syncer) AddToLibrary(ctx context.Context, c *sandbox.Client, packageName, sourceEntryID string) (sqlcgen.Medium, error) {
	ext, err := s.q.GetExtensionByPackageName(ctx, packageName)
	if err != nil {
		return sqlcgen.Medium{}, fmt.Errorf("lookup extension %s: %w", packageName, err)
	}

	details, err := c.GetDetails(ctx, packageName, sourceEntryID)
	if err != nil {
		return sqlcgen.Medium{}, fmt.Errorf("get details for %s/%s: %w", packageName, sourceEntryID, err)
	}

	entry, err := s.upsertEntryFromDetails(ctx, c, ext, details)
	if err != nil {
		return sqlcgen.Medium{}, err
	}
	// upsertEntryFromDetails is shared with RefreshMetadata, which must
	// never touch added_at, so marking the item as added is a deliberate
	// separate step done only on this path.
	return s.q.SetMediaAddedAt(ctx, entry.ID)
}

// RefreshMetadata re-fetches title/description/status/author/genres/cover
// from the source extension for an existing library entry and updates it
// in place. By default (syncChapters true) it also calls SyncChapters
// afterward to pick up new chapters; pass syncChapters false for a
// metadata-only refresh.
func (s *Syncer) RefreshMetadata(ctx context.Context, c *sandbox.Client, libraryEntryID int64, syncChapters bool) (sqlcgen.Medium, error) {
	entry, err := s.q.GetLibraryEntry(ctx, libraryEntryID)
	if err != nil {
		return sqlcgen.Medium{}, fmt.Errorf("get library entry %d: %w", libraryEntryID, err)
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

	entry, err := s.q.CreateLibraryEntry(ctx, sqlcgen.CreateLibraryEntryParams{
		ExtensionID:   sql.NullInt64{Int64: ext.ID, Valid: true},
		ExtensionName: ext.Name,
		ExternalID:    details.SourceEntryId,
		ContentType:   ext.ContentType,
		Title:         details.Title,
		CoverPath:     sql.NullString{String: details.CoverUrl, Valid: details.CoverUrl != ""},
		Description:   sql.NullString{String: details.Description, Valid: details.Description != ""},
		Status:        sql.NullString{String: details.Status, Valid: details.Status != ""},
		Author:        sql.NullString{String: author, Valid: author != ""},
		// Artist is not currently provided by the extension proto (only
		// Authors is available), so it is left unset here.
	})
	if err != nil {
		return sqlcgen.Medium{}, fmt.Errorf("upsert library entry: %w", err)
	}

	if details.CoverUrl != "" {
		img, err := c.GetImageBytes(ctx, ext.PackageName, details.CoverUrl)
		if err != nil {
			log.Printf("sync: caching cover for entry %d failed: %v", entry.ID, err)
		} else {
			coverPath, err := image.SaveBytesToFile(img.GetData(), img.GetContentType(), filepath.Join(s.mediaDir, "covers"), strconv.FormatInt(entry.ID, 10))
			if err != nil {
				log.Printf("sync: saving cover for entry %d failed: %v", entry.ID, err)
			} else if err := s.q.UpdateLibraryEntryCoverLocalPath(ctx, sqlcgen.UpdateLibraryEntryCoverLocalPathParams{
				CoverLocalPath: sql.NullString{String: coverPath, Valid: true},
				ID:             entry.ID,
			}); err != nil {
				log.Printf("sync: recording cover path for entry %d failed: %v", entry.ID, err)
			}
		}
	}

	if err := s.q.ClearGenresForEntry(ctx, entry.ID); err != nil {
		return sqlcgen.Medium{}, fmt.Errorf("clear genres for entry %d: %w", entry.ID, err)
	}
	for _, name := range details.GetGenres() {
		if name == "" {
			continue
		}
		genre, err := s.q.CreateGenre(ctx, name)
		if err != nil {
			return sqlcgen.Medium{}, fmt.Errorf("create genre %q: %w", name, err)
		}
		if err := s.q.AddGenreToEntry(ctx, sqlcgen.AddGenreToEntryParams{
			MediaID: entry.ID,
			GenreID: genre.ID,
		}); err != nil {
			return sqlcgen.Medium{}, fmt.Errorf("add genre %q to entry %d: %w", name, entry.ID, err)
		}
	}

	return entry, nil
}

func (s *Syncer) GetLibraryEntry(ctx context.Context, id int64) (sqlcgen.Medium, error) {
	return s.q.GetLibraryEntry(ctx, id)
}

func (s *Syncer) ListLibraryEntries(ctx context.Context, contentType string) ([]sqlcgen.Medium, error) {
	if contentType == "" {
		return s.q.ListLibraryEntries(ctx)
	}
	return s.q.ListLibraryEntriesByContentType(ctx, contentType)
}

// RemoveFromLibrary detaches a media item from the library without
// destroying it: added_at is cleared (library() queries and inLibrary
// checks stop returning it) and its folder memberships are dropped (folders
// are library organization, meaningless once removed). Chapters, reading
// progress, and downloaded files are left untouched — re-adding the item
// later brings all of that history back automatically.
func (s *Syncer) RemoveFromLibrary(ctx context.Context, libraryEntryID int64) error {
	if err := s.q.ClearFoldersForMedia(ctx, libraryEntryID); err != nil {
		return fmt.Errorf("clear folders for media %d: %w", libraryEntryID, err)
	}
	return s.q.ClearMediaAddedAt(ctx, libraryEntryID)
}

type chapterSummary struct {
	SourceID string
	Name     string
	Number   float64
	UploadTS int64
}

// fetchChapterSummaries hits the extension directly for its chapter/episode
// list, with no DB reads or writes. Shared by SyncChapters (which persists
// the result) and PreviewChapters (which does not).
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
	default: // manga, novel
		list, err := c.GetChapters(ctx, packageName, sourceEntryID)
		if err != nil {
			return nil, fmt.Errorf("get chapters for %s/%s: %w", packageName, sourceEntryID, err)
		}
		for _, ch := range list.Chapters {
			summaries = append(summaries, chapterSummary{ch.SourceChapterId, ch.Name, ch.Number, ch.UploadTimestamp})
		}
	}
	// Tachiyomi extensions conventionally return chapters newest-first.
	// sourceOrder (assigned by callers as the index into this slice) is
	// meant to mean "reading order" everywhere downstream, so reverse once
	// here rather than in every consumer. NOTE: unconfirmed whether this
	// holds for every source -- if some extension already returns
	// oldest-first, that source's order will come out backwards instead.
	for i, j := 0, len(summaries)-1; i < j; i, j = i+1, j-1 {
		summaries[i], summaries[j] = summaries[j], summaries[i]
	}
	return summaries, nil
}

// PreviewChapters fetches the chapter/episode list for a source entry that
// is not (yet) in the library, without writing anything to the DB. Used to
// preview chapter count/overlap before committing to addToLibrary.
func (s *Syncer) PreviewChapters(ctx context.Context, c *sandbox.Client, packageName, sourceEntryID string) ([]chapterSummary, error) {
	ext, err := s.q.GetExtensionByPackageName(ctx, packageName)
	if err != nil {
		return nil, fmt.Errorf("lookup extension %s: %w", packageName, err)
	}
	return s.fetchChapterSummaries(ctx, c, packageName, ext.ContentType, sourceEntryID)
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

	summaries, err := s.fetchChapterSummaries(ctx, c, ext.PackageName, ext.ContentType, entry.ExternalID)
	if err != nil {
		return nil, err
	}

	chapters := make([]sqlcgen.Chapter, 0, len(summaries))
	for idx, sm := range summaries {
		var uploadedAt sql.NullInt64
		if sm.UploadTS > 0 {
			uploadedAt = sql.NullInt64{Int64: sm.UploadTS / 1000, Valid: true}
		}
		chapter, err := s.q.CreateChapter(ctx, sqlcgen.CreateChapterParams{
			MediaID:     libraryEntryID,
			ExternalID:  sm.SourceID,
			Title:       sql.NullString{String: sm.Name, Valid: sm.Name != ""},
			Number:      sql.NullFloat64{Float64: sm.Number, Valid: true},
			UploadedAt:  uploadedAt,
			SourceOrder: sql.NullInt64{Int64: int64(idx), Valid: true},
		})
		if err != nil {
			return nil, fmt.Errorf("upsert chapter %s: %w", sm.SourceID, err)
		}
		chapters = append(chapters, chapter)
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
	return s.q.ListReadingProgressByLibraryEntry(ctx, libraryEntryID)
}
