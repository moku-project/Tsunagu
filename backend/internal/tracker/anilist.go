package tracker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	anilistAPI      = "https://graphql.anilist.co"
	anilistAuthBase = "https://anilist.co/api/v2/oauth/authorize"
	anilistTokenTTL = 365 * 24 * time.Hour
)

type AniList struct {
	clientID string
	http     *http.Client
}

func NewAniList(clientID string) *AniList {
	return &AniList{clientID: clientID, http: &http.Client{Timeout: 20 * time.Second}}
}

func (a *AniList) Key() string      { return "anilist" }
func (a *AniList) Name() string     { return "AniList" }
func (a *AniList) Configured() bool { return a.clientID != "" }
func (a *AniList) IconURL() string {
	return "https://anilist.co/img/icons/android-chrome-512x512.png"
}

func (a *AniList) AuthURL() string {
	if !a.Configured() {
		return ""
	}
	return fmt.Sprintf("%s?client_id=%s&response_type=token", anilistAuthBase, a.clientID)
}

func (a *AniList) query(ctx context.Context, token, doc string, vars map[string]any, out any) error {
	body, _ := json.Marshal(map[string]any{"query": doc, "variables": vars})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anilistAPI, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := a.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized {
		return ErrReauth
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("anilist: %s: %s", resp.Status, truncate(string(raw), 200))
	}
	var env struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("anilist: decode: %w", err)
	}
	if len(env.Errors) > 0 {
		return fmt.Errorf("anilist: %s", env.Errors[0].Message)
	}
	return json.Unmarshal(env.Data, out)
}

func (a *AniList) Exchange(ctx context.Context, pasted string) (Auth, error) {
	token := extractToken(pasted)
	if token == "" {
		return Auth{}, fmt.Errorf("no access token found in pasted value")
	}
	var out struct {
		Viewer struct {
			Name             string `json:"name"`
			MediaListOptions struct {
				ScoreFormat string `json:"scoreFormat"`
			} `json:"mediaListOptions"`
		} `json:"Viewer"`
	}
	doc := `query { Viewer { name mediaListOptions { scoreFormat } } }`
	if err := a.query(ctx, token, doc, nil, &out); err != nil {
		return Auth{}, err
	}
	exp := time.Now().Add(anilistTokenTTL)
	return Auth{
		AccessToken: token,
		ExpiresAt:   &exp,
		Username:    out.Viewer.Name,
		ScoreFormat: out.Viewer.MediaListOptions.ScoreFormat,
	}, nil
}

type alMedia struct {
	ID       int    `json:"id"`
	SiteURL  string `json:"siteUrl"`
	Chapters int    `json:"chapters"`
	Episodes int    `json:"episodes"`
	Type     string `json:"type"`
	Status   string `json:"status"`
	Title    struct {
		Romaji  string `json:"romaji"`
		English string `json:"english"`
		Native  string `json:"native"`
	} `json:"title"`
	Description string `json:"description"`
	CoverImage  struct {
		Large string `json:"large"`
	} `json:"coverImage"`
	MediaListEntry *alListEntry `json:"mediaListEntry"`
}

type alListEntry struct {
	ID          int         `json:"id"`
	Status      string      `json:"status"`
	Score       float64     `json:"score"`
	Progress    float64     `json:"progress"`
	Private     bool        `json:"private"`
	StartedAt   alFuzzyDate `json:"startedAt"`
	CompletedAt alFuzzyDate `json:"completedAt"`
}

type alFuzzyDate struct {
	Year  *int `json:"year"`
	Month *int `json:"month"`
	Day   *int `json:"day"`
}

func (d alFuzzyDate) time() *time.Time {
	if d.Year == nil || d.Month == nil || d.Day == nil {
		return nil
	}
	t := time.Date(*d.Year, time.Month(*d.Month), *d.Day, 0, 0, 0, 0, time.UTC)
	return &t
}

func (m alMedia) title() string {
	switch {
	case m.Title.English != "":
		return m.Title.English
	case m.Title.Romaji != "":
		return m.Title.Romaji
	default:
		return m.Title.Native
	}
}

