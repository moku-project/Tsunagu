package sandbox

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
	"time"
)

type SupervisedClient struct {
	binaryPath  string
	port        int
	extDir      string
	novelEnabled bool
	addr        string
	idleTimeout time.Duration

	mu         sync.Mutex
	cmd        *exec.Cmd
	client     *Client
	lastAccess time.Time

	stopReaper chan struct{}
}

type SupervisedConfig struct {
	BinaryPath     string
	Port           int
	ExtensionsDir  string
	NovelEnabled   bool
	Addr           string
	IdleTimeout    time.Duration
}

func NewSupervised(cfg SupervisedConfig) *SupervisedClient {
	sc := &SupervisedClient{
		binaryPath:   cfg.BinaryPath,
		port:         cfg.Port,
		extDir:       cfg.ExtensionsDir,
		novelEnabled: cfg.NovelEnabled,
		addr:         cfg.Addr,
		idleTimeout:  cfg.IdleTimeout,
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
	return sc.cmd.Process.Signal(syscall.Signal(0)) == nil
}

func (sc *SupervisedClient) spawnLocked() error {
	log.Printf("sandbox: spawning %s", sc.binaryPath)
	cmd := exec.Command(sc.binaryPath)
	cmd.Env = append(os.Environ(),
		"SANDBOX_PORT="+strconv.Itoa(sc.port),
		"SANDBOX_EXTENSIONS_DIR="+sc.extDir,
		"SANDBOX_ENABLE_NOVEL="+strconv.FormatBool(sc.novelEnabled),
	)
	cmd.Stdout = &logWriter{prefix: "[sandbox] "}
	cmd.Stderr = &logWriter{prefix: "[sandbox] "}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("spawning sandbox process: %w", err)
	}
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
		return // idle-reap disabled
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
}


func (sc *SupervisedClient) Shutdown() {
	close(sc.stopReaper)
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.killLocked()
}
