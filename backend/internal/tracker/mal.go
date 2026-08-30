package tracker

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	malAuthorize = "https://myanimelist.net/v1/oauth2/authorize"
	malToken     = "https://myanimelist.net/v1/oauth2/token"
	malAPI       = "https://api.myanimelist.net/v2"
)

type MAL struct {
	clientID     string
	clientSecret string
	callbackURL  string
	http         *http.Client

	mu   sync.Mutex
	pkce map[string]pkceEntry
}

type pkceEntry struct {
	verifier string
	at       time.Time
}

func NewMAL(cfg MALConfig) *MAL {
	return &MAL{
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		callbackURL:  cfg.CallbackURL,
		http:         &http.Client{Timeout: 20 * time.Second},
		pkce:         map[string]pkceEntry{},
	}
}

var errTokenRejected = errors.New("mal: token request rejected")

func (m *MAL) Key() string  { return "mal" }
func (m *MAL) Name() string { return "MyAnimeList" }

func (m *MAL) Configured() bool { return m.clientID != "" }

func (m *MAL) addClientAuth(form url.Values) {
	form.Set("client_id", m.clientID)
	if m.clientSecret != "" {
		form.Set("client_secret", m.clientSecret)
	}
}
func (m *MAL) IconURL() string {
	return "https://cdn.myanimelist.net/img/sp/icon/apple-touch-icon-256.png"
}

func randToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func (m *MAL) sweep() {
	cut := time.Now().Add(-15 * time.Minute)
	for k, e := range m.pkce {
		if e.at.Before(cut) {
			delete(m.pkce, k)
		}
	}

	for len(m.pkce) > 64 {
		var oldestK string
		var oldestT time.Time
		for k, e := range m.pkce {
			if oldestK == "" || e.at.Before(oldestT) {
				oldestK, oldestT = k, e.at
			}
		}
		delete(m.pkce, oldestK)
	}
}

func (m *MAL) AuthURL() string {
	if !m.Configured() {
		return ""
	}
	state := randToken(16)
	verifier := randToken(64)

	m.mu.Lock()
	m.sweep()
	m.pkce[state] = pkceEntry{verifier: verifier, at: time.Now()}
	m.mu.Unlock()

	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", m.clientID)
	q.Set("state", state)
	q.Set("code_challenge", verifier)
	q.Set("code_challenge_method", "plain")
	if m.callbackURL != "" {
		q.Set("redirect_uri", m.callbackURL)
	}
	return malAuthorize + "?" + q.Encode()
}

func (m *MAL) Exchange(ctx context.Context, pasted string) (Auth, error) {
	pasted = strings.TrimSpace(pasted)
	var q url.Values
	if u, err := url.Parse(pasted); err == nil && u.Query().Get("code") != "" {
		q = u.Query()
	} else {
		q, _ = url.ParseQuery(strings.TrimPrefix(pasted, "?"))
	}
	if q.Get("code") == "" {
		return Auth{}, fmt.Errorf("paste the full redirected URL from MyAnimeList (it contains ?code=…)")
	}
	return m.CompleteAuth(ctx, q)
}

func (m *MAL) CompleteAuth(ctx context.Context, q url.Values) (Auth, error) {
	if !m.Configured() {
		return Auth{}, fmt.Errorf("MyAnimeList is not configured on this server")
	}
	if e := q.Get("error"); e != "" {
		return Auth{}, fmt.Errorf("MyAnimeList denied authorization: %s", e)
	}
	code, state := q.Get("code"), q.Get("state")
	if code == "" || state == "" {
		return Auth{}, fmt.Errorf("callback missing code/state")
	}
	m.mu.Lock()
	entry, ok := m.pkce[state]
	delete(m.pkce, state)
	m.mu.Unlock()
	if !ok {
		return Auth{}, fmt.Errorf("login expired or already used -- start again")
	}

	form := url.Values{}
	m.addClientAuth(form)
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("code_verifier", entry.verifier)
	if m.callbackURL != "" {
		form.Set("redirect_uri", m.callbackURL)
	}
	return m.tokenRequest(ctx, form)
}

