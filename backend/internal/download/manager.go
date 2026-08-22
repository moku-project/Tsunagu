package download

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"tsunagu/backend/internal/db/sqlcgen"
	"tsunagu/backend/internal/sandbox"
)

type Manager struct {
	q        *sqlcgen.Queries
	sc       *sandbox.SupervisedClient
	mediaDir string

	pollInterval time.Duration
	workers      int

	wakeCh chan struct{}
	stopCh chan struct{}
	wg     sync.WaitGroup
}

func New(q *sqlcgen.Queries, sc *sandbox.SupervisedClient, mediaDir string) *Manager {
	return &Manager{
		q:            q,
		sc:           sc,
		mediaDir:     mediaDir,
		pollInterval: 2 * time.Second,
		workers:      2,
		wakeCh:       make(chan struct{}, 1),
		stopCh:       make(chan struct{}),
	}
}

func (m *Manager) Start() {
	for i := 0; i < m.workers; i++ {
		m.wg.Add(1)
		go m.workerLoop()
	}
}

func (m *Manager) Shutdown() {
	close(m.stopCh)
	m.wg.Wait()
}

func (m *Manager) Wake() {
	select {
	case m.wakeCh <- struct{}{}:
	default:
	}
}

func (m *Manager) workerLoop() {
	defer m.wg.Done()
	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.processOne()
		case <-m.wakeCh:
			m.processOne()
		}
	}
}

func (m *Manager) processOne() {
	ctx := context.Background()

	jobs, err := m.q.ListQueuedDownloads(ctx)
	if err != nil {
		log.Printf("download: list queued failed: %v", err)
		return
	}
	if len(jobs) == 0 {
		return
	}
	job := jobs[0]

	if err := m.q.UpdateDownloadProgress(ctx, sqlcgen.UpdateDownloadProgressParams{
		Status:   "downloading",
		Progress: 0,
		ID:       job.ID,
	}); err != nil {
		log.Printf("download: claim job %d failed: %v", job.ID, err)
		return
	}

	if err := m.runJob(ctx, job.ID, job.ChapterID); err != nil {
		log.Printf("download: job %d (chapter %d) failed: %v", job.ID, job.ChapterID, err)
		_ = m.q.FailDownload(ctx, sqlcgen.FailDownloadParams{
			Error: sql.NullString{String: err.Error(), Valid: true},
			ID:    job.ID,
		})
		return
	}

	if err := m.q.CompleteDownload(ctx, job.ID); err != nil {
		log.Printf("download: mark complete failed for job %d: %v", job.ID, err)
	}
}

func (m *Manager) runJob(ctx context.Context, jobID, chapterID int64) error {
	dctx, err := m.q.GetChapterDownloadContext(ctx, chapterID)
	if err != nil {
		return fmt.Errorf("resolving chapter context: %w", err)
	}

	client, err := m.sc.Ensure(ctx)
	if err != nil {
		return fmt.Errorf("sandbox unavailable: %w", err)
	}

	switch dctx.ContentType {
	case "manga":
		return m.downloadManga(ctx, jobID, client, dctx)
	case "novel":
		return m.downloadNovel(ctx, jobID, client, dctx)
	default:
		return fmt.Errorf("unsupported content type for download: %s", dctx.ContentType)
	}
}

