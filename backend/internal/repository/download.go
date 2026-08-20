package repository

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func DownloadJAR(cacheDir, packageName, version, jarURL string) (string, error) {
	return DownloadExtensionFile(cacheDir, packageName, version, jarURL, "jar")
}

func DownloadExtensionFile(cacheDir, packageName, version, downloadURL, ext string) (string, error) {
	if downloadURL == "" {
		return "", fetchErrorf("no download url available for %s", packageName)
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("create cache dir %s: %w", cacheDir, err)
	}

	dest := filepath.Join(cacheDir, fmt.Sprintf("%s-%s.%s", packageName, version, ext))
	if _, err := os.Stat(dest); err == nil {
		return dest, nil
	}

	resp, err := http.Get(downloadURL)
	if err != nil {
		return "", fetchErrorf("failed to fetch %s %s: %v", ext, downloadURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fetchErrorf("failed to fetch %s %s: HTTP %d", ext, downloadURL, resp.StatusCode)
	}

	tmp := dest + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(tmp)
		return "", fmt.Errorf("write %s to %s: %w", ext, tmp, err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return "", fmt.Errorf("close %s file: %w", ext, err)
	}
	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		return "", fmt.Errorf("finalize %s at %s: %w", ext, dest, err)
	}
	return dest, nil
}