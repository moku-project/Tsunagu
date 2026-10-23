package graph

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"testing"

	sandboxv1 "tsunagu/backend/internal/sandbox/gen/sandbox/v1"
)

func subHeadersOf(t *testing.T, proxyURL string) map[string]string {
	t.Helper()
	u, err := url.Parse(proxyURL)
	if err != nil {
		t.Fatal(err)
	}
	h := u.Query().Get("h")
	if h == "" {
		return nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(h)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

// Subtitle hosts check the video's Referer, so subtitles carry their source's headers.
func TestSubtitlesCarrySourceHeaders(t *testing.T) {
	info := &sandboxv1.StreamInfo{
		Headers:   map[string]string{"Referer": "https://stream.example/"},
		Subtitles: []*sandboxv1.SubtitleTrack{{Url: "https://subs.example/top.vtt", Lang: "English"}},
		Sources: []*sandboxv1.VideoSource{
			{
				Url:       "https://cdn.example/a.m3u8",
				Headers:   map[string]string{"Referer": "https://player.example/"},
				Subtitles: []*sandboxv1.SubtitleTrack{{Url: "https://subs.example/a.vtt", Lang: "English"}},
			},
			{
				Url:       "https://cdn.example/b.m3u8",
				Subtitles: []*sandboxv1.SubtitleTrack{{Url: "https://subs.example/b.vtt", Lang: "English"}},
			},
		},
	}
	vs := toVideoStream(info, "1", "2")
	want := []string{"https://stream.example/", "https://player.example/", "https://stream.example/"}
	if len(vs.Subtitles) != len(want) {
		t.Fatalf("got %d subtitles, want %d", len(vs.Subtitles), len(want))
	}
	for i, s := range vs.Subtitles {
		if got := subHeadersOf(t, s.URL)["Referer"]; got != want[i] {
			t.Errorf("subtitle %d Referer = %q, want %q", i, got, want[i])
		}
	}
}
