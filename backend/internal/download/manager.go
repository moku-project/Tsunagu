package download

import (
	"context"
	"crypto/sha1"
	"bytes"
	"bufio"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"io"
	"net/http"
	"math"
	"net/url"
	"regexp"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
	runningMu sync.Mutex
	running   map[int64]context.CancelFunc
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
		running:      make(map[int64]context.CancelFunc),
	}
}

func (m *Manager) Start() {
	if err := m.q.RequeueOrphanedDownloads(context.Background()); err != nil {
		log.Printf("download: requeue orphaned jobs failed: %v", err)
	}

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

	jobCtx, cancel := context.WithCancel(ctx)
	m.runningMu.Lock()
	m.running[job.ChapterID] = cancel
	m.runningMu.Unlock()
	defer func() {
		m.runningMu.Lock()
		delete(m.running, job.ChapterID)
		m.runningMu.Unlock()
		cancel()
	}()

	if err := m.runJob(jobCtx, job.ID, job.ChapterID); err != nil {
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
	case "anime":
		return m.downloadAnime(ctx, jobID, client, dctx)
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
		libraryFolderName(dctx.LibraryTitle, dctx.SourceEntryID),
		chapterLabel("manga", dctx.ChapterTitle, dctx.ChapterNumber, dctx.SourceChapterID),
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
		libraryFolderName(dctx.LibraryTitle, dctx.SourceEntryID),
	)
	if err := os.MkdirAll(novelDir, 0o755); err != nil {
		return fmt.Errorf("creating novel dir: %w", err)
	}

	ext := ".html"
	if text.GetFormat() != "" && text.GetFormat() != "html" {
		ext = "." + text.GetFormat()
	}
	localPath := filepath.Join(novelDir, chapterLabel("novel", dctx.ChapterTitle, dctx.ChapterNumber, dctx.SourceChapterID)+ext)
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

var illegalFilenameChars = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)

func sanitizeFilename(s string) string {
	s = strings.TrimSpace(s)
	s = illegalFilenameChars.ReplaceAllString(s, "_")
	s = strings.Trim(s, " .")
	if s == "" {
		return ""
	}
	if len(s) > 150 {
		s = s[:150]
	}
	return s
}

func libraryFolderName(title, fallbackID string) string {
	if name := sanitizeFilename(title); name != "" {
		return name
	}
	return safeSegment(fallbackID)
}

func formatChapterNumber(n float64) string {
	if n == math.Trunc(n) {
		return strconv.FormatFloat(n, 'f', 0, 64)
	}
	return strconv.FormatFloat(n, 'f', -1, 64)
}

func chapterLabel(contentType string, title sql.NullString, number sql.NullFloat64, fallbackID string) string {
	noun := "Chapter"
	if contentType == "anime" {
		noun = "Episode"
	}

	hasTitle := title.Valid && sanitizeFilename(title.String) != ""

	if hasTitle && number.Valid {
		numStr := formatChapterNumber(number.Float64)
		prefixPattern := regexp.MustCompile(`(?i)^` + regexp.QuoteMeta(noun) + `\s+0*` + regexp.QuoteMeta(numStr) + `\b`)
		if prefixPattern.MatchString(strings.TrimSpace(title.String)) {
			return sanitizeFilename(title.String)
		}
	}

	switch {
	case hasTitle && number.Valid:
		return sanitizeFilename(fmt.Sprintf("%s %s - %s", noun, formatChapterNumber(number.Float64), title.String))
	case hasTitle:
		return sanitizeFilename(title.String)
	case number.Valid:
		return sanitizeFilename(fmt.Sprintf("%s %s", noun, formatChapterNumber(number.Float64)))
	default:
		return safeSegment(fallbackID)
	}
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

func (m *Manager) downloadAnime(ctx context.Context, jobID int64, client *sandbox.Client, dctx sqlcgen.GetChapterDownloadContextRow) error {
	stream, err := client.GetVideoStream(ctx, dctx.ExtensionPackageName, dctx.SourceEntryID, dctx.SourceChapterID)
	if err != nil {
		return fmt.Errorf("fetching video stream: %w", err)
	}
	streamURL := stream.GetStreamUrl()
	if streamURL == "" {
		return fmt.Errorf("source returned no video stream")
	}

	animeDir := filepath.Join(
		m.mediaDir, "anime",
		libraryFolderName(dctx.LibraryTitle, dctx.SourceEntryID),
	)
	if err := os.MkdirAll(animeDir, 0o755); err != nil {
		return fmt.Errorf("creating anime dir: %w", err)
	}
	outputPath := filepath.Join(animeDir, chapterLabel("anime", dctx.ChapterTitle, dctx.ChapterNumber, dctx.SourceChapterID)+".mp4")

	keepaliveCtx, stopKeepalive := context.WithCancel(ctx)
	defer stopKeepalive()
	go m.keepSandboxAlive(keepaliveCtx)

	if err := m.q.UpdateDownloadProgress(ctx, sqlcgen.UpdateDownloadProgressParams{
		Status:   "downloading",
		Progress: 0.1,
		ID:       jobID,
	}); err != nil {
		log.Printf("download: progress update failed for job %d: %v", jobID, err)
	}

	args := []string{"-y"}
	if headers := stream.GetHeaders(); len(headers) > 0 {
		var sb strings.Builder
		for k, v := range headers {
			sb.WriteString(k)
			sb.WriteString(": ")
			sb.WriteString(v)
			sb.WriteString("\r\n")
		}
		args = append(args, "-headers", sb.String())
	}
	if strings.Contains(streamURL, ".m3u8") {
		args = append(args, "-allowed_extensions", "ALL", "-allowed_segment_extensions", "ALL")
	}

	args = append(args,
		"-reconnect", "1",
		"-reconnect_streamed", "1",
		"-reconnect_delay_max", "5",
		"-reconnect_on_network_error", "1",
	)
	args = append(args, "-i", streamURL, "-c", "copy", outputPath)

	if debugPath := os.Getenv("TSUNAGU_DEBUG_ANIME_M3U8"); debugPath != "" {
		if resp, dErr := http.Get(streamURL); dErr == nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			_ = os.WriteFile(debugPath, body, 0o644)
			log.Printf("download: wrote debug m3u8 (%d bytes, status %d) to %s", len(body), resp.StatusCode, debugPath)
		} else {
			log.Printf("download: debug m3u8 fetch failed: %v", dErr)
		}
	}

	progressPath := outputPath + ".progress"
	args = append([]string{"-progress", progressPath, "-nostats"}, args...)

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stopProgress := make(chan struct{})
	go m.watchFfmpegProgress(jobID, progressPath, stopProgress)

	err = cmd.Run()
	close(stopProgress)
	_ = os.Remove(progressPath)

	if err != nil {
		return fmt.Errorf("ffmpeg failed: %w (output: %s)", err, truncateForError(stderr.String()))
	}

	if _, err := os.Stat(outputPath); err != nil {
		return fmt.Errorf("ffmpeg reported success but output file missing: %w", err)
	}

	if err := m.q.UpsertAnimeEpisodeStream(ctx, sqlcgen.UpsertAnimeEpisodeStreamParams{
		ChapterID: dctx.ChapterID,
		StreamUrl: sql.NullString{String: streamURL, Valid: true},
		LocalPath: sql.NullString{String: outputPath, Valid: true},
	}); err != nil {
		return fmt.Errorf("recording episode stream: %w", err)
	}

	return m.q.UpdateDownloadProgress(ctx, sqlcgen.UpdateDownloadProgressParams{
		Status:   "downloading",
		Progress: 1,
		ID:       jobID,
	})
}

