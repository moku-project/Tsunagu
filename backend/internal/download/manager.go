package download

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

	wakeCh    chan struct{}
	stopCh    chan struct{}
	wg        sync.WaitGroup
	runningMu sync.Mutex
	running   map[int64]context.CancelFunc

	pausedMu sync.RWMutex
	paused   bool
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
			if !m.IsPaused() {
				m.processOne()
			}
		case <-m.wakeCh:
			if !m.IsPaused() {
				m.processOne()
			}
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

	err = m.runJob(jobCtx, job.ID, job.ChapterID)
	if jobCtx.Err() != nil {

		log.Printf("download: job %d (chapter %d) cancelled; discarding partial output", job.ID, job.ChapterID)
		m.discardPartial(job.ChapterID)
		_ = m.q.DeleteDownloadByChapter(context.Background(), job.ChapterID)
		return
	}
	if err != nil {
		log.Printf("download: job %d (chapter %d) failed: %v", job.ID, job.ChapterID, err)
		m.discardPartial(job.ChapterID)
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

func contentTypeDir(contentType string) string {
	switch contentType {
	case "novel":
		return "novels"
	default:
		return contentType
	}
}

func folderSegment(primary, fallback string) string {
	if name := sanitizeFilename(primary); name != "" {
		return name
	}
	return safeSegment(fallback)
}

func extensionFolderName(pkg string) string {
	return folderSegment(pkg, pkg)
}

func titleFolderName(title, fallbackID string) string {
	return folderSegment(title, fallbackID)
}

func (m *Manager) buildChapterDir(dctx sqlcgen.GetChapterDownloadContextRow) string {
	return filepath.Join(
		m.mediaDir,
		contentTypeDir(dctx.ContentType),
		extensionFolderName(dctx.ExtensionPackageName),
		titleFolderName(dctx.LibraryTitle, dctx.SourceEntryID),
		chapterFolderName(dctx.ContentType, dctx.ChapterTitle, dctx.ChapterNumber, dctx.SourceChapterID),
	)
}

func (m *Manager) removeEmptyDirs(dir string) {
	for {
		clean := filepath.Clean(dir)
		if clean == filepath.Clean(m.mediaDir) || clean == "." || clean == string(filepath.Separator) {
			return
		}
		if err := os.Remove(dir); err != nil {
			return
		}
		dir = filepath.Dir(dir)
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

	chapterDir := m.buildChapterDir(dctx)
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

	chapterDir := m.buildChapterDir(dctx)
	if err := os.MkdirAll(chapterDir, 0o755); err != nil {
		return fmt.Errorf("creating chapter dir: %w", err)
	}

	ext := ".html"
	if text.GetFormat() != "" && text.GetFormat() != "html" {
		ext = "." + text.GetFormat()
	}
	localPath := filepath.Join(chapterDir, "content"+ext)
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

var illegalFilenameChars = regexp.MustCompile(`[<>:"/\\|?*{}\x00-\x1f]`)

var looksLikeStructuredDataRe = regexp.MustCompile(`^\\s*[\\{\\[]`)

func sanitizeFilename(s string) string {
	if looksLikeStructuredDataRe.MatchString(s) {
		return ""
	}
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

func formatChapterNumber(n float64) string {
	if n == math.Trunc(n) {
		return strconv.FormatFloat(n, 'f', 0, 64)
	}
	return strconv.FormatFloat(n, 'f', -1, 64)
}

func chapterFolderName(contentType string, title sql.NullString, number sql.NullFloat64, fallbackID string) string {
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

	chapterDir := m.buildChapterDir(dctx)
	if err := os.MkdirAll(chapterDir, 0o755); err != nil {
		return fmt.Errorf("creating chapter dir: %w", err)
	}
	outputPath := filepath.Join(chapterDir, "episode.mp4")

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

	inputArgs := []string{"-y"}

	{
		sb := strings.Builder{}
		sb.WriteString("Accept-Encoding: identity\r\n")
		for k, v := range stream.GetHeaders() {
			switch strings.ToLower(k) {
			case "user-agent":
				inputArgs = append(inputArgs, "-user_agent", v)
			case "referer", "referrer":
				inputArgs = append(inputArgs, "-referer", v)
			case "accept-encoding":
			default:
				sb.WriteString(k)
				sb.WriteString(": ")
				sb.WriteString(v)
				sb.WriteString("\r\n")
			}
		}
		inputArgs = append(inputArgs, "-headers", sb.String())
	}

	if strings.Contains(streamURL, ".m3u8") || strings.Contains(streamURL, "/m3u8?") {
		inputArgs = append(inputArgs,
			"-protocol_whitelist", "file,http,https,tcp,tls,crypto,data",
			"-allowed_extensions", "ALL",
			"-allowed_segment_extensions", "ALL",
		)
	}

	inputArgs = append(inputArgs,
		"-reconnect", "1",
		"-reconnect_streamed", "1",
		"-reconnect_delay_max", "5",
		"-reconnect_on_network_error", "1",
		"-analyzeduration", "20M",
		"-probesize", "20M",
		"-fflags", "+genpts+igndts+discardcorrupt",
		"-err_detect", "ignore_err",
	)
	inputArgs = append(inputArgs, "-i", streamURL)

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

	passes := [][]string{
		{"-map", "0:v:0?", "-map", "0:a:0?", "-c", "copy", "-bsf:a", "aac_adtstoasc", "-movflags", "+faststart", outputPath},
		{"-map", "0:v:0?", "-map", "0:a:0?", "-c:v", "libx264", "-preset", "veryfast", "-crf", "21", "-c:a", "aac", "-b:a", "160k", "-movflags", "+faststart", outputPath},
	}

	var lastErr error
	var lastOut string
	for i, out := range passes {
		full := append([]string{"-progress", progressPath, "-nostats"}, inputArgs...)
		full = append(full, out...)

		cmd := exec.CommandContext(ctx, "ffmpeg", full...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		stopProgress := make(chan struct{})
		go m.watchFfmpegProgress(jobID, progressPath, stopProgress)

		lastErr = cmd.Run()
		close(stopProgress)
		_ = os.Remove(progressPath)

		if lastErr == nil {
			if fi, statErr := os.Stat(outputPath); statErr == nil && fi.Size() > 0 {
				break
			}
			lastErr = fmt.Errorf("ffmpeg reported success but produced no output")
		}
		lastOut = truncateForError(stderr.String())
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if i == 0 {
			log.Printf("download: job %d stream-copy failed, retrying with transcode", jobID)
			_ = os.Remove(outputPath)
		}
	}
	err = lastErr

	if err != nil {
		if segErr := m.muxViaSegments(ctx, jobID, streamURL, stream.GetHeaders(), outputPath); segErr != nil {
			return fmt.Errorf("ffmpeg failed: %w (output: %s); segment fallback: %v", err, lastOut, segErr)
		}
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

func (m *Manager) muxViaSegments(ctx context.Context, jobID int64, playlistURL string, headers map[string]string, outputPath string) error {
	setHdrs := func(req *http.Request) {
		req.Header.Set("Accept", "*/*")
		req.Header.Set("Accept-Encoding", "identity")
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, playlistURL, nil)
	if err != nil {
		return err
	}
	setHdrs(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch playlist: %w", err)
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("playlist HTTP %d", resp.StatusCode)
	}

	base, _ := url.Parse(playlistURL)
	var segs []string
	sc := bufio.NewScanner(bytes.NewReader(body))
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.ContainsRune(line, '�') {
			return fmt.Errorf("playlist looks corrupted (binary/compressed response not decoded)")
		}
		if u, uErr := url.Parse(line); uErr == nil {
			segs = append(segs, base.ResolveReference(u).String())
		}
	}
	if len(segs) == 0 {
		return fmt.Errorf("no segments in playlist")
	}

	dir := filepath.Dir(outputPath)
	rawPath := filepath.Join(dir, "raw.ts")
	raw, err := os.Create(rawPath)
	if err != nil {
		return err
	}
	defer os.Remove(rawPath)

	for i, seg := range segs {
		if ctx.Err() != nil {
			raw.Close()
			return ctx.Err()
		}
		sreq, rErr := http.NewRequestWithContext(ctx, http.MethodGet, seg, nil)
		if rErr != nil {
			raw.Close()
			return rErr
		}
		setHdrs(sreq)
		sresp, sErr := http.DefaultClient.Do(sreq)
		if sErr != nil {
			raw.Close()
			return fmt.Errorf("segment %d: %w", i, sErr)
		}
		if sresp.StatusCode != http.StatusOK {
			sresp.Body.Close()
			raw.Close()
			return fmt.Errorf("segment %d: HTTP %d", i, sresp.StatusCode)
		}
		_, cErr := io.Copy(raw, sresp.Body)
		sresp.Body.Close()
		if cErr != nil {
			raw.Close()
			return fmt.Errorf("segment %d write: %w", i, cErr)
		}
		if (i+1)%10 == 0 || i+1 == len(segs) {
			_ = m.q.UpdateDownloadProgress(ctx, sqlcgen.UpdateDownloadProgressParams{
				Status:   "downloading",
				Progress: 0.1 + 0.7*float64(i+1)/float64(len(segs)),
				ID:       jobID,
			})
		}
	}
	if err := raw.Close(); err != nil {
		return err
	}

	common := []string{
		"-y",
		"-analyzeduration", "40M", "-probesize", "40M",
		"-fflags", "+genpts+igndts+discardcorrupt", "-err_detect", "ignore_err",
		"-i", rawPath,
	}
	for _, out := range [][]string{
		{"-c", "copy", "-bsf:a", "aac_adtstoasc", "-movflags", "+faststart", outputPath},
		{"-c:v", "libx264", "-preset", "veryfast", "-crf", "21", "-c:a", "aac", "-b:a", "160k", "-movflags", "+faststart", outputPath},
	} {
		var stderr bytes.Buffer
		cmd := exec.CommandContext(ctx, "ffmpeg", append(append([]string{}, common...), out...)...)
		cmd.Stderr = &stderr
		if runErr := cmd.Run(); runErr == nil {
			if fi, sErr := os.Stat(outputPath); sErr == nil && fi.Size() > 0 {
				return nil
			}
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		_ = os.Remove(outputPath)
	}
	return fmt.Errorf("ffmpeg could not mux %d concatenated segments", len(segs))
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

func (m *Manager) discardPartial(chapterID int64) {
	ctx := context.Background()
	dctx, err := m.q.GetChapterDownloadContext(ctx, chapterID)
	if err != nil {
		log.Printf("download: cleanup for chapter %d: %v", chapterID, err)
		return
	}
	dir := m.buildChapterDir(dctx)
	if err := os.RemoveAll(dir); err != nil {
		log.Printf("download: cleanup: removing %s: %v", dir, err)
	}
	m.removeEmptyDirs(filepath.Dir(dir))
	_ = m.q.DeleteMangaPages(ctx, chapterID)
	_ = m.q.DeleteNovelChapterContent(ctx, chapterID)
	_ = m.q.DeleteAnimeEpisodeStream(ctx, chapterID)
}

func (m *Manager) Pause() {
	m.pausedMu.Lock()
	m.paused = true
	m.pausedMu.Unlock()
}

func (m *Manager) Resume() {
	m.pausedMu.Lock()
	m.paused = false
	m.pausedMu.Unlock()
	m.Wake()
}

func (m *Manager) IsPaused() bool {
	m.pausedMu.RLock()
	defer m.pausedMu.RUnlock()
	return m.paused
}

func (m *Manager) Reorder(ctx context.Context, chapterID int64, newPosition int64) error {
	target, err := m.q.GetQueuedDownloadByChapter(ctx, chapterID)
	if err != nil {
		return fmt.Errorf("chapter %d has no queued download: %w", chapterID, err)
	}

	queued, err := m.q.ListQueuedDownloads(ctx)
	if err != nil {
		return fmt.Errorf("listing queued downloads: %w", err)
	}

	reordered := make([]int64, 0, len(queued))
	for _, d := range queued {
		if d.ID != target.ID {
			reordered = append(reordered, d.ID)
		}
	}

	if newPosition < 1 {
		newPosition = 1
	}
	if newPosition > int64(len(reordered))+1 {
		newPosition = int64(len(reordered)) + 1
	}
	insertAt := int(newPosition) - 1

	final := make([]int64, 0, len(reordered)+1)
	final = append(final, reordered[:insertAt]...)
	final = append(final, target.ID)
	final = append(final, reordered[insertAt:]...)

	for i, id := range final {
		if err := m.q.SetDownloadPosition(ctx, sqlcgen.SetDownloadPositionParams{
			Position: sql.NullInt64{Int64: int64(i + 1), Valid: true},
			ID:       id,
		}); err != nil {
			return fmt.Errorf("updating position for download %d: %w", id, err)
		}
	}

	return nil
}

func (m *Manager) ClearQueue(ctx context.Context) error {
	return m.q.ClearDownloads(ctx)
}

func (m *Manager) DeleteChapterFiles(ctx context.Context, chapterID int64) error {
	dctx, err := m.q.GetChapterDownloadContext(ctx, chapterID)
	if err != nil {
		return fmt.Errorf("resolving chapter context: %w", err)
	}

	switch dctx.ContentType {
	case "manga":
		pages, err := m.q.ListMangaPages(ctx, chapterID)
		if err != nil {
			return fmt.Errorf("listing manga pages: %w", err)
		}
		var chapterDir string
		for _, p := range pages {
			if p.LocalPath.Valid {
				if chapterDir == "" {
					chapterDir = filepath.Dir(p.LocalPath.String)
				}
				if err := os.Remove(p.LocalPath.String); err != nil && !os.IsNotExist(err) {
					log.Printf("download: failed removing page file %s: %v", p.LocalPath.String, err)
				}
			}
		}
		if chapterDir != "" {
			m.removeEmptyDirs(chapterDir)
		}
		if err := m.q.DeleteMangaPages(ctx, chapterID); err != nil {
			return fmt.Errorf("deleting manga page rows: %w", err)
		}

	case "novel":
		content, err := m.q.GetNovelChapterContent(ctx, chapterID)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil
			}
			return fmt.Errorf("fetching novel content: %w", err)
		}
		if content.LocalPath.Valid {
			if err := os.Remove(content.LocalPath.String); err != nil && !os.IsNotExist(err) {
				log.Printf("download: failed removing novel file %s: %v", content.LocalPath.String, err)
			}
			m.removeEmptyDirs(filepath.Dir(content.LocalPath.String))
		}
		if err := m.q.DeleteNovelChapterContent(ctx, chapterID); err != nil {
			return fmt.Errorf("deleting novel content row: %w", err)
		}

	case "anime":
		stream, err := m.q.GetAnimeEpisodeStream(ctx, chapterID)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil
			}
			return fmt.Errorf("fetching anime stream: %w", err)
		}
		if stream.LocalPath.Valid {
			if err := os.Remove(stream.LocalPath.String); err != nil && !os.IsNotExist(err) {
				log.Printf("download: failed removing anime file %s: %v", stream.LocalPath.String, err)
			}
			m.removeEmptyDirs(filepath.Dir(stream.LocalPath.String))
		}
		if err := m.q.DeleteAnimeEpisodeStream(ctx, chapterID); err != nil {
			return fmt.Errorf("deleting anime stream row: %w", err)
		}

	default:
		return fmt.Errorf("unsupported content type for delete: %s", dctx.ContentType)
	}

	return nil
}
