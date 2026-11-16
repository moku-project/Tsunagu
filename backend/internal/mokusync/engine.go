package mokusync

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"
)

// Engine owns this instance's sync state in the Tsunagu database.
type Engine struct {
	db     *sql.DB
	node   string
	nameMu sync.Mutex
	name   string
	hlc    *HLC
	mu     sync.Mutex
	// wall overrides the time local changes are stamped with; tests use it to stage concurrent edits.
	wall     func() int64
	resolve  resolveState
	resolver Resolver
	notify   notifier
	// newOps is set while a write stores ops this instance hadn't seen; guarded by mu.
	newOps bool
	// remoteChanged tells the runner applied ops may have left titles to fetch.
	remoteChanged atomic.Bool
	// syncPort is the LAN port peers reach this instance on; 0 when it only dials out.
	syncPort atomic.Int32
	discover func(ctx context.Context, self string) ([]Nearby, error)
	// unreachable counts dialed peers whose watcher is failing; discovery then looks for their new address.
	unreachable  atomic.Int32
	pairFailures atomic.Int32
	// watchers are peers holding exchanges open with this instance; failing are peers this one can't reach.
	watchers inbound
	failing  sync.Map
	wake     notifier
	renamed  notifier
}

// notifier wakes everyone waiting for the next change; each change replaces the channel.
type notifier struct {
	mu sync.Mutex
	ch chan struct{}
}

func (n *notifier) wait() <-chan struct{} {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.ch == nil {
		n.ch = make(chan struct{})
	}
	return n.ch
}

func (n *notifier) broadcast() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.ch != nil {
		close(n.ch)
		n.ch = nil
	}
}

// SetResolver lets the runner fetch library titles this instance doesn't have yet.
func (e *Engine) SetResolver(r Resolver) { e.resolver = r }

// SetSyncPort records the port the peer listener took, for status and the mDNS advert.
func (e *Engine) SetSyncPort(port int) { e.syncPort.Store(int32(port)) }

// SyncPort is the port peers reach this instance on, or 0 when it only dials out.
func (e *Engine) SyncPort() int { return int(e.syncPort.Load()) }

// SetDiscovery turns on finding instances on the local network; call before Run.
func (e *Engine) SetDiscovery(find func(ctx context.Context, self string) ([]Nearby, error)) {
	e.discover = find
}

type querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Open loads or creates this instance's node ID, op counter and clock.
func Open(ctx context.Context, db *sql.DB) (*Engine, error) {
	e := &Engine{db: db, resolve: resolveState{failed: map[string]time.Time{}}}
	node, err := e.meta(ctx, db, "node")
	if err != nil {
		return nil, err
	}
	if node == "" {
		if node, err = initNode(ctx, db); err != nil {
			return nil, err
		}
	}
	e.node = node
	if e.name, err = e.meta(ctx, db, "name"); err != nil {
		return nil, err
	}
	last := Clock{}
	if s, err := e.meta(ctx, db, "clock"); err != nil {
		return nil, err
	} else if s != "" {
		if last, err = ParseClock(s); err != nil {
			return nil, err
		}
	}
	e.hlc = NewHLC(node, last)
	return e, nil
}

func (e *Engine) Node() string { return e.node }

func (e *Engine) Name() string {
	e.nameMu.Lock()
	defer e.nameMu.Unlock()
	return e.name
}

// maxNameLength keeps a name readable in a list on a phone screen.
const maxNameLength = 40

// ErrUnnamed is returned until someone names this device, so no other device sees it under a placeholder.
var ErrUnnamed = errors.New("name this device before pairing it")

