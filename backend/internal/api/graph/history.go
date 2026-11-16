package graph

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"tsunagu/backend/internal/api/graph/model"
	"tsunagu/backend/internal/db/sqlcgen"
	"tsunagu/backend/internal/history"
)

func toHistorySession(in *model.ReadingSessionInput) (history.Session, error) {
	if in == nil {
		return history.Session{}, errors.New("missing session")
	}
	mediaID, err := parseID(in.MediaID)
	if err != nil {
		return history.Session{}, fmt.Errorf("mediaId: %w", err)
	}
	s := history.Session{
		ID: in.ID, MediaID: mediaID, StartPage: int64(in.StartPage), EndPage: int64(in.EndPage),
		StartedAt: in.StartedAt.UnixMilli(), EndedAt: in.EndedAt.UnixMilli(), DurationMs: int64(in.DurationMs), Chapters: int64(in.Chapters),
	}
	for _, c := range []struct {
		in  *string
		out **int64
	}{{in.StartChapterID, &s.StartChapterID}, {in.EndChapterID, &s.EndChapterID}} {
		if c.in == nil || *c.in == "" {
			continue
		}
		id, err := parseID(*c.in)
		if err != nil {
			return history.Session{}, fmt.Errorf("chapter id: %w", err)
		}
		*c.out = &id
	}
	return s, nil
}

// toReadingSessions loads each session's title and chapters in two queries.
func (r *Resolver) toReadingSessions(ctx context.Context, sessions []history.Session) ([]*model.ReadingSession, error) {
	out := make([]*model.ReadingSession, 0, len(sessions))
	if len(sessions) == 0 {
		return out, nil
	}
	var mediaIDs, chapterIDs []int64
	for _, s := range sessions {
		mediaIDs = append(mediaIDs, s.MediaID)
		for _, c := range []*int64{s.StartChapterID, s.EndChapterID} {
			if c != nil {
				chapterIDs = append(chapterIDs, *c)
			}
		}
	}
	mediaRows, err := r.Q.ListMediaByIDs(ctx, mediaIDs)
	if err != nil {
		return nil, err
	}
	media := make(map[int64]sqlcgen.Medium, len(mediaRows))
	for _, m := range mediaRows {
		media[m.ID] = m
	}
	chapters := map[int64]sqlcgen.Chapter{}
	if len(chapterIDs) > 0 {
		rows, err := r.Q.ListChaptersByIDs(ctx, chapterIDs)
		if err != nil {
			return nil, err
		}
		for _, c := range rows {
			chapters[c.ID] = c
		}
	}
	chapter := func(id *int64) *model.Chapter {
		if id == nil {
			return nil
		}
		if c, ok := chapters[*id]; ok {
			return toChapter(c)
		}
		return nil
	}
	for _, s := range sessions {
		m, ok := media[s.MediaID]
		if !ok {
			continue
		}
		out = append(out, &model.ReadingSession{
			ID: s.ID, Media: toMedia(m, r.MediaDir), StartChapter: chapter(s.StartChapterID), EndChapter: chapter(s.EndChapterID),
			StartPage: int32(s.StartPage), EndPage: int32(s.EndPage),
			StartedAt: time.UnixMilli(s.StartedAt), EndedAt: time.UnixMilli(s.EndedAt),
			DurationMs: int32(s.DurationMs), Chapters: int32(s.Chapters),
		})
	}
	return out, nil
}

func toReadingStats(st history.Stats) *model.ReadingStats {
	out := &model.ReadingStats{
		TotalChaptersRead: int32(st.TotalChaptersRead), TotalTitlesRead: int32(st.TotalTitlesRead),
		TotalMinutesRead: int32(st.TotalMinutesRead), ChaptersThisWeek: int32(st.ChaptersThisWeek),
		CurrentStreakDays: int32(st.CurrentStreakDays), LongestStreakDays: int32(st.LongestStreakDays),
		LastStreakDate: st.LastStreakDate, Days: make([]*model.ReadingDay, 0, len(st.Days)),
	}
	if st.FirstReadAt > 0 {
		t := time.UnixMilli(st.FirstReadAt)
		out.FirstReadAt = &t
	}
	if st.LastReadAt > 0 {
		t := time.UnixMilli(st.LastReadAt)
		out.LastReadAt = &t
	}
	for _, d := range st.Days {
		out.Days = append(out.Days, &model.ReadingDay{Date: d.Date, Seconds: int32(d.Seconds), Chapters: int32(d.Chapters)})
	}
	return out
}

func optionalID(id *string) (*int64, error) {
	if id == nil || *id == "" {
		return nil, nil
	}
	n, err := strconv.ParseInt(*id, 10, 64)
	if err != nil {
		return nil, err
	}
	return &n, nil
}
