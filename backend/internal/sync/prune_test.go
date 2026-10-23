package sync

import (
	"context"
	"path/filepath"
	"testing"

	"tsunagu/backend/internal/db"
	"tsunagu/backend/internal/db/sqlcgen"
)

func TestPruneStaleChapters(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	if _, err := conn.Exec(`
		INSERT INTO media (id, external_id, content_type, title, added_at) VALUES (1, 'm', 'manga', 'Title', CURRENT_TIMESTAMP);
		INSERT INTO chapters (id, media_id, external_id, number, source_order) VALUES
			(1, 1, 'old/1', 1, 0), (2, 1, 'old/2', 2, 1), (3, 1, 'gone', 3, 2), (4, 1, 'kept', 4, 3),
			(5, 1, 'new/1', 1, 4), (6, 1, 'new/2', 2, 5);
		INSERT INTO reading_progress (media_id, chapter_id, progress, completed) VALUES (1, 1, 1, TRUE);
		INSERT INTO downloads (chapter_id, status, progress) VALUES (2, 'done', 1), (4, 'done', 1);
	`); err != nil {
		t.Fatal(err)
	}
	q := sqlcgen.New(conn)
	ctx := context.Background()
	before, _ := q.ListChaptersByMedia(ctx, 1)
	after := []sqlcgen.Chapter{before[4], before[5]} // the source now lists only new/1 and new/2
	before = before[:4]
	// Four known, two listed: not "far fewer", so unmatched chapters are pruned.
	if err := pruneStaleChapters(ctx, q, before, after); err != nil {
		t.Fatal(err)
	}

	left, _ := q.ListChaptersByMedia(ctx, 1)
	var ids []int64
	for _, ch := range left {
		ids = append(ids, ch.ID)
	}
	// old/1 and old/2 moved onto their replacements, gone was dropped, kept has a download.
	if want := []int64{4, 5, 6}; len(ids) != len(want) || ids[0] != 4 || ids[1] != 5 || ids[2] != 6 {
		t.Fatalf("chapters left = %v, want %v", ids, want)
	}
	var completed bool
	if err := conn.QueryRow(`SELECT completed FROM reading_progress WHERE chapter_id = 5`).Scan(&completed); err != nil || !completed {
		t.Fatalf("progress not carried to the replacement: %v %v", completed, err)
	}
	var n int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM downloads WHERE chapter_id = 6`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("download not carried to the replacement: %d %v", n, err)
	}
}
