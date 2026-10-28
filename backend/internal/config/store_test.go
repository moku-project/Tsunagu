package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileReplacesConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tsunagu.toml")
	s := &Store{path: path}
	if err := s.writeFileLocked([]byte("a = 1\n")); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(path); string(got) != "a = 1\n" {
		t.Fatalf("got %q", got)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("temp file left behind: %v", err)
	}
}

func TestWriteFileSkipsReadOnlyConfig(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores file permissions")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "tsunagu.toml")
	if err := os.WriteFile(path, []byte("a = 1\n"), 0o444); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	s := &Store{path: path}
	if err := s.writeFileLocked([]byte("a = 2\n")); err != nil {
		t.Fatalf("read-only config should be skipped, got %v", err)
	}
	if !s.readOnly {
		t.Fatal("readOnly not set")
	}
	if got, _ := os.ReadFile(path); string(got) != "a = 1\n" {
		t.Fatalf("read-only config changed: %q", got)
	}
}
