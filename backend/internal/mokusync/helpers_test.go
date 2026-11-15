package mokusync

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"tsunagu/backend/internal/db"
)

const (
	testTitles   = 6
	testChapters = 5
)

// node is one Tsunagu instance with its own database, seeded with the same titles as every other node.
type node struct {
	t    testing.TB
	ctx  context.Context
	e    *Engine
	db   *sql.DB
	wall int64
}

// newNode inserts titles in the given order, so local row IDs differ between nodes the way they do between devices.
func newNode(t testing.TB, name string, order []int) *node {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "tsunagu.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	ctx := context.Background()
	e, err := Open(ctx, conn)
	if err != nil {
		t.Fatal(err)
	}
	e.name = name
	n := &node{t: t, ctx: ctx, e: e, db: conn, wall: 1_700_000_000_000}
	e.wall = func() int64 { return n.wall }
	n.exec(`INSERT INTO repositories (index_url, name) VALUES ('https://example.com/index.json', 'Example')`)
	n.exec(`INSERT INTO extensions (repository_id, package_name, name, version, content_type, lang, apk_url, installed)
		VALUES (1, 'example.source', 'Example', '1', 'manga', 'en', 'x.apk', 1)`)
	if order == nil {
		order = make([]int, testTitles)
		for i := range order {
			order[i] = i
		}
	}
	for _, m := range order {
		n.addTitle(m)
	}
	n.capture()
	return n
}

func (n *node) addTitle(m int) {
	n.exec(`INSERT INTO media (extension_id, external_id, content_type, title) VALUES (1, ?, 'manga', ?)`, mediaExt(m), fmt.Sprintf("Title %d", m))
	for c := testChapters - 1; c >= 0; c-- {
		n.exec(`INSERT INTO chapters (media_id, external_id, number) VALUES ((SELECT id FROM media WHERE external_id = ?), ?, ?)`,
			mediaExt(m), chapterExt(c), c+1)
	}
}

func mediaExt(m int) string   { return fmt.Sprintf("/title/%d", m) }
func chapterExt(c int) string { return fmt.Sprintf("/chapter/%d", c) }
func mediaKey(m int) string   { return "example.source" + KeySep + mediaExt(m) }

func (n *node) exec(q string, args ...any) {
	n.t.Helper()
	if _, err := n.db.ExecContext(n.ctx, q, args...); err != nil {
		n.t.Fatalf("%s: %v", strings.Fields(q)[0], err)
	}
}

func (n *node) capture() int {
	n.t.Helper()
	k, err := n.e.Capture(n.ctx)
	if err != nil {
		n.t.Fatal(err)
	}
	return k
}

func (n *node) syncWith(o *node) {
	n.t.Helper()
	if err := n.e.SyncWith(n.ctx, LocalPeer{o.e}, o.e.Node()); err != nil {
		n.t.Fatalf("%s sync with %s: %v", n.e.name, o.e.name, err)
	}
}

const chapterIDs = `(SELECT c.id FROM chapters c JOIN media m ON m.id = c.media_id WHERE m.external_id = ? AND c.external_id = ?)`
const mediaID = `(SELECT id FROM media WHERE external_id = ?)`

func (n *node) setRead(m, c int, read bool) {
	n.exec(`INSERT INTO reading_progress (media_id, chapter_id, completed) VALUES (`+mediaID+`, `+chapterIDs+`, ?)
		ON CONFLICT(media_id, chapter_id) DO UPDATE SET completed = excluded.completed`, mediaExt(m), mediaExt(m), chapterExt(c), read)
}

func (n *node) setPosition(m, c int, progress float64) {
	n.exec(`INSERT INTO reading_progress (media_id, chapter_id, progress) VALUES (`+mediaID+`, `+chapterIDs+`, ?)
		ON CONFLICT(media_id, chapter_id) DO UPDATE SET progress = excluded.progress`, mediaExt(m), mediaExt(m), chapterExt(c), progress)
}

func (n *node) setEpisodePosition(m, c int, seconds, duration float64) {
	n.exec(`INSERT INTO reading_progress (media_id, chapter_id, progress, position_seconds, duration_seconds)
		VALUES (`+mediaID+`, `+chapterIDs+`, ?, ?, ?) ON CONFLICT(media_id, chapter_id) DO UPDATE SET
		progress = excluded.progress, position_seconds = excluded.position_seconds, duration_seconds = excluded.duration_seconds`,
		mediaExt(m), mediaExt(m), chapterExt(c), seconds/duration, seconds, duration)
}

func (n *node) setBookmark(m, c int, b bool) {
	n.exec(`UPDATE chapters SET bookmarked = ? WHERE id = `+chapterIDs, b, mediaExt(m), chapterExt(c))
}

func (n *node) setInLibrary(m int, in bool) {
	if in {
		n.exec(`UPDATE media SET added_at = COALESCE(added_at, CURRENT_TIMESTAMP) WHERE external_id = ?`, mediaExt(m))
	} else {
		n.exec(`UPDATE media SET added_at = NULL WHERE external_id = ?`, mediaExt(m))
	}
}

