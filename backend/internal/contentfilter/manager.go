package contentfilter

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"tsunagu/backend/internal/db/sqlcgen"
)

type Manager struct {
	q *sqlcgen.Queries

	mu    sync.RWMutex
	m     *matcher
	rules []Rule

	level       atomic.Int32
	recomputing atomic.Bool
}

func New(q *sqlcgen.Queries) (*Manager, error) {
	mgr := &Manager{q: q}
	if err := mgr.reload(context.Background()); err != nil {
		return nil, err
	}
	return mgr, nil
}

func (mgr *Manager) SetLevel(l Level) { mgr.level.Store(int32(l)) }
func (mgr *Manager) Level() Level     { return Level(mgr.level.Load()) }

func (mgr *Manager) reload(ctx context.Context) error {
	rows, err := mgr.q.ListContentFilterRules(ctx)
	if err != nil {
		return err
	}
	rules := make([]Rule, len(rows))
	for i, r := range rows {
		rules[i] = Rule{
			ID: r.ID, Category: r.Category, Field: r.Field, Keyword: r.Keyword,
			MinWeight: int(r.MinWeight), BlockLevel: int(r.BlockLevel), IsDefault: r.IsDefault != 0,
		}
	}
	mm := buildMatcher(rules)
	mgr.mu.Lock()
	mgr.m, mgr.rules = mm, rules
	mgr.mu.Unlock()
	return nil
}

func (mgr *Manager) snapshot() (*matcher, []Rule) {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	return mgr.m, mgr.rules
}

func (mgr *Manager) Rules() []Rule {
	_, r := mgr.snapshot()
	out := make([]Rule, len(r))
	copy(out, r)
	return out
}

func validField(f string) bool {
	switch f {
	case "genre", "tag", "title", "description":
		return true
	}
	return false
}

func (mgr *Manager) AddRule(ctx context.Context, category, field, keyword string, minWeight, blockLevel int) (Rule, error) {
	if category == "" || keyword == "" {
		return Rule{}, fmt.Errorf("category and keyword are required")
	}
	if !validField(field) {
		return Rule{}, fmt.Errorf("field must be genre/tag/title/description, got %q", field)
	}
	if blockLevel != 1 && blockLevel != 2 {
		return Rule{}, fmt.Errorf("blockLevel must be 1 (moderate) or 2 (strict)")
	}
	if minWeight < 0 {
		minWeight = 0
	}
	row, err := mgr.q.CreateContentFilterRule(ctx, sqlcgen.CreateContentFilterRuleParams{
		Category:   category,
		Field:      field,
		Keyword:    norm(keyword),
		MinWeight:  int64(minWeight),
		BlockLevel: int64(blockLevel),
	})
	if err != nil {
		return Rule{}, err
	}
	if err := mgr.reload(ctx); err != nil {
		return Rule{}, err
	}
	go mgr.RecomputeAll(context.Background())
	return Rule{
		ID: row.ID, Category: row.Category, Field: row.Field, Keyword: row.Keyword,
		MinWeight: int(row.MinWeight), BlockLevel: int(row.BlockLevel), IsDefault: row.IsDefault != 0,
	}, nil
}

func (mgr *Manager) RemoveRule(ctx context.Context, id int64) error {
	if err := mgr.q.DeleteContentFilterRule(ctx, id); err != nil {
		return err
	}
	if err := mgr.reload(ctx); err != nil {
		return err
	}
	go mgr.RecomputeAll(context.Background())
	return nil
}

func (mgr *Manager) ResetRules(ctx context.Context) error {
	if err := mgr.q.DeleteUserContentFilterRules(ctx); err != nil {
		return err
	}
	if err := mgr.reload(ctx); err != nil {
		return err
	}
	go mgr.RecomputeAll(context.Background())
	return nil
}

func (mgr *Manager) RecomputeMedia(ctx context.Context, mediaID int64) error {
	m, _ := mgr.snapshot()
	if !m.hasRules() {
		return mgr.q.SetMediaContentBlockRank(ctx, sqlcgen.SetMediaContentBlockRankParams{ID: mediaID})
	}
	in, err := mgr.q.GetContentFilterInputs(ctx, mediaID)
	if err != nil {
		return err
	}
	genreRows, _ := mgr.q.ListGenresForMedia(ctx, mediaID)
	genres := make([]string, len(genreRows))
	for i, g := range genreRows {
		genres[i] = g.Name
	}
	tagRows, _ := mgr.q.ListTagsWithWeightForMedia(ctx, mediaID)
	tags := make([]string, len(tagRows))
	weights := make([]int, len(tagRows))
	for i, t := range tagRows {
		tags[i] = t.Name
		weights[i] = int(t.Weight)
	}
	rank := m.blockRank(in.Title, in.Description.String, genres, tags, weights)
	var nr sql.NullInt64
	if rank > 0 {
		nr = sql.NullInt64{Int64: int64(rank), Valid: true}
	}
	return mgr.q.SetMediaContentBlockRank(ctx, sqlcgen.SetMediaContentBlockRankParams{ContentBlockRank: nr, ID: mediaID})
}

func (mgr *Manager) RecomputeAll(ctx context.Context) error {
	if !mgr.recomputing.CompareAndSwap(false, true) {
		return nil
	}
	defer mgr.recomputing.Store(false)

	ids, err := mgr.q.ListRecomputableMediaIDs(ctx)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := mgr.RecomputeMedia(ctx, id); err != nil {
			log.Printf("content filter: recompute media %d: %v", id, err)
		}
	}
	log.Printf("content filter: recomputed %d media", len(ids))
	return nil
}

func (mgr *Manager) TitleAllowed(title string) bool {
	return mgr.BrowseAllowed(title, nil)
}

func (mgr *Manager) BrowseAllowed(title string, genres []string) bool {
	lvl := mgr.Level()
	if lvl == Unrestricted {
		return true
	}
	m, _ := mgr.snapshot()
	rank := m.titleBlockRank(title)
	if len(genres) > 0 {
		if r := m.blockRank("", "", genres, nil, nil); r != 0 && (rank == 0 || r < rank) {
			rank = r
		}
	}
	return !Hidden(rank, lvl)
}
