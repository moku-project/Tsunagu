package localsource

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"tsunagu/backend/internal/db/sqlcgen"
)

type Enricher interface {
	AutoEnrich(ctx context.Context, mediaID int64) error
}

type Scanner struct {
	q        *sqlcgen.Queries
	mediaDir string
	enricher Enricher

	dirMu    sync.RWMutex
	localDir string // overrides the default {mediaDir}/local when set; see SetLocalDir
}

func New(q *sqlcgen.Queries, mediaDir string) *Scanner {
	return &Scanner{q: q, mediaDir: mediaDir}
}

// SetEnricher wires a metadata provider that newly-discovered local titles
// get auto-matched against (by folder/title name) right after ingestion, so
// they pick up a cover/synopsis without the user having to open the
// Metadata panel manually. Manual re-match via the panel always works
// regardless of whether this is set.
func (s *Scanner) SetEnricher(e Enricher) { s.enricher = e }

// LocalDir returns the directory currently scanned for local media —
// {mediaDir}/local by default, or the relocated directory set via
// SetLocalDir/Relocate.
func (s *Scanner) LocalDir() string {
	s.dirMu.RLock()
	defer s.dirMu.RUnlock()
	if s.localDir != "" {
		return s.localDir
	}
	return filepath.Join(s.mediaDir, "local")
}

// SetLocalDir changes the directory Scan() reads from. Pass "" to revert to
// the default {mediaDir}/local.
func (s *Scanner) SetLocalDir(dir string) {
	s.dirMu.Lock()
	s.localDir = strings.TrimSpace(dir)
	s.dirMu.Unlock()
}

type Result struct {
	MediaTouched  int `json:"mediaTouched"`
	ChaptersAdded int `json:"chaptersAdded"`
	FilesLinked   int `json:"filesLinked"`
	RowsPruned    int `json:"rowsPruned"`
}

var dirKind = map[string]string{
	"manga":  "manga",
	"anime":  "anime",
	"novels": "novel",
	"novel":  "novel",
}

// ParseLocalExternalID extracts the content type from a local-source
// external_id, e.g. "local:manga/Title" -> "manga".
func ParseLocalExternalID(externalID string) (contentType string, ok bool) {
	rest, found := strings.CutPrefix(externalID, "local:")
	if !found {
		return "", false
	}
	seg, _, _ := strings.Cut(rest, "/")
	ct, ok := dirKind[strings.ToLower(seg)]
	return ct, ok
}

var imageExts = map[string]bool{
	".webp": true, ".jpg": true, ".jpeg": true, ".png": true,
	".gif": true, ".avif": true, ".bmp": true,
}
var textExts = map[string]bool{
	".txt": true, ".html": true, ".htm": true, ".xhtml": true, ".md": true,
}
var bookExts = map[string]bool{
	".epub": true,
	".docx": true,
}
var videoExts = map[string]bool{
	".mp4": true, ".mkv": true, ".webm": true, ".m4v": true, ".avi": true, ".mov": true,
}
var archiveExts = map[string]bool{
	".cbz": true, ".zip": true, ".cbr": true, ".rar": true,
	".cb7": true, ".7z": true, ".cbt": true, ".tar": true, ".pdf": true,
}

func (s *Scanner) Scan(ctx context.Context) (Result, error) {
	var res Result

	root := s.LocalDir()
	if fi, err := os.Stat(root); err == nil && fi.IsDir() {
		kinds, _ := os.ReadDir(root)
		for _, kd := range kinds {
			if !kd.IsDir() {
				continue
			}
			ct, ok := dirKind[strings.ToLower(kd.Name())]
			if !ok {
				log.Printf("local source: skipping %q — top-level folder must be named manga, anime, or novels", kd.Name())
				continue
			}
			if err := s.ingestKind(ctx, filepath.Join(root, kd.Name()), kd.Name(), ct, &res); err != nil {
				return res, err
			}
		}
	}

	pruned, err := s.prune(ctx)
	if err != nil {
		return res, err
	}
	res.RowsPruned = pruned
	return res, nil
}

