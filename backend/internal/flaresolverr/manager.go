package flaresolverr

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	ModeDisabled = "disabled"
	ModeExternal = "external"
	ModeManaged  = "managed"

	solverPort = 8191
)

type Manager struct {
	dir string

	mu          sync.Mutex
	cmd         *exec.Cmd
	port        int
	mode        string
	externalURL string

	errMu   sync.Mutex
	lastErr string

	downloading atomic.Bool
	permille    atomic.Uint64
}

func NewManager(dir string) *Manager {
	return &Manager{dir: dir, port: solverPort, mode: ModeDisabled}
}

// ApplyConfig reconfigures the solver from the effective config. Called at
// startup and whenever cloudflare_solver_* changes.
func (m *Manager) ApplyConfig(mode, externalURL string) {
	if mode != ModeExternal && mode != ModeManaged {
		mode = ModeDisabled
	}
	m.mu.Lock()
	m.mode = mode
	m.externalURL = strings.TrimRight(strings.TrimSpace(externalURL), "/")
	m.mu.Unlock()
	if mode != ModeManaged {
		m.stop()
		m.setErr("")
		return
	}
	m.setErr("")
	if SupportedOnPlatform() && !m.Installed() && !m.downloading.Load() {
		log.Printf("flaresolverr: managed mode selected, downloading %s", PinnedVersion)
		_ = m.Install(context.Background())
	}
}

func (m *Manager) currentMode() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.mode
}

func (m *Manager) versionFile() string { return filepath.Join(m.dir, "VERSION") }
func (m *Manager) localURL() string    { return fmt.Sprintf("http://127.0.0.1:%d", m.port) }

func (m *Manager) setErr(s string) {
	m.errMu.Lock()
	m.lastErr = s
	m.errMu.Unlock()
}

func (m *Manager) errStr() string {
	m.errMu.Lock()
	defer m.errMu.Unlock()
	return m.lastErr
}

func (m *Manager) Installed() bool {
	a, ok := platformAsset()
	if !ok {
		return false
	}
	if b, err := os.ReadFile(m.versionFile()); err != nil || strings.TrimSpace(string(b)) != PinnedVersion {
		return false
	}
	st, err := os.Stat(filepath.Join(m.dir, a.binRel))
	return err == nil && !st.IsDir()
}

func (m *Manager) Running() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cmd != nil
}

func (m *Manager) EffectiveURL() string {
	m.mu.Lock()
	mode, ext := m.mode, m.externalURL
	m.mu.Unlock()
	switch mode {
	case ModeManaged:
		return m.localURL()
	case ModeExternal:
		return ext
	default:
		return ""
	}
}

func (m *Manager) Install(ctx context.Context) error {
	a, ok := platformAsset()
	if !ok {
		return fmt.Errorf("no managed FlareSolverr build for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if m.downloading.Load() {
		return fmt.Errorf("install already in progress")
	}
	if m.Installed() {
		return nil
	}
	m.downloading.Store(true)
	m.permille.Store(0)
	m.setErr("")
	go func() {
		defer m.downloading.Store(false)
		if err := m.doInstall(context.Background(), a); err != nil {
			log.Printf("flaresolverr: install failed: %v", err)
			m.setErr(err.Error())
			_ = os.RemoveAll(m.dir)
		}
	}()
	return nil
}

func (m *Manager) doInstall(ctx context.Context, a asset) error {
	if err := os.MkdirAll(m.dir, 0o755); err != nil {
		return err
	}
	tmp := filepath.Join(m.dir, "download.part")
	defer os.Remove(tmp)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: HTTP %d", a.url, resp.StatusCode)
	}

	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	pr := &progressReader{r: resp.Body, total: resp.ContentLength, pm: &m.permille}
	if _, err := io.Copy(f, pr); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	switch a.archive {
	case "tar.gz":
		if err := extractTarGz(tmp, m.dir); err != nil {
			return err
		}
	case "zip":
		if err := extractZip(tmp, m.dir); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown archive type %q", a.archive)
	}

	bin := filepath.Join(m.dir, a.binRel)
	if _, err := os.Stat(bin); err != nil {
		return fmt.Errorf("extracted archive is missing %s", a.binRel)
	}
	if err := os.Chmod(bin, 0o755); err != nil {
		return err
	}
	return os.WriteFile(m.versionFile(), []byte(PinnedVersion), 0o644)
}

func (m *Manager) EnsureRunning(ctx context.Context) error {
	if m.currentMode() != ModeManaged {
		return fmt.Errorf("managed solver not enabled")
	}
	if !m.Installed() {
		return fmt.Errorf("FlareSolverr is not installed")
	}
	m.mu.Lock()
	if m.cmd != nil {
		m.mu.Unlock()
		return nil
	}
	err := m.startLocked()
	m.mu.Unlock()
	if err != nil {
		m.setErr(err.Error())
		return err
	}
	if err := m.waitHealthy(ctx, 45*time.Second); err != nil {
		m.setErr(err.Error())
		m.stop()
		return err
	}
	m.setErr("")
	return nil
}

func (m *Manager) startLocked() error {
	a, _ := platformAsset()
	bin := filepath.Join(m.dir, a.binRel)
	cmd := exec.Command(bin)
	cmd.Dir = filepath.Dir(bin)
	cmd.Env = append(os.Environ(),
		"HOST=127.0.0.1",
		fmt.Sprintf("PORT=%d", m.port),
		"LOG_LEVEL=warning",
		"LOG_HTML=false",
	)
	cmd.Stdout = logWriter{}
	cmd.Stderr = logWriter{}
	if err := cmd.Start(); err != nil {
		return err
	}
	m.cmd = cmd
	go func(c *exec.Cmd) {
		_ = c.Wait()
		m.mu.Lock()
		if m.cmd == c {
			m.cmd = nil
		}
		m.mu.Unlock()
	}(cmd)
	return nil
}

