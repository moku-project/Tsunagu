package tracker

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/url"
	"sync"
	"time"

	"tsunagu/backend/internal/db/sqlcgen"
)

type Manager struct {
	q        *sqlcgen.Queries
	services map[string]Service
	order    []string

	acctMu   sync.RWMutex
	acctKeys map[int64]string
}

type MALConfig struct {
	ClientID     string
	ClientSecret string
	CallbackURL  string
}

func NewManager(q *sqlcgen.Queries, anilistClientID string, mal MALConfig) *Manager {
	m := &Manager{q: q, services: map[string]Service{}}
	for _, s := range []Service{NewAniList(anilistClientID), NewMAL(mal)} {
		m.services[s.Key()] = s
		m.order = append(m.order, s.Key())
	}
	return m
}

type Info struct {
	Key          string
	Name         string
	Configured   bool
	IsLoggedIn   bool
	AuthURL      string
	Username     string
	ScoreOptions []string
	IconURL      string
}

func (m *Manager) List(ctx context.Context) ([]Info, error) {
	accounts, err := m.q.ListTrackerAccounts(ctx)
	if err != nil {
		return nil, err
	}
	byType := map[string]sqlcgen.TrackerAccount{}
	for _, a := range accounts {
		byType[a.TrackerType] = a
	}
	out := make([]Info, 0, len(m.order))
	for _, key := range m.order {
		svc := m.services[key]
		info := Info{
			Key:        key,
			Name:       svc.Name(),
			Configured: svc.Configured(),
			AuthURL:    svc.AuthURL(),
			IconURL:    svc.IconURL(),
		}
		if acct, ok := byType[key]; ok {
			info.IsLoggedIn = true
			info.Username = acct.Username
			info.ScoreOptions = svc.ScoreOptions(authFromAccount(acct))
		}
		out = append(out, info)
	}
	return out, nil
}

func (m *Manager) Login(ctx context.Context, key, pasted string) (Info, error) {
	svc, ok := m.services[key]
	if !ok {
		return Info{}, fmt.Errorf("unknown tracker %q", key)
	}
	auth, err := svc.Exchange(ctx, pasted)
	if err != nil {
		return Info{}, err
	}
	return m.finishLogin(ctx, key, auth)
}

func (m *Manager) OAuthCallback(ctx context.Context, key string, q url.Values) (Info, error) {
	svc, ok := m.services[key]
	if !ok {
		return Info{}, fmt.Errorf("unknown tracker %q", key)
	}
	cb, ok := svc.(CallbackAuther)
	if !ok {
		return Info{}, fmt.Errorf("%s does not use OAuth callback", svc.Name())
	}
	auth, err := cb.CompleteAuth(ctx, q)
	if err != nil {
		return Info{}, err
	}
	return m.finishLogin(ctx, key, auth)
}

func (m *Manager) finishLogin(ctx context.Context, key string, auth Auth) (Info, error) {
	if _, err := m.q.UpsertTrackerAccount(ctx, sqlcgen.UpsertTrackerAccountParams{
		TrackerType:  key,
		AccessToken:  auth.AccessToken,
		RefreshToken: ns(auth.RefreshToken),
		ExpiresAt:    nt(auth.ExpiresAt),
		Username:     auth.Username,
		ScoreFormat:  auth.ScoreFormat,
	}); err != nil {
		return Info{}, err
	}
	m.invalidateAccountCache()
	list, err := m.List(ctx)
	if err != nil {
		return Info{}, err
	}
	for _, i := range list {
		if i.Key == key {
			return i, nil
		}
	}
	return Info{}, fmt.Errorf("tracker %q vanished after login", key)
}

func (m *Manager) Logout(ctx context.Context, key string) error {
	if _, ok := m.services[key]; !ok {
		return fmt.Errorf("unknown tracker %q", key)
	}
	m.invalidateAccountCache()
	return m.q.DeleteTrackerAccountByType(ctx, key)
}

func (m *Manager) Search(ctx context.Context, key, query, contentType string) ([]SearchResult, error) {
	acct, svc, auth, err := m.load(ctx, key)
	if err != nil {
		return nil, err
	}
	res, err := svc.Search(ctx, auth, query, contentType)
	return res, m.checkReauth(ctx, acct, err)
}

