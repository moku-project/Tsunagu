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

func customFolders(ctx context.Context, q querier) (map[string]bool, error) {
	rows, err := q.QueryContext(ctx, `SELECT sync_id FROM folders WHERE kind = 'custom'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

// canonicalFolder follows merged_into to the folder a merged folder now lives in. Merges always point
// at a smaller ID, so the chain ends; the depth cap only guards against corrupt data.
func canonicalFolder(ctx context.Context, q querier, syncID string) (string, error) {
	for range 32 {
		v, _, ok, err := register(ctx, q, KindFolder, syncID, "merged_into")
		if err != nil || !ok {
			return syncID, err
		}
		var next string
		if err := json.Unmarshal(v, &next); err != nil {
			return "", err
		}
		if next >= syncID {
			return syncID, nil
		}
		syncID = next
	}
	return syncID, nil
}

// folderAliases lists canon and every live folder merged into it, canon first.
func folderAliases(ctx context.Context, q querier, canon string) ([]string, error) {
	rows, err := q.QueryContext(ctx, `SELECT m.key FROM sync_registers m WHERE m.kind = 'folder' AND m.field = 'merged_into'
		AND NOT EXISTS (SELECT 1 FROM sync_registers d WHERE d.kind = 'folder' AND d.key = m.key AND d.field = 'deleted' AND d.value = 'true')
		ORDER BY m.key`)
	if err != nil {
		return nil, err
	}
	var merged []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			rows.Close()
			return nil, err
		}
		merged = append(merged, k)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	out := []string{canon}
	for _, k := range merged {
		c, err := canonicalFolder(ctx, q, k)
		if err != nil {
			return nil, err
		}
		if c == canon && k != canon {
			out = append(out, k)
		}
	}
	return out, nil
}

// projectAllMembers re-applies every synced title's folders: a merge, deletion or new folder can move titles
// several merges along. These events are rare, so the full pass is cheap overall.
func projectAllMembers(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `SELECT DISTINCT key, element FROM sync_set_adds WHERE kind = 'folder_member' ORDER BY element, key`)
	if err != nil {
		return err
	}
	var targets []target
	for rows.Next() {
		t := target{kind: KindFolderMember}
		if err := rows.Scan(&t.key, &t.field); err != nil {
			rows.Close()
			return err
		}
		targets = append(targets, t)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	done := map[string]bool{}
	for _, t := range targets {
		if done[t.field] {
			continue
		}
		done[t.field] = true
		if err := project(ctx, tx, t); err != nil {
			return err
		}
	}
	return nil
}

func folderDeleted(ctx context.Context, q querier, key string) (bool, error) {
	v, _, ok, err := register(ctx, q, KindFolder, key, "deleted")
	return ok && string(v) == "true", err
}

// projectTitleFolders works per title, not per folder, so the result is right however merges and deletions arrive.
func projectTitleFolders(ctx context.Context, tx *sql.Tx, element string) (bool, error) {
	mediaID, ok, err := resolveMedia(ctx, tx, element)
	if err != nil || !ok {
		return false, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT DISTINCT key FROM sync_set_adds WHERE kind = 'folder_member' AND element = ?`, element)
	if err != nil {
		return false, err
	}
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			rows.Close()
			return false, err
		}
		keys = append(keys, k)
	}
	if err := rows.Close(); err != nil {
		return false, err
	}
	complete := true
	want := map[int64]bool{}
	for _, k := range keys {
		canon, err := canonicalFolder(ctx, tx, k)
		if err != nil {
			return false, err
		}
		gone, err := folderDeleted(ctx, tx, k)
		if err != nil {
			return false, err
		}
		if gone {
			continue
		}
		if gone, err = folderDeleted(ctx, tx, canon); err != nil || gone {
			if err != nil {
				return false, err
			}
			continue
		}
		var id int64
		err = tx.QueryRowContext(ctx, `SELECT id FROM folders WHERE sync_id = ?`, canon).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			complete = false
			continue
		}
		if err != nil {
			return false, err
		}
		want[id] = true
	}
	have := map[int64]bool{}
	hrows, err := tx.QueryContext(ctx, `SELECT folder_id FROM media_folders WHERE media_id = ?`, mediaID)
	if err != nil {
		return false, err
	}
	for hrows.Next() {
		var id int64
		if err := hrows.Scan(&id); err != nil {
			hrows.Close()
			return false, err
		}
		have[id] = true
	}
	if err := hrows.Close(); err != nil {
		return false, err
	}
	for id := range want {
		if !have[id] {
			if _, err := tx.ExecContext(ctx, `INSERT INTO media_folders (media_id, folder_id) VALUES (?, ?)`, mediaID, id); err != nil {
				return false, err
			}
		}
	}
	for id := range have {
		if !want[id] {
			if _, err := tx.ExecContext(ctx, `DELETE FROM media_folders WHERE media_id = ? AND folder_id = ?`, mediaID, id); err != nil {
				return false, err
			}
		}
	}
	return complete, nil
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

