package metadata

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"tsunagu/backend/internal/db/sqlcgen"
)

type Manager struct {
	db        *sql.DB
	q         *sqlcgen.Queries
	providers map[string]Provider
}

func NewManager(db *sql.DB, q *sqlcgen.Queries) *Manager {
	return &Manager{
		db: db,
		q:  q,
		providers: map[string]Provider{
			"anilist": NewAniList(),
		},
	}
}

const DefaultProvider = "anilist"

func (m *Manager) AutoEnrich(ctx context.Context, mediaID int64) error {
	media, err := m.q.GetMedia(ctx, mediaID)
	if err != nil {
		return nil
	}
	links, err := m.q.ListMetadataLinksByMedia(ctx, mediaID)
	if err == nil && len(links) > 0 {
		return nil
	}
	thin, err := m.detailsAreThin(ctx, media)
	if err != nil || !thin {
		return nil
	}
	return m.tryMatch(ctx, media)
}

func (m *Manager) tryMatch(ctx context.Context, media sqlcgen.Medium) error {
	cands, err := m.providers[DefaultProvider].Search(ctx, media.Title, contentTypeOf(media))
	if err != nil {
		log.Printf("metadata: auto search %q: %v", media.Title, err)
		return nil
	}
	if len(cands) == 0 {
		return nil
	}
	best, score := bestMatch(media.Title, 0, cands)
	if score < AutoApplyScore {
		log.Printf("metadata: no confident match for %q (best %.2f)", media.Title, score)
		return nil
	}
	if _, err := m.applyCandidate(ctx, media.ID, DefaultProvider, best, score, false); err != nil {
		log.Printf("metadata: apply auto match for media %d: %v", media.ID, err)
	}
	return nil
}

func (m *Manager) EnrichLibrary(ctx context.Context) {
	ids, err := m.q.ListMediaIDsWithoutMetadataLink(ctx)
	if err != nil {
		log.Printf("metadata backfill: %v", err)
		return
	}
	if len(ids) == 0 {
		return
	}
	log.Printf("metadata backfill: %d unmatched media, working through them (~%dm)",
		len(ids), len(ids)*800/60000+1)
	matched := 0
	for i, id := range ids {
		if ctx.Err() != nil {
			return
		}
		media, err := m.q.GetMedia(ctx, id)
		if err != nil {
			continue
		}
		_ = m.tryMatch(ctx, media)
		if after, _ := m.q.ListMetadataLinksByMedia(ctx, id); len(after) > 0 {
			matched++
		}
		if (i+1)%50 == 0 {
			log.Printf("metadata backfill: %d/%d checked, %d matched", i+1, len(ids), matched)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(800 * time.Millisecond):
		}
	}
	log.Printf("metadata backfill: done -- matched %d/%d", matched, len(ids))
}

func (m *Manager) Search(ctx context.Context, provider, query string, ct ContentType) ([]Candidate, error) {
	p, ok := m.providers[provider]
	if !ok {
		return nil, fmt.Errorf("unknown metadata provider %q", provider)
	}
	return p.Search(ctx, query, ct)
}

func (m *Manager) Apply(ctx context.Context, mediaID int64, provider, providerID string) (sqlcgen.Medium, error) {
	p, ok := m.providers[provider]
	if !ok {
		return sqlcgen.Medium{}, fmt.Errorf("unknown metadata provider %q", provider)
	}
	cand, err := p.Fetch(ctx, providerID)
	if err != nil {
		return sqlcgen.Medium{}, err
	}
	return m.applyCandidate(ctx, mediaID, provider, *cand, 1, true)
}

func (m *Manager) Refresh(ctx context.Context, mediaID int64) (sqlcgen.Medium, error) {
	media, err := m.q.GetMedia(ctx, mediaID)
	if err != nil {
		return sqlcgen.Medium{}, err
	}
	link, err := m.q.GetMetadataLink(ctx, sqlcgen.GetMetadataLinkParams{MediaID: mediaID, Provider: DefaultProvider})
	switch {
	case err == nil && link.Locked != 0:
		return m.Apply(ctx, mediaID, link.Provider, link.ProviderID)
	case err != nil && !errors.Is(err, sql.ErrNoRows):
		return sqlcgen.Medium{}, err
	}
	cands, err := m.providers[DefaultProvider].Search(ctx, media.Title, contentTypeOf(media))
	if err != nil {
		return sqlcgen.Medium{}, err
	}
	if len(cands) == 0 {
		return media, nil
	}
	best, score := bestMatch(media.Title, 0, cands)
	if score < AutoApplyScore {
		return media, nil
	}
	return m.applyCandidate(ctx, mediaID, DefaultProvider, best, score, false)
}

func (m *Manager) Unlink(ctx context.Context, mediaID int64) error {
	return m.q.DeleteMetadataLink(ctx, sqlcgen.DeleteMetadataLinkParams{MediaID: mediaID, Provider: DefaultProvider})
}

