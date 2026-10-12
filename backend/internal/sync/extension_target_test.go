package sync

import (
	"database/sql"
	"testing"

	"tsunagu/backend/internal/db/sqlcgen"
)

func TestExtensionDownloadTarget(t *testing.T) {
	jar := func(u string) sql.NullString { return sql.NullString{String: u, Valid: u != ""} }
	cases := []struct {
		name    string
		ext     sqlcgen.Extension
		wantURL string
		wantExt string
	}{
		{"LNReader plugin", sqlcgen.Extension{ContentType: "novel", JarUrl: jar("https://x/plugin.js")}, "https://x/plugin.js", "js"},
		{"Tsundoku novel jar", sqlcgen.Extension{ContentType: "novel", JarUrl: jar("https://x/boxnovel.jar"), ApkUrl: "https://x/boxnovel.apk"}, "https://x/boxnovel.jar", "jar"},
		{"manga apk only", sqlcgen.Extension{ContentType: "manga", ApkUrl: "https://x/m.apk"}, "https://x/m.apk", "apk"},
	}
	for _, c := range cases {
		u, ext, err := extensionDownloadTarget(c.ext)
		if err != nil || u != c.wantURL || ext != c.wantExt {
			t.Errorf("%s: got (%q, %q, %v), want (%q, %q)", c.name, u, ext, err, c.wantURL, c.wantExt)
		}
	}
}