func (m *Manager) waitHealthy(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if m.pingLocal(ctx) {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
	return fmt.Errorf("FlareSolverr did not become healthy within %s", timeout)
}

func (m *Manager) pingLocal(ctx context.Context) bool {
	return reachable(ctx, m.localURL())
}

func reachable(ctx context.Context, base string) bool {
	if base == "" {
		return false
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+"/", nil)
	if err != nil {
		return false
	}
	resp, err := (&http.Client{Timeout: 4 * time.Second}).Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode < 500
}

func (m *Manager) stop() {
	m.mu.Lock()
	c := m.cmd
	m.cmd = nil
	m.mu.Unlock()
	if c != nil && c.Process != nil {
		_ = c.Process.Kill()
		_ = c.Wait()
	}
}

func (m *Manager) Shutdown() { m.stop() }

func (m *Manager) Uninstall() error {
	m.stop()
	m.setErr("")
	return os.RemoveAll(m.dir)
}

type Status struct {
	Mode      string
	State     string
	Progress  *float64
	Version   *string
	URL       *string
	Reachable bool
	Error     *string
	Supported bool
}

func (m *Manager) Status(ctx context.Context) Status {
	s := Status{Mode: m.currentMode(), Supported: SupportedOnPlatform()}
	if e := m.errStr(); e != "" {
		s.Error = &e
	}
	if u := m.EffectiveURL(); u != "" {
		s.URL = &u
		s.Reachable = reachable(ctx, u)
	}
	switch {
	case m.downloading.Load():
		s.State = "DOWNLOADING"
		p := float64(m.permille.Load()) / 1000
		s.Progress = &p
	case s.Error != nil:
		s.State = "ERROR"
	case s.Mode == ModeManaged && !m.Installed():
		s.State = "NOT_INSTALLED"
	case s.Mode == ModeManaged && m.Running():
		s.State = "RUNNING"
	case s.Mode == ModeManaged, s.Mode == ModeExternal:
		s.State = "INSTALLED"
	default:
		s.State = "NOT_INSTALLED"
	}
	if m.Installed() {
		v := PinnedVersion
		s.Version = &v
	}
	return s
}

// SolveHandler proxies POST /internal/flaresolverr/v1 from the sandbox to the
// active solver, lazily starting the managed process on first use.
func (m *Manager) SolveHandler(prefix string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isLoopback(r.RemoteAddr) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		ctx := r.Context()
		switch m.currentMode() {
		case ModeDisabled:
			http.Error(w, "cloudflare solver disabled", http.StatusServiceUnavailable)
			return
		case ModeManaged:
			if err := m.EnsureRunning(ctx); err != nil {
				http.Error(w, "flaresolverr unavailable: "+err.Error(), http.StatusBadGateway)
				return
			}
		}
		target := m.EffectiveURL()
		if target == "" {
			http.Error(w, "no solver url configured", http.StatusServiceUnavailable)
			return
		}
		sub := strings.TrimPrefix(r.URL.Path, prefix)
		if sub == "" {
			sub = "/"
		}
		out, err := http.NewRequestWithContext(ctx, r.Method, strings.TrimRight(target, "/")+sub, r.Body)
		if err != nil {
			http.Error(w, "bad proxy request", http.StatusBadGateway)
			return
		}
		if ct := r.Header.Get("Content-Type"); ct != "" {
			out.Header.Set("Content-Type", ct)
		}
		resp, err := (&http.Client{Timeout: 120 * time.Second}).Do(out)
		if err != nil {
			http.Error(w, "solver request failed: "+err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		for k, vs := range resp.Header {
			for _, v := range vs {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	})
}

func isLoopback(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

type logWriter struct{}

func (logWriter) Write(p []byte) (int, error) {
	log.Print("[flaresolverr] " + strings.TrimRight(string(p), "\n"))
	return len(p), nil
}

type progressReader struct {
	r     io.Reader
	total int64
	read  int64
	pm    *atomic.Uint64
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	p.read += int64(n)
	if p.total > 0 {
		v := p.read * 1000 / p.total
		if v > 1000 {
			v = 1000
		}
		p.pm.Store(uint64(v))
	}
	return n, err
}

func extractTarGz(archivePath, dest string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		target, ok := safeJoin(dest, hdr.Name)
		if !ok {
			continue
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(hdr.Mode)&0o777)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}
}

func extractZip(archivePath, dest string) error {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer zr.Close()
	for _, zf := range zr.File {
		target, ok := safeJoin(dest, zf.Name)
		if !ok {
			continue
		}
		if zf.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := zf.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
		if err != nil {
			rc.Close()
			return err
		}
		if _, err := io.Copy(out, rc); err != nil {
			out.Close()
			rc.Close()
			return err
		}
		out.Close()
		rc.Close()
	}
	return nil
}

func safeJoin(base, name string) (string, bool) {
	clean := filepath.Clean("/" + strings.ReplaceAll(name, `\`, "/"))
	joined := filepath.Join(base, clean)
	if joined != base && !strings.HasPrefix(joined, base+string(os.PathSeparator)) {
		return "", false
	}
	return joined, true
}