func (m *Manager) Link(ctx context.Context, mediaID int64) (*sqlcgen.MetadataLink, error) {
	link, err := m.q.GetMetadataLink(ctx, sqlcgen.GetMetadataLinkParams{MediaID: mediaID, Provider: DefaultProvider})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &link, nil
}

var malCache sync.Map

func (m *Manager) MalID(ctx context.Context, mediaID int64) (int, error) {
	if v, ok := malCache.Load(mediaID); ok {
		return v.(int), nil
	}
	al, _ := m.providers[DefaultProvider].(*AniList)

	var anilistID string
	if links, err := m.q.ListTrackerLinksByMedia(ctx, mediaID); err == nil {
		accts, _ := m.q.ListTrackerAccounts(ctx)
		typeByAcct := map[int64]string{}
		for _, a := range accts {
			typeByAcct[a.ID] = a.TrackerType
		}
		for _, l := range links {
			switch typeByAcct[l.TrackerAccountID] {
			case "mal":
				id := l.ExternalTrackerID
				if i := strings.IndexByte(id, ':'); i >= 0 {
					id = id[i+1:]
				}
				if n, err := strconv.Atoi(id); err == nil && n > 0 {
					malCache.Store(mediaID, n)
					return n, nil
				}
			case "anilist":
				anilistID = l.ExternalTrackerID
			}
		}
	}

	if link, err := m.Link(ctx, mediaID); err == nil && link != nil {
		anilistID = link.ProviderID
	} else if err != nil {
		return 0, err
	}

	if anilistID == "" || al == nil {
		return 0, nil
	}
	mal, err := al.MalID(ctx, anilistID)
	if err != nil {
		return 0, err
	}
	malCache.Store(mediaID, mal)
	return mal, nil
}

func (m *Manager) applyCandidate(ctx context.Context, mediaID int64, provider string, c Candidate, confidence float64, locked bool) (sqlcgen.Medium, error) {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return sqlcgen.Medium{}, err
	}
	defer tx.Rollback()
	qtx := m.q.WithTx(tx)

	updated, err := qtx.GapFillMediaMetadata(ctx, sqlcgen.GapFillMediaMetadataParams{
		Description: c.Description,
		Status:      c.Status,
		Author:      strings.Join(c.Authors, ", "),
		CoverPath:   c.CoverURL,
		ID:          mediaID,
	})
	if err != nil {
		return sqlcgen.Medium{}, fmt.Errorf("gap-fill media %d: %w", mediaID, err)
	}

	if len(c.Genres) > 0 {
		if existing, err := qtx.ListGenresForMedia(ctx, mediaID); err == nil && len(existing) == 0 {
			for _, name := range c.Genres {
				g, err := qtx.CreateGenre(ctx, name)
				if err != nil {
					return sqlcgen.Medium{}, err
				}
				if err := qtx.AddGenreToMedia(ctx, sqlcgen.AddGenreToMediaParams{MediaID: mediaID, GenreID: g.ID}); err != nil {
					return sqlcgen.Medium{}, err
				}
			}
		}
	}

	if len(c.Tags) > 0 {
		if existing, err := qtx.ListTagsForMedia(ctx, mediaID); err == nil && len(existing) == 0 {
			for _, name := range c.Tags {
				t, err := qtx.CreateTag(ctx, name)
				if err != nil {
					return sqlcgen.Medium{}, err
				}
				if err := qtx.AddTagToMedia(ctx, sqlcgen.AddTagToMediaParams{MediaID: mediaID, TagID: t.ID}); err != nil {
					return sqlcgen.Medium{}, err
				}
			}
		}
	}

	if _, err := qtx.UpsertMetadataLink(ctx, sqlcgen.UpsertMetadataLinkParams{
		MediaID:     mediaID,
		Provider:    provider,
		ProviderID:  c.ProviderID,
		ProviderUrl: c.URL,
		Confidence:  confidence,
		Locked:      b2i(locked),
	}); err != nil {
		return sqlcgen.Medium{}, err
	}

	if err := tx.Commit(); err != nil {
		return sqlcgen.Medium{}, err
	}
	return updated, nil
}

func (m *Manager) detailsAreThin(ctx context.Context, media sqlcgen.Medium) (bool, error) {
	missing := 0
	if !media.Description.Valid || media.Description.String == "" {
		missing++
	}
	if (!media.CoverPath.Valid || media.CoverPath.String == "") &&
		(!media.CoverLocalPath.Valid || media.CoverLocalPath.String == "") {
		missing++
	}
	if !media.Status.Valid || media.Status.String == "" {
		missing++
	}
	if !media.Author.Valid || media.Author.String == "" {
		missing++
	}
	if missing >= 2 {
		return true, nil
	}
	genres, err := m.q.ListGenresForMedia(ctx, media.ID)
	if err != nil {
		return false, err
	}
	return len(genres) == 0, nil
}

func contentTypeOf(media sqlcgen.Medium) ContentType {
	switch media.ContentType {
	case "anime":
		return Anime
	case "novel":
		return Novel
	default:
		return Manga
	}
}

func b2i(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