func (m *MAL) RefreshAuth(ctx context.Context, a Auth) (Auth, error) {
	if a.RefreshToken == "" {
		return Auth{}, ErrReauth
	}
	form := url.Values{}
	m.addClientAuth(form)
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", a.RefreshToken)
	fresh, err := m.tokenRequest(ctx, form)
	if errors.Is(err, errTokenRejected) {
		return Auth{}, ErrReauth
	}
	if err != nil {
		return Auth{}, err
	}

	if fresh.Username == "" {
		fresh.Username = a.Username
	}
	if fresh.ScoreFormat == "" {
		fresh.ScoreFormat = a.ScoreFormat
	}
	return fresh, nil
}

func (m *MAL) tokenRequest(ctx context.Context, form url.Values) (Auth, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, malToken, strings.NewReader(form.Encode()))
	if err != nil {
		return Auth{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := m.http.Do(req)
	if err != nil {
		return Auth{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Auth{}, fmt.Errorf("%w (HTTP %d): %s", errTokenRejected, resp.StatusCode, truncate(strings.TrimSpace(string(body)), 300))
	}
	var t struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &t); err != nil {
		return Auth{}, err
	}
	auth := Auth{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		ScoreFormat:  "POINT_10",
	}
	if t.ExpiresIn > 0 {
		exp := time.Now().Add(time.Duration(t.ExpiresIn) * time.Second)
		auth.ExpiresAt = &exp
	}
	if name, err := m.username(ctx, auth.AccessToken); err == nil {
		auth.Username = name
	}
	return auth, nil
}

func (m *MAL) username(ctx context.Context, token string) (string, error) {
	var out struct {
		Name string `json:"name"`
	}
	if err := m.get(ctx, token, "/users/@me?fields=name", &out); err != nil {
		return "", err
	}
	return out.Name, nil
}

func (m *MAL) get(ctx context.Context, token, path string, out any) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, malAPI+path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	return m.do(req, out)
}

func (m *MAL) patch(ctx context.Context, token, path string, form url.Values, out any) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodPatch, malAPI+path, strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return m.do(req, out)
}

