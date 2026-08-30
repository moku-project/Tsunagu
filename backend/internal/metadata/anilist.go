package metadata

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

const anilistAPI = "https://graphql.anilist.co"

const minTagRank = 60

const (
	maxTags    = 15
	maxAuthors = 4
)

type AniList struct{ http *http.Client }

func NewAniList() *AniList {
	return &AniList{http: &http.Client{Timeout: 15 * time.Second}}
}

func (a *AniList) Key() string { return "anilist" }

func (a *AniList) query(ctx context.Context, doc string, vars map[string]any, out any) error {
	body, _ := json.Marshal(map[string]any{"query": doc, "variables": vars})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anilistAPI, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := a.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("anilist: %s: %s", resp.Status, snip(string(raw), 200))
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

const mediaSelection = `
  id siteUrl format status genres synonyms isAdult
  title { romaji english native }
  description(asHtml: false)
  coverImage { extraLarge large }
  startDate { year }
  staff(perPage: 10, sort: RELEVANCE) { edges { role node { name { full } } } }
  tags { name rank isGeneralSpoiler isMediaSpoiler }
`

type alMedia struct {
	ID          int                                      `json:"id"`
	SiteURL     string                                   `json:"siteUrl"`
	Format      string                                   `json:"format"`
	Status      string                                   `json:"status"`
	Genres      []string                                 `json:"genres"`
	Synonyms    []string                                 `json:"synonyms"`
	IsAdult     bool                                     `json:"isAdult"`
	Title       struct{ Romaji, English, Native string } `json:"title"`
	Description string                                   `json:"description"`
	CoverImage  struct{ ExtraLarge, Large string }       `json:"coverImage"`
	StartDate   struct{ Year int }                       `json:"startDate"`
	Staff       struct {
		Edges []struct {
			Role string `json:"role"`
			Node struct {
				Name struct {
					Full string `json:"full"`
				} `json:"name"`
			} `json:"node"`
		} `json:"edges"`
	} `json:"staff"`
	Tags []struct {
		Name             string `json:"name"`
		Rank             int    `json:"rank"`
		IsGeneralSpoiler bool   `json:"isGeneralSpoiler"`
		IsMediaSpoiler   bool   `json:"isMediaSpoiler"`
	} `json:"tags"`
}

func (a *AniList) Search(ctx context.Context, query string, ct ContentType) ([]Candidate, error) {
	vars := map[string]any{"search": query, "type": "MANGA"}
	switch ct {
	case Anime:
		vars["type"] = "ANIME"
	case Novel:
		vars["format"] = "NOVEL"
	}
	doc := `query ($search: String, $type: MediaType, $format: MediaFormat) {
      Page(page: 1, perPage: 10) {
        media(search: $search, type: $type, format: $format, sort: SEARCH_MATCH) {` + mediaSelection + `}
      }
    }`
	var out struct {
		Page struct {
			Media []alMedia `json:"media"`
		} `json:"Page"`
	}
	if err := a.query(ctx, doc, vars, &out); err != nil {
		return nil, err
	}
	cands := make([]Candidate, 0, len(out.Page.Media))
	for _, m := range out.Page.Media {
		if ct == Manga && m.Format == "NOVEL" {
			continue
		}
		cands = append(cands, toCandidate(m))
	}
	return cands, nil
}

func (a *AniList) Fetch(ctx context.Context, providerID string) (*Candidate, error) {
	id, err := strconv.Atoi(providerID)
	if err != nil {
		return nil, fmt.Errorf("bad anilist id %q", providerID)
	}
	doc := `query ($id: Int) { Media(id: $id) {` + mediaSelection + `} }`
	var out struct {
		Media alMedia `json:"Media"`
	}
	if err := a.query(ctx, doc, map[string]any{"id": id}, &out); err != nil {
		return nil, err
	}
	c := toCandidate(out.Media)
	return &c, nil
}

func (a *AniList) MalID(ctx context.Context, providerID string) (int, error) {
	id, err := strconv.Atoi(providerID)
	if err != nil {
		return 0, fmt.Errorf("bad anilist id %q", providerID)
	}
	var out struct {
		Media struct {
			IDMal int `json:"idMal"`
		} `json:"Media"`
	}
	if err := a.query(ctx, `query ($id: Int) { Media(id: $id) { idMal } }`, map[string]any{"id": id}, &out); err != nil {
		return 0, err
	}
	return out.Media.IDMal, nil
}

func toCandidate(m alMedia) Candidate {
	titles := make([]string, 0, 3+len(m.Synonyms))
	for _, t := range []string{m.Title.English, m.Title.Romaji, m.Title.Native} {
		if t != "" {
			titles = append(titles, t)
		}
	}
	titles = append(titles, m.Synonyms...)

	primary := m.Title.English
	if primary == "" {
		primary = m.Title.Romaji
	}
	if primary == "" {
		primary = m.Title.Native
	}

	var authors []string
	for _, e := range m.Staff.Edges {
		r := strings.ToLower(e.Role)
		if strings.Contains(r, "story") || strings.Contains(r, "art") || strings.Contains(r, "original creator") {
			if n := e.Node.Name.Full; n != "" && !contains(authors, n) {
				authors = append(authors, n)
				if len(authors) == maxAuthors {
					break
				}
			}
		}
	}

	var tags []string
	for _, t := range m.Tags {
		if t.Rank < minTagRank || t.IsGeneralSpoiler || t.IsMediaSpoiler {
			continue
		}
		tags = append(tags, t.Name)
		if len(tags) == maxTags {
			break
		}
	}

	cover := m.CoverImage.ExtraLarge
	if cover == "" {
		cover = m.CoverImage.Large
	}

	return Candidate{
		ProviderID:   strconv.Itoa(m.ID),
		URL:          m.SiteURL,
		PrimaryTitle: primary,
		Titles:       titles,
		Description:  cleanDescription(m.Description),
		CoverURL:     cover,
		Status:       mapStatus(m.Status),
		Authors:      authors,
		Genres:       m.Genres,
		Tags:         tags,
		StartYear:    m.StartDate.Year,
		IsAdult:      m.IsAdult,
	}
}

func mapStatus(s string) string {
	switch s {
	case "RELEASING":
		return "Ongoing"
	case "FINISHED":
		return "Completed"
	case "HIATUS":
		return "Hiatus"
	case "CANCELLED":
		return "Cancelled"
	case "NOT_YET_RELEASED":
		return "Upcoming"
	default:
		return ""
	}
}

func cleanDescription(s string) string {
	s = strings.ReplaceAll(s, "<br>", "\n")
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = strings.ReplaceAll(s, "<br />", "\n")
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
	out := strings.TrimSpace(b.String())
	for strings.Contains(out, "\n\n\n") {
		out = strings.ReplaceAll(out, "\n\n\n", "\n\n")
	}
	return out
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func snip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
