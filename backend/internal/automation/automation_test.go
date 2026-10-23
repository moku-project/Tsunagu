package automation

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"tsunagu/backend/internal/db"
	"tsunagu/backend/internal/db/sqlcgen"
	"tsunagu/backend/internal/download"
	syncpkg "tsunagu/backend/internal/sync"
)

type fixture struct {
	t    *testing.T
	conn *sql.DB
	q    *sqlcgen.Queries
	m    *Manager
	dir  string
}

// newFixture creates one manga with five chapters in reading order (ids 1-5);
// downloaded lists the chapter ids that have a finished download on disk.
func newFixture(t *testing.T, downloaded ...int64) *fixture {
	t.Helper()
	dir := t.TempDir()
	conn, err := db.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	if _, err := conn.Exec(`
		INSERT INTO repositories (id, index_url) VALUES (1, 'https://repo.example/index.min.json');
		INSERT INTO extensions (id, repository_id, package_name, name, version, content_type, lang, apk_url) VALUES (1, 1, 'ext', 'Ext', '1', 'manga', 'en', '');
		INSERT INTO media (id, extension_id, external_id, content_type, title, added_at) VALUES (1, 1, 'm', 'manga', 'Title', CURRENT_TIMESTAMP);
		INSERT INTO chapters (id, media_id, external_id, source_order) VALUES
			(1, 1, 'c1', 1), (2, 1, 'c2', 2), (3, 1, 'c3', 3), (4, 1, 'c4', 4), (5, 1, 'c5', 5);
	`); err != nil {
		t.Fatal(err)
	}
	q := sqlcgen.New(conn)
	f := &fixture{t: t, conn: conn, q: q, dir: dir}
	for _, id := range downloaded {
		f.addDownload(id)
	}
	dm := download.New(q, nil, dir, dir, "loose")
	sy := syncpkg.New(conn, q, dir, dir)
	f.m = New(q, dm, sy, nil)
	return f
}

func (f *fixture) pagePath(chapterID int64) string {
	return filepath.Join(f.dir, "manga", "Ext", "Title", "Ch"+string(rune('0'+chapterID)), "1.jpg")
}

func (f *fixture) addDownload(chapterID int64) {
	p := f.pagePath(chapterID)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		f.t.Fatal(err)
	}
	f.exec(`INSERT INTO manga_pages (chapter_id, page_number, local_path) VALUES (?, 1, ?)`, chapterID, p)
	f.exec(`INSERT INTO downloads (chapter_id, status, progress) VALUES (?, 'done', 1)`, chapterID)
}

func (f *fixture) exec(query string, args ...any) {
	f.t.Helper()
	if _, err := f.conn.Exec(query, args...); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) markRead(ids ...int64) {
	for _, id := range ids {
		f.exec(`INSERT INTO reading_progress (media_id, chapter_id, progress, completed) VALUES (1, ?, 1, TRUE)`, id)
	}
}

func (f *fixture) state() (downloaded, pending []int64) {
	f.t.Helper()
	rows, err := f.q.ListChapterAutomationState(context.Background(), 1)
	if err != nil {
		f.t.Fatal(err)
	}
	for _, r := range rows {
		if r.IsDownloaded != 0 {
			downloaded = append(downloaded, r.ID)
		}
		if r.IsPending != 0 {
			pending = append(pending, r.ID)
		}
	}
	return downloaded, pending
}

func (f *fixture) fileExists(chapterID int64) bool {
	_, err := os.Stat(f.pagePath(chapterID))
	return err == nil
}

func TestRulesForPrecedence(t *testing.T) {
	global := sqlcgen.AutomationSetting{Enabled: 1, AutoDownload: 0, DownloadAhead: 2, DeleteOnRead: 1, RefreshInterval: IntervalDaily}
	ov := &sqlcgen.MediaAutomation{
		AutoDownload:    sql.NullInt64{Int64: 1, Valid: true},
		PauseUpdates:    sql.NullInt64{Int64: 1, Valid: true},
		RefreshInterval: sql.NullString{String: IntervalWeekly, Valid: true},
	}

	r := RulesFor(global, nil)
	if r.AutoDownload || r.DownloadAhead != 2 || !r.DeleteOnRead || r.PauseUpdates || r.RefreshInterval != IntervalDaily {
		t.Fatalf("global only: %+v", r)
	}
	r = RulesFor(global, ov)
	if !r.AutoDownload || r.DownloadAhead != 2 || !r.PauseUpdates || r.RefreshInterval != IntervalWeekly {
		t.Fatalf("override should win where set, inherit elsewhere: %+v", r)
	}

	enforced := global
	enforced.EnforceGlobal = 1
	if r := RulesFor(enforced, ov); r.AutoDownload || r.PauseUpdates || r.RefreshInterval != IntervalDaily {
		t.Fatalf("enforce global should ignore overrides: %+v", r)
	}

	disabled := global
	disabled.Enabled = 0
	if r := RulesFor(disabled, ov); r.AutoDownload || r.DownloadAhead != 0 || r.DeleteOnRead || !r.PauseUpdates {
		t.Fatalf("disabled should stop downloads/deletions but keep pause: %+v", r)
	}
}