func (m *Manager) Bind(ctx context.Context, key string, mediaID int64, remoteID string) (sqlcgen.TrackerLink, error) {
	acct, svc, auth, err := m.load(ctx, key)
	if err != nil {
		return sqlcgen.TrackerLink{}, err
	}
	t, err := svc.Bind(ctx, auth, remoteID)
	if err != nil {
		return sqlcgen.TrackerLink{}, m.checkReauth(ctx, acct, err)
	}
	t = m.pushLocalProgress(ctx, svc, auth, acct, mediaID, t)
	return m.persist(ctx, mediaID, acct.ID, t)
}

func (m *Manager) Update(ctx context.Context, linkID int64, status *int, score *float64, lastChapterRead *float64) (sqlcgen.TrackerLink, error) {
	link, err := m.q.GetTrackerLink(ctx, linkID)
	if err != nil {
		return sqlcgen.TrackerLink{}, err
	}
	acct, svc, auth, err := m.loadByAccountID(ctx, link.TrackerAccountID)
	if err != nil {
		return sqlcgen.TrackerLink{}, err
	}
	t := trackFromLink(link)
	if status != nil {
		t.Status = Status(*status)
	}
	if score != nil {
		t.Score = *score
	}
	if lastChapterRead != nil {
		t.LastChapterRead = *lastChapterRead
	}
	log.Printf("tracker update: link=%d %s/%s recv{status=%s score=%s prog=%s} -> push{status=%d score=%.1f prog=%.1f}",
		linkID, acct.TrackerType, link.ExternalTrackerID,
		ptrStr(status), ptrFloatStr(score), ptrFloatStr(lastChapterRead),
		int(t.Status), t.Score, t.LastChapterRead)
	pushed, err := svc.Push(ctx, auth, t)
	if err != nil {
		log.Printf("tracker update: link=%d push failed: %v", linkID, err)
		return sqlcgen.TrackerLink{}, m.checkReauth(ctx, acct, err)
	}
	log.Printf("tracker update: link=%d push ok -> {status=%d score=%.1f prog=%.1f libId=%s}",
		linkID, int(pushed.Status), pushed.Score, pushed.LastChapterRead, pushed.LibraryID)
	return m.persist(ctx, link.MediaID, acct.ID, pushed)
}

func ptrStr(p *int) string {
	if p == nil {
		return "nil"
	}
	return fmt.Sprintf("%d", *p)
}

func ptrFloatStr(p *float64) string {
	if p == nil {
		return "nil"
	}
	return fmt.Sprintf("%.1f", *p)
}

func (m *Manager) Unbind(ctx context.Context, linkID int64) error {
	return m.q.DeleteTrackerLink(ctx, linkID)
}

func (m *Manager) GetLink(ctx context.Context, linkID int64) (sqlcgen.TrackerLink, error) {
	return m.q.GetTrackerLink(ctx, linkID)
}

func (m *Manager) LinksByMedia(ctx context.Context, mediaID int64) ([]sqlcgen.TrackerLink, error) {
	return m.q.ListTrackerLinksByMedia(ctx, mediaID)
}

func (m *Manager) KeyForAccountID(ctx context.Context, accountID int64) string {
	m.acctMu.RLock()
	if m.acctKeys != nil {
		k, ok := m.acctKeys[accountID]
		m.acctMu.RUnlock()
		if ok {
			return k
		}
		return ""
	}
	m.acctMu.RUnlock()

	accounts, err := m.q.ListTrackerAccounts(ctx)
	if err != nil {
		return ""
	}
	fresh := make(map[int64]string, len(accounts))
	for _, a := range accounts {
		fresh[a.ID] = a.TrackerType
	}
	m.acctMu.Lock()
	m.acctKeys = fresh
	m.acctMu.Unlock()
	return fresh[accountID]
}

func (m *Manager) invalidateAccountCache() {
	m.acctMu.Lock()
	m.acctKeys = nil
	m.acctMu.Unlock()
}

