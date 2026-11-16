package history

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"tsunagu/backend/internal/db"
)

func setup(t *testing.T) (*sql.DB, context.Context) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	ctx := context.Background()
	for _, q := range []string{
		`INSERT INTO repositories (index_url, name) VALUES ('https://example.com/index.json', 'Example')`,
		`INSERT INTO extensions (repository_id, package_name, name, version, content_type, lang, apk_url, installed) VALUES (1, 'example.source', 'Example', '1', 'manga', 'en', 'x.apk', 1)`,
		`INSERT INTO media (id, extension_id, external_id, content_type, title) VALUES (1, 1, '/a', 'manga', 'A'), (2, 1, '/b', 'manga', 'B')`,
		`INSERT INTO chapters (id, media_id, external_id, number) VALUES (10, 1, '/a/1', 1), (11, 1, '/a/2', 2), (20, 2, '/b/1', 1)`,
	} {
		if _, err := conn.ExecContext(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	return conn, ctx
}

func ptr(v int64) *int64 { return &v }

var berlin, _ = time.LoadLocation("Europe/Berlin")

func at(s string) int64 {
	t, _ := time.ParseInLocation("2006-01-02 15:04", s, berlin)
	return t.UnixMilli()
}

func TestHistoryAndStats(t *testing.T) {
	conn, ctx := setup(t)
	record := func(s Session) {
		t.Helper()
		if _, err := Record(ctx, conn, s); err != nil {
			t.Fatal(err)
		}
	}
	// Three days in a row; the last session ends at 00:30 local time, which is still the previous day in UTC.
	record(Session{ID: "s1", MediaID: 1, StartChapterID: ptr(10), EndChapterID: ptr(11), StartedAt: at("2026-11-10 20:00"), EndedAt: at("2026-11-10 20:30"), DurationMs: 30 * 60_000, Chapters: 2})
	record(Session{ID: "s2", MediaID: 2, StartChapterID: ptr(20), EndChapterID: ptr(10), StartedAt: at("2026-11-11 20:00"), EndedAt: at("2026-11-11 23:00"), DurationMs: 3 * 3600_000, Chapters: 1})
	record(Session{ID: "s3", MediaID: 1, StartedAt: at("2026-11-12 00:10"), EndedAt: at("2026-11-12 00:30"), DurationMs: 20 * 60_000, Chapters: 1})
	record(Session{ID: "s3", MediaID: 1, StartedAt: at("2026-11-12 00:10"), EndedAt: at("2026-11-12 00:40"), DurationMs: 30 * 60_000, Chapters: 1})

	if _, err := Record(ctx, conn, Session{ID: "x", MediaID: 99, StartedAt: 1, EndedAt: 2}); !errors.Is(err, ErrUnknownTitle) {
		t.Fatalf("a session for a missing title: %v", err)
	}
	list, err := List(ctx, conn, 10, 0, nil)
	if err != nil || len(list) != 3 || list[0].ID != "s3" || list[2].ID != "s1" {
		t.Fatalf("history %+v %v", list, err)
	}
	if list[1].EndChapterID != nil || list[1].DurationMs != 3600_000 {
		t.Fatalf("a chapter from another title or an overlong session was kept: %+v", list[1])
	}

	st, err := ComputeStats(ctx, conn, berlin, time.UnixMilli(at("2026-11-13 09:00")))
	if err != nil {
		t.Fatal(err)
	}
	if st.CurrentStreakDays != 3 || st.LongestStreakDays != 3 || st.LastStreakDate != "2026-11-12" || len(st.Days) != 3 {
		t.Fatalf("streaks %+v", st)
	}
	if st.TotalMinutesRead != 30+60+30 || st.ChaptersThisWeek != 4 || st.TotalTitlesRead != 2 {
		t.Fatalf("totals %+v", st)
	}
	if late, _ := ComputeStats(ctx, conn, berlin, time.UnixMilli(at("2026-11-14 09:00"))); late.CurrentStreakDays != 0 {
		t.Fatalf("a streak survived a missed day: %d", late.CurrentStreakDays)
	}

	// Clearing history keeps the stats.
	if n, err := Clear(ctx, conn, ptr(1)); err != nil || n != 2 {
		t.Fatalf("cleared %d, %v", n, err)
	}
	if list, _ := List(ctx, conn, 10, 0, nil); len(list) != 1 || list[0].ID != "s2" {
		t.Fatalf("after clearing one title %+v", list)
	}
	if again, _ := ComputeStats(ctx, conn, berlin, time.UnixMilli(at("2026-11-13 09:00"))); again.TotalMinutesRead != st.TotalMinutesRead {
		t.Fatalf("clearing history changed stats: %d", again.TotalMinutesRead)
	}
	if err := Reset(ctx, conn); err != nil {
		t.Fatal(err)
	}
	if st, _ := ComputeStats(ctx, conn, berlin, time.Now()); st.TotalMinutesRead != 0 || len(st.Days) != 0 {
		t.Fatalf("after reset %+v", st)
	}
}

func TestImportKeepsOldStatsWithoutCountingTwice(t *testing.T) {
	conn, ctx := setup(t)
	sessions := []Session{
		{ID: "old1", MediaID: 1, StartedAt: at("2026-10-01 10:00"), EndedAt: at("2026-10-01 10:30"), DurationMs: 30 * 60_000, Chapters: 2},
		{ID: "gone", MediaID: 42, StartedAt: at("2026-10-01 11:00"), EndedAt: at("2026-10-01 11:30"), DurationMs: 30 * 60_000},
	}
	// The app's own totals cover more than its capped session list did.
	days := []Day{{Date: "2026-09-30", Seconds: 600, Chapters: 1}, {Date: "2026-10-01", Seconds: 3600, Chapters: 3}}
	imported, skipped, err := Import(ctx, conn, "moku-desktop", sessions, days)
	if err != nil || imported != 1 || skipped != 1 {
		t.Fatalf("imported %d skipped %d %v", imported, skipped, err)
	}
	if imported, skipped, _ := Import(ctx, conn, "moku-desktop", sessions, days); imported != 0 || skipped != 2 {
		t.Fatalf("a retried import added %d", imported)
	}
	st, err := ComputeStats(ctx, conn, berlin, time.UnixMilli(at("2026-10-02 09:00")))
	if err != nil {
		t.Fatal(err)
	}
	if st.TotalMinutesRead != 70 || st.CurrentStreakDays != 2 || st.FirstReadAt != at("2026-09-30 00:00") {
		t.Fatalf("imported stats %+v", st)
	}
	if list, _ := List(ctx, conn, 10, 0, nil); len(list) != 1 || list[0].ID != "old1" {
		t.Fatalf("imported history %+v", list)
	}
}
