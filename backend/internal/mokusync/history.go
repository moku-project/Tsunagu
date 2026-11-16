package mokusync

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

// Reading history kinds. Sessions are immutable apart from being hidden; imported days are fixed totals.
const (
	KindSession    = "session"     // key: session ID; field "data"
	KindReadingDay = "reading_day" // key: importing device and local date; field "totals"
)

// sessionValue names its title and chapters by key, so the session lands on the right rows on every device.
type sessionValue struct {
	Media     string `json:"media"`
	Start     string `json:"start,omitempty"`
	End       string `json:"end,omitempty"`
	StartPage int64  `json:"startPage"`
	EndPage   int64  `json:"endPage"`
	StartedAt int64  `json:"startedAt"`
	EndedAt   int64  `json:"endedAt"`
	Duration  int64  `json:"durationMs"`
	Chapters  int64  `json:"chapters"`
	Hidden    bool   `json:"hidden,omitempty"`
	Counted   bool   `json:"counted"`
}

var jsonNull = json.RawMessage("null")

func (e *Engine) captureSession(ctx context.Context, tx *sql.Tx, id string, wall int64) (int, error) {
	var v sessionValue
	err := tx.QueryRowContext(ctx, `SELECT mk.key, COALESCE(sk.key, ''), COALESCE(ek.key, ''), s.start_page, s.end_page,
		s.started_at, s.ended_at, s.duration_ms, s.chapters, s.hidden, s.counted
		FROM reading_sessions s JOIN sync_media_keys mk ON mk.media_id = s.media_id
		LEFT JOIN sync_chapter_keys sk ON sk.chapter_id = s.start_chapter_id
		LEFT JOIN sync_chapter_keys ek ON ek.chapter_id = s.end_chapter_id WHERE s.id = ?`, id).
		Scan(&v.Media, &v.Start, &v.End, &v.StartPage, &v.EndPage, &v.StartedAt, &v.EndedAt, &v.Duration, &v.Chapters, &v.Hidden, &v.Counted)
	if errors.Is(err, sql.ErrNoRows) {
		var exists bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM reading_sessions WHERE id = ?)`, id).Scan(&exists); err != nil || exists {
			// Titles from the local source have no key other devices could resolve.
			return 0, err
		}
		return e.clearRegister(ctx, tx, KindSession, id, "data", wall)
	}
	if err != nil {
		return 0, err
	}
	// A chapter this device never fetched keeps the key another device sent, instead of losing it on a later edit.
	if old, _, ok, err := register(ctx, tx, KindSession, id, "data"); err != nil {
		return 0, err
	} else if ok {
		var prev sessionValue
		if json.Unmarshal(old, &prev) == nil {
			if v.Start == "" {
				v.Start = prev.Start
			}
			if v.End == "" {
				v.End = prev.End
			}
		}
	}
	return e.emitSet(ctx, tx, KindSession, id, "data", mustJSON(v), wall)
}

func (e *Engine) captureReadingDay(ctx context.Context, tx *sql.Tx, key string, wall int64) (int, error) {
	p, ok := splitKey(key, 2)
	if !ok {
		return 0, nil
	}
	var totals [2]int64
	err := tx.QueryRowContext(ctx, `SELECT seconds, chapters FROM reading_days_imported WHERE source = ? AND day = ?`, p[0], p[1]).
		Scan(&totals[0], &totals[1])
	if errors.Is(err, sql.ErrNoRows) {
		return e.clearRegister(ctx, tx, KindReadingDay, key, "totals", wall)
	}
	if err != nil {
		return 0, err
	}
	return e.emitSet(ctx, tx, KindReadingDay, key, "totals", mustJSON(totals), wall)
}

// clearRegister records a deletion, only for something that was ever synced.
func (e *Engine) clearRegister(ctx context.Context, tx *sql.Tx, kind, key, field string, wall int64) (int, error) {
	if _, _, ok, err := register(ctx, tx, kind, key, field); err != nil || !ok {
		return 0, err
	}
	return e.emitSet(ctx, tx, kind, key, field, jsonNull, wall)
}

// projectSession writes a synced session; it stays pending until its title, and any chapters it names, exist here.
func projectSession(ctx context.Context, tx *sql.Tx, id string) (bool, error) {
	raw, _, ok, err := register(ctx, tx, KindSession, id, "data")
	if err != nil || !ok {
		return err == nil, err
	}
	if string(raw) == "null" {
		_, err := tx.ExecContext(ctx, `DELETE FROM reading_sessions WHERE id = ?`, id)
		return err == nil, err
	}
	var v sessionValue
	if err := json.Unmarshal(raw, &v); err != nil {
		return true, nil
	}
	mediaID, ok, err := resolveMedia(ctx, tx, v.Media)
	if err != nil || !ok {
		return false, err
	}
	complete := true
	chapter := func(key string) (any, error) {
		if key == "" {
			return nil, nil
		}
		id, _, ok, err := resolveChapter(ctx, tx, key)
		if err != nil || !ok {
			complete = false
			return nil, err
		}
		return id, nil
	}
	start, err := chapter(v.Start)
	if err != nil {
		return false, err
	}
	end, err := chapter(v.End)
	if err != nil {
		return false, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO reading_sessions (id, media_id, start_chapter_id, end_chapter_id, start_page, end_page,
		started_at, ended_at, duration_ms, chapters, hidden, counted) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET media_id = excluded.media_id, start_chapter_id = excluded.start_chapter_id,
		end_chapter_id = excluded.end_chapter_id, start_page = excluded.start_page, end_page = excluded.end_page,
		started_at = excluded.started_at, ended_at = excluded.ended_at, duration_ms = excluded.duration_ms,
		chapters = excluded.chapters, hidden = excluded.hidden, counted = excluded.counted`,
		id, mediaID, start, end, v.StartPage, v.EndPage, v.StartedAt, v.EndedAt, v.Duration, v.Chapters, v.Hidden, v.Counted)
	return complete && err == nil, err
}

func projectReadingDay(ctx context.Context, tx *sql.Tx, key string) (bool, error) {
	p, ok := splitKey(key, 2)
	if !ok {
		return true, nil
	}
	raw, _, ok, err := register(ctx, tx, KindReadingDay, key, "totals")
	if err != nil || !ok {
		return err == nil, err
	}
	var totals [2]int64
	if string(raw) == "null" || json.Unmarshal(raw, &totals) != nil {
		_, err := tx.ExecContext(ctx, `DELETE FROM reading_days_imported WHERE source = ? AND day = ?`, p[0], p[1])
		return err == nil, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO reading_days_imported (source, day, seconds, chapters) VALUES (?, ?, ?, ?)
		ON CONFLICT(source, day) DO UPDATE SET seconds = excluded.seconds, chapters = excluded.chapters`,
		p[0], p[1], totals[0], totals[1])
	return err == nil, err
}
