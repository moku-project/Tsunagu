package sync

import "testing"

func nums(vs ...float64) []chapterSummary {
	s := make([]chapterSummary, len(vs))
	for i, v := range vs {
		s[i] = chapterSummary{Number: v}
	}
	return s
}

func TestChaptersAreNewestFirst(t *testing.T) {
	cases := []struct {
		name string
		in   []chapterSummary
		want bool
	}{
		{"descending numbers", nums(10, 9, 8, 7, 1), true},
		{"ascending numbers", nums(1, 2, 3, 4, 10), false},
		{"single chapter", nums(1), false},
		{"one out-of-order special, still ascending", nums(1, 2, 0, 3, 4), false},
		{"no usable numbers falls back to convention", nums(-1, -1, -1), true},
		{
			"no numbers but upload ts descending",
			[]chapterSummary{{Number: -1, UploadTS: 300}, {Number: -1, UploadTS: 200}, {Number: -1, UploadTS: 100}},
			true,
		},
		{
			"no numbers but upload ts ascending",
			[]chapterSummary{{Number: -1, UploadTS: 100}, {Number: -1, UploadTS: 200}, {Number: -1, UploadTS: 300}},
			false,
		},
	}
	for _, c := range cases {
		if got := chaptersAreNewestFirst(c.in); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestDeriveScanlators(t *testing.T) {
	s := []chapterSummary{
		{Name: "Chapter 1", Number: 1},
		{Name: "Chapter 1", Number: 1},
		{Name: "Chapter 2", Number: 2},
		{Name: "Chapter 2", Number: 2},
		{Name: "Chapter 700: At His Own Pace", Number: 700},
	}
	want := []string{"1", "2", "1", "2", ""}
	for i, got := range deriveScanlators(s) {
		if got != want[i] {
			t.Errorf("deriveScanlators[%d] = %q, want %q", i, got, want[i])
		}
	}
}

func TestDeriveScanlatorsUnnumbered(t *testing.T) {
	s := []chapterSummary{
		{SourceID: "a", Name: "Chapter 1", Number: -1},
		{SourceID: "b", Name: "Chapter 1: Official Scan", Number: -1},
		{SourceID: "c", Name: "Chapter 2", Number: -1},
		{SourceID: "d", Name: "Chapter 2: Official Scan", Number: -1},
		{SourceID: "e", Name: "Chapter 699: Morning Edition", Number: -1},
		{SourceID: "f", Name: "Chapter 700: At His Own Pace", Number: -1},
	}
	want := []string{"1", "2", "1", "2", "", ""}
	for i, got := range deriveScanlators(s) {
		if got != want[i] {
			t.Errorf("[%d] %q = %q, want %q", i, s[i].Name, got, want[i])
		}
	}
}