func (n *node) createFolder(name string) string {
	n.exec(`INSERT INTO folders (name, kind) VALUES (?, 'custom')`, name)
	var id string
	if err := n.db.QueryRowContext(n.ctx, `SELECT sync_id FROM folders WHERE id = last_insert_rowid()`).Scan(&id); err != nil {
		n.t.Fatal(err)
	}
	return id
}

func (n *node) renameFolder(id, name string) {
	n.exec(`UPDATE folders SET name = ? WHERE sync_id = ?`, name, id)
}
func (n *node) deleteFolder(id string) { n.exec(`DELETE FROM folders WHERE sync_id = ?`, id) }

func (n *node) setInFolder(id string, m int, in bool) {
	if in {
		n.exec(`INSERT OR IGNORE INTO media_folders (media_id, folder_id) SELECT `+mediaID+`, id FROM folders WHERE sync_id = ?`, mediaExt(m), id)
	} else {
		n.exec(`DELETE FROM media_folders WHERE media_id = `+mediaID+` AND folder_id = (SELECT id FROM folders WHERE sync_id = ?)`, mediaExt(m), id)
	}
}

func (n *node) folderIDs() []string {
	return n.strings(`SELECT sync_id FROM folders WHERE kind = 'custom' ORDER BY sync_id`)
}

func (n *node) strings(q string, args ...any) []string {
	n.t.Helper()
	rows, err := n.db.QueryContext(n.ctx, q, args...)
	if err != nil {
		n.t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			n.t.Fatal(err)
		}
		out = append(out, s)
	}
	return out
}

// crdtState lists every register and present set element, with clocks.
func (n *node) crdtState() []string {
	regs := n.strings(`SELECT kind || ' ' || replace(key, char(31), '|') || ' ' || field || ' = ' || value || ' @' || clock
		FROM sync_registers ORDER BY kind, key, field`)
	elems := n.strings(`SELECT DISTINCT kind || ' ' || key || ' has ' || replace(element, char(31), '|')
		FROM sync_set_adds ORDER BY 1`)
	return append(regs, elems...)
}

// libraryState describes the library as the app sees it, keyed by source IDs instead of local row IDs.
// A chapter with no progress row reads the same as one with an untouched row.
func (n *node) libraryState() []string {
	out := n.strings(`SELECT 'library ' || external_id FROM media WHERE added_at IS NOT NULL ORDER BY external_id`)
	out = append(out, n.strings(`SELECT 'chapter ' || m.external_id || c.external_id || ' read=' || COALESCE(rp.completed, 0)
		|| ' p=' || COALESCE(rp.progress, 0) || ' s=' || COALESCE(rp.position_seconds, '-') || ' d=' || COALESCE(rp.duration_seconds, '-')
		|| ' bookmarked=' || c.bookmarked
		FROM chapters c JOIN media m ON m.id = c.media_id
		LEFT JOIN reading_progress rp ON rp.chapter_id = c.id AND rp.media_id = m.id
		ORDER BY m.external_id, c.external_id`)...)
	out = append(out, n.strings(`SELECT 'folder ' || sync_id || ' ' || name || ' order=' || sort_order || ' types=' || content_types
		|| ' update=' || include_in_update || ' download=' || include_in_download FROM folders ORDER BY sync_id`)...)
	out = append(out, n.strings(`SELECT 'member ' || f.sync_id || ' ' || m.external_id FROM media_folders mf
		JOIN folders f ON f.id = mf.folder_id JOIN media m ON m.id = mf.media_id ORDER BY 1`)...)
	return out
}

func (n *node) chapterRow(m, c int) (read bool, progress float64, bookmarked bool) {
	n.t.Helper()
	err := n.db.QueryRowContext(n.ctx, `SELECT COALESCE(rp.completed, 0), COALESCE(rp.progress, 0), c.bookmarked
		FROM chapters c JOIN media m ON m.id = c.media_id
		LEFT JOIN reading_progress rp ON rp.chapter_id = c.id AND rp.media_id = m.id
		WHERE m.external_id = ? AND c.external_id = ?`, mediaExt(m), chapterExt(c)).Scan(&read, &progress, &bookmarked)
	if err != nil {
		n.t.Fatal(err)
	}
	return
}

func (n *node) inLibrary(m int) bool {
	var in bool
	if err := n.db.QueryRowContext(n.ctx, `SELECT added_at IS NOT NULL FROM media WHERE external_id = ?`, mediaExt(m)).Scan(&in); err != nil {
		n.t.Fatal(err)
	}
	return in
}

func (n *node) opCount() int {
	var c int
	if err := n.db.QueryRowContext(n.ctx, `SELECT COUNT(*) FROM sync_ops`).Scan(&c); err != nil {
		n.t.Fatal(err)
	}
	return c
}

func diff(a, b []string) string {
	set := map[string]int{}
	for _, s := range a {
		set[s]++
	}
	for _, s := range b {
		set[s]--
	}
	var out []string
	for s, c := range set {
		if c > 0 {
			out = append(out, "- "+s)
		} else if c < 0 {
			out = append(out, "+ "+s)
		}
	}
	sort.Strings(out)
	return strings.Join(out, "\n")
}
