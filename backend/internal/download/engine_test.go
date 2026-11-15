package download

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"tsunagu/backend/internal/db"
	"tsunagu/backend/internal/db/sqlcgen"
)

type fakeEngine struct {
	got  VideoRequest
	run  func(ctx context.Context, req VideoRequest, progress func(VideoProgress)) error
	seen []VideoProgress
}

func (f *fakeEngine) Download(ctx context.Context, req VideoRequest, progress func(VideoProgress)) error {
	f.got = req
	return f.run(ctx, req, func(p VideoProgress) { f.seen = append(f.seen, p); progress(p) })
}

func engineManager(t *testing.T, engine VideoDownloadEngine) (*Manager, *sqlcgen.Queries, string) {
	t.Helper()
	tmp := t.TempDir()
	conn, err := db.Open(filepath.Join(tmp, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	q := sqlcgen.New(conn)
	m := &Manager{q: q}
	m.SetVideoEngine(engine)
	return m, q, filepath.Join(tmp, "episode.mp4")
}

func TestEngineWritesEpisodeWithHeaders(t *testing.T) {
	engine := &fakeEngine{run: func(_ context.Context, req VideoRequest, progress func(VideoProgress)) error {
		progress(VideoProgress{Fraction: 0.5, Bytes: 10})
		return os.WriteFile(req.OutputPath, []byte("mp4"), 0o644)
	}}
	m, _, out := engineManager(t, engine)
	headers := map[string]string{"Referer": "https://source.example/", "Cookie": "a=b"}
	if err := m.downloadWithEngine(context.Background(), 1, "https://cdn.example/v.m3u8", headers, out); err != nil {
		t.Fatal(err)
	}
	if engine.got.URL != "https://cdn.example/v.m3u8" || engine.got.Headers["Cookie"] != "a=b" || engine.got.OutputPath != out {
		t.Fatalf("engine got %+v", engine.got)
	}
	if len(engine.seen) != 1 {
		t.Fatalf("progress not reported: %v", engine.seen)
	}
}

func TestEngineFailureLeavesNoFile(t *testing.T) {
	m, _, out := engineManager(t, &fakeEngine{run: func(_ context.Context, req VideoRequest, _ func(VideoProgress)) error {
		_ = os.WriteFile(req.OutputPath, []byte("partial"), 0o644)
		return errors.New("stream ended early")
	}})
	if err := m.downloadWithEngine(context.Background(), 1, "https://cdn.example/v.mp4", nil, out); err == nil {
		t.Fatal("expected an error")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("partial output left behind")
	}
}

func TestEngineEmptyOutputIsAFailure(t *testing.T) {
	m, _, out := engineManager(t, &fakeEngine{run: func(context.Context, VideoRequest, func(VideoProgress)) error { return nil }})
	if err := m.downloadWithEngine(context.Background(), 1, "https://cdn.example/v.mp4", nil, out); err == nil {
		t.Fatal("success without output was accepted")
	}
}

func TestEngineCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	m, _, out := engineManager(t, &fakeEngine{run: func(ctx context.Context, req VideoRequest, _ func(VideoProgress)) error {
		_ = os.WriteFile(req.OutputPath, []byte("partial"), 0o644)
		cancel()
		<-ctx.Done()
		return nil
	}})
	done := make(chan error, 1)
	go func() { done <- m.downloadWithEngine(ctx, 1, "https://cdn.example/v.mp4", nil, out) }()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cancellation did not return")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("partial output left behind after cancel")
	}
}
