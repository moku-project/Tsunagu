package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"tsunagu/backend/internal/sandbox"
	"tsunagu/backend/internal/sync"
)

const maxExtensionUpload = 64 << 20

type ExtensionUploadHandler struct {
	Sy *sync.Syncer
	Sc *sandbox.SupervisedClient
}

func (h *ExtensionUploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxExtensionUpload)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		http.Error(w, "invalid upload", http.StatusBadRequest)
		return
	}
	defer r.MultipartForm.RemoveAll()
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(header.Filename)), ".")
	if ext != "apk" && ext != "jar" && ext != "js" {
		http.Error(w, "expected an .apk, .jar or .js file", http.StatusBadRequest)
		return
	}

	dest, err := h.Sy.SideloadTempPath(ext)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	defer os.Remove(dest)
	out, err := os.Create(dest)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	_, err = io.Copy(out, file)
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	c, err := h.Sc.Ensure(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("sandbox unavailable: %v", err), http.StatusServiceUnavailable)
		return
	}
	installed, err := h.Sy.InstallExtensionFile(r.Context(), c, dest)
	if err != nil {
		http.Error(w, fmt.Sprintf("install failed: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"packageName": installed.PackageName,
		"name":        installed.Name,
		"contentType": installed.ContentType,
	})
}