// SetName changes the name other instances show for this one; they pick it up on their next sync.
func (e *Engine) SetName(ctx context.Context, name string) error {
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		return errors.New("device name can't be empty")
	}
	if utf8.RuneCountInString(name) > maxNameLength {
		return fmt.Errorf("device name can't be longer than %d characters", maxNameLength)
	}
	if strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return errors.New("device name can't contain control characters")
	}
	if _, err := e.db.ExecContext(ctx, `INSERT INTO sync_meta (key, value) VALUES ('name', ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, name); err != nil {
		return err
	}
	e.nameMu.Lock()
	e.name = name
	e.nameMu.Unlock()
	e.renamed.broadcast()
	return nil
}

func (e *Engine) meta(ctx context.Context, q querier, key string) (string, error) {
	var v string
	err := q.QueryRowContext(ctx, `SELECT value FROM sync_meta WHERE key = ?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read sync meta %s: %w", key, err)
	}
	return v, nil
}

// write starts a transaction holding SQLite's write lock, so no other writer can slip a change in
// between projecting remote values and clearing the change records the projection caused.
func (e *Engine) write(ctx context.Context, fn func(tx *sql.Tx) error) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin sync tx: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE sync_meta SET value = value WHERE key = 'node'`); err != nil {
		return fmt.Errorf("lock sync tx: %w", err)
	}
	e.newOps = false
	if err := fn(tx); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO sync_meta (key, value) VALUES ('clock', ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, e.hlc.Last().String()); err != nil {
		return fmt.Errorf("save clock: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	if e.newOps {
		e.notify.broadcast()
	}
	return nil
}

// initNode creates the node ID and records the existing library in one transaction,
// so a crash can't leave a node whose earlier library never gets synced.
func initNode(ctx context.Context, db *sql.DB) (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	node := hex.EncodeToString(b)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	for _, kv := range [][2]string{{"node", node}, {"seq", "0"}, {"rows", "1"}} {
		if _, err := tx.ExecContext(ctx, `INSERT INTO sync_meta (key, value) VALUES (?, ?)`, kv[0], kv[1]); err != nil {
			return "", fmt.Errorf("init sync meta: %w", err)
		}
	}
	if err := enqueueExisting(ctx, tx); err != nil {
		return "", err
	}
	return node, tx.Commit()
}

// enqueueExisting records the library as it stood before sync existed, since the triggers only see later writes.
func enqueueExisting(ctx context.Context, db querier) error {
	const now = `CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER)`
	for _, q := range []string{
		`INSERT INTO sync_dirty (kind, key, field, op, recorded_at)
			SELECT 'folder', sync_id, '*', 'set', ` + now + ` FROM folders WHERE sync_id IS NOT NULL ORDER BY id`,
		`INSERT INTO sync_dirty (kind, key, field, op, recorded_at)
			SELECT 'library', '', mk.key, 'add', ` + now + ` FROM media m JOIN sync_media_keys mk ON mk.media_id = m.id
			WHERE m.added_at IS NOT NULL ORDER BY m.id`,
		`INSERT INTO sync_dirty (kind, key, field, op, recorded_at)
			SELECT 'folder_member', f.sync_id, mk.key, 'add', ` + now + ` FROM media_folders mf
			JOIN folders f ON f.id = mf.folder_id JOIN sync_media_keys mk ON mk.media_id = mf.media_id
			WHERE f.sync_id IS NOT NULL ORDER BY mf.media_id, mf.folder_id`,
		`INSERT INTO sync_dirty (kind, key, field, op, recorded_at)
			SELECT 'chapter', ck.key, f.field, 'set', ` + now + ` FROM reading_progress rp
			JOIN sync_chapter_keys ck ON ck.chapter_id = rp.chapter_id,
			(SELECT 'completed' AS field UNION ALL SELECT 'position') f ORDER BY rp.id`,
		`INSERT INTO sync_dirty (kind, key, field, op, recorded_at)
			SELECT 'chapter', ck.key, 'bookmarked', 'set', ` + now + ` FROM chapters c
			JOIN sync_chapter_keys ck ON ck.chapter_id = c.id WHERE c.bookmarked ORDER BY c.id`,
	} {
		if _, err := db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("record existing library: %w", err)
		}
	}
	return nil
}

// Capture turns local writes recorded by the triggers into ops. It returns how many ops it created.
func (e *Engine) Capture(ctx context.Context) (int, error) {
	// Checked before taking the write lock, so an idle library costs one read per tick.
	var pending bool
	if err := e.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM sync_dirty)`).Scan(&pending); err != nil {
		return 0, fmt.Errorf("check sync_dirty: %w", err)
	}
	if !pending {
		return 0, nil
	}
	n := 0
	err := e.write(ctx, func(tx *sql.Tx) error {
		var err error
		n, err = e.capture(ctx, tx)
		return err
	})
	return n, err
}

