package rest

import (
	"strings"
	"testing"
)

func TestSanitizeNovelHTML(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"keeps allowlist", "<p>hi <em>there</em></p>", "<p>hi <em>there</em></p>"},
		{"strips attributes", `<p class="x" onclick="evil()">t</p>`, "<p>t</p>"},
		{"drops script whole", `<p>a</p><script>alert(1)</script><p>b</p>`, "<p>a</p><p>b</p>"},
		{"drops style whole", `<style>p{x}</style><p>b</p>`, "<p>b</p>"},
		{"unwraps unknown tag keeps text", `<article><p>x</p></article>`, "<p>x</p>"},
		{"unwraps anchor, drops href", `<p>see <a href="javascript:evil">link</a></p>`, "<p>see link</p>"},
		{"neutralizes img", `<p><img src=x onerror=evil></p>`, "<p></p>"},
		{"void br no close", "a<br>b", "a<br>b"},
		{"escapes stray text", `2 < 3 & 4 > 1`, "2 &lt; 3 &amp; 4 &gt; 1"},
	}
	for _, c := range cases {
		if got := sanitizeNovelHTML(c.in); got != c.want {
			t.Errorf("%s:\n in:   %q\n got:  %q\n want: %q", c.name, c.in, got, c.want)
		}
	}
}

func TestSanitizeNovelHTMLNoScriptSurvives(t *testing.T) {
	out := sanitizeNovelHTML(`<div><script>x</script><p>ok</p><iframe src=evil></iframe></div>`)
	if strings.Contains(out, "script") || strings.Contains(out, "iframe") || strings.Contains(out, "evil") {
		t.Fatalf("dangerous content survived: %q", out)
	}
}