// project writes the merged value of one target into the library. A target whose title, chapter or
// folder isn't on this instance yet is kept in sync_pending until ProjectPending can apply it.
func project(ctx context.Context, tx *sql.Tx, t target) error {
	applied, err := projectTarget(ctx, tx, t)
	if err != nil {
		return err
	}
	if applied {
		_, err = tx.ExecContext(ctx, `DELETE FROM sync_pending WHERE kind = ? AND key = ? AND field = ?`, t.kind, t.key, t.field)
	} else {
		_, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO sync_pending (kind, key, field) VALUES (?, ?, ?)`, t.kind, t.key, t.field)
	}
	return err
}

// projectTarget reports false when the row the value belongs on doesn't exist here yet.
func projectTarget(ctx context.Context, tx *sql.Tx, t target) (bool, error) {
	switch t.kind {
	case KindChapter:
		return projectChapter(ctx, tx, t)
	case KindLibrary:
		mediaID, ok, err := resolveMedia(ctx, tx, t.field)
		if err != nil || !ok {
			return false, err
		}
		in, err := present(ctx, tx, KindLibrary, t.key, t.field)
		if err != nil {
			return false, err
		}
		if in {
			// Date added is the oldest add still standing, so it matches on every device.
			var first string
			if err := tx.QueryRowContext(ctx, `SELECT MIN(clock) FROM sync_set_adds WHERE kind = ? AND key = ? AND element = ?`,
				KindLibrary, t.key, t.field).Scan(&first); err != nil {
				return false, err
			}
			_, err = tx.ExecContext(ctx, `UPDATE media SET added_at = COALESCE(added_at, ?) WHERE id = ?`, clockTime(first), mediaID)
		} else {
			_, err = tx.ExecContext(ctx, `UPDATE media SET added_at = NULL WHERE id = ?`, mediaID)
		}
		return err == nil, err
	case KindFolderMember:
		return projectTitleFolders(ctx, tx, t.field)
	case KindFolder:
		return projectFolder(ctx, tx, t.key)
	case KindSession:
		return projectSession(ctx, tx, t.key)
	case KindReadingDay:
		return projectReadingDay(ctx, tx, t.key)
	}
	return true, nil
}

func projectChapter(ctx context.Context, tx *sql.Tx, t target) (bool, error) {
	chapterID, mediaID, ok, err := resolveChapter(ctx, tx, t.key)
	if err != nil || !ok {
		return false, err
	}
	v, clock, ok, err := register(ctx, tx, KindChapter, t.key, t.field)
	if err != nil || !ok {
		return err == nil, err
	}
	// History orders by updated_at, so it carries when the read happened, not when it arrived.
	at := clockTime(clock)
	switch t.field {
	case "completed":
		var completed bool
		if err := json.Unmarshal(v, &completed); err != nil {
			return false, err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO reading_progress (media_id, chapter_id, completed, updated_at) VALUES (?, ?, ?, ?)
			ON CONFLICT(media_id, chapter_id) DO UPDATE SET completed = excluded.completed,
			updated_at = MAX(reading_progress.updated_at, excluded.updated_at)`,
			mediaID, chapterID, completed, at)
	case "position":
		var p position
		if err := json.Unmarshal(v, &p); err != nil {
			return false, err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO reading_progress (media_id, chapter_id, progress, position_seconds, duration_seconds, updated_at)
			VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(media_id, chapter_id) DO UPDATE SET progress = excluded.progress,
			position_seconds = excluded.position_seconds, duration_seconds = excluded.duration_seconds,
			updated_at = MAX(reading_progress.updated_at, excluded.updated_at)`,
			mediaID, chapterID, p.Progress, p.Seconds, p.Duration, at)
	case "bookmarked":
		var b bool
		if err := json.Unmarshal(v, &b); err != nil {
			return false, err
		}
		_, err = tx.ExecContext(ctx, `UPDATE chapters SET bookmarked = ? WHERE id = ?`, b, chapterID)
	}
	return err == nil, err
}

func projectFolder(ctx context.Context, tx *sql.Tx, syncID string) (bool, error) {
	canon, err := canonicalFolder(ctx, tx, syncID)
	if err != nil {
		return false, err
	}
	if canon != syncID {
		// Merged away: its titles now show in the folder it merged into.
		if _, err := tx.ExecContext(ctx, `DELETE FROM folders WHERE sync_id = ? AND kind = 'custom'`, syncID); err != nil {
			return false, err
		}
		if _, err := projectFolder(ctx, tx, canon); err != nil {
			return false, err
		}
		return true, projectAllMembers(ctx, tx)
	}
	vals := map[string]json.RawMessage{}
	for _, f := range append([]string{"deleted"}, folderFields...) {
		v, _, ok, err := register(ctx, tx, KindFolder, syncID, f)
		if err != nil {
			return false, err
		}
		if ok {
			vals[f] = v
		}
	}
	var deleted bool
	if v, ok := vals["deleted"]; ok {
		if err := json.Unmarshal(v, &deleted); err != nil {
			return false, err
		}
	}
	if deleted {
		_, err := tx.ExecContext(ctx, `DELETE FROM folders WHERE sync_id = ? AND kind = 'custom'`, syncID)
		return err == nil, err
	}
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM folders WHERE sync_id = ?)`, syncID).Scan(&exists); err != nil {
		return false, err
	}
	if !exists {
		// System folders exist everywhere already; a custom one needs its name before it can be created.
		if strings.HasPrefix(syncID, "sys:") || vals["name"] == nil {
			return false, nil
		}
		var name string
		if err := json.Unmarshal(vals["name"], &name); err != nil {
			return false, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO folders (name, kind, sync_id) VALUES (?, 'custom', ?)`, name, syncID); err != nil {
			return false, fmt.Errorf("create folder: %w", err)
		}
		// Memberships that arrived before the folder, here or in folders merged into it, apply now.
		if err := projectAllMembers(ctx, tx); err != nil {
			return false, err
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
				return false, err
			}
			arg = s
		case "sort_order":
			var n int64
			if err := json.Unmarshal(v, &n); err != nil {
				return false, err
			}
			arg = n
		default:
			var b bool
			if err := json.Unmarshal(v, &b); err != nil {
				return false, err
			}
			arg = b
		}
		if _, err := tx.ExecContext(ctx, `UPDATE folders SET `+f+` = ? WHERE sync_id = ?`, arg, syncID); err != nil {
			return false, fmt.Errorf("update folder %s: %w", f, err)
		}
	}
	return true, nil
}