type dirtyRow struct {
	id                   int64
	kind, key, field, op string
	recordedAt           int64
}

type target struct{ kind, key, field string }

func (e *Engine) capture(ctx context.Context, tx *sql.Tx) (int, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id, kind, key, field, op, recorded_at FROM sync_dirty ORDER BY id`)
	if err != nil {
		return 0, fmt.Errorf("read sync_dirty: %w", err)
	}
	var dirty []dirtyRow
	for rows.Next() {
		var d dirtyRow
		if err := rows.Scan(&d.id, &d.kind, &d.key, &d.field, &d.op, &d.recordedAt); err != nil {
			rows.Close()
			return 0, err
		}
		dirty = append(dirty, d)
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	if len(dirty) == 0 {
		return 0, nil
	}

	n := 0
	// Registers only need their latest value, so repeated writes to one field become one op.
	seen := map[target]bool{}
	for i := len(dirty) - 1; i >= 0; i-- {
		d := dirty[i]
		if d.kind != KindChapter && d.kind != KindFolder {
			continue
		}
		t := target{d.kind, d.key, d.field}
		if seen[t] {
			dirty[i].op = "skip"
			continue
		}
		seen[t] = true
	}
	for _, d := range dirty {
		wall := d.recordedAt
		if e.wall != nil {
			wall = e.wall()
		}
		var made int
		var err error
		switch {
		case d.op == "skip":
		case d.kind == KindChapter:
			made, err = e.captureChapter(ctx, tx, d, wall)
		case d.kind == KindFolder:
			made, err = e.captureFolder(ctx, tx, d, wall)
		case d.kind == KindLibrary || d.kind == KindFolderMember:
			made, err = e.captureMember(ctx, tx, d, wall)
			if err == nil && d.kind == KindLibrary {
				var info int
				info, err = e.captureTitleInfo(ctx, tx, d.field, d.op == "add", wall)
				made += info
			}
		}
		if err != nil {
			return n, err
		}
		n += made
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM sync_dirty WHERE id <= ?`, dirty[len(dirty)-1].id); err != nil {
		return n, fmt.Errorf("clear sync_dirty: %w", err)
	}
	return n, nil
}

func (e *Engine) captureChapter(ctx context.Context, tx *sql.Tx, d dirtyRow, wall int64) (int, error) {
	state, ok, err := readChapter(ctx, tx, d.key)
	if err != nil || !ok {
		return 0, err
	}
	v, ok := state[d.field]
	if !ok {
		return 0, nil
	}
	return e.emitSet(ctx, tx, d.kind, d.key, d.field, v, wall)
}

func (e *Engine) captureFolder(ctx context.Context, tx *sql.Tx, d dirtyRow, wall int64) (int, error) {
	fields, ok, err := readFolder(ctx, tx, d.key)
	if err != nil {
		return 0, err
	}
	if !ok {
		// A folder merged into another loses its row here, but it wasn't deleted.
		if canon, err := canonicalFolder(ctx, tx, d.key); err != nil || canon != d.key {
			return 0, err
		}
		return e.emitSet(ctx, tx, KindFolder, d.key, "deleted", json.RawMessage("true"), wall)
	}
	n := 0
	for _, f := range folderFields {
		made, err := e.emitSet(ctx, tx, KindFolder, d.key, f, fields[f], wall)
		if err != nil {
			return n, err
		}
		n += made
	}
	return n, nil
}

