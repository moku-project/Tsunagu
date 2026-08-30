package chapternum

import "testing"

func TestFromTitle(t *testing.T) {
	cases := map[string]float64{
		"Chapter 700: At His Own Pace":       700,
		"Ch. 700.5":                          700.5,
		"700 - Morning Edition":              700,
		"700: Something":                     700,
		"Episode 12":                         12,
		"His Own PaceChapter 701: Adventure": 701,
		"Prologue":                           0,
		"":                                   0,
		"Vol. 5 Ch. 42 Title":                42,
	}
	for in, want := range cases {
		if got := FromTitle(in); got != want {
			t.Errorf("%q => %v, want %v", in, got, want)
		}
	}
}

func TestResolve(t *testing.T) {
	if Resolve(5, "Chapter 9") != 5 {
		t.Error("should keep a good number")
	}
	if Resolve(-1, "Chapter 9") != 9 {
		t.Error("should parse when number missing")
	}
	if Resolve(0, "Prologue") != 0 {
		t.Error("unparseable stays 0")
	}
}
