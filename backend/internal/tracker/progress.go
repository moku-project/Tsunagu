package tracker

import (
	"context"

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