func (e *Engine) captureMember(ctx context.Context, tx *sql.Tx, d dirtyRow, wall int64) (int, error) {
	keys := []string{d.key}
	if d.kind == KindFolderMember {
		canon, err := canonicalFolder(ctx, tx, d.key)
		if err != nil {
			return 0, err
		}
		if keys, err = folderAliases(ctx, tx, canon); err != nil {
			return 0, err
		}
	}
	tags := make([][]string, len(keys))
	found := false
	for i, k := range keys {
		var err error
		if tags[i], err = liveTags(ctx, tx, d.kind, k, d.field); err != nil {
			return 0, err
		}
		found = found || len(tags[i]) > 0
	}
	if d.op == "add" {
		if found {
			return 0, nil
		}
		return 1, e.emit(ctx, tx, Op{Kind: d.kind, Key: keys[0], Field: d.field, Type: OpAdd}, wall)
	}
	// A removal covers folders merged into this one too, or their copy would keep the title shown.
	n := 0
	for i, k := range keys {
		if len(tags[i]) == 0 {
			continue
		}
		if err := e.emit(ctx, tx, Op{Kind: d.kind, Key: k, Field: d.field, Type: OpRemove, Tags: tags[i]}, wall); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// mergeArrivedFolders merges a folder that just arrived into a same-name folder this device already
// had, so two "Reading" folders made before pairing become one. Same-name folders made on one device stay apart.
func (e *Engine) mergeArrivedFolders(ctx context.Context, tx *sql.Tx, before map[string]bool) error {
	rows, err := tx.QueryContext(ctx, `SELECT lower(trim(name)), sync_id FROM folders WHERE kind = 'custom' ORDER BY 1, 2`)
	if err != nil {
		return err
	}
	type pair struct{ a, b string }
	var pairs []pair
	local := map[string]string{} // name -> smallest folder this device already had
	var arrived [][2]string
	for rows.Next() {
		var name, id string
		if err := rows.Scan(&name, &id); err != nil {
			rows.Close()
			return err
		}
		if before[id] {
			if _, ok := local[name]; !ok {
				local[name] = id
			}
		} else {
			arrived = append(arrived, [2]string{name, id})
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, a := range arrived {
		if l, ok := local[a[0]]; ok {
			pairs = append(pairs, pair{a[1], l})
		}
	}
	wall := nowMillis()
	if e.wall != nil {
		wall = e.wall()
	}
	for _, p := range pairs {
		// The smaller ID wins, so every device that merges the same pair picks the same direction.
		winner, loser := min(p.a, p.b), max(p.a, p.b)
		if err := e.emit(ctx, tx, Op{Kind: KindFolder, Key: loser, Field: "merged_into", Type: OpSet, Value: mustJSON(winner)}, wall); err != nil {
			return err
		}
		if err := project(ctx, tx, target{KindFolder, loser, "merged_into"}); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) emitSet(ctx context.Context, tx *sql.Tx, kind, key, field string, value json.RawMessage, wall int64) (int, error) {
	var cur string
	err := tx.QueryRowContext(ctx, `SELECT value FROM sync_registers WHERE kind = ? AND key = ? AND field = ?`, kind, key, field).Scan(&cur)
	if err == nil && bytes.Equal([]byte(cur), value) {
		return 0, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("read register: %w", err)
	}
	return 1, e.emit(ctx, tx, Op{Kind: kind, Key: key, Field: field, Type: OpSet, Value: value}, wall)
}

func (e *Engine) emit(ctx context.Context, tx *sql.Tx, op Op, wall int64) error {
	var seq string
	if err := tx.QueryRowContext(ctx, `UPDATE sync_meta SET value = CAST(value AS INTEGER) + 1 WHERE key = 'seq' RETURNING value`).Scan(&seq); err != nil {
		return fmt.Errorf("next seq: %w", err)
	}
	n, err := strconv.ParseInt(seq, 10, 64)
	if err != nil {
		return err
	}
	op.Origin, op.Seq, op.Clock = e.node, n, e.hlc.Now(wall).String()
	_, err = e.applyOp(ctx, tx, op)
	return err
}

// Apply merges ops from another instance and writes the results into the library.
func (e *Engine) Apply(ctx context.Context, ops []Op) error {
	return e.write(ctx, func(tx *sql.Tx) error { return e.apply(ctx, tx, ops) })
}

func (e *Engine) apply(ctx context.Context, tx *sql.Tx, ops []Op) error {
	// Local edits not captured yet must become ops first, so they compete with these by clock.
	if _, err := e.capture(ctx, tx); err != nil {
		return err
	}
	var baseline int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(id), 0) FROM sync_dirty`).Scan(&baseline); err != nil {
		return err
	}
	changed := map[target]bool{}
	var order []target
	for _, op := range ops {
		if err := op.validate(); err != nil {
			return err
		}
		ok, err := e.applyOp(ctx, tx, op)
		if err != nil {
			return err
		}
		t := target{op.Kind, op.Key, op.Field}
		if ok && !changed[t] {
			changed[t] = true
			order = append(order, t)
		}
	}
	folderChanged := false
	for _, t := range order {
		folderChanged = folderChanged || t.kind == KindFolder
	}
	var before map[string]bool
	if folderChanged {
		ids, err := customFolders(ctx, tx)
		if err != nil {
			return err
		}
		before = ids
	}
	for _, t := range order {
		if err := project(ctx, tx, t); err != nil {
			return fmt.Errorf("project %s %q %s: %w", t.kind, t.key, t.field, err)
		}
	}
	if folderChanged {
		if err := e.mergeArrivedFolders(ctx, tx, before); err != nil {
			return fmt.Errorf("merge same-name folders: %w", err)
		}
	}
	if len(order) > 0 {
		e.remoteChanged.Store(true)
	}
	// The projection fired the triggers; those records describe remote changes, not local ones.
	if _, err := tx.ExecContext(ctx, `DELETE FROM sync_dirty WHERE id > ?`, baseline); err != nil {
		return fmt.Errorf("clear projected changes: %w", err)
	}
	return nil
}

// applyOp merges one op into the CRDT state. It reports whether the visible value changed.
func (e *Engine) applyOp(ctx context.Context, tx *sql.Tx, op Op) (bool, error) {
	var seen int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE((SELECT seq FROM sync_vv WHERE origin = ?), 0)`, op.Origin).Scan(&seen); err != nil {
		return false, err
	}
	if op.Seq <= seen {
		return false, nil
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO sync_vv (origin, seq) VALUES (?, ?)
		ON CONFLICT(origin) DO UPDATE SET seq = excluded.seq`, op.Origin, op.Seq); err != nil {
		return false, fmt.Errorf("advance version vector: %w", err)
	}
	e.newOps = true
	clock, err := ParseClock(op.Clock)
	if err != nil {
		return false, err
	}
	e.hlc.Observe(clock)

	switch op.Type {
	case OpSet:
		var curClock, curOrigin string
		var curSeq int64
		err := tx.QueryRowContext(ctx, `SELECT clock, origin, seq FROM sync_registers WHERE kind = ? AND key = ? AND field = ?`,
			op.Kind, op.Key, op.Field).Scan(&curClock, &curOrigin, &curSeq)
		switch {
		case err == nil:
			cur, err := ParseClock(curClock)
			if err != nil {
				return false, err
			}
			if !cur.Less(clock) {
				return false, nil
			}
			// The old winner can never win again, so no peer needs it.
			if _, err := tx.ExecContext(ctx, `DELETE FROM sync_ops WHERE origin = ? AND seq = ?`, curOrigin, curSeq); err != nil {
				return false, err
			}
		case !errors.Is(err, sql.ErrNoRows):
			return false, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO sync_registers (kind, key, field, value, clock, origin, seq) VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(kind, key, field) DO UPDATE SET value = excluded.value, clock = excluded.clock, origin = excluded.origin, seq = excluded.seq`,
			op.Kind, op.Key, op.Field, string(op.Value), op.Clock, op.Origin, op.Seq); err != nil {
			return false, fmt.Errorf("write register: %w", err)
		}
		return true, insertOp(ctx, tx, op)

	case OpAdd:
		var removed bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM sync_set_removed WHERE tag = ?)`, op.Tag()).Scan(&removed); err != nil {
			return false, err
		}
		if removed {
			return false, nil
		}
		was, err := present(ctx, tx, op.Kind, op.Key, op.Field)
		if err != nil {
			return false, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO sync_set_adds (kind, key, element, tag, clock) VALUES (?, ?, ?, ?, ?)`,
			op.Kind, op.Key, op.Field, op.Tag(), op.Clock); err != nil {
			return false, err
		}
		return !was, insertOp(ctx, tx, op)

	case OpRemove:
		was, err := present(ctx, tx, op.Kind, op.Key, op.Field)
		if err != nil {
			return false, err
		}
		for _, tag := range op.Tags {
			if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO sync_set_removed (tag) VALUES (?)`, tag); err != nil {
				return false, err
			}
			if _, err := tx.ExecContext(ctx, `DELETE FROM sync_set_adds WHERE tag = ?`, tag); err != nil {
				return false, err
			}
			origin, seq, _ := parseTag(tag)
			// A removed add is dead everywhere the remove reaches; the remove itself is the tombstone.
			if _, err := tx.ExecContext(ctx, `DELETE FROM sync_ops WHERE origin = ? AND seq = ? AND type = 'add'`, origin, seq); err != nil {
				return false, err
			}
		}
		now, err := present(ctx, tx, op.Kind, op.Key, op.Field)
		if err != nil {
			return false, err
		}
		return was && !now, insertOp(ctx, tx, op)
	}
	return false, fmt.Errorf("unknown op type %q", op.Type)
}

func insertOp(ctx context.Context, tx *sql.Tx, op Op) error {
	var tags any
	if len(op.Tags) > 0 {
		b, _ := json.Marshal(op.Tags)
		tags = string(b)
	}
	var value any
	if op.Value != nil {
		value = string(op.Value)
	}
	_, err := tx.ExecContext(ctx, `INSERT OR REPLACE INTO sync_ops (origin, seq, clock, kind, key, field, type, value, tags)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, op.Origin, op.Seq, op.Clock, op.Kind, op.Key, op.Field, string(op.Type), value, tags)
	if err != nil {
		return fmt.Errorf("log op: %w", err)
	}
	return nil
}

func present(ctx context.Context, q querier, kind, key, element string) (bool, error) {
	var ok bool
	err := q.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM sync_set_adds WHERE kind = ? AND key = ? AND element = ?)`, kind, key, element).Scan(&ok)
	return ok, err
}

func liveTags(ctx context.Context, q querier, kind, key, element string) ([]string, error) {
	rows, err := q.QueryContext(ctx, `SELECT tag FROM sync_set_adds WHERE kind = ? AND key = ? AND element = ? ORDER BY tag`, kind, key, element)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tags []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

// VersionVector returns the highest seq this instance has from each origin.
func (e *Engine) VersionVector(ctx context.Context) (VersionVector, error) {
	return versionVector(ctx, e.db)
}

func versionVector(ctx context.Context, q querier) (VersionVector, error) {
	rows, err := q.QueryContext(ctx, `SELECT origin, seq FROM sync_vv`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	vv := VersionVector{}
	for rows.Next() {
		var o string
		var s int64
		if err := rows.Scan(&o, &s); err != nil {
			return nil, err
		}
		vv[o] = s
	}
	return vv, rows.Err()
}

// mergeVV adopts other's entries after a complete delta from it: everything it had is now reflected here.
func mergeVV(ctx context.Context, tx *sql.Tx, other VersionVector) error {
	for origin, seq := range other {
		if _, err := tx.ExecContext(ctx, `INSERT INTO sync_vv (origin, seq) VALUES (?, ?)
			ON CONFLICT(origin) DO UPDATE SET seq = MAX(seq, excluded.seq)`, origin, seq); err != nil {
			return err
		}
	}
	return nil
}

// opsSince returns the ops a peer at vv lacks, oldest first per origin, up to limit.
func opsSince(ctx context.Context, q querier, vv VersionVector, limit int) ([]Op, bool, error) {
	mine, err := versionVector(ctx, q)
	if err != nil {
		return nil, false, err
	}
	origins := make([]string, 0, len(mine))
	for o, s := range mine {
		if s > vv[o] {
			origins = append(origins, o)
		}
	}
	sort.Strings(origins)
	var out []Op
	for _, o := range origins {
		rows, err := q.QueryContext(ctx, `SELECT origin, seq, clock, kind, key, field, type, value, tags FROM sync_ops
			WHERE origin = ? AND seq > ? ORDER BY seq LIMIT ?`, o, vv[o], limit-len(out)+1)
		if err != nil {
			return nil, false, err
		}
		for rows.Next() {
			if len(out) == limit {
				rows.Close()
				return out, true, nil
			}
			var op Op
			var typ string
			var value, tags sql.NullString
			if err := rows.Scan(&op.Origin, &op.Seq, &op.Clock, &op.Kind, &op.Key, &op.Field, &typ, &value, &tags); err != nil {
				rows.Close()
				return nil, false, err
			}
			op.Type = OpType(typ)
			if value.Valid {
				op.Value = json.RawMessage(value.String)
			}
			if tags.Valid {
				if err := json.Unmarshal([]byte(tags.String), &op.Tags); err != nil {
					rows.Close()
					return nil, false, err
				}
			}
			out = append(out, op)
		}
		if err := rows.Close(); err != nil {
			return nil, false, err
		}
	}
	return out, false, nil
}

// ProjectAll rewrites every synced value into the library. It picks up titles and chapters that
// arrived after their sync state did, for example once a source has been installed and fetched.
func (e *Engine) ProjectAll(ctx context.Context) error {
	return e.write(ctx, func(tx *sql.Tx) error {
		if _, err := e.capture(ctx, tx); err != nil {
			return err
		}
		var baseline int64
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(id), 0) FROM sync_dirty`).Scan(&baseline); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, `SELECT DISTINCT kind, key, field FROM sync_registers
			UNION SELECT DISTINCT kind, key, element FROM sync_set_adds ORDER BY 1, 2, 3`)
		if err != nil {
			return err
		}
		var targets []target
		for rows.Next() {
			var t target
			if err := rows.Scan(&t.kind, &t.key, &t.field); err != nil {
				rows.Close()
				return err
			}
			targets = append(targets, t)
		}
		if err := rows.Close(); err != nil {
			return err
		}
		for _, t := range targets {
			if err := project(ctx, tx, t); err != nil {
				return err
			}
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM sync_dirty WHERE id > ?`, baseline)
		return err
	})
}

func splitKey(key string, parts int) ([]string, bool) {
	p := strings.Split(key, KeySep)
	return p, len(p) == parts
}

func nowMillis() int64 { return time.Now().UnixMilli() }

// Named reports whether someone has named this device; pairing and announcing wait until then.
func (e *Engine) Named() bool { return e.Name() != "" }
