package rest

import (
	"net/url"
	"strings"
	"testing"
)

const sampleMPD = `<?xml version="1.0"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" type="static" mediaPresentationDuration="PT12.0S"><BaseURL>dash/abc/</BaseURL>
<Period>
<AdaptationSet contentType="video" lang="jpn"><Representation id="v" codecs="avc1.4d401f" bandwidth="1000" width="1280" height="720">
<SegmentTemplate timescale="1000" initialization="init_$RepresentationID$.m4s" media="$RepresentationID$-$Number%03d$.m4s" startNumber="1">
<SegmentTimeline><S t="0" d="5000" r="1"/><S d="2000"/></SegmentTimeline></SegmentTemplate></Representation></AdaptationSet>
<AdaptationSet contentType="audio" lang="jpn"><Representation id="a" codecs="mp4a.40.2" bandwidth="96">
<SegmentTemplate timescale="100" initialization="init_a.m4s" media="a_$Time$.m4s" duration="500"/></Representation></AdaptationSet>
</Period></MPD>`

func TestDASHBecomesHLS(t *testing.T) {
	self, _ := url.Parse("/content/1/2/video?format=hls")
	master, err := dashToHLS([]byte(sampleMPD), "", self)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`CODECS="avc1.4d401f,mp4a.40.2"`, "RESOLUTION=1280x720", `AUDIO="aud"`, `URI="video?format=hls&rep=a"`, "video?format=hls&rep=v"} {
		if !strings.Contains(string(master), want) {
			t.Fatalf("master lacks %q:\n%s", want, master)
		}
	}
	video, err := dashToHLS([]byte(sampleMPD), "v", self)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`#EXT-X-MAP:URI="dash/abc/init_v.m4s"`, "#EXTINF:5.000,\ndash/abc/v-001.m4s", "dash/abc/v-002.m4s", "#EXTINF:2.000,\ndash/abc/v-003.m4s", "#EXT-X-ENDLIST"} {
		if !strings.Contains(string(video), want) {
			t.Fatalf("video playlist lacks %q:\n%s", want, video)
		}
	}
	audio, err := dashToHLS([]byte(sampleMPD), "a", self)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(audio), "#EXTINF"); n != 3 || !strings.Contains(string(audio), "dash/abc/a_1000.m4s") {
		t.Fatalf("audio playlist from a duration template is wrong (%d segments):\n%s", n, audio)
	}
}
