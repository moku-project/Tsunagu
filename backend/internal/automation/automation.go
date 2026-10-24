// Package automation applies the library's download/update rules on the
// server: delete-after-read, download-ahead, max-kept chapters, auto-download
// of new chapters, and scheduled/app-open library updates. Clients only edit
// the rules; nothing here depends on a client being open.
package automation

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"slices"
	"sync"
	"time"

	"tsunagu/backend/internal/db/sqlcgen"
	"tsunagu/backend/internal/download"
	"tsunagu/backend/internal/sandbox"
	syncpkg "tsunagu/backend/internal/sync"
)

const (
	IntervalDaily  = "daily"
	IntervalWeekly = "weekly"
	IntervalManual = "manual"

	tickEvery = 10 * time.Minute
	// Reopening the app repeatedly shouldn't hammer every source each time.
	appOpenCooldown = 15 * time.Minute
)

// Trigger says why a library update was requested.
type Trigger int

const (
	TriggerManual Trigger = iota
	TriggerAppOpen
	TriggerScheduled
)

// Rules are the effective automation rules for one series.
type Rules struct {
	AutoDownload     bool
	DownloadAhead    int
	DeleteOnRead     bool
	DeleteDelayHours int
	MaxKeepChapters  int
	PauseUpdates     bool
	RefreshInterval  string
}

type Manager struct {
	q  *sqlcgen.Queries
	dm *download.Manager
	sy *syncpkg.Syncer
	sc *sandbox.SupervisedClient

	// startUpdate syncs the given series in the background; swapped out in tests.
	startUpdate func(ids []int64) bool

	// Serializes rule application so concurrent read marks for one series
	// don't both enqueue or delete the same chapters.
	mu sync.Mutex
}

func New(q *sqlcgen.Queries, dm *download.Manager, sy *syncpkg.Syncer, sc *sandbox.SupervisedClient) *Manager {
	m := &Manager{q: q, dm: dm, sy: sy, sc: sc}
	m.startUpdate = func(ids []int64) bool { return sy.StartLibraryUpdate(sc, ids) }
	sy.SetLibraryUpdateHooks(m.onSeriesChecked, m.onUpdateFinished)
	return m
}

// Start runs the scheduler: due delete-after-read jobs and interval-based
// library updates. It returns immediately.
func (m *Manager) Start(ctx context.Context) {
	go func() {
		// Give the sandbox and extension reload a moment after server start.
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Minute):
		}
		t := time.NewTicker(tickEvery)
		defer t.Stop()
		for {
			m.tick(ctx)
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
		}
	}()
}

func (m *Manager) tick(ctx context.Context) {
	m.runDueDeletions(ctx)
	if _, err := m.StartLibraryUpdate(ctx, nil, TriggerScheduled); err != nil {
		log.Printf("automation: scheduled library update: %v", err)
	}
}

// RulesFor resolves a series' effective rules from the global settings and its
// overrides (nil when it has none).
func RulesFor(s sqlcgen.AutomationSetting, ov *sqlcgen.MediaAutomation) Rules {
	r := Rules{
		AutoDownload:     s.AutoDownload != 0,
		DownloadAhead:    int(s.DownloadAhead),
		DeleteOnRead:     s.DeleteOnRead != 0,
		DeleteDelayHours: int(s.DeleteDelayHours),
		MaxKeepChapters:  int(s.MaxKeepChapters),
		RefreshInterval:  s.RefreshInterval,
	}
	if ov != nil && s.EnforceGlobal == 0 {
		if ov.AutoDownload.Valid {
			r.AutoDownload = ov.AutoDownload.Int64 != 0
		}
		if ov.DownloadAhead.Valid {
			r.DownloadAhead = int(ov.DownloadAhead.Int64)
		}
		if ov.DeleteOnRead.Valid {
			r.DeleteOnRead = ov.DeleteOnRead.Int64 != 0
		}
		if ov.DeleteDelayHours.Valid {
			r.DeleteDelayHours = int(ov.DeleteDelayHours.Int64)
		}
		if ov.MaxKeepChapters.Valid {
			r.MaxKeepChapters = int(ov.MaxKeepChapters.Int64)
		}
		if ov.PauseUpdates.Valid {
			r.PauseUpdates = ov.PauseUpdates.Int64 != 0
		}
		if ov.RefreshInterval.Valid {
			r.RefreshInterval = ov.RefreshInterval.String
		}
	}
	if s.Enabled == 0 {
		r.AutoDownload = false
		r.DownloadAhead = 0
		r.DeleteOnRead = false
		r.MaxKeepChapters = 0
	}
	return r
}

// Effective returns the rules that currently apply to a series.
func (m *Manager) Effective(ctx context.Context, mediaID int64) (Rules, error) {
	s, err := m.q.GetAutomationSettings(ctx)
	if err != nil {
		return Rules{}, err
	}
	ov, err := m.q.GetMediaAutomation(ctx, mediaID)
	if errors.Is(err, sql.ErrNoRows) {
		return RulesFor(s, nil), nil
	}
	if err != nil {
		return Rules{}, err
	}
	return RulesFor(s, &ov), nil
}