func (m *Manager) downloadManga(ctx context.Context, jobID int64, client *sandbox.Client, dctx sqlcgen.GetChapterDownloadContextRow) error {
	pages, err := client.GetPages(ctx, dctx.ExtensionPackageName, dctx.SourceEntryID, dctx.SourceChapterID)
	if err != nil {
		return fmt.Errorf("fetching page list: %w", err)
	}
	urls := pages.GetPageUrls()
	if len(urls) == 0 {
		return fmt.Errorf("source returned no pages")
	}

	chapterDir := filepath.Join(
		m.mediaDir, "manga",
		safeSegment(dctx.ExtensionPackageName),
		safeSegment(dctx.SourceEntryID),
		safeSegment(dctx.SourceChapterID),
	)
	if err := os.MkdirAll(chapterDir, 0o755); err != nil {
		return fmt.Errorf("creating chapter dir: %w", err)
	}

	total := len(urls)
	for i, pageURL := range urls {
		img, err := client.GetImageBytes(ctx, dctx.ExtensionPackageName, pageURL)
		if err != nil {
			return fmt.Errorf("fetching page %d: %w", i+1, err)
		}

		ext := extFromContentType(img.GetContentType())
		fileName := fmt.Sprintf("%03d%s", i+1, ext)
		localPath := filepath.Join(chapterDir, fileName)
		if err := os.WriteFile(localPath, img.GetData(), 0o644); err != nil {
			return fmt.Errorf("writing page %d: %w", i+1, err)
		}

		if err := m.q.UpsertMangaPage(ctx, sqlcgen.UpsertMangaPageParams{
			ChapterID:  dctx.ChapterID,
			PageNumber: int64(i + 1),
			LocalPath:  sql.NullString{String: localPath, Valid: true},
		}); err != nil {
			return fmt.Errorf("recording page %d: %w", i+1, err)
		}

		progress := float64(i+1) / float64(total)
		if err := m.q.UpdateDownloadProgress(ctx, sqlcgen.UpdateDownloadProgressParams{
			Status:   "downloading",
			Progress: progress,
			ID:       jobID,
		}); err != nil {
			log.Printf("download: progress update failed for job %d: %v", jobID, err)
		}
	}

	return nil
}

func (m *Manager) downloadNovel(ctx context.Context, jobID int64, client *sandbox.Client, dctx sqlcgen.GetChapterDownloadContextRow) error {
	text, err := client.GetChapterText(ctx, dctx.ExtensionPackageName, dctx.SourceEntryID, dctx.SourceChapterID)
	if err != nil {
		return fmt.Errorf("fetching chapter text: %w", err)
	}

	novelDir := filepath.Join(
		m.mediaDir, "novels",
		safeSegment(dctx.ExtensionPackageName),
		safeSegment(dctx.SourceEntryID),
	)
	if err := os.MkdirAll(novelDir, 0o755); err != nil {
		return fmt.Errorf("creating novel dir: %w", err)
	}

	ext := ".html"
	if text.GetFormat() != "" && text.GetFormat() != "html" {
		ext = "." + text.GetFormat()
	}
	localPath := filepath.Join(novelDir, safeSegment(dctx.SourceChapterID)+ext)
	if err := os.WriteFile(localPath, []byte(text.GetContent()), 0o644); err != nil {
		return fmt.Errorf("writing chapter content: %w", err)
	}

	if err := m.q.UpsertNovelChapterContent(ctx, sqlcgen.UpsertNovelChapterContentParams{
		ChapterID: dctx.ChapterID,
		LocalPath: sql.NullString{String: localPath, Valid: true},
	}); err != nil {
		return fmt.Errorf("recording chapter content: %w", err)
	}

	return m.q.UpdateDownloadProgress(ctx, sqlcgen.UpdateDownloadProgressParams{
		Status:   "downloading",
		Progress: 1,
		ID:       jobID,
	})
}

var safeSegmentRe = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

func safeSegment(raw string) string {
	s := raw
	if u, err := url.QueryUnescape(raw); err == nil {
		s = u
	}
	s = strings.TrimSpace(s)
	if s != "" && len(s) <= 120 && safeSegmentRe.MatchString(s) {
		return s
	}
	sum := sha1.Sum([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func extFromContentType(ct string) string {
	switch {
	case strings.Contains(ct, "png"):
		return ".png"
	case strings.Contains(ct, "webp"):
		return ".webp"
	case strings.Contains(ct, "gif"):
		return ".gif"
	default:
		return ".jpg"
	}
}
