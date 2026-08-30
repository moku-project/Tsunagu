package chapternum

import (
	"regexp"
	"strconv"
	"strings"
)

var regexes = []*regexp.Regexp{

	regexp.MustCompile(`(?i)(?:chapter|episode)\W*([0-9]+(?:\.[0-9]+)?)`),

	regexp.MustCompile(`(?i)\b(?:chap|ch|ep|#)\W*([0-9]+(?:\.[0-9]+)?)`),

	regexp.MustCompile(`^\s*([0-9]+(?:\.[0-9]+)?)\b`),
}

func FromTitle(title string) float64 {
	t := strings.TrimSpace(title)
	for _, re := range regexes {
		if m := re.FindStringSubmatch(t); m != nil {
			if f, err := strconv.ParseFloat(m[1], 64); err == nil && f > 0 {
				return f
			}
		}
	}
	return 0
}

func Resolve(num float64, title string) float64 {
	if num > 0 {
		return num
	}
	return FromTitle(title)
}
