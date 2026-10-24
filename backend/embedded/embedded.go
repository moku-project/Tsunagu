// Package embedded runs Tsunagu inside another app's process. Its API sticks to
// types gomobile can bind, so Android calls it directly from Kotlin.
//
// The app provides the extension host: a gRPC server implementing the sandbox
// service on loopback, checking the token it passes to Start.
package embedded

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"tsunagu/backend/internal/server"
)

var (
	mu       sync.Mutex
	running  *server.Server
	apiToken string
)

// Version is set at build time with -ldflags.
var Version = "dev"

// Start runs the server on a random loopback port. The app's extension host
// must be listening at sandboxAddr and accept sandboxToken.
func Start(dataDir, sandboxAddr, sandboxToken string) error {
	mu.Lock()
	defer mu.Unlock()
	if running != nil {
		return errors.New("tsunagu is already running")
	}
	setupLog(dataDir)

	token, err := randomToken()
	if err != nil {
		return err
	}
	srv, err := server.Start(server.Options{
		DataDir:         dataDir,
		Version:         Version,
		HTTPAddr:        "127.0.0.1:0",
		APIToken:        token,
		ExternalSandbox: true,
		SandboxAddr:     sandboxAddr,
		SandboxToken:    sandboxToken,
	})
	if err != nil {
		return err
	}
	running, apiToken = srv, token
	return nil
}

// Stop shuts the server down; Start may be called again afterwards.
func Stop() error {
	mu.Lock()
	defer mu.Unlock()
	if running == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := running.Shutdown(ctx)
	running, apiToken = nil, ""
	return err
}

// BaseURL is where the API listens, or "" when stopped.
func BaseURL() string {
	mu.Lock()
	defer mu.Unlock()
	if running == nil {
		return ""
	}
	return running.URL
}

// APIToken must be sent as a bearer token on every request. Other apps on the
// device can reach loopback ports too, so the API is never left open.
func APIToken() string {
	mu.Lock()
	defer mu.Unlock()
	return apiToken
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func setupLog(dataDir string) {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return
	}
	path := filepath.Join(dataDir, "tsunagu.log")
	if st, err := os.Stat(path); err == nil && st.Size() > 4<<20 {
		_ = os.Rename(path, path+".old")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	log.SetOutput(io.MultiWriter(os.Stderr, f))
}
