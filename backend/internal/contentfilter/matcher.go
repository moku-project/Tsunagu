package contentfilter

import "strings"

type Level int

const (
	Unrestricted Level = 0
	Moderate     Level = 1
	Strict       Level = 2
)

func ParseLevel(s string) Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "strict":
		return Strict
	case "moderate":
		return Moderate
	default:
		return Unrestricted
	}
}

func (l Level) String() string {
	switch l {
	case Strict:
		return "strict"
	case Moderate:
		return "moderate"
	default:
		return "unrestricted"
	}
}

// Hidden reports whether a precomputed content_block_rank is hidden at level.
// blockRank 0 = never filtered.
func Hidden(blockRank int, lvl Level) bool {
	return blockRank != 0 && blockRank <= int(lvl)
}

type Rule struct {
	ID         int64
	Category   string
	Field      string // genre | tag | title | description
	Keyword    string // lowercased, trimmed
	MinWeight  int
	BlockLevel int // 1 = hidden at Moderate+Strict, 2 = hidden at Strict only
	IsDefault  bool
}

// alias maps a genre/tag variant to a canonical term that appears in the
// default rule set, so a source emitting only "Mature" still trips "adult".
var alias = map[string]string{
	"mature":          "adult",
	"18+":             "adult",
	"r-18":            "adult",
	"r18":             "adult",
	"explicit":        "pornographic",
	"h":               "hentai",
	"doujinshi (18+)": "hentai",
	"guro":            "gore",
	"splatter":        "gore",
	"grotesque":       "gore",
}

func norm(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

type matcher struct {
	genre, tag, title, desc []Rule
}

func buildMatcher(rules []Rule) *matcher {
	m := &matcher{}
	for _, r := range rules {
		r.Keyword = norm(r.Keyword)
		if r.Keyword == "" {
			continue
		}
		switch r.Field {
		case "genre":
			m.genre = append(m.genre, r)
		case "tag":
			m.tag = append(m.tag, r)
		case "title":
			m.title = append(m.title, r)
		case "description":
			m.desc = append(m.desc, r)
		}
	}
	return m
}

func (m *matcher) hasRules() bool {
	return len(m.genre)+len(m.tag)+len(m.title)+len(m.desc) > 0
}

type weighted struct {
	s string
	w int
}

func (m *matcher) blockRank(title, desc string, genres, tags []string, weights []int) int {
	best := 0
	take := func(bl int) {
		if bl != 0 && (best == 0 || bl < best) {
			best = bl
		}
	}

	if len(m.title) > 0 {
		tl := norm(title)
		for _, r := range m.title {
			if strings.Contains(tl, r.Keyword) {
				take(r.BlockLevel)
			}
		}
	}
	if len(m.desc) > 0 && desc != "" {
		dl := norm(desc)
		for _, r := range m.desc {
			if strings.Contains(dl, r.Keyword) {
				take(r.BlockLevel)
			}
		}
	}

	if len(m.genre) > 0 {
		g := expand(genres)
		for _, r := range m.genre {
			for _, x := range g {
				if x == r.Keyword || strings.Contains(x, r.Keyword) {
					take(r.BlockLevel)
					break
				}
			}
		}
	}

	if len(m.tag) > 0 {
		t := expandWeighted(tags, weights)
		for _, r := range m.tag {
			for _, tw := range t {
				if tw.w >= r.MinWeight && (tw.s == r.Keyword || strings.Contains(tw.s, r.Keyword)) {
					take(r.BlockLevel)
					break
				}
			}
		}
	}

	return best
}

func (m *matcher) titleBlockRank(title string) int {
	best := 0
	tl := norm(title)
	for _, r := range m.title {
		if strings.Contains(tl, r.Keyword) && (best == 0 || r.BlockLevel < best) {
			best = r.BlockLevel
		}
	}
	return best
}

func expand(terms []string) []string {
	out := make([]string, 0, len(terms)*2)
	for _, t := range terms {
		t = norm(t)
		if t == "" {
			continue
		}
		out = append(out, t)
		if a, ok := alias[t]; ok {
			out = append(out, a)
		}
	}
	return out
}

func expandWeighted(terms []string, weights []int) []weighted {
	out := make([]weighted, 0, len(terms)*2)
	for i, t := range terms {
		t = norm(t)
		if t == "" {
			continue
		}
		w := 0
		if i < len(weights) {
			w = weights[i]
		}
		out = append(out, weighted{t, w})
		if a, ok := alias[t]; ok {
			out = append(out, weighted{a, w})
		}
	}
	return out
}
