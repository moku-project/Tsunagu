package mokusync

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// fakeResolver stands in for fetching from a source: it adds the title and its chapters to the node.
type fakeResolver struct {
	n     *node
	fail  bool
	calls []string
}

func (r *fakeResolver) Resolve(ctx context.Context, pkg, ext string) error {
	r.calls = append(r.calls, pkg+" "+ext)
	if r.fail {
		return errors.New("extension not installed")
	}
	m, err := strconv.Atoi(strings.TrimPrefix(ext, "/title/"))
	if err != nil {
		return err
	}
	var exists bool
	if err := r.n.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM media WHERE external_id = ?)`, ext).Scan(&exists); err != nil {
		return err
	}
	if exists {
		for c := 0; c < testChapters; c++ {
			r.n.exec(`INSERT OR IGNORE INTO chapters (media_id, external_id, number) VALUES (`+mediaID+`, ?, ?)`, ext, chapterExt(c), c+1)
		}
		return nil
	}
	r.n.addTitle(m)
	return nil
}

func (n *node) pendingCount() int {
	var c int
	if err := n.db.QueryRowContext(n.ctx, `SELECT COUNT(*) FROM sync_pending`).Scan(&c); err != nil {
		n.t.Fatal(err)
	}
	return c
}

func TestPendingValuesApplyWhenTheTitleArrives(t *testing.T) {
	a := newNode(t, "a", nil)
	b := newNode(t, "b", []int{0, 1})
	a.setInLibrary(4, true)
	a.setRead(4, 2, true)
	id := a.createFolder("Later")
	a.setInFolder(id, 4, true)
	a.syncWith(b)
	if b.pendingCount() == 0 {
		t.Fatal("values for a title b doesn't have weren't kept as pending")
	}
	if got, _ := b.e.PendingTitles(b.ctx); got != 1 {
		t.Fatalf("pending library titles = %d, want 1", got)
	}

	b.addTitle(4)
	if _, err := b.e.ProjectPending(b.ctx); err != nil {
		t.Fatal(err)
	}
	if !b.inLibrary(4) {
		t.Fatal("title 4 not added once it existed")
	}
	if read, _, _ := b.chapterRow(4, 2); !read {
		t.Fatal("read state not applied once the chapter existed")
	}
	if got := b.strings(`SELECT f.sync_id FROM media_folders mf JOIN folders f ON f.id = mf.folder_id`); len(got) != 1 || got[0] != id {
		t.Fatalf("folder membership = %v, want [%s]", got, id)
	}
	if got := b.pendingCount(); got != 0 {
		t.Fatalf("%d values still pending", got)
	}
	if got := b.capture(); got != 0 {
		t.Fatalf("applying pending values made %d local ops", got)
	}
	// b only has titles 0, 1 and 4, so compare those.
	var shared []string
	for _, line := range a.libraryState() {
		if !strings.Contains(line, mediaExt(2)) && !strings.Contains(line, mediaExt(3)) && !strings.Contains(line, mediaExt(5)) {
			shared = append(shared, line)
		}
	}
	if d := diff(shared, b.libraryState()); d != "" {
		t.Fatalf("libraries differ:\n%s", d)
	}
}

func TestProjectPendingSkipsWhenNothingNewArrived(t *testing.T) {
	a := newNode(t, "a", nil)
	b := newNode(t, "b", []int{0})
	a.setInLibrary(3, true)
	a.syncWith(b)
	if _, err := b.e.ProjectPending(b.ctx); err != nil {
		t.Fatal(err)
	}
	// Nothing was inserted, so the pass shouldn't even look at the pending values.
	b.exec(`UPDATE sync_pending SET field = 'poisoned' WHERE kind = 'library'`)
	if n, err := b.e.ProjectPending(b.ctx); err != nil || n != 0 {
		t.Fatalf("pass ran with no new rows: applied %d, err %v", n, err)
	}
}

func TestResolverFetchesOnlyMissingLibraryTitles(t *testing.T) {
	a := newNode(t, "a", nil)
	b := newNode(t, "b", []int{0, 1})
	b.exec(`INSERT INTO media (extension_id, external_id, content_type, title) VALUES (1, ?, 'manga', 'Title 3')`, mediaExt(3))
	a.setInLibrary(4, true) // b lacks the title
	a.setRead(4, 2, true)
	a.setInLibrary(3, true) // b has the title but not its chapters
	a.setRead(3, 1, true)
	a.setRead(5, 0, true)   // history only, not in the library
	a.setInLibrary(0, true) // b has it already
	a.syncWith(b)

	r := &fakeResolver{n: b}
	got, err := b.e.ResolveMissing(b.ctx, r)
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(r.calls)
	want := []string{"example.source " + mediaExt(3), "example.source " + mediaExt(4)}
	if got != 2 || strings.Join(r.calls, ",") != strings.Join(want, ",") {
		t.Fatalf("resolved %d via %v, want %v", got, r.calls, want)
	}
	for _, m := range []int{3, 4} {
		if !b.inLibrary(m) {
			t.Fatalf("title %d not in b's library after fetching", m)
		}
	}
	if read, _, _ := b.chapterRow(4, 2); !read {
		t.Fatal("title 4 chapter 2 not read after fetching")
	}
	if read, _, _ := b.chapterRow(3, 1); !read {
		t.Fatal("title 3 chapter 1 not read after fetching its chapters")
	}
}

func TestResolverBacksOffAfterAFailure(t *testing.T) {
	a := newNode(t, "a", nil)
	b := newNode(t, "b", []int{0})
	a.setInLibrary(2, true)
	a.syncWith(b)
	r := &fakeResolver{n: b, fail: true}
	for i := 0; i < 3; i++ {
		if _, err := b.e.ResolveMissing(b.ctx, r); err != nil {
			t.Fatal(err)
		}
	}
	if len(r.calls) != 1 {
		t.Fatalf("failing title fetched %d times, want once until the backoff passes", len(r.calls))
	}
}