// ChaptersRead applies on-read rules after chapters were marked read. Safe to
// call from a goroutine; failures are logged.
func (m *Manager) ChaptersRead(ctx context.Context, mediaID int64, chapterIDs []int64) {
	if len(chapterIDs) == 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	rules, err := m.Effective(ctx, mediaID)
	if err != nil {
		log.Printf("automation: rules for media %d: %v", mediaID, err)
		return
	}
	if !rules.DeleteOnRead && rules.DownloadAhead == 0 && rules.MaxKeepChapters == 0 {
		return
	}
	state, err := m.q.ListChapterAutomationState(ctx, mediaID)
	if err != nil {
		log.Printf("automation: chapters for media %d: %v", mediaID, err)
		return
	}

	justRead := make(map[int64]bool, len(chapterIDs))
	for _, id := range chapterIDs {
		justRead[id] = true
	}
	toDelete := map[int64]bool{}

	if rules.DeleteOnRead {
		deleteAt := time.Now().UTC().Add(time.Duration(rules.DeleteDelayHours) * time.Hour)
		for _, ch := range state {
			if !justRead[ch.ID] || ch.IsDownloaded == 0 {
				continue
			}
			if rules.DeleteDelayHours <= 0 {
				toDelete[ch.ID] = true
				continue
			}
			if err := m.q.ScheduleChapterDeletion(ctx, sqlcgen.ScheduleChapterDeletionParams{ChapterID: ch.ID, DeleteAt: deleteAt}); err != nil {
				log.Printf("automation: schedule deletion of chapter %d: %v", ch.ID, err)
			}
		}
	}

	var toQueue []int64
	if rules.DownloadAhead > 0 {
		last := -1
		for i, ch := range state {
			if justRead[ch.ID] {
				last = i
			}
		}
		if last >= 0 {
			end := min(last+1+rules.DownloadAhead, len(state))
			for _, ch := range state[last+1 : end] {
				if ch.IsRead == 0 && ch.IsDownloaded == 0 && ch.IsPending == 0 {
					toQueue = append(toQueue, ch.ID)
				}
			}
		}
	}

	if rules.MaxKeepChapters > 0 {
		// Keep unread chapters first (nearest first), then the newest read ones.
		queued := make(map[int64]bool, len(toQueue))
		for _, id := range toQueue {
			queued[id] = true
		}
		var ahead, read []sqlcgen.ListChapterAutomationStateRow
		for _, ch := range state {
			onDisk := ch.IsDownloaded != 0 && !toDelete[ch.ID]
			switch {
			case onDisk && (ch.IsRead != 0 || justRead[ch.ID]):
				read = append(read, ch)
			case onDisk || queued[ch.ID]:
				ahead = append(ahead, ch)
			}
		}
		room := rules.MaxKeepChapters
		for _, ch := range ahead {
			if room > 0 {
				room--
				continue
			}
			if queued[ch.ID] {
				delete(queued, ch.ID)
			} else {
				toDelete[ch.ID] = true
			}
		}
		for i := 0; i < len(read)-room; i++ {
			toDelete[read[i].ID] = true
		}
		toQueue = slices.DeleteFunc(toQueue, func(id int64) bool { return !queued[id] })
	}

	if len(toDelete) > 0 {
		ids := make([]int64, 0, len(toDelete))
		for id := range toDelete {
			ids = append(ids, id)
		}
		if err := m.dm.DeleteChapters(ctx, ids); err != nil {
			log.Printf("automation: delete chapters of media %d: %v", mediaID, err)
		} else if err := m.q.CancelChapterDeletions(ctx, ids); err != nil {
			log.Printf("automation: clear scheduled deletions: %v", err)
		}
	}
	m.enqueue(ctx, toQueue)
}

// ChaptersUnread cancels pending delete-after-read jobs for chapters that were
// marked unread again.
func (m *Manager) ChaptersUnread(ctx context.Context, chapterIDs []int64) {
	if len(chapterIDs) == 0 {
		return
	}
	if err := m.q.CancelChapterDeletions(ctx, chapterIDs); err != nil {
		log.Printf("automation: cancel scheduled deletions: %v", err)
	}
}

func (m *Manager) enqueue(ctx context.Context, chapterIDs []int64) {
	if len(chapterIDs) == 0 {
		return
	}
	for _, id := range chapterIDs {
		if _, err := m.q.EnqueueDownload(ctx, id); err != nil && !errors.Is(err, sql.ErrNoRows) {
			log.Printf("automation: enqueue chapter %d: %v", id, err)
		}
	}
	m.dm.Wake()
}

func (m *Manager) runDueDeletions(ctx context.Context) {
	due, err := m.q.ListDueChapterDeletions(ctx, time.Now().UTC())
	if err != nil {
		log.Printf("automation: list due deletions: %v", err)
		return
	}
	if len(due) == 0 {
		return
	}
	ids := make([]int64, 0, len(due))
	for _, d := range due {
		ids = append(ids, d.ChapterID)
	}
	if err := m.dm.DeleteChapters(ctx, ids); err != nil {
		log.Printf("automation: delete-after-read: %v", err)
	}
	// Drop the jobs even on failure so one bad chapter can't wedge the queue.
	if err := m.q.CancelChapterDeletions(ctx, ids); err != nil {
		log.Printf("automation: clear scheduled deletions: %v", err)
	}
}