func (s *Scanner) ingestKind(ctx context.Context, kindDir, kindSegment, ct string, res *Result) error {
	titles, err := os.ReadDir(kindDir)
	if err != nil {
		return err
	}
	for _, td := range titles {
		if !td.IsDir() {
			continue
		}
		title := td.Name()
		titleDir := filepath.Join(kindDir, title)
		externalID := "local:" + strings.ToLower(kindSegment) + "/" + title

		media, err := s.upsertMedia(ctx, externalID, ct, title, findCover(titleDir))
		if err != nil {
			return fmt.Errorf("local media %q: %w", title, err)
		}
		res.MediaTouched++

		// A chapter is either a subfolder of loose files, or — for manga —
		// a single .cbz/.zip archive sitting directly in the title folder.
		entries, _ := os.ReadDir(titleDir)
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			if e.IsDir() {
				names = append(names, e.Name())
				continue
			}
			if ct == "manga" && archiveExts[strings.ToLower(filepath.Ext(e.Name()))] {
				names = append(names, e.Name())
				continue
			}
			if ct == "novel" && bookExts[strings.ToLower(filepath.Ext(e.Name()))] {
				names = append(names, e.Name())
				continue
			}
			if isCoverFilename(e.Name()) {
				continue
			}
			log.Printf("local source: skipping %q in %q — loose files aren't chapters; put them in a chapter subfolder"+
				" or, for manga, a .cbz/.zip file", e.Name(), titleDir)
		}
		sort.Sort(naturalSort(names))

		for idx, chapName := range names {
			chapPath := filepath.Join(titleDir, chapName)
			var (
				created bool
				linked  int
				err     error
			)
			ext := strings.ToLower(filepath.Ext(chapName))
			switch {
			case ct == "manga" && archiveExts[ext]:
				name := strings.TrimSuffix(chapName, filepath.Ext(chapName))
				created, linked, err = s.ingestArchiveChapter(ctx, media.ID, externalID, name, chapPath, idx)
			case ct == "novel" && ext == ".epub":
				created, linked, err = s.ingestEpubBook(ctx, media.ID, externalID, chapPath, idx)
			case ct == "novel" && ext == ".docx":
				created, linked, err = s.ingestDocxBook(ctx, media.ID, externalID, chapPath, idx)
			default:
				created, linked, err = s.ingestChapter(ctx, media.ID, ct, externalID, chapName, chapPath, idx)
			}
			if err != nil {
				return fmt.Errorf("local chapter %q/%q: %w", title, chapName, err)
			}
			if created {
				res.ChaptersAdded++
			}
			res.FilesLinked += linked
		}
	}
	return nil
}

func (s *Scanner) upsertChapterRow(ctx context.Context, mediaID int64, mediaExternalID, chapName string, idx int) (sqlcgen.Chapter, bool, error) {
	return s.upsertChapterRowNamed(ctx, mediaID, mediaExternalID, chapName, chapName, idx)
}

func (s *Scanner) upsertChapterRowNamed(ctx context.Context, mediaID int64, mediaExternalID, key, title string, idx int) (sqlcgen.Chapter, bool, error) {
	chExternalID := mediaExternalID + "/" + key
	_, getErr := s.q.GetChapterByMediaAndExternalID(ctx, sqlcgen.GetChapterByMediaAndExternalIDParams{
		MediaID:    mediaID,
		ExternalID: chExternalID,
	})
	created := getErr == sql.ErrNoRows

	num := parseLeadingNumber(key)
	if num < 0 {
		num = float64(idx + 1)
	}
	ch, err := s.q.CreateChapter(ctx, sqlcgen.CreateChapterParams{
		MediaID:     mediaID,
		ExternalID:  chExternalID,
		Title:       sql.NullString{String: title, Valid: true},
		Number:      sql.NullFloat64{Float64: num, Valid: true},
		SourceOrder: sql.NullInt64{Int64: int64(idx), Valid: true},
	})
	return ch, created, err
}

// ingestArchiveChapter treats a single .cbz/.zip file as one chapter, reading
// its image entries directly (no extraction to disk) — mirrors ingestChapter
// but sources pages from inside the archive instead of a folder.
func (s *Scanner) ingestArchiveChapter(ctx context.Context, mediaID int64, mediaExternalID, chapName, archivePath string, idx int) (created bool, linked int, err error) {
	names, err := listArchiveImageNames(archivePath)
	if err != nil {
		return false, 0, err
	}
	sort.Sort(naturalSort(names))

	ch, created, err := s.upsertChapterRow(ctx, mediaID, mediaExternalID, chapName, idx)
	if err != nil {
		return created, 0, err
	}

	if err := s.q.DeleteMangaPages(ctx, ch.ID); err != nil {
		return created, 0, err
	}
	page := 0
	for _, name := range names {
		page++
		if err := s.q.UpsertMangaPage(ctx, sqlcgen.UpsertMangaPageParams{
			ChapterID:  ch.ID,
			PageNumber: int64(page),
			LocalPath:  sql.NullString{String: ZipPagePath(archivePath, name), Valid: true},
		}); err != nil {
			return created, linked, err
		}
		linked++
	}
	return created, linked, nil
}

