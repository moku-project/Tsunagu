package localsource

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"tsunagu/backend/internal/db/sqlcgen"
)

type Scanner struct {
	q        *sqlcgen.Queries
	mediaDir string
}

func New(q *sqlcgen.Queries, mediaDir string) *Scanner {
	return &Scanner{q: q, mediaDir: mediaDir}
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

var imageExts = map[string]bool{
	".webp": true, ".jpg": true, ".jpeg": true, ".png": true,
	".gif": true, ".avif": true, ".bmp": true,
}
var textExts = map[string]bool{
	".txt": true, ".html": true, ".htm": true, ".xhtml": true, ".epub": true, ".md": true,
}
var videoExts = map[string]bool{
	".mp4": true, ".mkv": true, ".webm": true, ".m4v": true, ".avi": true, ".mov": true,
}

func (s *Scanner) Scan(ctx context.Context) (Result, error) {
	var res Result

	root := filepath.Join(s.mediaDir, "local")
	if fi, err := os.Stat(root); err == nil && fi.IsDir() {
		kinds, _ := os.ReadDir(root)
		for _, kd := range kinds {
			if !kd.IsDir() {
				continue
			}
			ct, ok := dirKind[strings.ToLower(kd.Name())]
			if !ok {
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

		chapDirs, _ := os.ReadDir(titleDir)
		names := make([]string, 0, len(chapDirs))
		for _, cd := range chapDirs {
			if cd.IsDir() {
				names = append(names, cd.Name())
			}
		}
		sort.Sort(naturalSort(names))

		for idx, chapName := range names {
			chapDir := filepath.Join(titleDir, chapName)
			created, linked, err := s.ingestChapter(ctx, media.ID, ct, externalID, chapName, chapDir, idx)
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

	chExternalID := mediaExternalID + "/" + chapName
	_, getErr := s.q.GetChapterByMediaAndExternalID(ctx, sqlcgen.GetChapterByMediaAndExternalIDParams{
		MediaID:    mediaID,
		ExternalID: chExternalID,
	})
	created = getErr == sql.ErrNoRows

	num := parseLeadingNumber(chapName)
	if num < 0 {
		num = float64(idx + 1)
	}
	ch, err := s.q.CreateChapter(ctx, sqlcgen.CreateChapterParams{
		MediaID:     mediaID,
		ExternalID:  chExternalID,
		Title:       sql.NullString{String: chapName, Valid: true},
		Number:      sql.NullFloat64{Float64: num, Valid: true},
		SourceOrder: sql.NullInt64{Int64: int64(idx), Valid: true},
	})
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
		return s.q.CreateLocalMedia(ctx, sqlcgen.CreateLocalMediaParams{
			ExternalID:     externalID,
			ContentType:    ct,
			Title:          title,
			CoverLocalPath: coverArg,
		})
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

func gone(path string) bool {
	if path == "" {
		return true
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

func findCover(dir string) string {
	for _, name := range []string{"cover.jpg", "cover.jpeg", "cover.png", "cover.webp", "cover.avif"} {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
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
