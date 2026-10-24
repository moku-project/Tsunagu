package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"tsunagu/backend/internal/config"
	"tsunagu/backend/internal/server"
)

var (
	serverVersion   = "dev"
	serverBuildTime = "unknown"
)

func setupFileLog(dataDir string) {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	dir := dataDir
	if dir == "" {
		dir = "."
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	path := filepath.Join(dir, "tsunagu.log")
	if st, err := os.Stat(path); err == nil && st.Size() > 4<<20 {
		_ = os.Rename(path, path+".old")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	log.SetOutput(io.MultiWriter(os.Stderr, f))
	log.Printf("=== %s %s (%s) starting, pid %d ===", server.Name, serverVersion, serverBuildTime, os.Getpid())
}

func main() {
	dataDir := flag.String("data-dir", "", "root directory for the DB, caches, extensions and downloads")
	configPath := flag.String("config", "", "path to tsunagu.toml (defaults to <data-dir>/tsunagu.toml)")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Printf("%s %s (%s)\n", server.Name, serverVersion, serverBuildTime)
		return
	}
	if flag.NArg() > 0 && flag.Arg(0) == "config" {
		runConfigCLI(flag.Args()[1:], config.Options{ConfigPath: *configPath, DataDir: *dataDir})
		return
	}

	setupFileLog(*dataDir)

	srv, err := server.Start(server.Options{
		DataDir:    *dataDir,
		ConfigPath: *configPath,
		Version:    serverVersion,
		BuildTime:  serverBuildTime,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("TSUNAGU_READY url=%s version=%s\n", srv.URL, serverVersion)
	_ = os.Stdout.Sync()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Print("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
