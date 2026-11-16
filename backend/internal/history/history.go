// Package history keeps reading sessions and the stats built from them, so every app shows the same history.
package history

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"
)

// MaxSession caps one session's counted time, so a reader left open overnight doesn't count as reading.
const MaxSession = time.Hour

type Session struct {
	ID             string
	MediaID        int64
	StartChapterID *int64
	EndChapterID   *int64
	StartPage      int64
	EndPage        int64
	StartedAt      int64 // unix ms
	EndedAt        int64
	DurationMs     int64
	Chapters       int64
}

// Day is reading done on one local date.
type Day struct {
	Date     string
	Seconds  int64
	Chapters int64
}

var ErrUnknownTitle = errors.New("no such title on this server")

// Record saves a finished session, or updates it when an app sends the same session again.
func Record(ctx context.Context, db *sql.DB, s Session) (Session, error) {
	s, err := clean(ctx, db, s)
	if err != nil {
		return Session{}, err
	}
	_, err = db.ExecContext(ctx, `INSERT INTO reading_sessions (id, media_id, start_chapter_id, end_chapter_id, start_page, end_page,
		started_at, ended_at, duration_ms, chapters) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET start_chapter_id = excluded.start_chapter_id, end_chapter_id = excluded.end_chapter_id,
		start_page = excluded.start_page, end_page = excluded.end_page, started_at = excluded.started_at,
		ended_at = excluded.ended_at, duration_ms = excluded.duration_ms, chapters = excluded.chapters
		WHERE reading_sessions.media_id = excluded.media_id`,
		s.ID, s.MediaID, s.StartChapterID, s.EndChapterID, s.StartPage, s.EndPage, s.StartedAt, s.EndedAt, s.DurationMs, s.Chapters)
	return s, err
}

// clean checks a session against this server's titles; chapters it doesn't know are dropped rather than refused.
func clean(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, s Session) (Session, error) {
	if s.ID == "" {
		return s, errors.New("session needs an ID")
	}
	var ok bool
	if err := q.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM media WHERE id = ?)`, s.MediaID).Scan(&ok); err != nil {
		return s, err
	}
	if !ok {
		return s, ErrUnknownTitle
	}
	for _, ch := range []**int64{&s.StartChapterID, &s.EndChapterID} {
		if *ch == nil {
			continue
		}
		if err := q.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM chapters WHERE id = ? AND media_id = ?)`, **ch, s.MediaID).Scan(&ok); err != nil {
			return s, err
		}
		if !ok {
			*ch = nil
		}
	}
	if s.EndedAt < s.StartedAt {
		return s, errors.New("session ends before it starts")
	}
	s.DurationMs = max(0, min(s.DurationMs, s.EndedAt-s.StartedAt, MaxSession.Milliseconds()))
	s.Chapters = max(1, s.Chapters)
	s.StartPage, s.EndPage = max(0, s.StartPage), max(0, s.EndPage)
	return s, nil
}

