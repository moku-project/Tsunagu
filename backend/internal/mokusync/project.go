package mokusync

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// position is the chapter's resume point: progress through it, plus video time for episodes.
type position struct {
	Progress float64  `json:"p"`
	Seconds  *float64 `json:"s,omitempty"`
	Duration *float64 `json:"d,omitempty"`
}

var folderFields = []string{"name", "sort_order", "include_in_update", "include_in_download", "content_types"}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func resolveMedia(ctx context.Context, q querier, key string) (int64, bool, error) {
	p, ok := splitKey(key, 2)
	if !ok {
		return 0, false, nil
	}
	var id int64
	err := q.QueryRowContext(ctx, `SELECT m.id FROM extensions e JOIN media m ON m.extension_id = e.id
		WHERE e.package_name = ? AND m.external_id = ?`, p[0], p[1]).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	return id, err == nil, err
}

func resolveChapter(ctx context.Context, q querier, key string) (chapterID, mediaID int64, ok bool, err error) {
	p, ok := splitKey(key, 3)
	if !ok {
		return 0, 0, false, nil
	}
	err = q.QueryRowContext(ctx, `SELECT c.id, c.media_id FROM extensions e JOIN media m ON m.extension_id = e.id
		JOIN chapters c ON c.media_id = m.id WHERE e.package_name = ? AND m.external_id = ? AND c.external_id = ?`,
		p[0], p[1], p[2]).Scan(&chapterID, &mediaID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, false, nil
	}
	return chapterID, mediaID, err == nil, err
}

// readChapter returns the chapter's synced fields as register values.
func readChapter(ctx context.Context, q querier, key string) (map[string]json.RawMessage, bool, error) {
	chapterID, mediaID, ok, err := resolveChapter(ctx, q, key)
	if err != nil || !ok {
		return nil, false, err
	}
	var bookmarked bool
	if err := q.QueryRowContext(ctx, `SELECT bookmarked FROM chapters WHERE id = ?`, chapterID).Scan(&bookmarked); err != nil {
		return nil, false, err
	}
	var completed bool
	var pos position
	var seconds, duration sql.NullFloat64
	err = q.QueryRowContext(ctx, `SELECT completed, progress, position_seconds, duration_seconds FROM reading_progress
		WHERE media_id = ? AND chapter_id = ?`, mediaID, chapterID).Scan(&completed, &pos.Progress, &seconds, &duration)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, false, err
	}
	if seconds.Valid {
		pos.Seconds = &seconds.Float64
	}
	if duration.Valid {
		pos.Duration = &duration.Float64
	}
	return map[string]json.RawMessage{
		"completed":  mustJSON(completed),
		"position":   mustJSON(pos),
		"bookmarked": mustJSON(bookmarked),
	}, true, nil
}

func readFolder(ctx context.Context, q querier, syncID string) (map[string]json.RawMessage, bool, error) {
	var name, types string
	var order, update, download int64
	err := q.QueryRowContext(ctx, `SELECT name, sort_order, include_in_update, include_in_download, content_types
		FROM folders WHERE sync_id = ?`, syncID).Scan(&name, &order, &update, &download, &types)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return map[string]json.RawMessage{
		"name":                mustJSON(name),
		"sort_order":          mustJSON(order),
		"include_in_update":   mustJSON(update != 0),
		"include_in_download": mustJSON(download != 0),
		"content_types":       mustJSON(types),
	}, true, nil
}

