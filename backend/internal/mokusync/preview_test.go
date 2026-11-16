package mokusync

import (
	"reflect"
	"testing"
)

func TestPreviewShowsWhatPairingMerges(t *testing.T) {
	hub := newNode(t, "desktop", nil)
	phone := newNode(t, "phone", nil)
	for _, m := range []int{0, 1, 2} {
		hub.setInLibrary(m, true)
	}
	hub.setRead(0, 0, true)
	hub.setRead(0, 1, true)
	hub.createFolder("Reading")
	hub.createFolder("Done")
	phone.setInLibrary(2, true)
	phone.setInLibrary(3, true)
	phone.createFolder(" reading")
	srv := serve(t, hub)
	code, _, _ := hub.e.PairingCode(hub.ctx)

	if _, err := phone.e.Preview(phone.ctx, srv.Client(), srv.URL, "WRONGONE"); err == nil {
		t.Fatal("previewed with a wrong code")
	}
	got, err := phone.e.Preview(phone.ctx, srv.Client(), srv.URL, code)
	if err != nil {
		t.Fatal(err)
	}
	want := PairPreview{Node: hub.e.Node(), Name: "desktop", Titles: 3, NewTitles: 2, SentTitles: 1, Folders: 2,
		ReadChapters: 2, MergingFolders: []string{"Reading"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("preview\n got %+v\nwant %+v", got, want)
	}

	// Pairing with a device other than the previewed one fails and leaves the code usable.
	if _, err := phone.e.ConnectTo(phone.ctx, srv.Client(), srv.URL, code, "someone-else"); err == nil {
		t.Fatal("paired with a device that wasn't previewed")
	}
	if _, err := phone.e.ConnectTo(phone.ctx, srv.Client(), srv.URL, code, got.Node); err != nil {
		t.Fatalf("pairing after the preview: %v", err)
	}
}
