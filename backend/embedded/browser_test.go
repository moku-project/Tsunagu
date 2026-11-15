package embedded

import (
	"context"
	"encoding/json"
	"net"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/grpc"

	"tsunagu/backend/internal/db"
	sandboxv1 "tsunagu/backend/internal/sandbox/gen/sandbox/v1"
)

// cookieHost records BrowserCookies calls and echoes back what was applied.
type cookieHost struct {
	host
	calls chan *sandboxv1.BrowserCookiesRequest
	saved string
}

func (h *cookieHost) BrowserCookies(_ context.Context, r *sandboxv1.BrowserCookiesRequest) (*sandboxv1.BrowserCookiesResponse, error) {
	h.calls <- r
	if r.Apply {
		h.saved = r.StateJson
	}
	state := h.saved
	if state == "" {
		state = `{"cookies":[],"userAgent":""}`
	}
	return &sandboxv1.BrowserCookiesResponse{StateJson: state}, nil
}

func TestBrowserVerificationRejectsLocalAndInvalidURLs(t *testing.T) {
	for _, target := range []string{
		"ftp://source.example/", "https://user@source.example/", "http://localhost:8080/",
		"https://127.0.0.1/", "https://192.168.1.10/", "https://[::1]/", "not a url",
	} {
		if _, err := BeginBrowserVerification("pkg", target); err == nil {
			t.Errorf("BeginBrowserVerification(%q) succeeded", target)
		}
	}
}

func TestBrowserVerificationRoundTrip(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	h := &cookieHost{calls: make(chan *sandboxv1.BrowserCookiesRequest, 8)}
	g := grpc.NewServer()
	sandboxv1.RegisterExtensionServiceServer(g, h)
	go g.Serve(lis)
	defer g.Stop()

	dir := t.TempDir()
	if err := Start(dir, lis.Addr().String(), "host-token"); err != nil {
		t.Fatal(err)
	}
	defer Stop()

	if _, err := BeginBrowserVerification("eu.kanade.unknown", "https://source.example/"); err == nil ||
		!strings.Contains(err.Error(), "unknown source") {
		t.Fatalf("unknown source: %v", err)
	}

	conn, err := db.Open(filepath.Join(dir, "tsunagu.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`
		INSERT INTO repositories (id, name, index_url) VALUES (1, 'r', 'https://repo.example/index.min.json');
		INSERT INTO extensions (repository_id, package_name, name, version, content_type, lang, apk_url, installed)
		VALUES (1, 'eu.kanade.source', 'Source', '1', 'manga', 'en', 'https://repo.example/s.apk', TRUE);`); err != nil {
		t.Fatal(err)
	}
	conn.Close()
	for len(h.calls) > 0 {
		<-h.calls
	}

	raw, err := BeginBrowserVerification("eu.kanade.source", "https://source.example/series/1")
	if err != nil {
		t.Fatal(err)
	}
	var session struct{ ID, ExtensionID, URL, State string }
	if err := json.Unmarshal([]byte(raw), &session); err != nil {
		t.Fatal(err)
	}
	if session.ID == "" || session.ExtensionID != "eu.kanade.source" || session.State != "pending" {
		t.Fatalf("session = %s", raw)
	}
	<-h.calls // the read Begin makes to confirm the host

	if _, err := BrowserCookies(session.ID); err != nil {
		t.Fatal(err)
	}
	if r := <-h.calls; r.Apply || r.Url != "https://source.example/series/1" {
		t.Fatalf("read call = %+v", r)
	}

	state := `{"cookies":[{"name":"cf_clearance","value":"ok","domain":"source.example"}],"userAgent":"Browser"}`
	if err := CompleteBrowserVerification(session.ID, state); err != nil {
		t.Fatal(err)
	}
	if r := <-h.calls; !r.Apply || r.StateJson != state || r.Url != "https://source.example/series/1" {
		t.Fatalf("apply call = %+v", r)
	}
	if err := CompleteBrowserVerification(session.ID, state); err == nil {
		t.Fatal("a finished session was reusable")
	}
}