func alSearchResult(m alMedia) SearchResult {
	count := m.Chapters
	if m.Type == "ANIME" {
		count = m.Episodes
	}
	return SearchResult{
		RemoteID:         strconv.Itoa(m.ID),
		Title:            m.title(),
		URL:              m.SiteURL,
		CoverURL:         m.CoverImage.Large,
		Summary:          truncate(stripTags(m.Description), 500),
		TotalChapters:    count,
		PublishingStatus: m.Status,
		MediaType:        m.Type,
	}
}

const alMediaFields = `id siteUrl chapters episodes type status
  title { romaji english native }
  description(asHtml: false)
  coverImage { large }`

func (a *AniList) Search(ctx context.Context, auth Auth, q, contentType string) ([]SearchResult, error) {

	var pages []struct{ alias, args string }
	switch contentType {
	case "anime":
		pages = []struct{ alias, args string }{{"anime", "type: ANIME"}}
	case "novel":
		pages = []struct{ alias, args string }{{"novel", "type: MANGA, format: NOVEL"}}
	case "manga":
		pages = []struct{ alias, args string }{{"manga", "type: MANGA, format_not: NOVEL"}}
	default:
		pages = []struct{ alias, args string }{
			{"manga", "type: MANGA, format_not: NOVEL"},
			{"novel", "type: MANGA, format: NOVEL"},
			{"anime", "type: ANIME"},
		}
	}

	var b strings.Builder
	b.WriteString("query($q: String) {\n")
	for _, p := range pages {
		fmt.Fprintf(&b, "  %s: Page(page: 1, perPage: 15) { media(search: $q, %s, sort: SEARCH_MATCH) { %s } }\n",
			p.alias, p.args, alMediaFields)
	}
	b.WriteString("}")

	var out map[string]struct {
		Media []alMedia `json:"media"`
	}
	if err := a.query(ctx, auth.AccessToken, b.String(), map[string]any{"q": q}, &out); err != nil {
		return nil, err
	}

	lists := make([][]alMedia, len(pages))
	maxLen := 0
	for i, p := range pages {
		lists[i] = out[p.alias].Media
		if len(lists[i]) > maxLen {
			maxLen = len(lists[i])
		}
	}
	var res []SearchResult
	for i := 0; i < maxLen; i++ {
		for _, l := range lists {
			if i < len(l) {
				res = append(res, alSearchResult(l[i]))
			}
		}
	}
	return res, nil
}

func (a *AniList) fetchMedia(ctx context.Context, auth Auth, remoteID string) (alMedia, error) {
	id, err := strconv.Atoi(remoteID)
	if err != nil {
		return alMedia{}, fmt.Errorf("bad anilist media id %q", remoteID)
	}
	var out struct {
		Media alMedia `json:"Media"`
	}
	doc := `query($id: Int) {
      Media(id: $id) {
        id siteUrl chapters status
        title { romaji english native }
        mediaListEntry {
          id status score progress private
          startedAt { year month day }
          completedAt { year month day }
        }
      }
    }`
	if err := a.query(ctx, auth.AccessToken, doc, map[string]any{"id": id}, &out); err != nil {
		return alMedia{}, err
	}
	return out.Media, nil
}

func (a *AniList) trackFromMedia(m alMedia) Track {
	t := Track{
		RemoteID:      strconv.Itoa(m.ID),
		Title:         m.title(),
		URL:           m.SiteURL,
		TotalChapters: m.Chapters,
		Status:        StatusPlanToRead,
	}
	if e := m.MediaListEntry; e != nil {
		t.LibraryID = strconv.Itoa(e.ID)
		t.Status = statusFromAniList(e.Status)
		t.Score = e.Score
		t.LastChapterRead = e.Progress
		t.Private = e.Private
		t.StartedAt = e.StartedAt.time()
		t.FinishedAt = e.CompletedAt.time()
	}
	return t
}

func (a *AniList) Bind(ctx context.Context, auth Auth, remoteID string) (Track, error) {
	m, err := a.fetchMedia(ctx, auth, remoteID)
	if err != nil {
		return Track{}, err
	}
	return a.trackFromMedia(m), nil
}

