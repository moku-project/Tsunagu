package rest

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"tsunagu/backend/internal/db/sqlcgen"
	"tsunagu/backend/internal/localsource"
	"tsunagu/backend/internal/sandbox"
	sandboxv1 "tsunagu/backend/internal/sandbox/gen/sandbox/v1"
	"tsunagu/backend/internal/streamresolve"
)

type ContentHandler struct {
	Q  *sqlcgen.Queries
	Sc *sandbox.SupervisedClient
	Sr *streamresolve.Resolver
}

func (h *ContentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/content/")
	parts := strings.SplitN(path, "/", 4)
	if len(parts) < 3 {
		http.Error(w, "expected /content/{mediaId}/{chapterId}/(pages/{n}|video|text)", http.StatusBadRequest)
		return
	}

	mediaID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "invalid media id", http.StatusBadRequest)
		return
	}
	chapterID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		http.Error(w, "invalid chapter id", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	ch, err := h.Q.GetChapter(ctx, chapterID)
	if err != nil {
		http.Error(w, "chapter not found", http.StatusNotFound)
		return
	}
	if ch.MediaID != mediaID {
		http.Error(w, fmt.Sprintf("chapter %d does not belong to media %d", chapterID, mediaID), http.StatusNotFound)
		return
	}

	switch parts[2] {
	case "pages":
		if len(parts) != 4 {
			http.Error(w, "expected /content/{mediaId}/{chapterId}/pages/{pageNumber}", http.StatusBadRequest)
			return
		}
		pageNumber, err := strconv.Atoi(parts[3])
		if err != nil || pageNumber < 1 {
			http.Error(w, "invalid page number", http.StatusBadRequest)
			return
		}
		h.servePage(w, r, chapterID, pageNumber)
	case "prefetch":
		h.prefetchChapter(w, r, chapterID)
	case "video":
		switch {
		case len(parts) == 4 && parts[3] == "probe":
			h.probeVideo(w, r, chapterID)
		case len(parts) == 4 && parts[3] == "events":
			h.videoEvents(w, r, chapterID)
		default:
			h.serveVideo(w, r, chapterID)
		}
	case "hls":
		h.serveHLS(w, r, chapterID)
	case "dash":
		if len(parts) < 4 {
			http.Error(w, "bad dash path", http.StatusBadRequest)
			return
		}
		h.serveDASH(w, r, parts[3], chapterID)
	case "subtitle":
		h.serveSubtitle(w, r, chapterID)
	case "text":
		h.serveText(w, r, chapterID)
	case "pdf":
		h.servePdf(w, r, chapterID)
	case "docx":
		h.serveDocx(w, r, chapterID)
	default:
		http.Error(w, "unknown content type", http.StatusNotFound)
	}
}