func (s *Scanner) ingestEpubBook(ctx context.Context, mediaID int64, mediaExternalID, epubPath string, startIdx int) (created bool, linked int, err error) {
	chapters, err := ParseEpubSpine(epubPath)
	if err != nil {
		return false, 0, err
	}
	for i, c := range chapters {
		ch, chCreated, err := s.upsertChapterRowNamed(ctx, mediaID, mediaExternalID, c.EntryPath, c.Title, startIdx+i)
		if err != nil {
			return created, linked, err
		}
		if chCreated {
			created = true
		}
		if err := s.q.UpsertNovelChapterContent(ctx, sqlcgen.UpsertNovelChapterContentParams{
			ChapterID: ch.ID,
			LocalPath: sql.NullString{String: ZipPagePath(epubPath, c.EntryPath), Valid: true},
		}); err != nil {
			return created, linked, err
		}
		linked++
	}
	return created, linked, nil
}

func (s *Scanner) ingestDocxBook(ctx context.Context, mediaID int64, mediaExternalID, docxPath string, idx int) (created bool, linked int, err error) {
	key := filepath.Base(docxPath)
	title := strings.TrimSuffix(key, filepath.Ext(key))
	ch, created, err := s.upsertChapterRowNamed(ctx, mediaID, mediaExternalID, key, title, idx)
	if err != nil {
		return created, 0, err
	}
	if err := s.q.UpsertNovelChapterContent(ctx, sqlcgen.UpsertNovelChapterContentParams{
		ChapterID: ch.ID,
		LocalPath: sql.NullString{String: DocxBookPath(docxPath), Valid: true},
	}); err != nil {
		return created, 0, err
	}
	return created, 1, nil
}

func (s *Scanner) ingestChapter(ctx context.Context, mediaID int64, ct, mediaExternalID, chapName, chapDir string, idx int) (created bool, linked int, err error) {
	entries, err := os.ReadDir(chapDir)
	if err != nil {
		return false, 0, err
	}
	files := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			files = append(files, e.Name())
		}
	}
	sort.Sort(naturalSort(files))

	ch, created, err := s.upsertChapterRow(ctx, mediaID, mediaExternalID, chapName, idx)
	if err != nil {
		return created, 0, err
	}

	switch ct {
	case "manga":

		if err := s.q.DeleteMangaPages(ctx, ch.ID); err != nil {
			return created, 0, err
		}
		page := 0
		for _, f := range files {
			if !imageExts[strings.ToLower(filepath.Ext(f))] {
				continue
			}
			page++
			if err := s.q.UpsertMangaPage(ctx, sqlcgen.UpsertMangaPageParams{
				ChapterID:  ch.ID,
				PageNumber: int64(page),
				LocalPath:  sql.NullString{String: filepath.Join(chapDir, f), Valid: true},
			}); err != nil {
				return created, linked, err
			}
			linked++
		}
	case "novel":
		if f := firstMatch(files, textExts); f != "" {
			if err := s.q.UpsertNovelChapterContent(ctx, sqlcgen.UpsertNovelChapterContentParams{
				ChapterID: ch.ID,
				LocalPath: sql.NullString{String: filepath.Join(chapDir, f), Valid: true},
			}); err != nil {
				return created, linked, err
			}
			linked++
		}
	case "anime":
		if f := firstMatch(files, videoExts); f != "" {
			if err := s.q.UpsertAnimeEpisodeStream(ctx, sqlcgen.UpsertAnimeEpisodeStreamParams{
				ChapterID: ch.ID,
				StreamUrl: sql.NullString{},
				LocalPath: sql.NullString{String: filepath.Join(chapDir, f), Valid: true},
			}); err != nil {
				return created, linked, err
			}
			linked++
		}
	}
	return created, linked, nil
}

func (s *Scanner) upsertMedia(ctx context.Context, externalID, ct, title, cover string) (sqlcgen.Medium, error) {
	var coverArg sql.NullString
	if cover != "" {
		coverArg = sql.NullString{String: cover, Valid: true}
	}
	existing, err := s.q.GetLocalMediaByExternalID(ctx, externalID)
	if err == sql.ErrNoRows {
		created, err := s.q.CreateLocalMedia(ctx, sqlcgen.CreateLocalMediaParams{
			ExternalID:     externalID,
			ContentType:    ct,
			Title:          title,
			CoverLocalPath: coverArg,
		})
		if err == nil && s.enricher != nil {
			s.maybeEnrich(created.ID)
		}
		return created, err
	}
	if err != nil {
		return sqlcgen.Medium{}, err
	}
	return s.q.UpdateLocalMedia(ctx, sqlcgen.UpdateLocalMediaParams{
		Title:          title,
		CoverLocalPath: coverArg,
		ID:             existing.ID,
	})
}