func (m *MAL) do(req *http.Request, out any) error {
	resp, err := m.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == http.StatusUnauthorized {
		return ErrReauth
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("mal: HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(body, out)
}

func malSplitID(remoteID string) (kind, id string) {
	if i := strings.IndexByte(remoteID, ':'); i > 0 {
		return remoteID[:i], remoteID[i+1:]
	}
	return "manga", remoteID
}

func malJoinID(kind, id string) string { return kind + ":" + id }

type malNode struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	MainPic struct {
		Medium string `json:"medium"`
		Large  string `json:"large"`
	} `json:"main_picture"`
	Synopsis     string `json:"synopsis"`
	NumChapters  int    `json:"num_chapters"`
	NumEpisodes  int    `json:"num_episodes"`
	Status       string `json:"status"`
	MediaType    string `json:"media_type"`
	MyListStatus *struct {
		Status          string  `json:"status"`
		Score           float64 `json:"score"`
		NumChaptersRead float64 `json:"num_chapters_read"`
		NumEpWatched    float64 `json:"num_episodes_watched"`
		IsRereading     bool    `json:"is_rereading"`
		StartDate       string  `json:"start_date"`
		FinishDate      string  `json:"finish_date"`
	} `json:"my_list_status"`
}

func (m *MAL) Search(ctx context.Context, a Auth, q, contentType string) ([]SearchResult, error) {
	if len([]rune(q)) < 3 {
		return nil, nil
	}
	kinds := []string{"manga", "anime"}
	switch contentType {
	case "anime":
		kinds = []string{"anime"}
	case "manga", "novel":
		kinds = []string{"manga"}
	}

	var res []SearchResult
	for _, kind := range kinds {
		fields := "id,title,main_picture,synopsis,status,media_type,num_chapters"
		if kind == "anime" {
			fields = "id,title,main_picture,synopsis,status,media_type,num_episodes"
		}
		var out struct {
			Data []struct {
				Node malNode `json:"node"`
			} `json:"data"`
		}
		p := fmt.Sprintf("/%s?q=%s&limit=15&fields=%s", kind, url.QueryEscape(q), fields)
		if err := m.get(ctx, a.AccessToken, p, &out); err != nil {
			return nil, err
		}
		for _, d := range out.Data {
			n := d.Node
			count := n.NumChapters
			mt := "MANGA"
			if kind == "anime" {
				count, mt = n.NumEpisodes, "ANIME"
			}

			if contentType == "novel" && !strings.Contains(n.MediaType, "novel") {
				continue
			}
			res = append(res, SearchResult{
				RemoteID:         malJoinID(kind, strconv.Itoa(n.ID)),
				Title:            n.Title,
				URL:              fmt.Sprintf("https://myanimelist.net/%s/%d", kind, n.ID),
				CoverURL:         firstNonEmpty(n.MainPic.Large, n.MainPic.Medium),
				Summary:          truncate(n.Synopsis, 500),
				TotalChapters:    count,
				PublishingStatus: n.Status,
				MediaType:        mt,
			})
		}
	}
	return res, nil
}

func (m *MAL) node(ctx context.Context, token, kind, id string) (malNode, error) {
	fields := "id,title,num_chapters,status,my_list_status{status,score,num_chapters_read,is_rereading,start_date,finish_date}"
	if kind == "anime" {
		fields = "id,title,num_episodes,status,my_list_status{status,score,num_episodes_watched,is_rereading,start_date,finish_date}"
	}
	var n malNode
	err := m.get(ctx, token, fmt.Sprintf("/%s/%s?fields=%s", kind, id, fields), &n)
	return n, err
}

func (m *MAL) trackFromNode(kind, id string, n malNode) Track {
	t := Track{
		RemoteID:      malJoinID(kind, id),
		Title:         n.Title,
		URL:           fmt.Sprintf("https://myanimelist.net/%s/%s", kind, id),
		TotalChapters: n.NumChapters,
		Status:        StatusPlanToRead,
	}
	if kind == "anime" {
		t.TotalChapters = n.NumEpisodes
	}
	if s := n.MyListStatus; s != nil {
		t.Score = s.Score
		if kind == "anime" {
			t.LastChapterRead = s.NumEpWatched
		} else {
			t.LastChapterRead = s.NumChaptersRead
		}
		t.Status = malStatusToCanonical(s.Status, s.IsRereading)
		t.StartedAt = parseMALDate(s.StartDate)
		t.FinishedAt = parseMALDate(s.FinishDate)
	}
	return t
}

func (m *MAL) Bind(ctx context.Context, a Auth, remoteID string) (Track, error) {
	kind, id := malSplitID(remoteID)
	n, err := m.node(ctx, a.AccessToken, kind, id)
	if err != nil {
		return Track{}, err
	}
	return m.trackFromNode(kind, id, n), nil
}

func (m *MAL) Refresh(ctx context.Context, a Auth, t Track) (Track, error) {
	return m.Bind(ctx, a, t.RemoteID)
}

func (m *MAL) Push(ctx context.Context, a Auth, t Track) (Track, error) {
	kind, id := malSplitID(t.RemoteID)
	form := url.Values{}
	form.Set("status", canonicalToMALStatus(t.Status, kind))
	form.Set("score", strconv.Itoa(int(t.Score+0.5)))
	if kind == "anime" {
		form.Set("num_watched_episodes", strconv.Itoa(int(t.LastChapterRead+0.5)))
	} else {
		form.Set("num_chapters_read", strconv.Itoa(int(t.LastChapterRead+0.5)))
	}
	if t.Status == StatusRereading {
		form.Set("is_rereading", "true")
	}
	if err := m.patch(ctx, a.AccessToken, fmt.Sprintf("/%s/%s/my_list_status", kind, id), form, nil); err != nil {
		return Track{}, err
	}
	return m.Refresh(ctx, a, t)
}

func (m *MAL) ScoreOptions(a Auth) []string {
	out := make([]string, 11)
	for i := 0; i <= 10; i++ {
		out[i] = strconv.Itoa(i)
	}
	return out
}

func malStatusToCanonical(s string, rereading bool) Status {
	switch s {
	case "reading", "watching":
		if rereading {
			return StatusRereading
		}
		return StatusReading
	case "completed":
		return StatusCompleted
	case "on_hold":
		return StatusOnHold
	case "dropped":
		return StatusDropped
	case "plan_to_read", "plan_to_watch":
		return StatusPlanToRead
	default:
		return StatusPlanToRead
	}
}

func canonicalToMALStatus(s Status, kind string) string {
	switch s {
	case StatusReading, StatusRereading:
		if kind == "anime" {
			return "watching"
		}
		return "reading"
	case StatusCompleted:
		return "completed"
	case StatusOnHold:
		return "on_hold"
	case StatusDropped:
		return "dropped"
	default:
		if kind == "anime" {
			return "plan_to_watch"
		}
		return "plan_to_read"
	}
}

func parseMALDate(s string) *time.Time {
	if s == "" {
		return nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return &t
	}
	return nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
