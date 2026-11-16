package mokusync

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"sync"
	"time"
)

// staleAfter drops a device that hasn't been heard from out of the deletion-record check, so a lost phone can't
// keep every deletion forever. If it does come back, titles removed while it was away can reappear.
var staleAfter = 60 * 24 * time.Hour

// mergeSeen records what node just said it holds, and what it relayed about other devices.
func (e *Engine) mergeSeen(ctx context.Context, tx *sql.Tx, node string, vv VersionVector, relayed map[string]Seen) error {
	if node != "" && node != e.node {
		if err := upsertSeen(ctx, tx, node, Seen{VV: vv, At: nowMillis()}); err != nil {
			return err
		}
	}
	for n, s := range relayed {
		if n == e.node || n == node || n == "" {
			continue
		}
		if err := upsertSeen(ctx, tx, n, s); err != nil {
			return err
		}
	}
	return nil
}

func upsertSeen(ctx context.Context, tx *sql.Tx, node string, s Seen) error {
	var raw string
	var at int64
	cur := VersionVector{}
	err := tx.QueryRowContext(ctx, `SELECT vv, reported_at FROM sync_seen WHERE node = ?`, node).Scan(&raw, &at)
	switch {
	case err == nil:
		if err := json.Unmarshal([]byte(raw), &cur); err != nil {
			return err
		}
	case !errors.Is(err, sql.ErrNoRows):
		return err
	}
	grew := false
	for o, n := range s.VV {
		if n > cur[o] {
			cur[o], grew = n, true
		}
	}
	// Report times only matter at the scale of staleAfter, so a quiet exchange writes nothing.
	if !grew && s.At-at < time.Minute.Milliseconds() {
		return nil
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO sync_seen (node, vv, reported_at) VALUES (?, ?, ?)
		ON CONFLICT(node) DO UPDATE SET vv = excluded.vv, reported_at = excluded.reported_at`, node, string(mustJSON(cur)), max(at, s.At))
	return err
}

// seenMatrix is what this device knows every device holds, itself included.
func (e *Engine) seenMatrix(ctx context.Context, q querier) (map[string]Seen, error) {
	mine, err := versionVector(ctx, q)
	if err != nil {
		return nil, err
	}
	out := map[string]Seen{e.node: {VV: mine, At: nowMillis()}}
	rows, err := q.QueryContext(ctx, `SELECT node, vv, reported_at FROM sync_seen`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var node, raw string
		var s Seen
		if err := rows.Scan(&node, &raw, &s.At); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(raw), &s.VV); err != nil {
			return nil, err
		}
		if node != e.node {
			out[node] = s
		}
	}
	return out, rows.Err()
}

// Compact drops deletion records every device has already applied. Each one only exists to cancel an add that
// some device might still send; once all have the deletion, no device holds that add any more.
func (e *Engine) Compact(ctx context.Context) (int, error) {
	dropped := 0
	err := e.write(ctx, func(tx *sql.Tx) error {
		seen, err := e.seenMatrix(ctx, tx)
		if err != nil {
			return err
		}
		mine := seen[e.node].VV
		cutoff := nowMillis() - staleAfter.Milliseconds()
		stable := VersionVector{}
		for o, n := range mine {
			stable[o] = n
		}
		others := 0
		for node, s := range seen {
			if node == e.node || s.At < cutoff {
				continue
			}
			others++
			for o := range stable {
				stable[o] = min(stable[o], s.VV[o])
			}
		}
		rows, err := tx.QueryContext(ctx, `SELECT origin, seq, tags FROM sync_ops WHERE type = 'remove'`)
		if err != nil {
			return err
		}
		type removal struct {
			origin string
			seq    int64
			tags   []string
		}
		var gone []removal
		for rows.Next() {
			var r removal
			var tags sql.NullString
			if err := rows.Scan(&r.origin, &r.seq, &tags); err != nil {
				rows.Close()
				return err
			}
			limit, ok := stable[r.origin]
			if others == 0 {
				limit, ok = math.MaxInt64, true
			}
			if !ok || r.seq > limit {
				continue
			}
			if tags.Valid {
				if err := json.Unmarshal([]byte(tags.String), &r.tags); err != nil {
					rows.Close()
					return err
				}
			}
			gone = append(gone, r)
		}
		if err := rows.Close(); err != nil {
			return err
		}
		for _, r := range gone {
			if _, err := tx.ExecContext(ctx, `DELETE FROM sync_ops WHERE origin = ? AND seq = ?`, r.origin, r.seq); err != nil {
				return err
			}
			for _, t := range r.tags {
				if _, err := tx.ExecContext(ctx, `DELETE FROM sync_set_removed WHERE tag = ?`, t); err != nil {
					return err
				}
			}
		}
		dropped = len(gone)
		return nil
	})
	return dropped, err
}

// inboundGrace covers the gap between one held exchange ending and the watcher's next one starting.
const inboundGrace = 10 * time.Second

// inbound tracks which peers are watching this instance with held exchanges.
type inbound struct {
	mu sync.Mutex
	m  map[string]*watchMark
}

type watchMark struct {
	holding int
	last    time.Time
}

func (in *inbound) mark(node string, delta int) {
	in.mu.Lock()
	defer in.mu.Unlock()
	if in.m == nil {
		in.m = map[string]*watchMark{}
	}
	w := in.m[node]
	if w == nil {
		w = &watchMark{}
		in.m[node] = w
	}
	w.holding += delta
	w.last = time.Now()
}

func (in *inbound) begin(node string) { in.mark(node, 1) }
func (in *inbound) end(node string)   { in.mark(node, -1) }

func (in *inbound) watching(node string) bool {
	in.mu.Lock()
	defer in.mu.Unlock()
	w := in.m[node]
	return w != nil && (w.holding > 0 || time.Since(w.last) < inboundGrace)
}
