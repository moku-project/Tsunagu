package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCallerDataDirReplacesSavedPaths(t *testing.T) {
	oldDir := "/var/mobile/Containers/Data/Application/OLD/Backend"
	newDir := t.TempDir()
	toml := "data_dir = \"" + oldDir + "\"\ndb_path = \"" + oldDir + "/tsunagu.db\"\nmedia_dir = \"" + oldDir + "/media\"\njar_cache_dir = \"/elsewhere/jars\"\n"
	if err := os.WriteFile(filepath.Join(newDir, "tsunagu.toml"), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	c, _, _, err := Load(Options{DataDir: newDir})
	if err != nil {
		t.Fatal(err)
	}
	if c.DataDir != newDir || c.DBPath != filepath.Join(newDir, "tsunagu.db") || c.MediaDir != filepath.Join(newDir, "media") {
		t.Fatalf("paths not rebased: %+v", c)
	}
	if c.JarCacheDir != "/elsewhere/jars" {
		t.Fatalf("path outside the old data dir changed: %s", c.JarCacheDir)
	}
}
