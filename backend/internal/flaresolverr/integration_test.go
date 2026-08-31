package flaresolverr

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestManagedInstallAndRun(t *testing.T) {
	if os.Getenv("FS_INTEG") == "" {
		t.Skip("set FS_INTEG=1 to run the FlareSolverr download+run integration test")
	}
	dir := os.Getenv("FS_DIR")
	if dir == "" {
		dir = filepath.Join(t.TempDir(), "flaresolverr")
	}
	m := NewManager(dir)
	ctx := context.Background()

	if !SupportedOnPlatform() {
		t.Fatalf("platform not supported")
	}

	if !m.Installed() {
		t.Log("installing FlareSolverr", PinnedVersion, "->", dir)
		if err := m.Install(ctx); err != nil {
			t.Fatalf("Install: %v", err)
		}
		deadline := time.Now().Add(8 * time.Minute)
		for m.downloading.Load() {
			if time.Now().After(deadline) {
				t.Fatal("install timed out")
			}
			s := m.Status(ctx)
			t.Logf("  state=%s progress=%.0f%% err=%v", s.State, deref(s.Progress)*100, deref(s.Error))
			time.Sleep(5 * time.Second)
		}
	}
	s := m.Status(ctx)
	if !m.Installed() {
		t.Fatalf("not installed after install: state=%s err=%v", s.State, deref(s.Error))
	}
	t.Log("installed OK, version marker present")

	m.ApplyConfig("managed", "")
	defer m.Shutdown()

	err := m.EnsureRunning(ctx)
	s = m.Status(ctx)
	t.Logf("EnsureRunning err=%v  state=%s reachable=%v url=%v solverErr=%v",
		err, s.State, s.Reachable, deref(s.URL), deref(s.Error))
	if err != nil {
		t.Fatalf("EnsureRunning failed: %v", err)
	}
	if !s.Reachable {
		t.Fatalf("running but /health not reachable")
	}
	t.Log("FlareSolverr managed process is up and healthy")
}

func deref[T any](p *T) T {
	var z T
	if p != nil {
		return *p
	}
	return z
}
