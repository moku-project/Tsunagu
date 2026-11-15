package mokusync

import (
	"context"
	"path/filepath"
	"testing"

	"tsunagu/backend/internal/db"
)

func TestCaptureTurnsLocalWritesIntoOps(t *testing.T) {
	a := newNode(t, "a", nil)
	if got := a.capture(); got != 0 {
		t.Fatalf("seeding a library that nobody has touched made %d ops", got)
	}
	a.setRead(0, 0, true)
	a.setBookmark(0, 1, true)
	a.setInLibrary(0, true)
	if got := a.capture(); got != 4 {
		t.Fatalf("read (completed + position), bookmark and library add made %d ops, want 4", got)
	}
	if got := a.capture(); got != 0 {
		t.Fatalf("second capture made %d ops", got)
	}
	// Writing the value a register already holds isn't a change.
	a.setRead(0, 0, false)
	a.setRead(0, 0, true)
	if got := a.capture(); got != 0 {
		t.Fatalf("toggling back and forth before a capture made %d ops", got)
	}
}

func TestLibraryFromBeforeSyncIsSentOnFirstPairing(t *testing.T) {
	// Build a library with sync's tables in place but before the engine has ever started.
	conn, err := db.Open(filepath.Join(t.TempDir(), "old.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	old := &node{t: t, ctx: context.Background(), db: conn}
	old.exec(`INSERT INTO repositories (index_url, name) VALUES ('https://example.com/index.json', 'Example')`)
	old.exec(`INSERT INTO extensions (repository_id, package_name, name, version, content_type, lang, apk_url, installed)
		VALUES (1, 'example.source', 'Example', '1', 'manga', 'en', 'x.apk', 1)`)
	for m := 0; m < testTitles; m++ {
		old.addTitle(m)
	}
	old.setInLibrary(2, true)
	old.setRead(2, 1, true)
	old.setBookmark(2, 3, true)
	id := old.createFolder("Shelf")
	old.setInFolder(id, 2, true)
	// Drop what the triggers saw, so only the start-up snapshot can carry this library.
	old.exec(`DELETE FROM sync_dirty`)

	e, err := Open(old.ctx, conn)
	if err != nil {
		t.Fatal(err)
	}
	old.e = e
	old.wall = 1_700_000_000_000
	e.wall = func() int64 { return old.wall }

	fresh := newNode(t, "fresh", nil)
	old.syncWith(fresh)
	if d := diff(old.libraryState(), fresh.libraryState()); d != "" {
		t.Fatalf("libraries differ after first pairing:\n%s", d)
	}
}

func TestSyncMatchesTitlesBySourceNotLocalID(t *testing.T) {
	a := newNode(t, "a", []int{0, 1, 2, 3, 4, 5})
	b := newNode(t, "b", []int{5, 4, 3, 2, 1, 0})
	a.setInLibrary(2, true)
	a.setRead(2, 3, true)
	a.setPosition(2, 4, 0.5)
	a.setBookmark(2, 1, true)
	a.syncWith(b)

	if !b.inLibrary(2) {
		t.Fatal("title 2 missing from b's library")
	}
	if read, _, _ := b.chapterRow(2, 3); !read {
		t.Fatal("chapter 3 not read on b")
	}
	if _, p, _ := b.chapterRow(2, 4); p != 0.5 {
		t.Fatalf("chapter 4 position on b = %v, want 0.5", p)
	}
	if _, _, bm := b.chapterRow(2, 1); !bm {
		t.Fatal("chapter 1 bookmark missing on b")
	}
	if d := diff(a.libraryState(), b.libraryState()); d != "" {
		t.Fatalf("libraries differ:\n%s", d)
	}
}

func TestLatestPositionWinsWhicheverWayItTravels(t *testing.T) {
	for _, phoneFirst := range []bool{true, false} {
		phone := newNode(t, "phone", nil)
		desktop := newNode(t, "desktop", nil)
		// Read on the train with the phone offline, then on the desktop later.
		phone.wall += 1000
		phone.setPosition(0, 2, 0.8)
		phone.capture()
		desktop.wall += 5000
		desktop.setPosition(0, 2, 0.3)
		desktop.capture()

		if phoneFirst {
			phone.syncWith(desktop)
		} else {
			desktop.syncWith(phone)
		}
		for _, n := range []*node{phone, desktop} {
			if _, p, _ := n.chapterRow(0, 2); p != 0.3 {
				t.Fatalf("phoneFirst=%v: %s resumes at %v, want the later read at 0.3", phoneFirst, n.e.name, p)
			}
		}
	}
}

func TestRemoteReadsKeepWhenTheyHappened(t *testing.T) {
	a := newNode(t, "a", nil)
	b := newNode(t, "b", nil)
	a.wall = 1_700_000_000_000 // 2023-11-14 22:13:20 UTC, long before the sync
	a.setInLibrary(0, true)
	a.setRead(0, 1, true)
	a.capture()
	a.syncWith(b)
	want := "2023-11-14 22:13:20"
	if got := b.strings(`SELECT strftime('%Y-%m-%d %H:%M:%S', updated_at) FROM reading_progress`); len(got) != 1 || got[0] != want {
		t.Fatalf("b's history time = %v, want %s", got, want)
	}
	if got := b.strings(`SELECT strftime('%Y-%m-%d %H:%M:%S', added_at) FROM media WHERE added_at IS NOT NULL`); len(got) != 1 || got[0] != want {
		t.Fatalf("b's date added = %v, want %s", got, want)
	}
}

func TestUnreadAfterReadWinsAndReadAfterUnreadWins(t *testing.T) {
	a := newNode(t, "a", nil)
	b := newNode(t, "b", nil)
	a.setRead(1, 1, true)
	a.syncWith(b)

	b.wall += 100
	b.setRead(1, 1, false)
	b.capture()
	b.syncWith(a)
	if read, _, _ := a.chapterRow(1, 1); read {
		t.Fatal("a later unread on b did not reach a")
	}

	a.wall += 200
	a.setRead(1, 1, true)
	a.capture()
	a.syncWith(b)
	if read, _, _ := b.chapterRow(1, 1); !read {
		t.Fatal("a later read on a did not reach b")
	}
}

func TestReAddSurvivesAConcurrentRemove(t *testing.T) {
	a := newNode(t, "a", nil)
	b := newNode(t, "b", nil)
	a.setInLibrary(3, true)
	a.syncWith(b)

	// a removes the title. b, not having heard, removes it and adds it back.
	a.setInLibrary(3, false)
	a.capture()
	b.setInLibrary(3, false)
	b.capture()
	b.setInLibrary(3, true)
	b.capture()

	a.syncWith(b)
	if !a.inLibrary(3) || !b.inLibrary(3) {
		t.Fatalf("re-add lost: a=%v b=%v", a.inLibrary(3), b.inLibrary(3))
	}

	// A remove that has seen every add still removes it.
	a.setInLibrary(3, false)
	a.capture()
	a.syncWith(b)
	if a.inLibrary(3) || b.inLibrary(3) {
		t.Fatalf("remove lost: a=%v b=%v", a.inLibrary(3), b.inLibrary(3))
	}
}

func TestFoldersSyncByIDNotName(t *testing.T) {
	a := newNode(t, "a", nil)
	b := newNode(t, "b", nil)
	a.setInLibrary(0, true)
	id := a.createFolder("Reading")
	a.setInFolder(id, 0, true)
	a.capture()
	a.syncWith(b)
	if got := b.folderIDs(); len(got) != 1 || got[0] != id {
		t.Fatalf("b folders = %v, want [%s]", got, id)
	}
	if d := diff(a.libraryState(), b.libraryState()); d != "" {
		t.Fatalf("libraries differ after folder create:\n%s", d)
	}

	// Renamed differently on both before they talk: the later rename wins on both.
	a.wall += 10
	a.renameFolder(id, "Reading now")
	a.capture()
	b.wall += 20
	b.renameFolder(id, "Currently reading")
	b.capture()
	a.syncWith(b)
	for _, n := range []*node{a, b} {
		if got := n.strings(`SELECT name FROM folders WHERE sync_id = ?`, id); len(got) != 1 || got[0] != "Currently reading" {
			t.Fatalf("%s folder name = %v", n.e.name, got)
		}
	}

	a.deleteFolder(id)
	a.capture()
	a.syncWith(b)
	if got := b.folderIDs(); len(got) != 0 {
		t.Fatalf("deleted folder still on b: %v", got)
	}
	if !b.inLibrary(0) {
		t.Fatal("deleting a folder removed its title from the library")
	}
}

func TestValuesWaitForATitleThisDeviceDoesNotHaveYet(t *testing.T) {
	a := newNode(t, "a", nil)
	b := newNode(t, "b", []int{0, 1})
	a.setInLibrary(4, true)
	a.setRead(4, 2, true)
	a.syncWith(b)

	// b has no row for title 4 until its source is fetched there.
	b.addTitle(4)
	if b.inLibrary(4) {
		t.Fatal("projection ran before ProjectAll")
	}
	if err := b.e.ProjectAll(b.ctx); err != nil {
		t.Fatal(err)
	}
	if !b.inLibrary(4) {
		t.Fatal("title 4 not added once it existed")
	}
	if read, _, _ := b.chapterRow(4, 2); !read {
		t.Fatal("chapter read state not applied once the chapter existed")
	}
	if got := b.capture(); got != 0 {
		t.Fatalf("projecting remote values made %d local ops", got)
	}
}

func TestApplyingRemoteValuesDoesNotEchoBack(t *testing.T) {
	a := newNode(t, "a", nil)
	b := newNode(t, "b", nil)
	a.setInLibrary(1, true)
	id := a.createFolder("Later")
	a.setInFolder(id, 1, true)
	a.setEpisodePosition(1, 0, 600, 1440)
	a.capture()
	a.syncWith(b)
	if got := b.capture(); got != 0 {
		t.Fatalf("b turned %d applied changes into its own ops", got)
	}
	if d := diff(a.crdtState(), b.crdtState()); d != "" {
		t.Fatalf("sync state differs:\n%s", d)
	}
}

func TestApplyIsIdempotent(t *testing.T) {
	a := newNode(t, "a", nil)
	b := newNode(t, "b", nil)
	a.setInLibrary(2, true)
	a.setRead(2, 0, true)
	a.capture()
	ops, _, err := opsSince(a.ctx, a.db, VersionVector{}, batchLimit)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := b.e.Apply(b.ctx, ops); err != nil {
			t.Fatal(err)
		}
	}
	if d := diff(a.crdtState(), b.crdtState()); d != "" {
		t.Fatalf("state after repeated apply differs:\n%s", d)
	}
}

func TestSupersededOpsArePruned(t *testing.T) {
	a := newNode(t, "a", nil)
	for i := 1; i <= 50; i++ {
		a.wall++
		a.setPosition(0, 0, float64(i)/100)
		a.capture()
	}
	// One completed and one position op survive; the 49 older positions can never win again.
	if got := a.opCount(); got != 2 {
		t.Fatalf("op log holds %d ops, want 2", got)
	}
	a.setInLibrary(0, true)
	a.capture()
	a.setInLibrary(0, false)
	a.capture()
	// The add is gone; the remove stays as the tombstone.
	if got := a.opCount(); got != 3 {
		t.Fatalf("op log holds %d ops after add and remove, want 3", got)
	}
}

func TestSlowClockStillOrdersAfterWhatItHasSeen(t *testing.T) {
	a := newNode(t, "a", nil)
	b := newNode(t, "b", nil)
	a.wall += 3_600_000 // a's clock runs an hour ahead
	a.setPosition(0, 0, 0.9)
	a.capture()
	a.syncWith(b)

	// b reads afterwards with its own, slower clock. Having seen a's edit, its stamp still comes later.
	b.wall += 1000
	b.setPosition(0, 0, 0.1)
	b.capture()
	b.syncWith(a)
	if _, p, _ := a.chapterRow(0, 0); p != 0.1 {
		t.Fatalf("a resumes at %v, want b's later read at 0.1", p)
	}
}

func TestSmallBatchesStillConverge(t *testing.T) {
	defer func(l int) { batchLimit = l }(batchLimit)
	batchLimit = 3
	a := newNode(t, "a", nil)
	b := newNode(t, "b", nil)
	for m := 0; m < testTitles; m++ {
		a.setInLibrary(m, true)
		for c := 0; c < testChapters; c++ {
			a.setRead(m, c, true)
		}
	}
	a.capture()
	b.setBookmark(5, 4, true)
	b.capture()
	a.syncWith(b)
	if d := diff(a.libraryState(), b.libraryState()); d != "" {
		t.Fatalf("libraries differ:\n%s", d)
	}
}