// maybeEnrich runs a best-effort metadata auto-match for a newly-discovered
// local title in the background, so scanning doesn't block on network calls.
func (s *Scanner) maybeEnrich(mediaID int64) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := s.enricher.AutoEnrich(ctx, mediaID); err != nil {
			log.Printf("localsource: metadata auto-enrich for media %d: %v", mediaID, err)
		}
	}()
}

func gone(path string) bool {
	if path == "" {
		return true
	}
	if archivePath, entryName, ok := ParseZipPagePath(path); ok {
		if _, err := os.Stat(archivePath); err != nil {
			return os.IsNotExist(err)
		}
		return !ArchiveHasEntry(archivePath, entryName)
	}
	if docxPath, ok := ParseDocxBookPath(path); ok {
		_, err := os.Stat(docxPath)
		return err != nil && os.IsNotExist(err)
	}
	_, err := os.Stat(path)
	return err != nil && os.IsNotExist(err)
}

func (s *Scanner) prune(ctx context.Context) (int, error) {
	n := 0

	pages, err := s.q.ListAllMangaPagePaths(ctx)
	if err != nil {
		return n, err
	}
	for _, r := range pages {
		if !gone(r.LocalPath.String) {
			continue
		}
		if err := s.q.DeleteMangaPage(ctx, sqlcgen.DeleteMangaPageParams{
			ChapterID: r.ChapterID, PageNumber: r.PageNumber,
		}); err != nil {
			return n, err
		}
		n++
	}

	novels, err := s.q.ListAllNovelContentPaths(ctx)
	if err != nil {
		return n, err
	}
	for _, r := range novels {
		if !gone(r.LocalPath.String) {
			continue
		}
		if err := s.q.DeleteNovelChapterContent(ctx, r.ChapterID); err != nil {
			return n, err
		}
		n++
	}

	streams, err := s.q.ListAllEpisodeStreamPaths(ctx)
	if err != nil {
		return n, err
	}
	for _, r := range streams {
		if !gone(r.LocalPath.String) {
			continue
		}
		if err := s.q.DeleteAnimeEpisodeStream(ctx, r.ChapterID); err != nil {
			return n, err
		}
		n++
	}

	return n, nil
}

var coverFilenames = []string{"cover.jpg", "cover.jpeg", "cover.png", "cover.webp", "cover.avif"}

func findCover(dir string) string {
	for _, name := range coverFilenames {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func isCoverFilename(name string) bool {
	lower := strings.ToLower(name)
	for _, c := range coverFilenames {
		if lower == c {
			return true
		}
	}
	return false
}

func firstMatch(files []string, exts map[string]bool) string {
	for _, f := range files {
		if exts[strings.ToLower(filepath.Ext(f))] {
			return f
		}
	}
	return ""
}

func parseLeadingNumber(s string) float64 {
	start := -1
	for i, r := range s {
		if unicode.IsDigit(r) {
			start = i
			break
		}
	}
	if start < 0 {
		return -1
	}
	end := start
	seenDot := false
	for end < len(s) {
		c := s[end]
		if c >= '0' && c <= '9' {
			end++
			continue
		}
		if c == '.' && !seenDot && end+1 < len(s) && s[end+1] >= '0' && s[end+1] <= '9' {
			seenDot = true
			end++
			continue
		}
		break
	}
	v, err := strconv.ParseFloat(s[start:end], 64)
	if err != nil {
		return -1
	}
	return v
}

type naturalSort []string

func (n naturalSort) Len() int      { return len(n) }
func (n naturalSort) Swap(i, j int) { n[i], n[j] = n[j], n[i] }
func (n naturalSort) Less(i, j int) bool {
	a, b := n[i], n[j]
	for len(a) > 0 && len(b) > 0 {
		ad, bd := a[0] >= '0' && a[0] <= '9', b[0] >= '0' && b[0] <= '9'
		if ad && bd {
			ai, an := leadingDigits(a)
			bi, bn := leadingDigits(b)
			if ai != bi {
				return ai < bi
			}
			a, b = an, bn
			continue
		}
		if a[0] != b[0] {
			return a[0] < b[0]
		}
		a, b = a[1:], b[1:]
	}
	return len(a) < len(b)
}

func leadingDigits(s string) (int64, string) {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	v, _ := strconv.ParseInt(s[:i], 10, 64)
	return v, s[i:]
}
