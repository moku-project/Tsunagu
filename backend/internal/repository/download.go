package repository

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func DownloadJAR(cacheDir, packageName, version, jarURL string) (string, error) {
	if jarURL == "" {
		return "", fetchErrorf("no jar_url available for %s", packageName)
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("create cache dir %s: %w", cacheDir, err)
	}

	dest := filepath.Join(cacheDir, fmt.Sprintf("%s-%s.jar", packageName, version))
	if _, err := os.Stat(dest); err == nil {
		return dest, nil
	}

	resp, err := http.Get(jarURL)
	if err != nil {
		return "", fetchErrorf("failed to fetch jar %s: %v", jarURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fetchErrorf("failed to fetch jar %s: HTTP %d", jarURL, resp.StatusCode)
	}

	tmp := dest + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(tmp)
		return "", fmt.Errorf("write jar to %s: %w", tmp, err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return "", fmt.Errorf("close jar file: %w", err)
	}
	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		return "", fmt.Errorf("finalize jar at %s: %w", dest, err)
	}
	return dest, nil
}