func (m *Manager) keepSandboxAlive(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := m.sc.Ensure(ctx); err != nil {
				log.Printf("download: sandbox keepalive failed: %v", err)
			}
		}
	}
}

func truncateForError(s string) string {
	const max = 500
	if len(s) <= max {
		return s
	}
	return s[len(s)-max:]
}

func (m *Manager) watchFfmpegProgress(jobID int64, progressPath string, stop <-chan struct{}) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	ctx := context.Background()

	var lastSize int64
	var lastSampleAt time.Time

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			totalSize, outTimeMs := readFfmpegProgressFile(progressPath)
			if totalSize == 0 && outTimeMs == 0 {
				continue
			}

			var bytesPerSec float64
			now := time.Now()
			if !lastSampleAt.IsZero() && totalSize > lastSize {
				if elapsed := now.Sub(lastSampleAt).Seconds(); elapsed > 0 {
					bytesPerSec = float64(totalSize-lastSize) / elapsed
				}
			}
			lastSize = totalSize
			lastSampleAt = now

			progress := 0.1
			if totalSize > 0 {
				progress = float64(totalSize) / (300 * 1024 * 1024)
				if progress > 0.95 {
					progress = 0.95
				}
				if progress < 0.1 {
					progress = 0.1
				}
			}
			if err := m.q.UpdateDownloadStats(ctx, sqlcgen.UpdateDownloadStatsParams{
				Status:          "downloading",
				Progress:        progress,
				DownloadedBytes: sql.NullInt64{Int64: totalSize, Valid: totalSize > 0},
				BytesPerSec:     sql.NullFloat64{Float64: bytesPerSec, Valid: bytesPerSec > 0},
				ID:              jobID,
			}); err != nil {
				log.Printf("download: progress update failed for job %d: %v", jobID, err)
			}
		}
	}
}

func readFfmpegProgressFile(path string) (totalSize int64, outTimeMs int64) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "total_size="):
			v := strings.TrimPrefix(line, "total_size=")
			if n, err := parseInt64(v); err == nil {
				totalSize = n
			}
		case strings.HasPrefix(line, "out_time_ms="):
			v := strings.TrimPrefix(line, "out_time_ms=")
			if n, err := parseInt64(v); err == nil {
				outTimeMs = n
			}
		}
	}
	return totalSize, outTimeMs
}

func parseInt64(s string) (int64, error) {
	var n int64
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

func (m *Manager) Cancel(ctx context.Context, chapterID int64) error {
	m.runningMu.Lock()
	cancel, ok := m.running[chapterID]
	m.runningMu.Unlock()
	if ok {
		cancel()
	}
	return nil
}
