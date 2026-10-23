package sync

import (
	"database/sql"
	"slices"
	"testing"

	"tsunagu/backend/internal/db/sqlcgen"
)

func chapters(ids ...string) []sqlcgen.Chapter {
	out := make([]sqlcgen.Chapter, len(ids))
	for i, id := range ids {
		out[i] = sqlcgen.Chapter{ID: int64(i + 1), ExternalID: id}
	}
	return out
}

func numbered(id int64, ext string, num float64) sqlcgen.Chapter {
	return sqlcgen.Chapter{ID: id, ExternalID: ext, Number: sql.NullFloat64{Float64: num, Valid: true}}
}

func TestNewChapterIDs(t *testing.T) {
	cases := []struct {
		name   string
		before []sqlcgen.Chapter
		after  []sqlcgen.Chapter
		want   []int64
	}{
		{"first-ever sync is a backlog, not new", nil, chapters("a", "b", "c"), nil},
		{"nothing new", chapters("a", "b"), chapters("a", "b"), nil},
		{"new releases", chapters("a", "b"), chapters("a", "b", "c", "d"), []int64{3, 4}},
		{"new chapter inserted mid-list", chapters("a", "c"), chapters("a", "b", "c"), []int64{2}},
		{"source dropped a chapter", chapters("a", "b", "c"), chapters("a", "c"), nil},
		{
			"source changed every chapter's ID",
			[]sqlcgen.Chapter{numbered(1, "old/1", 1), numbered(2, "old/2", 2)},
			[]sqlcgen.Chapter{numbered(3, "new/1", 1), numbered(4, "new/2", 2), numbered(5, "new/3", 3)},
			[]int64{5},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := newChapterIDs(c.before, c.after); !slices.Equal(got, c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
		})
	}
}
