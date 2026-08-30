package rest

import (
	"net/url"
	"strings"
	"testing"
)

func BenchmarkRewriteHLSPlaylist(b *testing.B) {
	base, _ := url.Parse("https://cdn.example/stream/1080.m3u8")
	var sb strings.Builder
	sb.WriteString("#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:6\n")
	for i := 0; i < 400; i++ {
		sb.WriteString("#EXTINF:6.0,\nseg_")
		sb.WriteString(strings.Repeat("0", 4))
		sb.WriteString(".ts\n")
	}
	sb.WriteString("#EXT-X-ENDLIST\n")
	body := []byte(sb.String())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rewriteHLSPlaylist(body, base, "")
	}
}

func BenchmarkSanitizeNovelHTML(b *testing.B) {
	in := strings.Repeat(`<p class="x" style="color:red">Some <b>bold</b> and <i>italic</i> text with a <a href="http://x">link</a> and <script>evil()</script>.</p>`, 200)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sanitizeNovelHTML(in)
	}
}

func BenchmarkSRTToVTT(b *testing.B) {
	var sb strings.Builder
	for i := 0; i < 500; i++ {
		sb.WriteString("1\r\n00:00:01,000 --> 00:00:04,000\r\nLine of dialogue here.\r\n\r\n")
	}
	in := []byte(sb.String())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = srtToVTT(in)
	}
}
