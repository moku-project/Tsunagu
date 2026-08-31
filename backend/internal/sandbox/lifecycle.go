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
	jarPath      string
	port         int
	extDir       string
	storageDir   string
	novelEnabled bool
	addr         string
	idleTimeout  time.Duration
	heapMB       int

	mu         sync.Mutex
	cmd        *exec.Cmd
	client     *Client
	lastAccess time.Time

	stopReaper chan struct{}
}

type SupervisedConfig struct {
	JarPath       string
	Port          int
	ExtensionsDir string
	StorageDir    string
	NovelEnabled  bool
	Addr          string
	IdleTimeout   time.Duration
	HeapMB        int
}

func NewSupervised(cfg SupervisedConfig) *SupervisedClient {
	sc := &SupervisedClient{
		jarPath:      cfg.JarPath,
		port:         cfg.Port,
		extDir:       cfg.ExtensionsDir,
		storageDir:   cfg.StorageDir,
		novelEnabled: cfg.NovelEnabled,
		addr:         cfg.Addr,
		idleTimeout:  cfg.IdleTimeout,
		heapMB:       cfg.HeapMB,
		stopReaper:   make(chan struct{}),
	}
	go sc.reapLoop()
	return sc
}

func (sc *SupervisedClient) Ensure(ctx context.Context) (*Client, error) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.lastAccess = time.Now()

	if sc.client != nil && sc.processAlive() {
		return sc.client, nil
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
	if sc.cmd.ProcessState != nil {
		return false
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
		heap = 512
	}
	cmd := exec.Command(resolved.JavaBin,
		"-Dpolyglot.engine.WarnInterpreterOnly=false",
		fmt.Sprintf("-Xmx%dm", heap),
		"-XX:+UseSerialGC",
		"-XX:TieredStopAtLevel=1",
		"-XX:+ExitOnOutOfMemoryError",
		"-Xss512k",
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
	// On some Windows hosts, AF_UNIX connect() fails with WSAEINVAL for socket
	// files created under %USERPROFILE%\AppData (AV/EDR filter, Controlled
	// Folder Access, etc.). The JVM's NIO Selector self-pipe puts its socket
	// in the OS temp dir (GetTempPath -> %TMP%/%TEMP%), which defaults to
	// %LOCALAPPDATA%\Temp, so the sandbox dies in NioEventLoop.openSelector
	// before it can serve gRPC. If the current temp dir can't host an AF_UNIX
	// socket, redirect the child's TMP/TEMP to one that can. -Djava.io.tmpdir
	// does NOT help here: the native selector code reads the env var.
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
	sc.cmd = cmd
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
	if sc.cmd == nil || sc.client == nil {
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
		_ = sc.cmd.Wait()
	}
	sc.cmd = nil
	_ = os.Remove(sc.pidFile())
}

func (sc *SupervisedClient) Shutdown() {
	close(sc.stopReaper)
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.killLocked()
}