func TestDeleteOnReadAndDownloadAhead(t *testing.T) {
	f := newFixture(t, 1, 2, 3)
	f.exec(`UPDATE automation_settings SET delete_on_read = 1, download_ahead = 2`)
	f.markRead(2)

	f.m.ChaptersRead(context.Background(), 1, []int64{2})

	downloaded, pending := f.state()
	if !slices.Equal(downloaded, []int64{1, 3}) {
		t.Fatalf("downloaded = %v, want [1 3] (chapter 2 deleted after read)", downloaded)
	}
	if f.fileExists(2) {
		t.Fatal("chapter 2 file still on disk")
	}
	// Next two after chapter 2 are 3 (already downloaded) and 4.
	if !slices.Equal(pending, []int64{4}) {
		t.Fatalf("queued = %v, want [4]", pending)
	}
}

func TestDelayedDeleteOnRead(t *testing.T) {
	f := newFixture(t, 1)
	f.exec(`UPDATE automation_settings SET delete_on_read = 1, delete_delay_hours = 24`)
	f.markRead(1)
	ctx := context.Background()

	f.m.ChaptersRead(ctx, 1, []int64{1})
	if !f.fileExists(1) {
		t.Fatal("delayed delete removed the file immediately")
	}
	f.m.runDueDeletions(ctx)
	if !f.fileExists(1) {
		t.Fatal("deleted before the delay elapsed")
	}

	f.exec(`UPDATE scheduled_deletions SET delete_at = ?`, time.Now().UTC().Add(-time.Minute))
	f.m.runDueDeletions(ctx)
	if f.fileExists(1) {
		t.Fatal("due deletion did not run")
	}
	var n int
	if err := f.conn.QueryRow(`SELECT COUNT(*) FROM scheduled_deletions`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("scheduled deletions left: %d err=%v", n, err)
	}
}

func TestUnreadCancelsDelayedDelete(t *testing.T) {
	f := newFixture(t, 1)
	f.exec(`UPDATE automation_settings SET delete_on_read = 1, delete_delay_hours = 1`)
	ctx := context.Background()
	f.m.ChaptersRead(ctx, 1, []int64{1})
	f.m.ChaptersUnread(ctx, []int64{1})
	f.exec(`UPDATE scheduled_deletions SET delete_at = ?`, time.Now().UTC().Add(-time.Minute))
	f.m.runDueDeletions(ctx)
	if !f.fileExists(1) {
		t.Fatal("marking unread should cancel the pending deletion")
	}
}

// The cap drops read chapters first and keeps the next ones to read.
func TestMaxKeepKeepsNextUnread(t *testing.T) {
	f := newFixture(t, 1, 2, 3)
	f.exec(`UPDATE automation_settings SET max_keep_chapters = 1`)
	f.markRead(1)

	f.m.ChaptersRead(context.Background(), 1, []int64{1})

	if downloaded, _ := f.state(); !slices.Equal(downloaded, []int64{2}) {
		t.Fatalf("downloaded = %v, want [2]", downloaded)
	}
}

// Download-ahead must not queue past the cap only for the cap to delete it again.
func TestMaxKeepLimitsDownloadAhead(t *testing.T) {
	f := newFixture(t, 1)
	f.exec(`UPDATE automation_settings SET max_keep_chapters = 2, download_ahead = 4`)
	f.markRead(1)

	f.m.ChaptersRead(context.Background(), 1, []int64{1})

	downloaded, pending := f.state()
	// The read chapter makes room for the two queued ones.
	if len(downloaded) != 0 || !slices.Equal(pending, []int64{2, 3}) {
		t.Fatalf("downloaded=%v pending=%v, want [] and [2 3]", downloaded, pending)
	}
}