// Import takes in history an app kept on its own. Sessions for titles this server lacks are skipped. When the
// app also sends its per-day totals, those carry the stats and the imported sessions only fill the history list.
func Import(ctx context.Context, db *sql.DB, source string, sessions []Session, days []Day) (imported, skipped int, err error) {
	if source == "" {
		return 0, 0, errors.New("import needs a source")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()
	counted := len(days) == 0
	for _, s := range sessions {
		s, err := clean(ctx, tx, s)
		if errors.Is(err, ErrUnknownTitle) {
			skipped++
			continue
		}
		if err != nil {
			skipped++
			continue
		}
		res, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO reading_sessions (id, media_id, start_chapter_id, end_chapter_id,
			start_page, end_page, started_at, ended_at, duration_ms, chapters, counted) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			s.ID, s.MediaID, s.StartChapterID, s.EndChapterID, s.StartPage, s.EndPage, s.StartedAt, s.EndedAt, s.DurationMs, s.Chapters, counted)
		if err != nil {
			return 0, 0, err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			imported++
		} else {
			skipped++
		}
	}
	for _, d := range days {
		if _, err := time.Parse(time.DateOnly, d.Date); err != nil {
			return 0, 0, fmt.Errorf("day %q: want YYYY-MM-DD", d.Date)
		}
		if d.Seconds <= 0 && d.Chapters <= 0 {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO reading_days_imported (source, day, seconds, chapters) VALUES (?, ?, ?, ?)
			ON CONFLICT(source, day) DO UPDATE SET seconds = excluded.seconds, chapters = excluded.chapters`,
			source, d.Date, max(0, d.Seconds), max(0, d.Chapters)); err != nil {
			return 0, 0, err
		}
	}
	return imported, skipped, tx.Commit()
}

// List returns the history newest first, before a time when given, for one title when mediaID is set.
func List(ctx context.Context, db *sql.DB, limit int, before int64, mediaID *int64) ([]Session, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	if before <= 0 {
		before = 1 << 62
	}
	rows, err := db.QueryContext(ctx, `SELECT id, media_id, start_chapter_id, end_chapter_id, start_page, end_page,
		started_at, ended_at, duration_ms, chapters FROM reading_sessions
		WHERE hidden = 0 AND ended_at < ? AND (? IS NULL OR media_id = ?) ORDER BY ended_at DESC, id LIMIT ?`,
		before, mediaID, mediaID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Session{}
	for rows.Next() {
		var s Session
		var start, end sql.NullInt64
		if err := rows.Scan(&s.ID, &s.MediaID, &start, &end, &s.StartPage, &s.EndPage, &s.StartedAt, &s.EndedAt, &s.DurationMs, &s.Chapters); err != nil {
			return nil, err
		}
		if start.Valid {
			s.StartChapterID = &start.Int64
		}
		if end.Valid {
			s.EndChapterID = &end.Int64
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// Clear hides sessions from the history list, for one title or all; stats keep counting them.
func Clear(ctx context.Context, db *sql.DB, mediaID *int64) (int64, error) {
	res, err := db.ExecContext(ctx, `UPDATE reading_sessions SET hidden = 1 WHERE hidden = 0 AND (? IS NULL OR media_id = ?)`, mediaID, mediaID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Reset deletes all history and the stats built on it.
func Reset(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, q := range []string{`DELETE FROM reading_sessions`, `DELETE FROM reading_days_imported`} {
		if _, err := tx.ExecContext(ctx, q); err != nil {
			return err
		}
	}
	return tx.Commit()
}

type Stats struct {
	TotalChaptersRead int64
	TotalTitlesRead   int64
	TotalMinutesRead  int64
	ChaptersThisWeek  int64
	FirstReadAt       int64
	LastReadAt        int64
	CurrentStreakDays int
	LongestStreakDays int
	LastStreakDate    string
	Days              []Day
}

// ComputeStats builds reading stats with days counted in loc, the reader's time zone.
func ComputeStats(ctx context.Context, db *sql.DB, loc *time.Location, now time.Time) (Stats, error) {
	var st Stats
	days := map[string]*Day{}
	add := func(date string, seconds, chapters int64) {
		d := days[date]
		if d == nil {
			d = &Day{Date: date}
			days[date] = d
		}
		d.Seconds += seconds
		d.Chapters += chapters
	}
	rows, err := db.QueryContext(ctx, `SELECT started_at, ended_at, duration_ms, chapters, counted FROM reading_sessions`)
	if err != nil {
		return st, err
	}
	for rows.Next() {
		var started, ended, dur, chapters int64
		var counted bool
		if err := rows.Scan(&started, &ended, &dur, &chapters, &counted); err != nil {
			rows.Close()
			return st, err
		}
		if counted {
			add(time.UnixMilli(ended).In(loc).Format(time.DateOnly), (dur+500)/1000, chapters)
		}
		if st.FirstReadAt == 0 || started < st.FirstReadAt {
			st.FirstReadAt = started
		}
		st.LastReadAt = max(st.LastReadAt, ended)
	}
	if err := rows.Close(); err != nil {
		return st, err
	}
	rows, err = db.QueryContext(ctx, `SELECT day, SUM(seconds), SUM(chapters) FROM reading_days_imported GROUP BY day`)
	if err != nil {
		return st, err
	}
	for rows.Next() {
		var date string
		var seconds, chapters int64
		if err := rows.Scan(&date, &seconds, &chapters); err != nil {
			rows.Close()
			return st, err
		}
		add(date, seconds, chapters)
		if t, err := time.ParseInLocation(time.DateOnly, date, loc); err == nil && (st.FirstReadAt == 0 || t.UnixMilli() < st.FirstReadAt) {
			st.FirstReadAt = t.UnixMilli()
		}
	}
	if err := rows.Close(); err != nil {
		return st, err
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM reading_progress WHERE completed`).Scan(&st.TotalChaptersRead); err != nil {
		return st, err
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM (SELECT media_id FROM reading_sessions
		UNION SELECT media_id FROM reading_progress WHERE completed)`).Scan(&st.TotalTitlesRead); err != nil {
		return st, err
	}

	st.Days = make([]Day, 0, len(days))
	for _, d := range days {
		st.Days = append(st.Days, *d)
	}
	sort.Slice(st.Days, func(i, j int) bool { return st.Days[i].Date < st.Days[j].Date })
	today := dayIndex(now.In(loc).Format(time.DateOnly))
	var seconds int64
	streak := 0
	for i, d := range st.Days {
		seconds += d.Seconds
		if today-dayIndex(d.Date) < 7 {
			st.ChaptersThisWeek += d.Chapters
		}
		if i > 0 && dayIndex(d.Date)-dayIndex(st.Days[i-1].Date) == 1 {
			streak++
		} else {
			streak = 1
		}
		st.LongestStreakDays = max(st.LongestStreakDays, streak)
	}
	st.TotalMinutesRead = (seconds + 30) / 60
	if n := len(st.Days); n > 0 {
		st.LastStreakDate = st.Days[n-1].Date
		// A streak survives until the end of the day after the last reading day.
		if today-dayIndex(st.LastStreakDate) <= 1 {
			st.CurrentStreakDays = streak
		}
	}
	return st, nil
}

func dayIndex(date string) int {
	t, err := time.Parse(time.DateOnly, date)
	if err != nil {
		return 0
	}
	return int(t.Unix() / 86400)
}
