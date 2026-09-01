package metadata

import (
	"context"
	"regexp"
	"strings"
)

type Provider interface {
	Key() string
	Search(ctx context.Context, query string, ct ContentType) ([]Candidate, error)
	Fetch(ctx context.Context, providerID string) (*Candidate, error)
}

type ContentType string

const (
	Manga ContentType = "manga"
	Novel ContentType = "novel"
	Anime ContentType = "anime"
)

type Candidate struct {
	ProviderID   string
	URL          string
	PrimaryTitle string
	Titles       []string
	Description  string
	CoverURL     string
	Status       string
	Authors      []string
	Genres       []string
	Tags         []string
	TagWeights   []int
	StartYear    int
	IsAdult      bool
}

const (
	AutoApplyScore = 0.87
)

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

var stripWords = map[string]bool{
	"vol": true, "volume": true, "season": true, "part": true,
	"the": true, "manga": true, "novel": true, "ln": true,
}

func normalizeTitle(s string) string {
	s = strings.ToLower(s)
	s = nonAlnum.ReplaceAllString(s, " ")
	fields := strings.Fields(s)
	out := fields[:0]
	for _, f := range fields {
		if stripWords[f] {
			continue
		}
		out = append(out, f)
	}
	return strings.Join(out, " ")
}

func bestMatch(sourceTitle string, sourceYear int, cands []Candidate) (Candidate, float64) {
	want := normalizeTitle(sourceTitle)
	var best Candidate
	bestScore := -1.0
	for _, c := range cands {
		s := 0.0
		for _, t := range c.Titles {
			if v := diceCoefficient(want, normalizeTitle(t)); v > s {
				s = v
			}
		}
		if sourceYear > 0 && c.StartYear > 0 {
			switch d := sourceYear - c.StartYear; {
			case d == 0:
				s += 0.05
			case d < -1 || d > 1:
				s -= 0.10
			}
		}
		if s > bestScore {
			bestScore, best = s, c
		}
	}
	if bestScore < 0 {
		bestScore = 0
	}
	if bestScore > 1 {
		bestScore = 1
	}
	return best, bestScore
}

func diceCoefficient(a, b string) float64 {
	if a == b {
		if a == "" {
			return 0
		}
		return 1
	}
	if len(a) < 2 || len(b) < 2 {
		return 0
	}
	ba := bigrams(a)
	bb := bigrams(b)
	inter := 0
	for g, n := range ba {
		if m, ok := bb[g]; ok {
			inter += min(n, m)
		}
	}
	total := 0
	for _, n := range ba {
		total += n
	}
	for _, n := range bb {
		total += n
	}
	if total == 0 {
		return 0
	}
	return 2 * float64(inter) / float64(total)
}

func bigrams(s string) map[string]int {
	r := []rune(s)
	m := make(map[string]int, len(r))
	for i := 0; i+1 < len(r); i++ {
		if r[i] == ' ' || r[i+1] == ' ' {
			continue
		}
		m[string(r[i:i+2])]++
	}
	return m
}