func TestDisabledAutomationDoesNothing(t *testing.T) {
	f := newFixture(t, 1)
	f.exec(`UPDATE automation_settings SET enabled = 0, delete_on_read = 1, download_ahead = 3`)
	f.markRead(1)
	f.m.ChaptersRead(context.Background(), 1, []int64{1})
	downloaded, pending := f.state()
	if !slices.Equal(downloaded, []int64{1}) || len(pending) != 0 {
		t.Fatalf("downloaded=%v pending=%v, want untouched", downloaded, pending)
	}
}

func TestAutoDownloadNewChapters(t *testing.T) {
	ctx := context.Background()

	f := newFixture(t)
	f.m.onSeriesChecked(ctx, 1, []int64{4, 5})
	if _, pending := f.state(); len(pending) != 0 {
		t.Fatalf("queued %v with auto-download off", pending)
	}
	var checked sql.NullTime
	if err := f.conn.QueryRow(`SELECT chapters_checked_at FROM media WHERE id = 1`).Scan(&checked); err != nil || !checked.Valid {
		t.Fatalf("chapters_checked_at not recorded: %v %v", checked, err)
	}

	f.exec(`UPDATE automation_settings SET auto_download = 1`)
	f.m.onSeriesChecked(ctx, 1, []int64{5, 4})
	if _, pending := f.state(); !slices.Equal(pending, []int64{4, 5}) {
		t.Fatalf("queued %v, want [4 5]", pending)
	}

	// A folder flagged for auto-download works without the global switch.
	g := newFixture(t)
	g.exec(`INSERT INTO folders (id, name, kind, include_in_download) VALUES (90, 'Reading Now', 'custom', 1)`)
	g.exec(`INSERT INTO media_folders (media_id, folder_id) VALUES (1, 90)`)
	g.m.onSeriesChecked(ctx, 1, []int64{5})
	if _, pending := g.state(); !slices.Equal(pending, []int64{5}) {
		t.Fatalf("folder auto-download queued %v, want [5]", pending)
	}
}

func TestFilterTargets(t *testing.T) {
	f := newFixture(t)
	f.exec(`INSERT INTO media (id, extension_id, external_id, content_type, title, added_at) VALUES
		(2, 1, 'paused', 'manga', 'Paused', CURRENT_TIMESTAMP),
		(3, 1, 'manual', 'manga', 'Manual', CURRENT_TIMESTAMP),
		(4, 1, 'weekly', 'manga', 'Weekly', CURRENT_TIMESTAMP)`)
	f.exec(`INSERT INTO media_automation (media_id, pause_updates) VALUES (2, 1)`)
	f.exec(`INSERT INTO media_automation (media_id, refresh_interval) VALUES (3, 'manual'), (4, 'weekly')`)
	now := time.Now().UTC()
	f.exec(`UPDATE media SET chapters_checked_at = ? WHERE id IN (1, 4)`, now.Add(-30*time.Hour))
	ctx := context.Background()
	s, err := f.q.GetAutomationSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	all := []int64{1, 2, 3, 4}

	got, _ := f.m.filterTargets(ctx, s, all, TriggerManual, now)
	if !slices.Equal(got, []int64{1, 3, 4}) {
		t.Fatalf("manual = %v, want [1 3 4] (paused skipped)", got)
	}
	got, _ = f.m.filterTargets(ctx, s, all, TriggerAppOpen, now)
	if !slices.Equal(got, []int64{1, 4}) {
		t.Fatalf("app open = %v, want [1 4] (paused and manual skipped)", got)
	}
	// Daily series checked 30h ago is due; weekly series checked 30h ago isn't.
	got, _ = f.m.filterTargets(ctx, s, all, TriggerScheduled, now)
	if !slices.Equal(got, []int64{1}) {
		t.Fatalf("scheduled = %v, want [1]", got)
	}
}

// stubUpdates replaces the real sync with a recorder and returns the IDs of
// every update started.
func (f *fixture) stubUpdates() *[][]int64 {
	var started [][]int64
	f.m.startUpdate = func(ids []int64) bool {
		started = append(started, slices.Clone(ids))
		return true
	}
	return &started
}