func (h *ContentHandler) servePdf(w http.ResponseWriter, r *http.Request, chapterID int64) {
	rows, err := h.Q.ListMangaPages(r.Context(), chapterID)
	if err != nil {
		http.Error(w, "chapter not found", http.StatusNotFound)
		return
	}
	for _, row := range rows {
		if !row.LocalPath.Valid {
			continue
		}
		archivePath, _, ok := localsource.ParseZipPagePath(row.LocalPath.String)
		if !ok || strings.ToLower(filepath.Ext(archivePath)) != ".pdf" {
			continue
		}
		if _, statErr := os.Stat(archivePath); statErr != nil {
			continue
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		http.ServeFile(w, r, archivePath)
		return
	}
	http.Error(w, "pdf not found", http.StatusNotFound)
}

func (h *ContentHandler) serveDocx(w http.ResponseWriter, r *http.Request, chapterID int64) {
	row, err := h.Q.GetNovelChapterContent(r.Context(), chapterID)
	if err != nil || !row.LocalPath.Valid {
		http.Error(w, "docx not found", http.StatusNotFound)
		return
	}
	docxPath, ok := localsource.ParseDocxBookPath(row.LocalPath.String)
	if !ok {
		http.Error(w, "not a docx chapter", http.StatusNotFound)
		return
	}
	if _, statErr := os.Stat(docxPath); statErr != nil {
		http.Error(w, "docx not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	http.ServeFile(w, r, docxPath)
}

func (h *ContentHandler) servePage(w http.ResponseWriter, r *http.Request, chapterID int64, pageNumber int) {
	ctx := r.Context()

	rows, err := h.Q.ListMangaPages(ctx, chapterID)
	if err == nil {
		for _, row := range rows {
			if int(row.PageNumber) != pageNumber || !row.LocalPath.Valid || row.LocalPath.String == "" {
				continue
			}
			if archivePath, entryName, ok := localsource.ParseZipPagePath(row.LocalPath.String); ok {
				if serveZipEntry(w, archivePath, entryName) {
					return
				}
				continue
			}
			if _, statErr := os.Stat(row.LocalPath.String); statErr == nil {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				http.ServeFile(w, r, row.LocalPath.String)
				return
			}
		}
	}

	dctx, err := h.Q.GetChapterDownloadContext(ctx, chapterID)
	if err != nil {
		http.Error(w, "chapter not found", http.StatusNotFound)
		return
	}
	if dctx.ContentType != "manga" {
		http.Error(w, "page content only supports manga chapters", http.StatusBadRequest)
		return
	}

	client, err := h.Sc.Ensure(ctx)
	if err != nil {
		http.Error(w, "sandbox unavailable", http.StatusServiceUnavailable)
		return
	}

	urls, err := client.GetPageURLs(ctx, dctx.ExtensionPackageName, dctx.SourceEntryID, dctx.SourceChapterID)
	if err != nil {
		http.Error(w, "fetching page list failed", http.StatusBadGateway)
		return
	}
	if pageNumber > len(urls) {
		http.Error(w, "page number out of range", http.StatusNotFound)
		return
	}

	data, ct, err := fetchMangaImage(ctx, client, dctx.ExtensionPackageName, urls[pageNumber-1])
	if err != nil {
		http.Error(w, "fetching image failed", http.StatusBadGateway)
		return
	}

	if end := pageNumber + 3; pageNumber < len(urls) {
		if end > len(urls) {
			end = len(urls)
		}
		prefetchMangaImages(client, dctx.ExtensionPackageName, urls[pageNumber:end])
	}

	if ct == "" {
		ct = "image/jpeg"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(data)
}

func (h *ContentHandler) prefetchChapter(w http.ResponseWriter, r *http.Request, chapterID int64) {
	if rows, err := h.Q.ListMangaPages(r.Context(), chapterID); err == nil {
		for _, row := range rows {
			if row.LocalPath.Valid && row.LocalPath.String != "" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
	}

	dctx, err := h.Q.GetChapterDownloadContext(r.Context(), chapterID)
	if err != nil || dctx.ContentType != "manga" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	client, err := h.Sc.Ensure(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		urls, err := client.GetPageURLs(ctx, dctx.ExtensionPackageName, dctx.SourceEntryID, dctx.SourceChapterID)
		if err != nil {
			return
		}
		k := 6
		if k > len(urls) {
			k = len(urls)
		}
		prefetchMangaImages(client, dctx.ExtensionPackageName, urls[:k])
	}()
	w.WriteHeader(http.StatusNoContent)
}

func (h *ContentHandler) serveVideo(w http.ResponseWriter, r *http.Request, chapterID int64) {
	ctx := r.Context()

	if row, err := h.Q.GetAnimeEpisodeStream(ctx, chapterID); err == nil && row.LocalPath.Valid && row.LocalPath.String != "" {
		if _, statErr := os.Stat(row.LocalPath.String); statErr == nil {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			http.ServeFile(w, r, row.LocalPath.String)
			return
		}
	}

	dctx, err := h.Q.GetChapterDownloadContext(ctx, chapterID)
	if err != nil {
		http.Error(w, "chapter not found", http.StatusNotFound)
		return
	}
	if dctx.ContentType != "anime" {
		http.Error(w, "video content only supports anime episodes", http.StatusBadRequest)
		return
	}

	info, err := h.Sr.Resolve(ctx, dctx.ExtensionPackageName, dctx.SourceEntryID, dctx.SourceChapterID)
	if err != nil {
		http.Error(w, "resolving stream failed", http.StatusBadGateway)
		return
	}

	streamURL, headers := info.GetStreamUrl(), info.GetHeaders()
	if q := r.URL.Query().Get("quality"); q != "" {
		if src := pickVideoSource(info.GetSources(), q); src != nil {
			streamURL = src.GetUrl()
			if len(src.GetHeaders()) > 0 {
				headers = src.GetHeaders()
			}
		}
	}
	if streamURL == "" {
		log.Printf("serveVideo chapter=%d: source returned no stream url (sources=%d)", chapterID, len(info.GetSources()))
		http.Error(w, "source returned no stream url", http.StatusBadGateway)
		return
	}
	streamURL, headers = unwrapLocalProxyURL(streamURL, headers)

	if looksHLS(streamURL) {
		tgt, ok := streamHTTPURL(streamURL)
		if !ok {
			log.Printf("serveVideo chapter=%d: hls url rejected: %q", chapterID, streamURL)
			http.Error(w, "stream url rejected", http.StatusBadGateway)
			return
		}
		h.streamHLS(w, r, tgt, headers, chapterID)
		return
	}

	if looksDASH(streamURL) {
		tgt, ok := streamHTTPURL(streamURL)
		if !ok {
			log.Printf("serveVideo chapter=%d: dash url rejected: %q", chapterID, streamURL)
			http.Error(w, "stream url rejected", http.StatusBadGateway)
			return
		}
		h.streamDASHManifest(w, r, tgt, headers)
		return
	}

	up, err := http.NewRequestWithContext(ctx, http.MethodGet, streamURL, nil)
	if err != nil {
		http.Error(w, "bad stream url", http.StatusBadGateway)
		return
	}

	applyUpstreamHeaders(up, headers)
	if rng := r.Header.Get("Range"); rng != "" {
		up.Header.Set("Range", rng)
	}

	resp, err := proxyClient.Do(up)
	if err != nil {
		http.Error(w, "upstream stream fetch failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	base, _ := url.Parse(streamURL)
	switch ct := strings.ToLower(resp.Header.Get("Content-Type")); {
	case strings.Contains(ct, "mpegurl") && base != nil:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(rewriteHLSPlaylist(body, base, encodeHeaderParam(headers)))
		return
	case strings.Contains(ct, "dash+xml") && base != nil:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		w.Header().Set("Content-Type", "application/dash+xml")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(rewriteDASHManifest(body, base, headers))
		return
	}

	for _, hk := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges"} {
		if v := resp.Header.Get(hk); v != "" {
			w.Header().Set(hk, v)
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func classifyStream(rawURL string) string {
	switch {
	case looksHLS(rawURL):
		return "hls"
	case looksDASH(rawURL):
		return "dash"
	default:
		return "progressive"
	}
}

func verifyStreamURL(ctx context.Context, rawURL string, headers map[string]string) (int, error) {
	u, ok := streamHTTPURL(rawURL)
	if !ok {
		return 0, fmt.Errorf("stream url rejected: %q", rawURL)
	}

	lower := strings.ToLower(u.Path)
	isManifest := strings.Contains(lower, ".m3u8") || strings.Contains(lower, ".mpd")

	var lastErr error
	for _, method := range []string{http.MethodHead, http.MethodGet} {
		req, err := http.NewRequestWithContext(ctx, method, u.String(), nil)
		if err != nil {
			return 0, err
		}
		applyUpstreamHeaders(req, headers)
		if method == http.MethodGet && !isManifest {
			req.Header.Set("Range", "bytes=0-0")
		}
		resp, err := proxyClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		resp.Body.Close()
		if resp.StatusCode < 400 {
			return resp.StatusCode, nil
		}
		if (resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusForbidden) && method == http.MethodHead {
			continue
		}
		return resp.StatusCode, fmt.Errorf("stream URL returned HTTP %d", resp.StatusCode)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("stream unreachable")
	}
	return 0, lastErr
}

type probeResult struct {
	OK          bool   `json:"ok"`
	Stage       string `json:"stage,omitempty"`
	StreamType  string `json:"streamType,omitempty"`
	Sources     int    `json:"sources"`
	Subtitles   int    `json:"subtitles"`
	AudioTracks int    `json:"audioTracks"`
	SkipMarkers int    `json:"skipMarkers"`
	Status      int    `json:"status,omitempty"`
	Error       string `json:"error,omitempty"`
	ElapsedMs   int64  `json:"elapsedMs"`
}

func (h *ContentHandler) probeVideo(w http.ResponseWriter, r *http.Request, chapterID int64) {
	start := time.Now()
	res := h.runProbe(r.Context(), chapterID, nil)
	res.ElapsedMs = time.Since(start).Milliseconds()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if !res.OK {
		w.WriteHeader(http.StatusBadGateway)
	}
	_ = json.NewEncoder(w).Encode(res)
}

func (h *ContentHandler) videoEvents(w http.ResponseWriter, r *http.Request, chapterID int64) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Connection", "keep-alive")

	start := time.Now()
	send := func(event string, payload any) {
		b, _ := json.Marshal(payload)
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, b)
		flusher.Flush()
	}
	stage := func(name string, extra map[string]any) {
		m := map[string]any{"stage": name, "elapsedMs": time.Since(start).Milliseconds()}
		for k, v := range extra {
			m[k] = v
		}
		send("stage", m)
	}

	stage("resolving", nil)
	res := h.runProbe(r.Context(), chapterID, func(s string) {
		stage(s, nil)
	})
	res.ElapsedMs = time.Since(start).Milliseconds()
	if res.OK {
		send("ready", res)
	} else {
		send("error", map[string]any{
			"stage": res.Stage, "error": res.Error, "status": res.Status,
			"elapsedMs": res.ElapsedMs,
		})
	}
}

func (h *ContentHandler) runProbe(ctx context.Context, chapterID int64, onStage func(string)) probeResult {
	dctx, err := h.Q.GetChapterDownloadContext(ctx, chapterID)
	if err != nil {
		return probeResult{Stage: "resolve", Error: "chapter not found"}
	}
	if dctx.ContentType != "anime" {
		return probeResult{Stage: "resolve", Error: "not an anime episode"}
	}

	if row, err := h.Q.GetAnimeEpisodeStream(ctx, chapterID); err == nil && row.LocalPath.Valid && row.LocalPath.String != "" {
		return probeResult{OK: true, StreamType: "file"}
	}

	info, err := h.Sr.Resolve(ctx, dctx.ExtensionPackageName, dctx.SourceEntryID, dctx.SourceChapterID)
	if err != nil {
		return probeResult{Stage: "resolve", Error: err.Error()}
	}
	primaryURL, primaryHeaders := info.GetStreamUrl(), info.GetHeaders()

	// Try the primary stream URL first, then fall back through the other
	// sources the extension returned. Extractors routinely hand back a broken
	// primary URL alongside a dozen working alternates (a flaky host, a
	// mapper entry missing its `url`, etc.); playback already picks among
	// GetSources() for quality, so the probe should not fail the episode when
	// only the primary is bad.
	type streamCandidate struct {
		url     string
		headers map[string]string
	}
	var candidates []streamCandidate
	if primaryURL != "" {
		candidates = append(candidates, streamCandidate{primaryURL, primaryHeaders})
	}
	for _, s := range info.GetSources() {
		u := s.GetUrl()
		if u == "" || u == primaryURL {
			continue
		}
		hdr := primaryHeaders
		if len(s.GetHeaders()) > 0 {
			hdr = s.GetHeaders()
		}
		candidates = append(candidates, streamCandidate{u, hdr})
	}
	if len(candidates) == 0 {
		return probeResult{Stage: "resolve", Error: "source returned no stream url"}
	}

	if onStage != nil {
		onStage("verifying")
	}

	res := probeResult{
		Sources:     len(info.GetSources()),
		Subtitles:   len(info.GetSubtitles()),
		AudioTracks: len(info.GetAudioTracks()),
		SkipMarkers: len(info.GetTimestamps()),
	}

	var lastErr error
	var lastStatus int
	for i, c := range candidates {
		streamURL, headers := unwrapLocalProxyURL(c.url, c.headers)
		streamType := classifyStream(streamURL)

		status, verr := verifyStreamURL(ctx, streamURL, headers)
		if verr != nil {
			lastErr, lastStatus = verr, status
			log.Printf("probeVideo chapter=%d: candidate %d/%d (%s) verify failed: status=%d %v url=%q",
				chapterID, i+1, len(candidates), streamType, status, verr, streamURL)
			continue
		}
		if streamType == "hls" {
			if cerr := hlsContentCheck(ctx, streamURL, headers, 2); cerr != nil {
				lastErr, lastStatus = cerr, status
				log.Printf("probeVideo chapter=%d: candidate %d/%d hls content check failed: %v url=%q",
					chapterID, i+1, len(candidates), cerr, streamURL)
				continue
			}
		}
		res.OK = true
		res.StreamType = streamType
		res.Status = status
		return res
	}

	h.Sr.Invalidate(dctx.ExtensionPackageName, dctx.SourceEntryID, dctx.SourceChapterID)
	res.Stage = "verify"
	res.Status = lastStatus
	if lastErr != nil {
		res.Error = lastErr.Error()
	}
	return res
}

func hlsContentCheck(ctx context.Context, rawURL string, headers map[string]string, depth int) error {
	u, ok := streamHTTPURL(rawURL)
	if !ok {
		return fmt.Errorf("stream url rejected: %q", rawURL)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	applyUpstreamHeaders(req, headers)
	resp, err := proxyClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("stream URL returned HTTP %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	var nested []string
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		abs := resolveMaybeRelative(u, line)
		if strings.Contains(strings.ToLower(abs), ".m3u8") {
			nested = append(nested, abs)
			continue
		}
		if !isAdSegment(abs) {
			return nil
		}
	}
	if depth > 0 {
		for _, n := range nested {
			if hlsContentCheck(ctx, n, headers, depth-1) == nil {
				return nil
			}
		}
	}
	return fmt.Errorf("stream has no playable segments (ads only) -- try another source")
}

func (h *ContentHandler) serveHLS(w http.ResponseWriter, r *http.Request, chapterID int64) {
	raw, err := base64.RawURLEncoding.DecodeString(r.URL.Query().Get("u"))
	if err != nil {
		http.Error(w, "bad hls target", http.StatusBadRequest)
		return
	}
	tgt, ok := streamHTTPURL(string(raw))
	if !ok {
		http.Error(w, "hls target rejected", http.StatusBadRequest)
		return
	}
	var headers map[string]string
	if hp := r.URL.Query().Get("h"); hp != "" {
		if hj, derr := base64.RawURLEncoding.DecodeString(hp); derr == nil {
			_ = json.Unmarshal(hj, &headers)
		}
	}
	h.streamHLS(w, r, tgt, headers, chapterID)
}

func (h *ContentHandler) invalidateResolveFor(ctx context.Context, chapterID int64) {
	if dctx, err := h.Q.GetChapterDownloadContext(ctx, chapterID); err == nil {
		h.Sr.Invalidate(dctx.ExtensionPackageName, dctx.SourceEntryID, dctx.SourceChapterID)
	}
}

func (h *ContentHandler) streamHLS(w http.ResponseWriter, r *http.Request, target *url.URL, headers map[string]string, chapterID int64) {
	isRanged := r.Header.Get("Range") != ""
	looksSegment := !strings.Contains(strings.ToLower(target.Path), ".m3u8")
	if looksSegment && !isRanged {
		if b, ct, ok := streamSegments.get(target.String()); ok {
			writeCachedSegment(w, b, ct)
			return
		}
	}

	up, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target.String(), nil)
	if err != nil {
		http.Error(w, "bad hls url", http.StatusBadGateway)
		return
	}
	applyUpstreamHeaders(up, headers)
	if rng := r.Header.Get("Range"); rng != "" {
		up.Header.Set("Range", rng)
	}

	resp, err := proxyClient.Do(up)
	if err != nil {
		http.Error(w, "hls upstream fetch failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		log.Printf("hls upstream %d for %s (referer=%q ua=%q)",
			resp.StatusCode, target.String(), up.Header.Get("Referer"), up.Header.Get("User-Agent"))
		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusGone {
			h.invalidateResolveFor(r.Context(), chapterID)
		}
	}

	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.Contains(ct, "mpegurl") || strings.Contains(strings.ToLower(target.Path), ".m3u8") {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(rewriteHLSPlaylist(body, target, encodeHeaderParam(headers)))
		return
	}

	h.writeSegment(w, r, target.String(), resp)
}

func (h *ContentHandler) writeSegment(w http.ResponseWriter, r *http.Request, key string, resp *http.Response) {
	respCT := resp.Header.Get("Content-Type")
	cacheable := resp.StatusCode == http.StatusOK && r.Header.Get("Range") == "" && resp.ContentLength <= segCacheMaxObject

	for _, hk := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges"} {
		if v := resp.Header.Get(hk); v != "" {
			w.Header().Set(hk, v)
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(resp.StatusCode)

	// Deliver bytes immediately. The cache limit limits retained memory, never
	// the response body: large segments must still reach the player in full.
	controller := http.NewResponseController(w)
	var cached []byte
	var delivered int64
	buffer := make([]byte, 32<<10)
	for {
		n, readErr := resp.Body.Read(buffer)
		if n > 0 {
			written, writeErr := w.Write(buffer[:n])
			delivered += int64(written)
			if writeErr != nil || written != n {
				panic(http.ErrAbortHandler)
			}
			if err := controller.Flush(); err != nil && !errors.Is(err, http.ErrNotSupported) {
				panic(http.ErrAbortHandler)
			}
			if cacheable {
				if len(cached)+n <= segCacheMaxObject {
					cached = append(cached, buffer[:n]...)
				} else {
					cacheable = false
					cached = nil
				}
			}
		}
		if readErr == io.EOF {
			if resp.ContentLength >= 0 && delivered != resp.ContentLength {
				panic(http.ErrAbortHandler)
			}
			if cacheable {
				streamSegments.put(key, cached, respCT)
			}
			return
		}
		if readErr != nil {
			// An interrupted upstream response must not look complete or enter
			// the cache. net/http closes the stream so the player can retry it.
			panic(http.ErrAbortHandler)
		}
	}
}

func writeCachedSegment(w http.ResponseWriter, data []byte, ct string) {
	if ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(data)
}

const segCacheMaxObject = 24 << 20

const defaultBrowserUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

func applyUpstreamHeaders(req *http.Request, headers map[string]string) {
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", defaultBrowserUA)
	}
	if req.Header.Get("Referer") == "" && req.URL != nil {
		req.Header.Set("Referer", req.URL.Scheme+"://"+req.URL.Host+"/")
	}
}

var streamSegments = newSegCache(256 << 20)

var proxyClient = &http.Client{
	Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   8,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 20 * time.Second,
		ExpectContinueTimeout: time.Second,
	},
}

var hlsURIAttr = regexp.MustCompile(`URI="([^"]*)"`)

var adSegmentHosts = []string{
	"ibyteimg.com", "ad-site-i18n", "/ads/", "/adserver",
	"doubleclick", "googlesyndication", "2mdn.net", "adsafeprotected", "imasdk",
}

func isAdSegment(uri string) bool {
	l := strings.ToLower(uri)
	for _, s := range adSegmentHosts {
		if strings.Contains(l, s) {
			return true
		}
	}
	return false
}

func perSegmentTag(t string) bool {
	return strings.HasPrefix(t, "#EXTINF") ||
		strings.HasPrefix(t, "#EXT-X-DISCONTINUITY") ||
		strings.HasPrefix(t, "#EXT-X-PROGRAM-DATE-TIME") ||
		strings.HasPrefix(t, "#EXT-X-BYTERANGE")
}

func rewriteHLSPlaylist(body []byte, base *url.URL, hdrParam string) []byte {
	out := make([]byte, 0, len(body)+len(body)/3)
	var pending []string
	flush := func() {
		for _, p := range pending {
			out = append(out, p...)
			out = append(out, '\n')
		}
		pending = pending[:0]
	}
	for _, line := range strings.Split(string(body), "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "":
			flush()
			out = append(out, line...)
			out = append(out, '\n')
		case strings.HasPrefix(trimmed, "#"):
			if m := hlsURIAttr.FindStringSubmatchIndex(line); m != nil && m[2] != m[3] {
				abs := resolveRef(base, line[m[2]:m[3]])
				flush()
				out = append(out, line[:m[2]]...)
				out = append(out, proxiedRef(abs, hdrParam)...)
				out = append(out, line[m[3]:]...)
				out = append(out, '\n')
			} else if perSegmentTag(trimmed) {
				pending = append(pending, line)
			} else {
				flush()
				out = append(out, line...)
				out = append(out, '\n')
			}
		default:
			abs := resolveRef(base, trimmed)
			if isAdSegment(abs) {
				pending = pending[:0]
				continue
			}
			flush()
			out = append(out, proxiedRef(abs, hdrParam)...)
			out = append(out, '\n')
		}
	}
	flush()
	return out
}

func resolveRef(base *url.URL, ref string) string {
	if base == nil {
		return ref
	}
	if !strings.Contains(ref, "://") && !strings.HasPrefix(ref, "/") &&
		!strings.Contains(ref, "../") && !strings.Contains(ref, "./") {
		bp := base.Path
		if i := strings.LastIndexByte(bp, '/'); i >= 0 {
			bp = bp[:i+1]
		} else {
			bp = "/"
		}
		return base.Scheme + "://" + base.Host + bp + ref
	}
	if u, err := url.Parse(ref); err == nil {
		return base.ResolveReference(u).String()
	}
	return ref
}

func resolveMaybeRelative(base *url.URL, ref string) string { return resolveRef(base, ref) }

func proxiedRef(abs, hdrParam string) string {
	u := base64.RawURLEncoding.EncodeToString([]byte(abs))
	if hdrParam == "" {
		return "hls?u=" + u
	}
	return "hls?u=" + u + "&h=" + hdrParam
}

func looksHLS(rawURL string) bool {
	u := strings.ToLower(rawURL)
	return strings.Contains(u, ".m3u8") || strings.Contains(u, ".m3u")
}

func looksDASH(rawURL string) bool {
	return strings.Contains(strings.ToLower(rawURL), ".mpd")
}

func pickVideoSource(sources []*sandboxv1.VideoSource, want string) *sandboxv1.VideoSource {
	want = strings.TrimSpace(want)
	for _, s := range sources {
		if strings.EqualFold(s.GetLabel(), want) {
			return s
		}
	}
	if n, err := strconv.Atoi(strings.TrimSuffix(strings.ToLower(want), "p")); err == nil {
		for _, s := range sources {
			if int(s.GetResolution()) == n {
				return s
			}
		}
	}
	return nil
}

func (h *ContentHandler) serveSubtitle(w http.ResponseWriter, r *http.Request, chapterID int64) {
	raw, err := base64.RawURLEncoding.DecodeString(r.URL.Query().Get("u"))
	if err != nil {
		http.Error(w, "bad subtitle target", http.StatusBadRequest)
		return
	}
	tgt, ok := publicHTTPURL(string(raw))
	if !ok {
		http.Error(w, "subtitle target rejected", http.StatusBadRequest)
		return
	}
	var headers map[string]string
	if hp := r.URL.Query().Get("h"); hp != "" {
		if hj, derr := base64.RawURLEncoding.DecodeString(hp); derr == nil {
			_ = json.Unmarshal(hj, &headers)
		}
	}

	body, ok := h.fetchSubtitleViaSandbox(r.Context(), chapterID, tgt.String())
	if !ok {
		body, ok = h.fetchSubtitleDirect(r.Context(), tgt, headers)
		if !ok {
			http.Error(w, "subtitle fetch failed", http.StatusBadGateway)
			return
		}
	}

	head := strings.ToLower(strings.TrimLeft(strings.TrimPrefix(string(body), "\ufeff"), " \t\r\n"))
	ext := strings.ToLower(path.Ext(tgt.Path))
	w.Header().Set("Cache-Control", "no-store")
	switch {
	case strings.HasPrefix(head, "[script info]") || ext == ".ass" || ext == ".ssa":
		w.Header().Set("X-Subtitle-Format", "ass")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write(body)
	case strings.HasPrefix(head, "webvtt"):
		w.Header().Set("X-Subtitle-Format", "vtt")
		w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
		_, _ = w.Write(body)
	case ext == ".srt" || srtHeader.Match(body):
		w.Header().Set("X-Subtitle-Format", "srt")
		w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
		_, _ = w.Write(srtToVTT(body))
	default:

		w.Header().Set("X-Subtitle-Format", "vtt")
		w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
		if !strings.HasPrefix(head, "webvtt") {
			_, _ = w.Write([]byte("WEBVTT\n\n"))
		}
		_, _ = w.Write(body)
	}
}

func (h *ContentHandler) fetchSubtitleViaSandbox(ctx context.Context, chapterID int64, subURL string) ([]byte, bool) {
	dctx, err := h.Q.GetChapterDownloadContext(ctx, chapterID)
	if err != nil || dctx.ExtensionPackageName == "" {
		return nil, false
	}
	client, err := h.Sc.Ensure(ctx)
	if err != nil {
		return nil, false
	}
	img, err := client.GetImageBytes(ctx, dctx.ExtensionPackageName, subURL)
	if err != nil || len(img.GetData()) == 0 {
		log.Printf("serveSubtitle: sandbox fetch failed for %s: %v", subURL, err)
		return nil, false
	}
	return img.GetData(), true
}

func (h *ContentHandler) fetchSubtitleDirect(ctx context.Context, tgt *url.URL, passthrough map[string]string) ([]byte, bool) {
	origin := tgt.Scheme + "://" + tgt.Host
	attempts := []map[string]string{
		{"User-Agent": defaultBrowserUA},
		{"User-Agent": defaultBrowserUA, "Referer": origin + "/", "Origin": origin},
		passthrough,
	}
	for i, hdr := range attempts {
		up, err := http.NewRequestWithContext(ctx, http.MethodGet, tgt.String(), nil)
		if err != nil {
			return nil, false
		}
		for k, v := range hdr {
			up.Header.Set(k, v)
		}
		got, err := proxyClient.Do(up)
		if err != nil {
			continue
		}
		if got.StatusCode < 400 {
			b, _ := io.ReadAll(io.LimitReader(got.Body, 8<<20))
			got.Body.Close()
			return b, true
		}
		got.Body.Close()
		if i == len(attempts)-1 {
			log.Printf("serveSubtitle: upstream HTTP %d for %s (tried %d header sets)", got.StatusCode, tgt.String(), len(attempts))
		}
	}
	return nil, false
}

var (
	srtHeader   = regexp.MustCompile(`^\x{feff}?\s*\d+\s*\r?\n\d\d:\d\d:\d\d,\d\d\d\s*-->`)
	srtTimeline = regexp.MustCompile(`(\d\d:\d\d:\d\d),(\d\d\d)`)
)

func srtToVTT(srt []byte) []byte {
	s := strings.TrimPrefix(string(srt), "\ufeff")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = srtTimeline.ReplaceAllString(s, "$1.$2")
	return []byte("WEBVTT\n\n" + strings.TrimLeft(s, "\n"))
}

func encodeHeaderParam(headers map[string]string) string {
	if len(headers) == 0 {
		return ""
	}
	j, err := json.Marshal(headers)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(j)
}

func dashSegmentBase(manifest *url.URL, headers map[string]string) string {
	dir := *manifest
	if i := strings.LastIndex(dir.Path, "/"); i >= 0 {
		dir.Path = dir.Path[:i+1]
	}
	dir.RawQuery, dir.Fragment = "", ""
	payload, _ := json.Marshal(struct {
		B string            `json:"b"`
		H map[string]string `json:"h,omitempty"`
	}{B: dir.String(), H: headers})
	return base64.RawURLEncoding.EncodeToString(payload)
}

var (
	dashBaseURLTag = regexp.MustCompile(`(?s)<BaseURL>\s*(https?://[^<]+?)\s*</BaseURL>`)
	dashMPDOpenTag = regexp.MustCompile(`<MPD\b[^>]*>`)
)

func rewriteDASHManifest(body []byte, manifest *url.URL, headers map[string]string) []byte {
	s := string(body)

	s = dashBaseURLTag.ReplaceAllStringFunc(s, func(m string) string {
		abs := dashBaseURLTag.FindStringSubmatch(m)[1]
		u, err := url.Parse(strings.TrimSpace(abs))
		if err != nil {
			return m
		}
		return "<BaseURL>dash/" + dashSegmentBase(manifest.ResolveReference(u), headers) + "/</BaseURL>"
	})

	inject := "<BaseURL>dash/" + dashSegmentBase(manifest, headers) + "/</BaseURL>"
	if loc := dashMPDOpenTag.FindStringIndex(s); loc != nil {
		return []byte(s[:loc[1]] + inject + s[loc[1]:])
	}
	return []byte(inject + s)
}

func (h *ContentHandler) serveDASH(w http.ResponseWriter, r *http.Request, rest string, chapterID int64) {
	slash := strings.IndexByte(rest, '/')
	if slash < 0 {
		http.Error(w, "bad dash path", http.StatusBadRequest)
		return
	}
	raw, err := base64.RawURLEncoding.DecodeString(rest[:slash])
	if err != nil {
		http.Error(w, "bad dash base", http.StatusBadRequest)
		return
	}
	var blob struct {
		B string            `json:"b"`
		H map[string]string `json:"h"`
	}
	if err := json.Unmarshal(raw, &blob); err != nil {
		http.Error(w, "bad dash base", http.StatusBadRequest)
		return
	}
	baseURL, err := url.Parse(blob.B)
	if err != nil {
		http.Error(w, "bad dash base", http.StatusBadRequest)
		return
	}
	seg, err := url.Parse(rest[slash+1:])
	if err != nil {
		http.Error(w, "bad dash segment", http.StatusBadRequest)
		return
	}
	tgt, ok := streamHTTPURL(baseURL.ResolveReference(seg).String())
	if !ok {
		http.Error(w, "dash target rejected", http.StatusBadRequest)
		return
	}

	if r.Header.Get("Range") == "" {
		if b, ct, ok := streamSegments.get(tgt.String()); ok {
			writeCachedSegment(w, b, ct)
			return
		}
	}

	up, err := http.NewRequestWithContext(r.Context(), http.MethodGet, tgt.String(), nil)
	if err != nil {
		http.Error(w, "bad dash url", http.StatusBadGateway)
		return
	}
	applyUpstreamHeaders(up, blob.H)
	if rng := r.Header.Get("Range"); rng != "" {
		up.Header.Set("Range", rng)
	}
	resp, err := proxyClient.Do(up)
	if err != nil {
		http.Error(w, "dash upstream fetch failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusGone {
		log.Printf("dash upstream %d for %s", resp.StatusCode, tgt.String())
		h.invalidateResolveFor(r.Context(), chapterID)
	}
	h.writeSegment(w, r, tgt.String(), resp)
}

func (h *ContentHandler) streamDASHManifest(w http.ResponseWriter, r *http.Request, target *url.URL, headers map[string]string) {
	up, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target.String(), nil)
	if err != nil {
		http.Error(w, "bad dash url", http.StatusBadGateway)
		return
	}
	applyUpstreamHeaders(up, headers)
	resp, err := proxyClient.Do(up)
	if err != nil {
		http.Error(w, "dash manifest fetch failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	w.Header().Set("Content-Type", "application/dash+xml")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(rewriteDASHManifest(body, target, headers))
}

func decodeB64Any(s string) ([]byte, error) {
	for _, enc := range []*base64.Encoding{
		base64.StdEncoding, base64.RawStdEncoding,
		base64.URLEncoding, base64.RawURLEncoding,
	} {
		if b, err := enc.DecodeString(s); err == nil {
			return b, nil
		}
	}
	return nil, fmt.Errorf("not base64: %q", s)
}

func unwrapLocalProxyURL(raw string, headers map[string]string) (string, map[string]string) {
	u, err := url.Parse(raw)
	if err != nil {
		return raw, headers
	}
	isProxy := strings.Contains(u.Path, "/proxy/")
	isM3u8 := u.Path == "/m3u8" || strings.HasSuffix(u.Path, "/m3u8")
	if !isProxy && !isM3u8 {
		return raw, headers
	}
	switch u.Hostname() {
	case "127.0.0.1", "localhost", "::1", "0.0.0.0":
	default:
		return raw, headers
	}
	q := u.Query()
	if isProxy {
		real, err := decodeB64Any(q.Get("url"))
		if err != nil || len(real) == 0 {
			return raw, headers
		}
		merged := map[string]string{}
		for k, v := range headers {
			merged[k] = v
		}
		if hb, err := decodeB64Any(q.Get("headers")); err == nil {
			for _, line := range strings.Split(string(hb), "\n") {
				if i := strings.IndexByte(line, ':'); i > 0 {
					merged[strings.TrimSpace(line[:i])] = strings.TrimSpace(line[i+1:])
				}
			}
		}
		return string(real), merged
	}
	realStr := q.Get("url")
	if _, ok := publicHTTPURL(realStr); !ok {
		if b, err := decodeB64Any(q.Get("url")); err == nil && len(b) > 0 {
			realStr = string(b)
		}
	}
	if _, ok := publicHTTPURL(realStr); !ok {
		return raw, headers
	}
	merged := map[string]string{}
	for k, v := range headers {
		merged[k] = v
	}
	if v := q.Get("referer"); v != "" {
		merged["Referer"] = v
	}
	for _, key := range []string{"useragent", "user-agent", "userAgent"} {
		if v := q.Get(key); v != "" {
			merged["User-Agent"] = v
			break
		}
	}
	return realStr, merged
}

func publicHTTPURL(raw string) (*url.URL, bool) {
	raw = strings.TrimSpace(raw)

	if strings.HasPrefix(raw, "//") {
		raw = "https:" + raw
	} else if !strings.Contains(raw, "://") && !strings.HasPrefix(raw, "/") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, false
	}
	host := u.Hostname()
	if host == "" || strings.EqualFold(host, "localhost") {
		return nil, false
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
			return nil, false
		}
	}
	return u, true
}

var reservedLoopbackPorts = map[string]bool{"6007": true, "50051": true}

// streamHTTPURL is publicHTTPURL for video/HLS/DASH targets. Some anime extensions
// hand back URLs pointing at their own in-sandbox HLS proxy on a loopback
// ephemeral port (e.g. http://127.0.0.1:45221/pl/1); that endpoint is trusted and
// reachable, so loopback is allowed for it while every other private range and our
// own service ports stay blocked.
func streamHTTPURL(raw string) (*url.URL, bool) {
	if u, ok := publicHTTPURL(raw); ok {
		return u, true
	}
	raw = strings.TrimSpace(raw)
	if !strings.Contains(raw, "://") && !strings.HasPrefix(raw, "//") {
		return nil, false
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, false
	}
	ip := net.ParseIP(u.Hostname())
	if ip == nil || !ip.IsLoopback() {
		return nil, false
	}
	port := u.Port()
	if port == "" || reservedLoopbackPorts[port] {
		return nil, false
	}
	if n, perr := strconv.Atoi(port); perr != nil || n < 1024 {
		return nil, false
	}
	return u, true
}

func (h *ContentHandler) serveText(w http.ResponseWriter, r *http.Request, chapterID int64) {
	ctx := r.Context()

	if row, err := h.Q.GetNovelChapterContent(ctx, chapterID); err == nil && row.LocalPath.Valid && row.LocalPath.String != "" {
		if _, ok := localsource.ParseDocxBookPath(row.LocalPath.String); ok {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"docxSource":true}`))
			return
		}

		var data []byte
		var rerr error
		entryPath := row.LocalPath.String
		if archivePath, entryName, ok := localsource.ParseZipPagePath(row.LocalPath.String); ok {
			entryPath = entryName
			data, rerr = localsource.ReadArchiveEntry(archivePath, entryName)
		} else {
			data, rerr = os.ReadFile(row.LocalPath.String)
		}
		if rerr == nil {
			ct := textContentType(entryPath)
			if strings.HasPrefix(ct, "text/html") {
				data = []byte(sanitizeNovelHTML(string(data)))
			}
			w.Header().Set("Content-Type", ct)
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			_, _ = w.Write(data)
			return
		}
	}

	dctx, err := h.Q.GetChapterDownloadContext(ctx, chapterID)
	if err != nil {
		http.Error(w, "chapter not found", http.StatusNotFound)
		return
	}
	if dctx.ContentType != "novel" {
		http.Error(w, "text content only supports novel chapters", http.StatusBadRequest)
		return
	}

	client, err := h.Sc.Ensure(ctx)
	if err != nil {
		http.Error(w, "sandbox unavailable", http.StatusServiceUnavailable)
		return
	}

	resp, err := client.GetChapterText(ctx, dctx.ExtensionPackageName, dctx.SourceEntryID, dctx.SourceChapterID)
	if err != nil {
		http.Error(w, "fetching chapter text failed", http.StatusBadGateway)
		return
	}
	content := resp.GetContent()
	ct := "text/plain; charset=utf-8"
	if strings.EqualFold(resp.GetFormat(), "html") {
		ct = "text/html; charset=utf-8"
		content = sanitizeNovelHTML(content)
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(content))
}

func textContentType(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".html", ".htm", ".xhtml":
		return "text/html; charset=utf-8"
	default:
		return "text/plain; charset=utf-8"
	}
}

func imageContentType(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".avif":
		return "image/avif"
	case ".bmp":
		return "image/bmp"
	default:
		return "image/jpeg"
	}
}

// serveZipEntry reads a single page directly out of a CBZ/ZIP or CBR/RAR
// archive without extracting it to disk. Returns false if the entry
// couldn't be served, so the caller can fall through to other page sources.
func serveZipEntry(w http.ResponseWriter, archivePath, entryName string) bool {
	return localsource.ServeArchiveEntry(w, func(name string, size int64) {
		w.Header().Set("Content-Type", imageContentType(name))
		w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}, archivePath, entryName)
}

var _ = sql.ErrNoRows
