package image

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func ExtFromContentType(ct string) string {
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

func DownloadToFile(url, destDir, destName string) (string, error) {
	return DownloadToFileWith(&http.Client{Timeout: 15 * time.Second}, url, destDir, destName)
}

func DownloadToFileWith(client *http.Client, url, destDir, destName string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("fetching %s: %w", url, err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetching %s: status %d", url, resp.StatusCode)
	}
	// Bot checks such as DDoS-Guard answer 200 with an HTML page; caching that would serve it as the cover forever.
	body := bufio.NewReader(resp.Body)
	if head, _ := body.Peek(512); IsWebPage(head) {
		return "", fmt.Errorf("fetching %s: got a web page, not an image", url)
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("creating dir %s: %w", destDir, err)
	}

	ext := ExtFromContentType(resp.Header.Get("Content-Type"))
	path := filepath.Join(destDir, destName+ext)

	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("creating file %s: %w", path, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, body); err != nil {
		return "", fmt.Errorf("writing file %s: %w", path, err)
	}
	return path, nil
}

// IsWebPage reports whether b sniffs as HTML. Only HTML: SVG covers sniff as XML or plain text and are real images.
func IsWebPage(b []byte) bool {
	return len(b) > 0 && strings.HasPrefix(http.DetectContentType(b), "text/html")
}

func SaveBytesToFile(data []byte, contentType, destDir, destName string) (string, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("creating dir %s: %w", destDir, err)
	}
	ext := ExtFromContentType(contentType)
	path := filepath.Join(destDir, destName+ext)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("writing file %s: %w", path, err)
	}
	return path, nil
}

func ClearDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading dir %s: %w", dir, err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
			return fmt.Errorf("removing %s: %w", e.Name(), err)
		}
	}
	return nil
}