// onSeriesChecked runs after a library update synced one series.
func (m *Manager) onSeriesChecked(ctx context.Context, mediaID int64, newChapterIDs []int64) {
	if err := m.q.SetMediaChaptersCheckedAt(ctx, sqlcgen.SetMediaChaptersCheckedAtParams{
		ChaptersCheckedAt: sql.NullTime{Time: time.Now().UTC(), Valid: true},
		ID:                mediaID,
	}); err != nil {
		log.Printf("automation: mark media %d checked: %v", mediaID, err)
	}
	if len(newChapterIDs) == 0 {
		return
	}
	rules, err := m.Effective(ctx, mediaID)
	if err != nil {
		log.Printf("automation: rules for media %d: %v", mediaID, err)
		return
	}
	auto := rules.AutoDownload
	if !auto {
		inFolder, err := m.q.MediaInAutoDownloadFolder(ctx, mediaID)
		if err != nil {
			log.Printf("automation: folder flags for media %d: %v", mediaID, err)
		}
		auto = inFolder
	}
	if !auto {
		return
	}

	isNew := make(map[int64]bool, len(newChapterIDs))
	for _, id := range newChapterIDs {
		isNew[id] = true
	}
	state, err := m.q.ListChapterAutomationState(ctx, mediaID)
	if err != nil {
		log.Printf("automation: chapters for media %d: %v", mediaID, err)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var toQueue []int64
	for _, ch := range state { // reading order
		if isNew[ch.ID] && ch.IsRead == 0 && ch.IsDownloaded == 0 && ch.IsPending == 0 {
			toQueue = append(toQueue, ch.ID)
		}
	}
	m.enqueue(ctx, toQueue)
}

func (m *Manager) onUpdateFinished(ctx context.Context) {
	if err := m.q.SetLastLibraryUpdateAt(ctx, sql.NullTime{Time: time.Now().UTC(), Valid: true}); err != nil {
		log.Printf("automation: record library update time: %v", err)
	}
}

func intervalDuration(interval string) time.Duration {
	switch interval {
	case IntervalDaily:
		return 24 * time.Hour
	case IntervalWeekly:
		return 7 * 24 * time.Hour
	}
	return 0
}

// StartLibraryUpdate starts a library update for the given trigger, applying
// pause/interval rules. It returns false when nothing was started (already
// running, disabled, on cooldown, or nothing due).
func (m *Manager) StartLibraryUpdate(ctx context.Context, folderID *int64, trigger Trigger) (bool, error) {
	if m.sy.LibraryUpdateStatus().Running {
		return false, nil
	}
	s, err := m.q.GetAutomationSettings(ctx)
	if err != nil {
		return false, err
	}
	now := time.Now().UTC()
	switch trigger {
	case TriggerAppOpen:
		if s.UpdateOnOpen == 0 {
			return false, nil
		}
		if s.LastLibraryUpdateAt.Valid && now.Sub(s.LastLibraryUpdateAt.Time) < appOpenCooldown {
			return false, nil
		}
	case TriggerScheduled:
		if s.Enabled == 0 {
			return false, nil
		}
	}

	ids, err := m.sy.LibraryUpdateTargets(ctx, folderID)
	if err != nil {
		return false, err
	}
	// An explicit folder refresh updates exactly that folder.
	if !(trigger == TriggerManual && folderID != nil) {
		ids, err = m.filterTargets(ctx, s, ids, trigger, now)
		if err != nil {
			return false, err
		}
	}
	if len(ids) == 0 {
		return false, nil
	}
	return m.startUpdate(ids), nil
}

func (m *Manager) filterTargets(ctx context.Context, s sqlcgen.AutomationSetting, ids []int64, trigger Trigger, now time.Time) ([]int64, error) {
	overrides, err := m.q.ListMediaAutomation(ctx)
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]*sqlcgen.MediaAutomation, len(overrides))
	for i := range overrides {
		byID[overrides[i].MediaID] = &overrides[i]
	}
	var checked map[int64]sql.NullTime
	if trigger == TriggerScheduled && len(ids) > 0 {
		rows, err := m.q.ListUpdateTargetsWithCheckedAt(ctx, ids)
		if err != nil {
			return nil, err
		}
		checked = make(map[int64]sql.NullTime, len(rows))
		for _, r := range rows {
			checked[r.ID] = r.ChaptersCheckedAt
		}
	}

	out := ids[:0:0]
	for _, id := range ids {
		r := RulesFor(s, byID[id])
		if r.PauseUpdates {
			continue
		}
		if trigger != TriggerManual && r.RefreshInterval == IntervalManual {
			continue
		}
		if trigger == TriggerScheduled {
			// Small slack so a tick landing just short of the interval still counts.
			if at := checked[id]; at.Valid && now.Sub(at.Time) < intervalDuration(r.RefreshInterval)-tickEvery {
				continue
			}
		}
		out = append(out, id)
	}
	return out, nil
}
