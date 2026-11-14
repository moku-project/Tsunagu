package graph

import (
	"context"
	"path/filepath"
	"testing"

	"tsunagu/backend/internal/db"
	"tsunagu/backend/internal/db/sqlcgen"
)

func TestInstanceIDIsCreatedOnceAndKept(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	r := &Resolver{Q: sqlcgen.New(conn)}

	first, err := r.instanceID(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 32 {
		t.Fatalf("want a 32-char hex id, got %q", first)
	}
	again, err := r.instanceID(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if again != first {
		t.Fatalf("id changed between calls: %q then %q", first, again)
	}
}
