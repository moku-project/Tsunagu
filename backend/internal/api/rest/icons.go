package rest

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"tsunagu/backend/internal/db/sqlcgen"
	"tsunagu/backend/internal/image"
)

type IconProxyHandler struct {
	Q            *sqlcgen.Queries
	IconCacheDir string
}

func (h *IconProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/proxy/icon/"), "/")
	if idStr == "" {
		http.Error(w, "expected /proxy/icon/{extensionId}", http.StatusBadRequest)
		return
	}
	extID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid extension id", http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	ext, err := h.Q.GetExtension(ctx, extID)
	if err != nil {
		http.Error(w, "extension not found", http.StatusNotFound)
		return
	}
	if ext.IconLocalPath.Valid && ext.IconLocalPath.String != "" {
		if data, ct, ok := readCachedCover(ext.IconLocalPath.String); ok {
			w.Header().Set("Content-Type", ct)
			w.Header().Set("Cache-Control", "public, max-age=86400")
			_, _ = w.Write(data)
			return
		}
	}
	if !ext.IconUrl.Valid || ext.IconUrl.String == "" {
		http.Error(w, "no icon available for this extension", http.StatusNotFound)
		return
	}
	destName := strconv.FormatInt(extID, 10)
	localPath, err := image.DownloadToFile(ext.IconUrl.String, h.IconCacheDir, destName)
	if err != nil {
		http.Error(w, "fetching icon failed", http.StatusBadGateway)
		return
	}
	_ = h.Q.UpdateExtensionIconLocalPath(ctx, sqlcgen.UpdateExtensionIconLocalPathParams{
		ID:            extID,
		IconLocalPath: sql.NullString{String: localPath, Valid: true},
	})
	data, ct, ok := readCachedCover(localPath)
	if !ok {
		http.Error(w, "reading fetched icon failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(data)
}
