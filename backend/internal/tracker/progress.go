package tracker

import (
	"context"
	"time"

	"tsunagu/backend/internal/chapternum"
)

func (m *Manager) maxRead(ctx context.Context, mediaID int64) (float64, error) {
	n, err := m.q.MaxReadChapterNumber(ctx, mediaID)
	if err != nil {
		return 0, err
	}
	if n > 0 {
		return n, nil
	}
	rows, err := m.q.ListCompletedChapterTitles(ctx, mediaID)
	if err != nil {
		return 0, err
	}
	best := 0.0
	for _, r := range rows {
		v := 0.0
		if r.Number.Valid && r.Number.Float64 > 0 {
			v = r.Number.Float64
		} else if r.Title.Valid {
			v = chapternum.FromTitle(r.Title.String)
		}
		if v > best {
			best = v
		}
	}
	return best, nil
}

// advanceStatus marks a planned title reading (dating the start) and a finished one completed.
// An existing start date is kept, since stamping today would overwrite real history on the tracker.
func advanceStatus(t *Track) {
	now := time.Now()
	if t.Status == StatusPlanToRead || t.Status == 0 {
		t.Status = StatusReading
		if t.StartedAt == nil {
			t.StartedAt = &now
		}
	}
	if t.TotalChapters > 0 && t.LastChapterRead >= float64(t.TotalChapters) &&
		(t.Status == StatusReading || t.Status == StatusRereading) {
		t.Status = StatusCompleted
		if t.FinishedAt == nil {
			t.FinishedAt = &now
		}
	}
}
