package sync

import (
	"database/sql"
	"testing"

	"tsunagu/backend/internal/db/sqlcgen"
)

func TestExtensionDownloadTarget(t *testing.T) {
	jar := func(u string) sql.NullString { return sql.NullString{String: u, Valid: u != ""} }
	cases := []struct {
		name      string
		ext       sqlcgen.Extension
		wantURL   string
		preferAPK bool
		wantExt   string
	}{
		{"LNReader plugin", sqlcgen.Extension{ContentType: "novel", JarUrl: jar("https://x/plugin.js")}, "https://x/plugin.js", false, "js"},
		{"Tsundoku novel jar", sqlcgen.Extension{ContentType: "novel", JarUrl: jar("https://x/novel.jar"), ApkUrl: "https://x/novel.apk"}, "https://x/novel.jar", false, "jar"},
		{"manga apk only", sqlcgen.Extension{ContentType: "manga", ApkUrl: "https://x/m.apk"}, "https://x/m.apk", false, "apk"},
		{"manga jar and apk", sqlcgen.Extension{ContentType: "manga", JarUrl: jar("https://x/m.jar"), ApkUrl: "https://x/m.apk"}, "https://x/m.jar", false, "jar"},
		{"manga jar and apk, APK preferred", sqlcgen.Extension{ContentType: "manga", JarUrl: jar("https://x/m.jar"), ApkUrl: "https://x/m.apk"}, "https://x/m.apk", true, "apk"},
		{"LNReader plugin, APK preferred", sqlcgen.Extension{ContentType: "novel", JarUrl: jar("https://x/p.js")}, "https://x/p.js", true, "js"},
	}
	for _, c := range cases {
		u, ext, err := extensionDownloadTarget(c.ext, c.preferAPK)
		if err != nil || u != c.wantURL || ext != c.wantExt {
			t.Errorf("%s: got (%q, %q, %v), want (%q, %q)", c.name, u, ext, err, c.wantURL, c.wantExt)
		}
	}
}
