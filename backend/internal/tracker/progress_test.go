package tracker

import (
	"testing"
	"time"
)

func TestAdvanceStatus(t *testing.T) {
	cases := []struct {
		name    string
		in      Track
		want    Status
		ended   bool
		started bool
	}{
		{"planned becomes reading", Track{Status: StatusPlanToRead, LastChapterRead: 3, TotalChapters: 10}, StatusReading, false, true},
		{"last chapter completes", Track{Status: StatusReading, LastChapterRead: 10, TotalChapters: 10}, StatusCompleted, true, false},
		{"reread completes", Track{Status: StatusRereading, LastChapterRead: 12, TotalChapters: 12}, StatusCompleted, true, false},
		{"unknown length stays reading", Track{Status: StatusReading, LastChapterRead: 500}, StatusReading, false, false},
		{"paused stays paused", Track{Status: StatusOnHold, LastChapterRead: 10, TotalChapters: 10}, StatusOnHold, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.in
			advanceStatus(&got)
			if got.Status != c.want {
				t.Fatalf("status = %v, want %v", got.Status, c.want)
			}
			if (got.FinishedAt != nil) != c.ended {
				t.Fatalf("finishedAt set = %v, want %v", got.FinishedAt != nil, c.ended)
			}
			if (got.StartedAt != nil) != c.started {
				t.Fatalf("startedAt set = %v, want %v", got.StartedAt != nil, c.started)
			}
		})
	}
}

func TestAdvanceStatusKeepsAnExistingStartDate(t *testing.T) {
	then := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	got := Track{Status: StatusPlanToRead, LastChapterRead: 1, TotalChapters: 10, StartedAt: &then}
	advanceStatus(&got)
	if got.StartedAt == nil || !got.StartedAt.Equal(then) {
		t.Fatalf("startedAt = %v, want %v", got.StartedAt, then)
	}
}
