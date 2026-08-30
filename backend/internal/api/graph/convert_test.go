package graph

import "testing"

func TestParseSourceLabel(t *testing.T) {
	cases := []struct{ in, kind, server string }{
		{"HD-2 - Dub - 1080p (1920x1080) - 1.14 MB/s", "dub", "HD-2"},
		{"HD-1 - Soft Sub - 720p (1280x720)", "softsub", "HD-1"},
		{"StreamHG - Hard Sub - 480p", "hardsub", "StreamHG"},
		{"Doodstream - Dub - Doodstream mirror", "dub", "Doodstream"},
		{"1080p", "", ""},
		{"Vidstreaming", "", ""},
	}
	for _, c := range cases {
		k, s := parseSourceLabel(c.in)
		if k != c.kind || s != c.server {
			t.Errorf("%q => (%q,%q), want (%q,%q)", c.in, k, s, c.kind, c.server)
		}
	}
}
