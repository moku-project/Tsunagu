package sandbox

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type logWriter struct{ prefix string }

func (w *logWriter) Write(p []byte) (int, error) {
	log.Print(w.prefix + string(p))
	return len(p), nil
}

type SupervisedClient struct {
	jarPath           string
	port              int
	extDir            string
	storageDir        string
	novelEnabled      bool
	addr              string
	idleTimeout       time.Duration
	heapMB            int
	flaresolverrURLFn func() string

	mu         sync.Mutex
	cmd        *exec.Cmd
	exited     chan struct{} // closed once cmd has been reaped
	client     *Client
	adopted    bool // client talks to a sandbox this process did not spawn
	lastAccess time.Time
	holds      int // long jobs that reuse one *Client; the reaper waits for them

	stopReaper chan struct{}
}

type SupervisedConfig struct {
	JarPath             string
	Port                int
	ExtensionsDir       string
	StorageDir          string
	NovelEnabled        bool
	Addr                string
	IdleTimeout         time.Duration
	HeapMB              int
	FlareSolverrURLFunc func() string
}

func NewSupervised(cfg SupervisedConfig) *SupervisedClient {
	sc := &SupervisedClient{
		jarPath:           cfg.JarPath,
		port:              cfg.Port,
		extDir:            cfg.ExtensionsDir,
		storageDir:        cfg.StorageDir,
		novelEnabled:      cfg.NovelEnabled,
		addr:              cfg.Addr,
		idleTimeout:       cfg.IdleTimeout,
		heapMB:            cfg.HeapMB,
		flaresolverrURLFn: cfg.FlareSolverrURLFunc,
		stopReaper:        make(chan struct{}),
	}
	go sc.reapLoop()
	return sc
}

func (sc *SupervisedClient) Ensure(ctx context.Context) (*Client, error) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.lastAccess = time.Now()

	if sc.client != nil && (sc.processAlive() || sc.adopted && sc.client.usable()) {
		return sc.client, nil
	}
	if sc.cmd != nil {
		log.Printf("sandbox: process exited unexpectedly, respawning")
		sc.killLocked()
	}
	if sc.client != nil {
		sc.client.Close()
		sc.client = nil
	}
	sc.adopted = false

	if sc.cmd == nil {
		if client, err := NewClient(sc.addr); err == nil {
			checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			_, pingErr := client.ListLoadedExtensions(checkCtx)
			cancel()
			if pingErr == nil {
				log.Printf("sandbox: adopting existing process at %s", sc.addr)
				sc.client = client
				sc.adopted = true
				return sc.client, nil
			}
			client.Close()
		}
	}

	if err := sc.spawnLocked(); err != nil {
		return nil, err
	}
	if err := sc.waitReadyLocked(ctx); err != nil {
		return nil, err
	}
	return sc.client, nil
}

func (sc *SupervisedClient) processAlive() bool {
	if sc.cmd == nil || sc.cmd.Process == nil {
		return false
	}
	select {
	case <-sc.exited:
		return false
	default:
	}
	return processAlive(sc.cmd)
}

func (sc *SupervisedClient) pidFile() string {
	return filepath.Join(sc.storageDir, "sandbox.pid")
}