func TestAppOpenUpdate(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	f.exec(`INSERT INTO media (id, extension_id, external_id, content_type, title, added_at) VALUES
		(2, 1, 'paused', 'manga', 'Paused', CURRENT_TIMESTAMP)`)
	f.exec(`INSERT INTO media_automation (media_id, pause_updates) VALUES (2, 1)`)
	started := f.stubUpdates()

	ok, err := f.m.StartLibraryUpdate(ctx, nil, TriggerAppOpen)
	if err != nil || !ok {
		t.Fatalf("app open on a fresh server should update: ok=%v err=%v", ok, err)
	}
	if len(*started) != 1 || !slices.Equal((*started)[0], []int64{1}) {
		t.Fatalf("updated %v, want [[1]] (paused series skipped)", *started)
	}

	// Finishing the update starts the cooldown, so a quick reopen doesn't refetch everything.
	f.m.onUpdateFinished(ctx)
	if ok, _ := f.m.StartLibraryUpdate(ctx, nil, TriggerAppOpen); ok {
		t.Fatal("reopening within the cooldown started another update")
	}
	// A manual update ignores the cooldown.
	if ok, _ := f.m.StartLibraryUpdate(ctx, nil, TriggerManual); !ok {
		t.Fatal("manual update refused during the app-open cooldown")
	}

	f.exec(`UPDATE automation_settings SET last_library_update_at = ?`, time.Now().UTC().Add(-time.Hour))
	if ok, _ := f.m.StartLibraryUpdate(ctx, nil, TriggerAppOpen); !ok {
		t.Fatal("app open after the cooldown didn't update")
	}

	f.exec(`UPDATE automation_settings SET update_on_open = 0, last_library_update_at = NULL`)
	if ok, _ := f.m.StartLibraryUpdate(ctx, nil, TriggerAppOpen); ok {
		t.Fatal("update on open is off but an update started")
	}
	if len(*started) != 3 {
		t.Fatalf("started %d updates, want 3", len(*started))
	}
}

func TestAppOpenUpdateNothingDue(t *testing.T) {
	f := newFixture(t)
	f.exec(`INSERT INTO media_automation (media_id, refresh_interval) VALUES (1, 'manual')`)
	started := f.stubUpdates()
	if ok, _ := f.m.StartLibraryUpdate(context.Background(), nil, TriggerAppOpen); ok || len(*started) != 0 {
		t.Fatalf("only a manual-refresh series: ok=%v started=%v", ok, *started)
	}
}

func TestScheduledUpdate(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	started := f.stubUpdates()

	// Never checked → due.
	if ok, _ := f.m.StartLibraryUpdate(ctx, nil, TriggerScheduled); !ok {
		t.Fatal("never-checked series wasn't updated on schedule")
	}
	f.m.onSeriesChecked(ctx, 1, nil)
	if ok, _ := f.m.StartLibraryUpdate(ctx, nil, TriggerScheduled); ok {
		t.Fatal("series checked just now was updated again on schedule")
	}

	f.exec(`UPDATE media SET chapters_checked_at = ?`, time.Now().UTC().Add(-25*time.Hour))
	f.exec(`UPDATE automation_settings SET enabled = 0`)
	if ok, _ := f.m.StartLibraryUpdate(ctx, nil, TriggerScheduled); ok {
		t.Fatal("scheduled update ran with automation disabled")
	}
	f.exec(`UPDATE automation_settings SET enabled = 1`)
	if ok, _ := f.m.StartLibraryUpdate(ctx, nil, TriggerScheduled); !ok {
		t.Fatal("daily series checked 25h ago wasn't updated")
	}
	if len(*started) != 2 {
		t.Fatalf("started %d updates, want 2", len(*started))
	}
}

func TestManualFolderUpdateIgnoresPause(t *testing.T) {
	f := newFixture(t)
	f.exec(`INSERT INTO folders (id, name, kind) VALUES (90, 'Shelf', 'custom')`)
	f.exec(`INSERT INTO media_folders (media_id, folder_id) VALUES (1, 90)`)
	f.exec(`INSERT INTO media_automation (media_id, pause_updates) VALUES (1, 1)`)
	started := f.stubUpdates()

	if ok, _ := f.m.StartLibraryUpdate(context.Background(), nil, TriggerManual); ok {
		t.Fatal("library-wide manual update included a paused series")
	}
	folder := int64(90)
	if ok, _ := f.m.StartLibraryUpdate(context.Background(), &folder, TriggerManual); !ok {
		t.Fatal("refreshing a folder explicitly should update its paused series too")
	}
	if !slices.Equal((*started)[0], []int64{1}) {
		t.Fatalf("updated %v, want [1]", *started)
	}
}