func (m *Manager) SyncMediaProgress(ctx context.Context, mediaID int64) {
	links, err := m.q.ListTrackerLinksByMedia(ctx, mediaID)
	if err != nil || len(links) == 0 {
		return
	}
	maxRead, _ := m.maxRead(ctx, mediaID)

	for _, link := range links {
		if !link.SyncProgress {
			continue
		}
		acct, svc, auth, err := m.loadByAccountID(ctx, link.TrackerAccountID)
		if err != nil {
			log.Printf("tracker sync: media %d account %d: %v", mediaID, link.TrackerAccountID, err)
			continue
		}
		local := trackFromLink(link)
		t := local
		if remote, err := svc.Refresh(ctx, auth, t); err != nil {
			log.Printf("tracker sync: refresh media %d on %s: %v", mediaID, acct.TrackerType, err)
			_ = m.checkReauth(ctx, acct, err)
			continue
		} else {
			t = remote
		}

		if t.LastChapterRead > maxRead {
			if err := m.q.MarkChaptersReadUpToNumber(ctx, sqlcgen.MarkChaptersReadUpToNumberParams{
				MediaID: mediaID,
				UpTo:    sql.NullFloat64{Float64: t.LastChapterRead, Valid: true},
			}); err != nil {
				log.Printf("tracker sync: pull media %d: %v", mediaID, err)
			} else {
				log.Printf("tracker sync: media %d <- %s pulled progress %.1f (local was %.1f)",
					mediaID, acct.TrackerType, t.LastChapterRead, maxRead)
			}
			maxRead = t.LastChapterRead
		}

		if maxRead > t.LastChapterRead {
			t.LastChapterRead = maxRead
			if t.Status == StatusPlanToRead || t.Status == 0 {
				t.Status = StatusReading
			}
			pushed, err := svc.Push(ctx, auth, t)
			if err != nil {
				log.Printf("tracker sync: push media %d on %s: %v", mediaID, acct.TrackerType, err)
				_ = m.checkReauth(ctx, acct, err)
				continue
			}
			log.Printf("tracker sync: media %d -> %s pushed progress %.1f (was %.1f)",
				mediaID, acct.TrackerType, maxRead, local.LastChapterRead)
			t = pushed
		}

		if _, err := m.persist(ctx, mediaID, acct.ID, t); err != nil {
			log.Printf("tracker sync: persist media %d: %v", mediaID, err)
		}
	}
}

func (m *Manager) PollAll(ctx context.Context) {
	ids, err := m.q.ListMediaIDsWithTrackerLinks(ctx)
	if err != nil {
		log.Printf("tracker poll: %v", err)
		return
	}
	for _, id := range ids {
		m.SyncMediaProgress(ctx, id)
	}
	if len(ids) > 0 {
		log.Printf("tracker poll: reconciled %d media", len(ids))
	}
}

func (m *Manager) pushLocalProgress(ctx context.Context, svc Service, auth Auth, acct sqlcgen.TrackerAccount, mediaID int64, t Track) Track {
	maxRead, err := m.maxRead(ctx, mediaID)
	if err != nil || maxRead <= 0 || maxRead <= t.LastChapterRead {
		return t
	}
	t.LastChapterRead = maxRead
	if t.Status == StatusPlanToRead || t.Status == 0 {
		t.Status = StatusReading
	}
	if pushed, err := svc.Push(ctx, auth, t); err != nil {
		log.Printf("tracker bind: push media %d on %s: %v", mediaID, acct.TrackerType, err)
	} else {
		return pushed
	}
	return t
}

func (m *Manager) persist(ctx context.Context, mediaID, accountID int64, t Track) (sqlcgen.TrackerLink, error) {
	return m.q.UpsertTrackerLink(ctx, sqlcgen.UpsertTrackerLinkParams{
		MediaID:           mediaID,
		TrackerAccountID:  accountID,
		ExternalTrackerID: t.RemoteID,
		LibraryID:         ns(t.LibraryID),
		TrackerTitle:      t.Title,
		RemoteUrl:         t.URL,
		Status:            int64(t.Status),
		LastChapterRead:   t.LastChapterRead,
		TotalChapters:     int64(t.TotalChapters),
		Score:             t.Score,
		StartedAt:         nt(t.StartedAt),
		FinishedAt:        nt(t.FinishedAt),
		Private:           b2i(t.Private),
		SyncProgress:      true,
	})
}

