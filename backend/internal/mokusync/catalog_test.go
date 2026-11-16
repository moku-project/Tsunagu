package mokusync

import (
	"reflect"
	"testing"
)

// bare is a device with no extensions or repositories at all.
func bare(t *testing.T, name string) *node {
	n := newNode(t, name, []int{})
	n.exec(`DELETE FROM extensions`)
	n.exec(`DELETE FROM repositories`)
	return n
}

func TestMissingExtensionsSayWhatToInstall(t *testing.T) {
	a := newNode(t, "a", nil)
	for _, m := range []int{1, 2, 3} {
		a.setInLibrary(m, true)
	}
	a.setRead(2, 0, true)
	b := bare(t, "b")
	a.syncWith(b)

	got, err := b.e.MissingExtensions(b.ctx)
	if err != nil {
		t.Fatal(err)
	}
	want := []MissingExtension{{
		Package: "example.source", Name: "Example", Lang: "en", ContentType: "manga",
		Repository: "https://example.com/index.json", Titles: 3, Examples: []string{"Title 1", "Title 2", "Title 3"},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("missing extensions\n got %+v\nwant %+v", got, want)
	}
	if repo, err := b.e.ExtensionRepository(b.ctx, "example.source"); err != nil || repo != "https://example.com/index.json" {
		t.Fatalf("repository %q, %v", repo, err)
	}
	if repo, _ := b.e.ExtensionRepository(b.ctx, "example"); repo != "" {
		t.Fatalf("a package prefix matched another package: %q", repo)
	}

	// Fetching can only fail without the extension, so nothing is tried until it's installed.
	r := &fakeResolver{n: b}
	if _, err := b.e.ResolveMissing(b.ctx, r); err != nil || len(r.calls) != 0 {
		t.Fatalf("tried %v (%v) with the extension missing", r.calls, err)
	}
	b.exec(`INSERT INTO repositories (index_url, name) VALUES ('https://example.com/index.json', 'Example')`)
	b.exec(`INSERT INTO extensions (id, repository_id, package_name, name, version, content_type, lang, apk_url, installed)
		VALUES (1, (SELECT id FROM repositories), 'example.source', 'Example', '1', 'manga', 'en', 'x.apk', 0)`)
	got, _ = b.e.MissingExtensions(b.ctx)
	if len(got) != 1 || !got[0].Available {
		t.Fatalf("an extension the device lists but hasn't installed: %+v", got)
	}
	b.exec(`UPDATE extensions SET installed = 1`)
	b.e.ExtensionsChanged()
	if n, err := b.e.ResolveMissing(b.ctx, r); err != nil || n != 3 {
		t.Fatalf("resolved %d (%v) after installing, want 3", n, err)
	}
	if got, _ := b.e.MissingExtensions(b.ctx); len(got) != 0 {
		t.Fatalf("still missing after install: %+v", got)
	}
	if read, _, _ := b.chapterRow(2, 0); !read || !b.inLibrary(1) {
		t.Fatal("waiting values didn't apply once the titles arrived")
	}
}

func TestTitleInfoConverges(t *testing.T) {
	a := newNode(t, "a", nil)
	b := newNode(t, "b", nil)
	a.setInLibrary(1, true)
	b.wall++
	b.setInLibrary(1, true)
	a.syncWith(b)
	if d := diff(a.crdtState(), b.crdtState()); d != "" {
		t.Fatalf("sync state differs:\n%s", d)
	}
	info, ok, err := readTitleInfo(b.ctx, b.db, mediaKey(1))
	if err != nil || !ok || info.Title != "Title 1" || info.Extension.Name != "Example" {
		t.Fatalf("title info %+v %v %v", info, ok, err)
	}
	if got := a.capture(); got != 0 {
		t.Fatalf("re-capturing unchanged title info made %d ops", got)
	}
}