func elements(ctx context.Context, q querier, kind, key string) ([]string, error) {
	rows, err := q.QueryContext(ctx, `SELECT DISTINCT element FROM sync_set_adds WHERE kind = ? AND key = ? ORDER BY element`, kind, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func register(ctx context.Context, q querier, kind, key, field string) (json.RawMessage, string, bool, error) {
	var v, clock string
	err := q.QueryRowContext(ctx, `SELECT value, clock FROM sync_registers WHERE kind = ? AND key = ? AND field = ?`,
		kind, key, field).Scan(&v, &clock)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", false, nil
	}
	return json.RawMessage(v), clock, err == nil, err
}

// clockTime renders a clock's wall time the way SQLite's CURRENT_TIMESTAMP does.
func clockTime(clock string) string {
	c, err := ParseClock(clock)
	if err != nil {
		return time.Now().UTC().Format(time.DateTime)
	}
	return time.UnixMilli(c.Wall).UTC().Format(time.DateTime)
}

// project writes the merged value of one target into the library. Targets whose title, chapter or
// folder isn't on this instance yet are skipped; ProjectAll applies them once they exist.
func project(ctx context.Context, tx *sql.Tx, t target) error {
	switch t.kind {
	case KindChapter:
		return projectChapter(ctx, tx, t)
	case KindLibrary:
		mediaID, ok, err := resolveMedia(ctx, tx, t.field)
		if err != nil || !ok {
			return err
		}
		in, err := present(ctx, tx, KindLibrary, t.key, t.field)
		if err != nil {
			return err
		}
		if in {
			// Date added is the oldest add still standing, so it matches on every device.
			var first string
			if err := tx.QueryRowContext(ctx, `SELECT MIN(clock) FROM sync_set_adds WHERE kind = ? AND key = ? AND element = ?`,
				KindLibrary, t.key, t.field).Scan(&first); err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, `UPDATE media SET added_at = COALESCE(added_at, ?) WHERE id = ?`, clockTime(first), mediaID)
		} else {
			_, err = tx.ExecContext(ctx, `UPDATE media SET added_at = NULL WHERE id = ?`, mediaID)
		}
		return err
	case KindFolderMember:
		mediaID, ok, err := resolveMedia(ctx, tx, t.field)
		if err != nil || !ok {
			return err
		}
		var folderID int64
		err = tx.QueryRowContext(ctx, `SELECT id FROM folders WHERE sync_id = ?`, t.key).Scan(&folderID)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		in, err := present(ctx, tx, KindFolderMember, t.key, t.field)
		if err != nil {
			return err
		}
		if in {
			_, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO media_folders (media_id, folder_id) VALUES (?, ?)`, mediaID, folderID)
		} else {
			_, err = tx.ExecContext(ctx, `DELETE FROM media_folders WHERE media_id = ? AND folder_id = ?`, mediaID, folderID)
		}
		return err
	case KindFolder:
		return projectFolder(ctx, tx, t.key)
	}
	return nil
}

func projectChapter(ctx context.Context, tx *sql.Tx, t target) error {
	chapterID, mediaID, ok, err := resolveChapter(ctx, tx, t.key)
	if err != nil || !ok {
		return err
	}
	v, clock, ok, err := register(ctx, tx, KindChapter, t.key, t.field)
	if err != nil || !ok {
		return err
	}
	// History orders by updated_at, so it carries when the read happened, not when it arrived.
	at := clockTime(clock)
	switch t.field {
	case "completed":
		var completed bool
		if err := json.Unmarshal(v, &completed); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO reading_progress (media_id, chapter_id, completed, updated_at) VALUES (?, ?, ?, ?)
			ON CONFLICT(media_id, chapter_id) DO UPDATE SET completed = excluded.completed,
			updated_at = MAX(reading_progress.updated_at, excluded.updated_at)`,
			mediaID, chapterID, completed, at)
	case "position":
		var p position
		if err := json.Unmarshal(v, &p); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO reading_progress (media_id, chapter_id, progress, position_seconds, duration_seconds, updated_at)
			VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(media_id, chapter_id) DO UPDATE SET progress = excluded.progress,
			position_seconds = excluded.position_seconds, duration_seconds = excluded.duration_seconds,
			updated_at = MAX(reading_progress.updated_at, excluded.updated_at)`,
			mediaID, chapterID, p.Progress, p.Seconds, p.Duration, at)
	case "bookmarked":
		var b bool
		if err := json.Unmarshal(v, &b); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE chapters SET bookmarked = ? WHERE id = ?`, b, chapterID)
	}
	return err
}

func projectFolder(ctx context.Context, tx *sql.Tx, syncID string) error {
	vals := map[string]json.RawMessage{}
	for _, f := range append([]string{"deleted"}, folderFields...) {
		v, _, ok, err := register(ctx, tx, KindFolder, syncID, f)
		if err != nil {
			return err
		}
		if ok {
			vals[f] = v
		}
	}
	var deleted bool
	if v, ok := vals["deleted"]; ok {
		if err := json.Unmarshal(v, &deleted); err != nil {
			return err
		}
	}
	if deleted {
		_, err := tx.ExecContext(ctx, `DELETE FROM folders WHERE sync_id = ? AND kind = 'custom'`, syncID)
		return err
	}
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM folders WHERE sync_id = ?)`, syncID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		// System folders exist everywhere already; a custom one needs its name before it can be created.
		if strings.HasPrefix(syncID, "sys:") || vals["name"] == nil {
			return nil
		}
		var name string
		if err := json.Unmarshal(vals["name"], &name); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO folders (name, kind, sync_id) VALUES (?, 'custom', ?)`, name, syncID); err != nil {
			return fmt.Errorf("create folder: %w", err)
		}
		// Memberships that arrived before the folder were skipped; apply them now.
		members, err := elements(ctx, tx, KindFolderMember, syncID)
		if err != nil {
			return err
		}
		for _, m := range members {
			if err := project(ctx, tx, target{KindFolderMember, syncID, m}); err != nil {
				return err
			}
		}
	}
	for _, f := range folderFields {
		v, ok := vals[f]
		if !ok {
			continue
		}
		var arg any
		switch f {
		case "name", "content_types":
			var s string
			if err := json.Unmarshal(v, &s); err != nil {
				return err
			}
			arg = s
		case "sort_order":
			var n int64
			if err := json.Unmarshal(v, &n); err != nil {
				return err
			}
			arg = n
		default:
			var b bool
			if err := json.Unmarshal(v, &b); err != nil {
				return err
			}
			arg = b
		}
		if _, err := tx.ExecContext(ctx, `UPDATE folders SET `+f+` = ? WHERE sync_id = ?`, arg, syncID); err != nil {
			return fmt.Errorf("update folder %s: %w", f, err)
		}
	}
	return nil
}
