package sync

import (
	"context"
	"path/filepath"
	"testing"

	"tsunagu/backend/internal/db"
	"tsunagu/backend/internal/db/sqlcgen"
	"tsunagu/backend/internal/repository"
)

func TestMirroredPackageKeepsItsRepo(t *testing.T) {
	dir := t.TempDir()
	conn, err := db.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	if _, err := conn.Exec(`INSERT INTO repositories (id, index_url) VALUES (1, 'https://a/index.min.json'), (2, 'https://b/index.min.json')`); err != nil {
		t.Fatal(err)
	}
	q := sqlcgen.New(conn)
	s := New(conn, q, dir, dir)
	ctx := context.Background()
	pkg := "eu.kanade.tachiyomi.extension.en.x"
	index := func(apk string) []repository.ParsedExtension {
		return []repository.ParsedExtension{{PackageName: pkg, Name: "X", VersionName: "1", Lang: "en", ApkURL: apk}}
	}

	if err := s.applyRepoIndex(ctx, 1, index("https://a/x.apk")); err != nil {
		t.Fatal(err)
	}
	if err := s.applyRepoIndex(ctx, 2, index("https://b/x.apk")); err != nil {
		t.Fatal(err)
	}
	ext, err := q.GetExtensionByPackageName(ctx, pkg)
	if err != nil || ext.RepositoryID != 1 || ext.ApkUrl != "https://a/x.apk" {
		t.Fatalf("mirror overwrote the owning repo's row: %+v %v", ext, err)
	}

	if _, err := conn.Exec(`UPDATE extensions SET installed = TRUE, installed_version = '1'`); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteRepository(ctx, 1); err != nil {
		t.Fatal(err)
	}
	ext, err = q.GetExtensionByPackageName(ctx, pkg)
	if err != nil || !ext.Installed {
		t.Fatalf("installed extension lost with its repo: %+v %v", ext, err)
	}
	// The mirror can now provide updates for it.
	if err := s.applyRepoIndex(ctx, 2, index("https://b/x.apk")); err != nil {
		t.Fatal(err)
	}
	if ext, _ = q.GetExtensionByPackageName(ctx, pkg); ext.RepositoryID != 2 || !ext.Installed {
		t.Fatalf("mirror did not take over: %+v", ext)
	}
}
