package mokusync

import (
	"context"
	"database/sql"
	"sort"
	"sync"
	"time"
)

// Resolver fetches a title and its chapters from its source, so synced values have rows to land on.
type Resolver interface {
	Resolve(ctx context.Context, packageName, externalID string) error
}

const (
	resolveBatch   = 20
	resolveBackoff = 30 * time.Minute
)

type resolveState struct {
	mu     sync.Mutex
	failed map[string]time.Time
	// Inserted-row count at the last pending pass. Only a new title, chapter or folder can let a
	// pending value apply, so an unchanged count skips the pass.
	rows int64
}

// ProjectPending applies synced values whose title, chapter or folder has appeared since they arrived.
func (e *Engine) ProjectPending(ctx context.Context) (int, error) {
	var rows int64
	if err := e.db.QueryRowContext(ctx, `SELECT CAST(value AS INTEGER) FROM sync_meta WHERE key = 'rows'`).Scan(&rows); err != nil {
		return 0, err
	}
	e.resolve.mu.Lock()
	unchanged := rows == e.resolve.rows
	e.resolve.mu.Unlock()
	if unchanged {
		return 0, nil
	}
	applied := 0
	err := e.write(ctx, func(tx *sql.Tx) error {
		if _, err := e.capture(ctx, tx); err != nil {
			return err
		}
		var baseline int64
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(id), 0) FROM sync_dirty`).Scan(&baseline); err != nil {
			return err
		}
		targets, err := pendingTargets(ctx, tx)
		if err != nil {
			return err
		}
		for _, t := range targets {
			if err := project(ctx, tx, t); err != nil {
				return err
			}
		}
		var left int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM sync_pending`).Scan(&left); err != nil {
			return err
		}
		applied = len(targets) - left
		_, err = tx.ExecContext(ctx, `DELETE FROM sync_dirty WHERE id > ?`, baseline)
		return err
	})
	if err == nil {
		e.resolve.mu.Lock()
		e.resolve.rows = rows
		e.resolve.mu.Unlock()
	}
	return applied, err
}

func pendingTargets(ctx context.Context, q querier) ([]target, error) {
	rows, err := q.QueryContext(ctx, `SELECT kind, key, field FROM sync_pending ORDER BY kind = 'folder' DESC, kind, key, field`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []target
	for rows.Next() {
		var t target
		if err := rows.Scan(&t.kind, &t.key, &t.field); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ResolveMissing fetches library titles that have synced values but no rows here, a batch at a time.
// Other titles wait until opened, so a fresh phone doesn't fetch a whole reading history at once.
func (e *Engine) ResolveMissing(ctx context.Context, r Resolver) (int, error) {
	targets, err := pendingTargets(ctx, e.db)
	if err != nil {
		return 0, err
	}
	wanted := map[string]bool{}
	for _, t := range targets {
		var media string
		switch t.kind {
		case KindLibrary, KindFolderMember:
			media = t.field
		case KindChapter:
			p, ok := splitKey(t.key, 3)
			if !ok {
				continue
			}
			media = p[0] + KeySep + p[1]
		default:
			continue
		}
		wanted[media] = true
	}
	keys := make([]string, 0, len(wanted))
	for k := range wanted {
		in, err := present(ctx, e.db, KindLibrary, "", k)
		if err != nil {
			return 0, err
		}
		if in {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	installed, _, err := localExtensions(ctx, e.db)
	if err != nil {
		return 0, err
	}
	resolved := 0
	now := time.Now()
	for _, k := range keys {
		if resolved == resolveBatch {
			break
		}
		e.resolve.mu.Lock()
		last, failed := e.resolve.failed[k]
		e.resolve.mu.Unlock()
		if failed && now.Sub(last) < resolveBackoff {
			continue
		}
		p, ok := splitKey(k, 2)
		// Without its extension the fetch can only fail; ExtensionsChanged retries once it's installed.
		if !ok || !installed[p[0]] {
			continue
		}
		if err := r.Resolve(ctx, p[0], p[1]); err != nil {
			if ctx.Err() != nil {
				return resolved, ctx.Err()
			}
			e.resolve.mu.Lock()
			e.resolve.failed[k] = now
			e.resolve.mu.Unlock()
			continue
		}
		resolved++
	}
	if resolved > 0 {
		if _, err := e.ProjectPending(ctx); err != nil {
			return resolved, err
		}
	}
	return resolved, nil
}

// PendingTitles counts library titles waiting for their source to be fetched here.
func (e *Engine) PendingTitles(ctx context.Context) (int, error) {
	var n int
	err := e.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT p.field) FROM sync_pending p JOIN sync_set_adds a
		ON a.kind = 'library' AND a.element = p.field WHERE p.kind = 'library'`).Scan(&n)
	return n, err
}