func (sc *SupervisedClient) killStalePid() {
	b, err := os.ReadFile(sc.pidFile())
	if err != nil {
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || pid <= 0 {
		return
	}
	if p, err := os.FindProcess(pid); err == nil {
		_ = p.Kill()
	}
	_ = os.Remove(sc.pidFile())
}

func (sc *SupervisedClient) spawnLocked() error {
	sc.killStalePid()
	resolved, err := resolveRuntime(sc.jarPath)
	if err != nil {
		return fmt.Errorf("resolving sandbox runtime: %w", err)
	}
	log.Printf("sandbox: spawning %s -cp %s tsunagu.MainKt (source: %s)",
		resolved.JavaBin, resolved.JarPath, resolved.Source)
	heap := sc.heapMB
	if heap <= 0 {
		heap = 1024
	}
	cmd := exec.Command(resolved.JavaBin,
		"-Dpolyglot.engine.WarnInterpreterOnly=false",
		"-Djava.net.preferIPv4Stack=true",
		fmt.Sprintf("-Xmx%dm", heap),
		"-XX:+UseSerialGC",
		"-XX:TieredStopAtLevel=1",
		"-XX:+ExitOnOutOfMemoryError",
		"-XX:+HeapDumpOnOutOfMemoryError",
		"-XX:HeapDumpPath="+filepath.Join(sc.storageDir, "sandbox-oom.hprof"),
		// Java's regex engine recurses per repetition, unlike Android's ICU, so
		// extension patterns over large pages (Next.js payload scrapers) overflow
		// small stacks, including on threads extensions start themselves. Stacks
		// are reserved, not committed: memory is only used as deep calls touch it.
		"-Xss64m",
		"-cp", resolved.JarPath,
		"tsunagu.MainKt",
	)
	cmd.SysProcAttr = childSysProcAttr()
	cmd.Env = append(os.Environ(),
		"SANDBOX_PORT="+strconv.Itoa(sc.port),
		"SANDBOX_EXTENSIONS_DIR="+sc.extDir,
		"SANDBOX_STORAGE_DIR="+sc.storageDir,
		"SANDBOX_ENABLE_NOVEL="+strconv.FormatBool(sc.novelEnabled),
	)
	if sc.flaresolverrURLFn != nil {
		if u := sc.flaresolverrURLFn(); u != "" {
			cmd.Env = append(cmd.Env, "SANDBOX_FLARESOLVERR_URL="+u)
		}
	}

	if tmp := usableSandboxTempDir(); tmp != "" {
		cmd.Env = append(cmd.Env, "TMP="+tmp, "TEMP="+tmp)
		log.Printf("sandbox: redirecting child TMP/TEMP to %s (default temp dir rejects AF_UNIX sockets)", tmp)
	}
	cmd.Stdout = &logWriter{prefix: "[sandbox] "}
	cmd.Stderr = &logWriter{prefix: "[sandbox] "}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("spawning sandbox process: %w", err)
	}
	if err := afterStart(cmd); err != nil {
		log.Printf("sandbox: afterStart: %v", err)
	}
	_ = os.MkdirAll(sc.storageDir, 0o755)
	_ = os.WriteFile(sc.pidFile(), []byte(strconv.Itoa(cmd.Process.Pid)), 0o644)
	// Reap immediately: an unreaped child stays a zombie, which still answers
	// signal 0 and would make processAlive report a crashed sandbox as healthy.
	exited := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(exited)
	}()
	sc.cmd = cmd
	sc.exited = exited
	return nil
}

func (sc *SupervisedClient) waitReadyLocked(ctx context.Context) error {
	deadline := time.Now().Add(30 * time.Second)
	backoff := 200 * time.Millisecond
	for time.Now().Before(deadline) {
		client, err := NewClient(sc.addr)
		if err == nil {
			checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			_, pingErr := client.ListLoadedExtensions(checkCtx)
			cancel()
			if pingErr == nil {
				sc.client = client
				log.Printf("sandbox: ready at %s", sc.addr)
				return nil
			}
			client.Close()
		}
		time.Sleep(backoff)
		if backoff < 2*time.Second {
			backoff *= 2
		}
	}
	return fmt.Errorf("sandbox did not become ready within timeout")
}

func (sc *SupervisedClient) reapLoop() {
	interval := 1 * time.Minute
	if sc.idleTimeout > 0 {
		interval = sc.idleTimeout / 4
		if interval < 5*time.Second {
			interval = 5 * time.Second
		}
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			sc.maybeReap()
		case <-sc.stopReaper:
			return
		}
	}
}

func (sc *SupervisedClient) maybeReap() {
	if sc.idleTimeout <= 0 {
		return
	}
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if sc.cmd == nil || sc.client == nil || sc.holds > 0 {
		return
	}
	if time.Since(sc.lastAccess) < sc.idleTimeout {
		return
	}
	log.Printf("sandbox: idle for %s, reaping process", sc.idleTimeout)
	sc.killLocked()
}

func (sc *SupervisedClient) killLocked() {
	if sc.client != nil {
		sc.client.Close()
		sc.client = nil
	}
	if sc.cmd != nil && sc.cmd.Process != nil {
		_ = sc.cmd.Process.Kill()
		<-sc.exited
	}
	sc.cmd = nil
	sc.exited = nil
	_ = os.Remove(sc.pidFile())
}

// Hold keeps the sandbox from being reaped as idle until release is called.
func (sc *SupervisedClient) Hold() (release func()) {
	sc.mu.Lock()
	sc.holds++
	sc.lastAccess = time.Now()
	sc.mu.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			sc.mu.Lock()
			sc.holds--
			sc.lastAccess = time.Now()
			sc.mu.Unlock()
		})
	}
}

// SetSpawnOptions changes settings the sandbox only reads at startup; call Restart to apply them.
func (sc *SupervisedClient) SetSpawnOptions(novelEnabled bool, heapMB int) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.novelEnabled = novelEnabled
	sc.heapMB = heapMB
}

// Restart kills the running process so the next Ensure respawns it with fresh env vars.
func (sc *SupervisedClient) Restart() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if sc.cmd == nil {
		return
	}
	log.Printf("sandbox: restarting to pick up new config")
	sc.killLocked()
}

func (sc *SupervisedClient) Shutdown() {
	close(sc.stopReaper)
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.killLocked()
}