func (m *Manager) load(ctx context.Context, key string) (sqlcgen.TrackerAccount, Service, Auth, error) {
	svc, ok := m.services[key]
	if !ok {
		return sqlcgen.TrackerAccount{}, nil, Auth{}, fmt.Errorf("unknown tracker %q", key)
	}
	acct, err := m.q.GetTrackerAccount(ctx, key)
	if err == sql.ErrNoRows {
		return sqlcgen.TrackerAccount{}, nil, Auth{}, fmt.Errorf("not logged in to %s", svc.Name())
	}
	if err != nil {
		return sqlcgen.TrackerAccount{}, nil, Auth{}, err
	}
	auth := authFromAccount(acct)

	if r, ok := svc.(Refresher); ok && auth.RefreshToken != "" &&
		auth.ExpiresAt != nil && time.Until(*auth.ExpiresAt) < 5*time.Minute {
		if fresh, rerr := r.RefreshAuth(ctx, auth); rerr == nil {
			auth = fresh
			if _, perr := m.q.UpsertTrackerAccount(ctx, sqlcgen.UpsertTrackerAccountParams{
				TrackerType: key, AccessToken: auth.AccessToken,
				RefreshToken: ns(auth.RefreshToken), ExpiresAt: nt(auth.ExpiresAt),
				Username: auth.Username, ScoreFormat: auth.ScoreFormat,
			}); perr != nil {
				log.Printf("tracker %s: persist refreshed token: %v", key, perr)
			}
		} else if errors.Is(rerr, ErrReauth) {
			_ = m.q.DeleteTrackerAccountByType(ctx, key)
			return sqlcgen.TrackerAccount{}, nil, Auth{}, fmt.Errorf("%s session expired, log in again", svc.Name())
		} else {
			log.Printf("tracker %s: token refresh failed: %v", key, rerr)
		}
	}
	return acct, svc, auth, nil
}

func (m *Manager) loadByAccountID(ctx context.Context, accountID int64) (sqlcgen.TrackerAccount, Service, Auth, error) {
	key := m.KeyForAccountID(ctx, accountID)
	if key == "" {
		return sqlcgen.TrackerAccount{}, nil, Auth{}, fmt.Errorf("tracker account %d not found", accountID)
	}
	return m.load(ctx, key)
}

func (m *Manager) checkReauth(ctx context.Context, acct sqlcgen.TrackerAccount, err error) error {
	if errors.Is(err, ErrReauth) {
		_ = m.q.DeleteTrackerAccountByType(ctx, acct.TrackerType)
		return fmt.Errorf("%s session expired, log in again", acct.TrackerType)
	}
	return err
}

func authFromAccount(a sqlcgen.TrackerAccount) Auth {
	var exp *time.Time
	if a.ExpiresAt.Valid {
		t := a.ExpiresAt.Time
		exp = &t
	}
	return Auth{
		AccessToken:  a.AccessToken,
		RefreshToken: a.RefreshToken.String,
		ExpiresAt:    exp,
		Username:     a.Username,
		ScoreFormat:  a.ScoreFormat,
	}
}

func trackFromLink(l sqlcgen.TrackerLink) Track {
	t := Track{
		RemoteID:        l.ExternalTrackerID,
		LibraryID:       l.LibraryID.String,
		Title:           l.TrackerTitle,
		URL:             l.RemoteUrl,
		Status:          Status(l.Status),
		LastChapterRead: l.LastChapterRead,
		TotalChapters:   int(l.TotalChapters),
		Score:           l.Score,
		Private:         l.Private != 0,
	}
	if l.StartedAt.Valid {
		s := l.StartedAt.Time
		t.StartedAt = &s
	}
	if l.FinishedAt.Valid {
		f := l.FinishedAt.Time
		t.FinishedAt = &f
	}
	return t
}

func ns(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nt(t *time.Time) sql.NullTime {
	if t == nil || t.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func b2i(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
