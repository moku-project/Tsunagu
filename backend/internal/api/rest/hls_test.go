package rest

import (
	"net/url"
	"strings"
	"testing"
)

func TestRewriteHLSPlaylist(t *testing.T) {
	base, _ := url.Parse("https://cdn.example.com/anime/x/index-f1-v1-a1.m3u8")
	in := strings.Join([]string{
		"#EXTM3U",
		"#EXT-X-KEY:METHOD=AES-128,URI=\"key.bin\"",
		"#EXTINF:6.0,",
		"seg-1.ts",
		"#EXTINF:6.0,",
		"https://other.cdn/seg-2.ts",
		"",
	}, "\n")
	out := string(rewriteHLSPlaylist([]byte(in), base, "aGRy"))

	if !strings.Contains(out, "#EXTM3U") {
		t.Fatal("dropped header")
	}
	if !strings.Contains(out, `URI="hls?`) {
		t.Errorf("key URI not rewritten:\n%s", out)
	}
	if strings.Contains(out, "seg-1.ts\n") || strings.Contains(out, "https://other.cdn/seg-2.ts") {
		t.Errorf("segment URLs not rewritten:\n%s", out)
	}
	if !strings.Contains(out, "h=aGRy") {
		t.Errorf("header param not carried:\n%s", out)
	}

	if strings.Count(out, "#EXTINF:6.0,") != 2 {
		t.Errorf("EXTINF lines mangled:\n%s", out)
	}
}

func TestPublicHTTPURL(t *testing.T) {
	ok := []string{"https://cdn.example.com/a.m3u8", "http://1.2.3.4/x"}
	bad := []string{"ftp://x/y", "https://localhost/x", "http://127.0.0.1/x", "http://192.168.1.10/x", "http://[::1]/x", "not a url", ""}
	for _, u := range ok {
		if _, good := publicHTTPURL(u); !good {
			t.Errorf("want accept %q", u)
		}
	}
	for _, u := range bad {
		if _, good := publicHTTPURL(u); good {
			t.Errorf("want reject %q", u)
		}
	}
}

func TestRewriteDASHManifest(t *testing.T) {
	man, _ := url.Parse("https://cdn.example.com/anime/x/index.mpd")
	in := `<?xml version="1.0"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" type="static">
<Period><AdaptationSet><Representation id="0">
<SegmentTemplate initialization="stream_init$RepresentationID$.m4s" media="$RepresentationID$-seg_$Number%08d$.m4s" startNumber="1"/>
</Representation></AdaptationSet></Period>
</MPD>`
	out := string(rewriteDASHManifest([]byte(in), man, map[string]string{"Referer": "https://src/"}))

	if !strings.Contains(out, "<BaseURL>dash/") {
		t.Fatalf("no injected BaseURL:\n%s", out)
	}

	if strings.Index(out, "<BaseURL>dash/") < strings.Index(out, "<Period>") {

	} else {
		t.Errorf("BaseURL not injected at MPD head:\n%s", out)
	}

	if !strings.Contains(out, `media="$RepresentationID$-seg_$Number%08d$.m4s"`) {
		t.Errorf("segment template mangled:\n%s", out)
	}
}

func TestRewriteDASHAbsoluteBaseURL(t *testing.T) {
	man, _ := url.Parse("https://cdn.example.com/x/index.mpd")
	in := `<MPD type="static"><BaseURL>https://other.cdn/v/</BaseURL><Period/></MPD>`
	out := string(rewriteDASHManifest([]byte(in), man, nil))
	if strings.Contains(out, "https://other.cdn") {
		t.Errorf("absolute BaseURL not neutralized:\n%s", out)
	}
	if strings.Count(out, "<BaseURL>dash/") != 2 {
		t.Errorf("expected 2 proxied BaseURLs:\n%s", out)
	}
}

func TestSRTToVTT(t *testing.T) {
	srt := "1\r\n00:00:01,000 --> 00:00:04,000\r\nHello\r\n\r\n2\r\n00:00:05,500 --> 00:00:07,250\r\nWorld\r\n"
	out := string(srtToVTT([]byte(srt)))
	if !strings.HasPrefix(out, "WEBVTT\n\n") {
		t.Fatalf("no WEBVTT header: %q", out)
	}
	if strings.Contains(out, ",000") || strings.Contains(out, ",250") {
		t.Errorf("comma separators not converted: %q", out)
	}
	if !strings.Contains(out, "00:00:01.000 --> 00:00:04.000") {
		t.Errorf("timeline wrong: %q", out)
	}
	if !srtHeader.Match([]byte(srt)) {
		t.Error("srtHeader should match a real SRT")
	}
	if srtHeader.Match([]byte("WEBVTT\n\n00:00:01.000 --> ...")) {
		t.Error("srtHeader should not match VTT")
	}
}

func TestUnwrapLocalProxyURL(t *testing.T) {
	raw := "http://127.0.0.1:38205/proxy/playlist.m3u8?url=aHR0cHM6Ly92aXZpYmViZS5zaXRlL3B1YmxpYy9zdHJlYW0vZTZlNDkyMTRkMjJhN2YzYi8zMTE4NDdfMTA4MC5tM3U4&headers=VXNlci1BZ2VudDpNb3ppbGxhLzUuMCAoV2luZG93cyBOVCAxMC4wOyBXaW42NDsgeDY0KSBBcHBsZVdlYktpdC81MzcuMzYgKEtIVE1MLCBsaWtlIEdlY2tvKSBDaHJvbWUvMTIwLjAuMC4wIFNhZmFyaS81MzcuMzYKUmVmZXJlcjpodHRwczovL2FuaW5la28udG8vCg"
	got, h := unwrapLocalProxyURL(raw, map[string]string{"X": "y"})
	if got != "https://vivibebe.site/public/stream/e6e49214d22a7f3b/311847_1080.m3u8" {
		t.Fatalf("url = %q", got)
	}
	if h["Referer"] != "https://anineko.to/" || h["User-Agent"] == "" || h["X"] != "y" {
		t.Fatalf("headers = %#v", h)
	}
	if u, hh := unwrapLocalProxyURL("https://real.cdn/x.m3u8", nil); u != "https://real.cdn/x.m3u8" || hh != nil {
		t.Fatalf("passthrough broke: %q", u)
	}
}

func TestRewriteHLSPlaylistDropsAds(t *testing.T) {
	base, _ := url.Parse("https://vivibebe.site/public/stream/x/311847_1080.m3u8")
	in := "#EXTM3U\n#EXT-X-TARGETDURATION:10\n" +
		"#EXT-X-DISCONTINUITY\n#EXTINF:2.0,\nhttps://p16-ad-sg.ibyteimg.com/obj/ad-site-i18n/abc\n" +
		"#EXTINF:2.0,\nhttps://p16-ad-sg.ibyteimg.com/obj/ad-site-i18n/def\n" +
		"#EXT-X-DISCONTINUITY\n#EXTINF:10.0,\n311847_1080_00001.ts\n"
	out := string(rewriteHLSPlaylist([]byte(in), base, ""))
	if strings.Contains(out, "ibyteimg") {
		t.Fatalf("ad segment survived:\n%s", out)
	}
	if !strings.Contains(out, "311847_1080_00001.ts") && !strings.Contains(out, "hls?u=") {
		t.Fatalf("content segment lost:\n%s", out)
	}
	if strings.Count(out, "#EXTINF") != 1 {
		t.Fatalf("stale #EXTINF left:\n%s", out)
	}
}
