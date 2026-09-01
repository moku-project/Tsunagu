package rest

import (
	"database/sql"
	"encoding/base64"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"tsunagu/backend/internal/db/sqlcgen"
	"tsunagu/backend/internal/image"
)

type CoverProxyHandler struct {
	Q             *sqlcgen.Queries
	CoverCacheDir string
}

func (h *CoverProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/proxy/cover/"), "/")
	if idStr == "" {
		http.Error(w, "expected /proxy/cover/{libraryEntryId}", http.StatusBadRequest)
		return
	}

	entryID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid library entry id", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	entry, err := h.Q.GetMedia(ctx, entryID)
	if err != nil {
		http.Error(w, "library entry not found", http.StatusNotFound)
		return
	}

	if entry.CoverLocalPath.Valid && entry.CoverLocalPath.String != "" {
		if data, ct, ok := readCachedCover(entry.CoverLocalPath.String); ok {
			w.Header().Set("Content-Type", ct)
			w.Header().Set("Cache-Control", "public, max-age=86400")
			_, _ = w.Write(data)
			return
		}
	}

	var candidates []string
	if entry.CoverOverride.Valid && entry.CoverOverride.String != "" {
		candidates = append(candidates, entry.CoverOverride.String)
	}
	if entry.CoverPath.Valid && entry.CoverPath.String != "" {
		candidates = append(candidates, entry.CoverPath.String)
	}
	if mc, err := h.Q.GetMediaMetadataCover(ctx, entryID); err == nil && mc != "" {
		candidates = append(candidates, mc)
	}
	if len(candidates) == 0 {
		http.Error(w, "no cover available for this entry", http.StatusNotFound)
		return
	}

	destName := strconv.FormatInt(entryID, 10)
	var localPath string
	for _, u := range candidates {
		if localPath, err = image.DownloadToFile(u, h.CoverCacheDir, destName); err == nil {
			break
		}
	}
	if localPath == "" {
		http.Error(w, "fetching cover failed", http.StatusBadGateway)
		return
	}

	_ = h.Q.UpdateMediaCoverLocalPath(ctx, sqlcgen.UpdateMediaCoverLocalPathParams{
		ID:             entryID,
		CoverLocalPath: sql.NullString{String: localPath, Valid: true},
	})

	data, ct, ok := readCachedCover(localPath)
	if !ok {
		http.Error(w, "reading fetched cover failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(data)
}

type RemoteCoverProxyHandler struct {
	CoverCacheDir string
}

func (h *RemoteCoverProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	for _, pre := range []string{"/proxy/cover/remote/", "/proxy/img/"} {
		p = strings.TrimPrefix(p, pre)
	}
	encoded := strings.TrimSuffix(p, "/")
	if encoded == "" {
		http.Error(w, "expected /proxy/img/{base64Url}", http.StatusBadRequest)
		return
	}

	raw, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		http.Error(w, "invalid encoded url", http.StatusBadRequest)
		return
	}
	upstreamURL := string(raw)

	if _, ok := publicHTTPURL(upstreamURL); !ok {
		http.Error(w, "image url rejected", http.StatusBadRequest)
		return
	}

	destName := base64.URLEncoding.EncodeToString([]byte(upstreamURL))
	localPath, err := image.DownloadToFile(upstreamURL, h.CoverCacheDir, destName)
	if err != nil {
		http.Error(w, "fetching cover failed", http.StatusBadGateway)
		return
	}

	data, ct, ok := readCachedCover(localPath)
	if !ok {
		http.Error(w, "reading fetched cover failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(data)
}

func readCachedCover(path string) ([]byte, string, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, "", false
	}
	var contentType string
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		contentType = "image/png"
	case ".webp":
		contentType = "image/webp"
	case ".gif":
		contentType = "image/gif"
	default:
		contentType = "image/jpeg"
	}
	return b, contentType, true
}
