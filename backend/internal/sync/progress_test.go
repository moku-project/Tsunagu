package sync

import (
	"context"
	"path/filepath"
	"testing"

	"tsunagu/backend/internal/db"
	"tsunagu/backend/internal/db/sqlcgen"
)

func TestRecordProgressKeepsCompleted(t *testing.T) {
	dir := t.TempDir()
	conn, err := db.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	if _, err := conn.Exec(`
		INSERT INTO media (id, external_id, content_type, title, added_at) VALUES (1, 'm', 'manga', 'Title', CURRENT_TIMESTAMP);
		INSERT INTO chapters (id, media_id, external_id, source_order) VALUES (1, 1, 'c1', 1);
	`); err != nil {
		t.Fatal(err)
	}
	q := sqlcgen.New(conn)
	s := New(conn, q, dir, dir)
	ctx := context.Background()

	if _, err := s.MarkChapterRead(ctx, 1, 1); err != nil {
		t.Fatal(err)
	}
	// Re-reading or re-watching reports a partial position without completed.
	rp, err := s.RecordProgress(ctx, 1, 1, 0.2, false, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !rp.Completed || rp.Progress != 0.2 {
		t.Fatalf("got completed=%v progress=%v, want completed kept and progress 0.2", rp.Completed, rp.Progress)
	}

	// An explicit mark-unread still clears it.
	rp, err = q.UpsertReadingProgress(ctx, sqlcgen.UpsertReadingProgressParams{MediaID: 1, ChapterID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if rp.Completed {
		t.Fatal("explicit unread did not clear completed")
	}
}
