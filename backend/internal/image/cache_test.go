package image

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDownloadToFileRejectsWebPages(t *testing.T) {
	png := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		if r.URL.Path == "/challenge" {
			_, _ = w.Write([]byte("<!doctype html><html><body>Checking that you're not a bot...</body></html>"))
			return
		}
		_, _ = w.Write(png)
	}))
	defer srv.Close()

	dir := t.TempDir()
	if _, err := DownloadToFile(srv.URL+"/challenge", dir, "a"); err == nil {
		t.Fatal("a bot-check page was saved as a cover")
	}
	if _, err := DownloadToFile(srv.URL+"/cover", dir, "b"); err != nil {
		t.Fatalf("real image rejected: %v", err)
	}
}

func TestSVGCoversAreNotWebPages(t *testing.T) {
	for _, svg := range []string{`<svg xmlns="http://www.w3.org/2000/svg"></svg>`, `<?xml version="1.0"?><svg></svg>`} {
		if IsWebPage([]byte(svg)) {
			t.Errorf("%q was taken for a web page", svg)
		}
	}
}
