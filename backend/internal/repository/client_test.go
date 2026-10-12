package repository

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
)

func bytesField(b []byte, num protowire.Number, v []byte) []byte {
	b = protowire.AppendTag(b, num, protowire.BytesType)
	return protowire.AppendBytes(b, v)
}

// novelIndexPB is a one-extension index_v2 shaped like NovelSourcery's.
func novelIndexPB() []byte {
	var res []byte
	res = bytesField(res, 501, []byte("https://example.com/boxnovel.jar"))
	var src []byte
	src = bytesField(src, 3, []byte("en"))
	var ext []byte
	ext = bytesField(ext, 1, []byte("BoxNovel"))
	ext = bytesField(ext, 2, []byte("eu.kanade.tachiyomi.novelextension.en.boxnovel"))
	ext = bytesField(ext, 3, res)
	ext = bytesField(ext, 6, []byte("1.6.12"))
	ext = bytesField(ext, 8, src)
	list := bytesField(nil, 1, ext)
	idx := bytesField(nil, 1, []byte("NovelSourcery"))
	return bytesField(idx, 101, list)
}

const decoyIndex = `[{"name":"Outdated App","pkg":"eu.kanade.tachiyomi.extension.all.keiyoushi","apk":"x.apk","lang":"all","version":"1.4.1"}]`

func TestFetchIndexFollowsIndexV2(t *testing.T) {
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()
	mux.HandleFunc("/repo/index.min.json", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte(decoyIndex)) })
	mux.HandleFunc("/repo/repo.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"index_v2":"` + srv.URL + `/repo/index.pb","meta":{"name":"NovelSourcery"}}`))
	})
	mux.HandleFunc("/repo/index.pb", func(w http.ResponseWriter, _ *http.Request) { w.Write(novelIndexPB()) })
	mux.HandleFunc("/plain/index.min.json", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte(decoyIndex)) })

	for _, u := range []string{srv.URL + "/repo/index.min.json", srv.URL + "/repo/repo.json"} {
		exts, err := FetchIndex(u)
		if err != nil {
			t.Fatalf("%s: %v", u, err)
		}
		if len(exts) != 1 || exts[0].Name != "BoxNovel" || exts[0].JarURL != "https://example.com/boxnovel.jar" || exts[0].Lang != "en" {
			t.Fatalf("%s: got %+v, want the BoxNovel entry from index_v2", u, exts)
		}
	}

	// No repo.json: the index is used as given.
	exts, err := FetchIndex(srv.URL + "/plain/index.min.json")
	if err != nil || len(exts) != 1 || exts[0].Name != "Outdated App" {
		t.Fatalf("plain index: %+v %v", exts, err)
	}
}

func TestClassifyContentType(t *testing.T) {
	for pkg, want := range map[string]string{
		"eu.kanade.tachiyomi.extension.en.mangadex":      "manga",
		"eu.kanade.tachiyomi.animeextension.en.anikoto":  "anime",
		"eu.kanade.tachiyomi.novelextension.en.boxnovel": "novel",
	} {
		if got := ClassifyContentType(pkg); got != want {
			t.Errorf("ClassifyContentType(%q) = %q, want %q", pkg, got, want)
		}
	}
}