func TestSeriesOverrideAndEnforceGlobal(t *testing.T) {
	ctx := context.Background()

	// The series opts out of the global delete-after-read.
	f := newFixture(t, 1)
	f.exec(`UPDATE automation_settings SET delete_on_read = 1`)
	f.exec(`INSERT INTO media_automation (media_id, delete_on_read) VALUES (1, 0)`)
	f.m.ChaptersRead(ctx, 1, []int64{1})
	if !f.fileExists(1) {
		t.Fatal("series override should keep the download")
	}

	// Enforcing global rules ignores that opt-out.
	f.exec(`UPDATE automation_settings SET enforce_global = 1`)
	f.m.ChaptersRead(ctx, 1, []int64{1})
	if f.fileExists(1) {
		t.Fatal("enforce global should delete despite the series override")
	}

	// A series can turn on a rule that's off globally.
	g := newFixture(t, 1)
	g.exec(`INSERT INTO media_automation (media_id, delete_on_read) VALUES (1, 1)`)
	g.m.ChaptersRead(ctx, 1, []int64{1})
	if g.fileExists(1) {
		t.Fatal("series override should delete after read")
	}
}

func TestDownloadAheadSkipsReadAndQueued(t *testing.T) {
	f := newFixture(t)
	f.exec(`UPDATE automation_settings SET download_ahead = 3`)
	f.markRead(1, 3)
	f.exec(`INSERT INTO downloads (chapter_id, status, progress) VALUES (4, 'queued', 0)`)

	f.m.ChaptersRead(context.Background(), 1, []int64{1})

	// Next three after 1 are 2, 3 (read) and 4 (already queued).
	if _, pending := f.state(); !slices.Equal(pending, []int64{2, 4}) {
		t.Fatalf("pending = %v, want [2 4]", pending)
	}
	var n int
	if err := f.conn.QueryRow(`SELECT COUNT(*) FROM downloads WHERE chapter_id = 4`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("chapter 4 has %d download rows, want 1 (err=%v)", n, err)
	}
}

func TestAutoDownloadSkipsReadAndDownloaded(t *testing.T) {
	f := newFixture(t, 4)
	f.exec(`UPDATE automation_settings SET auto_download = 1`)
	f.markRead(3)
	f.m.onSeriesChecked(context.Background(), 1, []int64{3, 4, 5})
	if _, pending := f.state(); !slices.Equal(pending, []int64{5}) {
		t.Fatalf("queued %v, want [5]", pending)
	}
}

func TestAutoDownloadOffWhenAutomationDisabled(t *testing.T) {
	f := newFixture(t)
	f.exec(`UPDATE automation_settings SET auto_download = 1, enabled = 0`)
	f.m.onSeriesChecked(context.Background(), 1, []int64{5})
	if _, pending := f.state(); len(pending) != 0 {
		t.Fatalf("queued %v with automation disabled", pending)
	}
}

// A series whose extension was uninstalled must still have deletable downloads.
func TestDeleteDownloadsOfOrphanedSeries(t *testing.T) {
	f := newFixture(t, 1, 2)
	f.exec(`UPDATE media SET extension_id = NULL WHERE id = 1`)
	var ext sql.NullInt64
	if err := f.conn.QueryRow(`SELECT extension_id FROM media WHERE id = 1`).Scan(&ext); err != nil || ext.Valid {
		t.Fatalf("fixture series should have no extension: %v %v", ext, err)
	}
	if err := f.m.dm.DeleteChapters(context.Background(), []int64{1, 2}); err != nil {
		t.Fatal(err)
	}
	if f.fileExists(1) || f.fileExists(2) {
		t.Fatal("files left on disk")
	}
	if downloaded, _ := f.state(); len(downloaded) != 0 {
		t.Fatalf("still downloaded: %v", downloaded)
	}
}

// Local series have no source to sync; including them made every scheduled tick
// fail them and respawn the sandbox.
func TestUpdateTargetsSkipLocalMedia(t *testing.T) {
	f := newFixture(t)
	f.exec(`INSERT INTO media (id, external_id, content_type, title, added_at) VALUES (6, 'local', 'manga', 'Local', CURRENT_TIMESTAMP)`)
	ids, err := f.q.ListUpdateTargetMediaIDs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(ids, []int64{1}) {
		t.Fatalf("targets = %v, want [1]", ids)
	}
}

// A chapter that is queued or already downloaded must not get a second job.
func TestEnqueueDownloadSkipsActiveChapters(t *testing.T) {
	f := newFixture(t, 1)
	ctx := context.Background()
	if _, err := f.q.EnqueueDownload(ctx, 1); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("downloaded chapter re-queued: %v", err)
	}
	if _, err := f.q.EnqueueDownload(ctx, 2); err != nil {
		t.Fatal(err)
	}
	if _, err := f.q.EnqueueDownload(ctx, 2); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("queued chapter queued twice: %v", err)
	}
}