func (a *AniList) Refresh(ctx context.Context, auth Auth, t Track) (Track, error) {
	return a.Bind(ctx, auth, t.RemoteID)
}

func (a *AniList) Push(ctx context.Context, auth Auth, t Track) (Track, error) {
	id, err := strconv.Atoi(t.RemoteID)
	if err != nil {
		return Track{}, fmt.Errorf("bad anilist media id %q", t.RemoteID)
	}
	vars := map[string]any{
		"mediaId":  id,
		"status":   statusToAniList(t.Status),
		"progress": int(t.LastChapterRead),
		"score":    t.Score,
		"private":  t.Private,
	}
	var out struct {
		SaveMediaListEntry alListEntry `json:"SaveMediaListEntry"`
	}
	doc := `mutation($mediaId: Int, $status: MediaListStatus, $progress: Int, $score: Float, $private: Boolean) {
      SaveMediaListEntry(mediaId: $mediaId, status: $status, progress: $progress, score: $score, private: $private) {
        id status score progress private
        startedAt { year month day }
        completedAt { year month day }
      }
    }`
	if err := a.query(ctx, auth.AccessToken, doc, vars, &out); err != nil {
		return Track{}, err
	}
	e := out.SaveMediaListEntry
	t.LibraryID = strconv.Itoa(e.ID)
	t.Status = statusFromAniList(e.Status)
	t.Score = e.Score
	t.LastChapterRead = e.Progress
	t.Private = e.Private
	if st := e.StartedAt.time(); st != nil {
		t.StartedAt = st
	}
	if ft := e.CompletedAt.time(); ft != nil {
		t.FinishedAt = ft
	}
	return t, nil
}

func (a *AniList) ScoreOptions(auth Auth) []string {
	switch auth.ScoreFormat {
	case "POINT_100":
		return numRange(0, 100, 1)
	case "POINT_10_DECIMAL":
		out := []string{}
		for v := 0.0; v <= 10.0+1e-9; v += 0.5 {
			out = append(out, strconv.FormatFloat(v, 'f', 1, 64))
		}
		return out
	case "POINT_5":
		return numRange(0, 5, 1)
	case "POINT_3":
		return []string{"–", "😦", "😐", "😊"}
	default:
		return numRange(0, 10, 1)
	}
}

func statusFromAniList(s string) Status {
	switch s {
	case "CURRENT":
		return StatusReading
	case "PLANNING":
		return StatusPlanToRead
	case "COMPLETED":
		return StatusCompleted
	case "DROPPED":
		return StatusDropped
	case "PAUSED":
		return StatusOnHold
	case "REPEATING":
		return StatusRereading
	default:
		return StatusPlanToRead
	}
}

func statusToAniList(s Status) string {
	switch s {
	case StatusReading:
		return "CURRENT"
	case StatusPlanToRead:
		return "PLANNING"
	case StatusCompleted:
		return "COMPLETED"
	case StatusDropped:
		return "DROPPED"
	case StatusOnHold:
		return "PAUSED"
	case StatusRereading:
		return "REPEATING"
	default:
		return "PLANNING"
	}
}

func extractToken(s string) string {
	s = strings.TrimSpace(s)
	if !strings.Contains(s, "access_token=") {
		return s
	}
	s = strings.ReplaceAll(s, "#", "&")
	for _, part := range strings.Split(s[strings.Index(s, "?")+1:], "&") {
		if strings.HasPrefix(part, "access_token=") {
			return strings.TrimPrefix(part, "access_token=")
		}
	}
	return ""
}

func numRange(lo, hi, step int) []string {
	out := make([]string, 0, (hi-lo)/step+1)
	for v := lo; v <= hi; v += step {
		out = append(out, strconv.Itoa(v))
	}
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func stripTags(s string) string {
	var b strings.Builder
	depth := 0
	for _, r := range s {
		switch r {
		case '<':
			depth++
		case '>':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 {
				b.WriteRune(r)
			}
		}
	}
	return strings.TrimSpace(b.String())
}
