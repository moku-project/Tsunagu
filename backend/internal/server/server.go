// Package server wires Tsunagu's services together. The desktop binary and the
// embedded mobile build both start the server through it.
package server

import (
	"context"
	"fmt"
	"html"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tsunagu/backend/internal/api/graph"
	"tsunagu/backend/internal/api/rest"
	"tsunagu/backend/internal/auth"
	"tsunagu/backend/internal/automation"
	"tsunagu/backend/internal/backup"
	"tsunagu/backend/internal/config"
	"tsunagu/backend/internal/contentfilter"
	"tsunagu/backend/internal/db"
	"tsunagu/backend/internal/db/sqlcgen"
	"tsunagu/backend/internal/download"
	"tsunagu/backend/internal/flaresolverr"
	"tsunagu/backend/internal/localsource"
	"tsunagu/backend/internal/metadata"
	"tsunagu/backend/internal/sandbox"
	"tsunagu/backend/internal/streamresolve"
	"tsunagu/backend/internal/sync"
	"tsunagu/backend/internal/tracker"
)

const Name = "Tsunagu"

type Options struct {
	DataDir    string
	ConfigPath string
	Version    string
	BuildTime  string

	// Set by embedders; each overrides its config value without persisting it.
	HTTPAddr string
	APIToken string

	// ExternalSandbox means the embedder runs the extension host at SandboxAddr;
	// the server connects to it and never spawns one.
	ExternalSandbox bool
	SandboxAddr     string
	SandboxToken    string
}

type Server struct {
	// URL is the loopback base URL clients on this machine should use.
	URL string

	http     *http.Server
	cleanups []func()
}

func Start(opts Options) (*Server, error) {
	s := &Server{}
	ok := false
	defer func() {
		if !ok {
			s.runCleanups()
		}
	}()

	bootCfg, tomlPath, activeKeys, err := config.Load(config.Options{ConfigPath: opts.ConfigPath, DataDir: opts.DataDir})
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	if opts.HTTPAddr != "" {
		bootCfg.HTTPAddr = opts.HTTPAddr
	}
	if wd, wdErr := os.Getwd(); wdErr == nil {
		log.Printf("working directory: %s", wd)
	}
	if absData, absErr := filepath.Abs(bootCfg.DataDir); absErr == nil {
		log.Printf("data directory: %s", absData)
	} else {
		log.Printf("data directory: %s (unresolved: %v)", bootCfg.DataDir, absErr)
	}
	log.Printf("config file: %s", tomlPath)

	wantAddr := bootCfg.HTTPAddr
	if !strings.Contains(wantAddr, ":") {
		wantAddr = ":" + wantAddr
	}
	ln, err := net.Listen("tcp", wantAddr)
	if err != nil {
		host, _, _ := net.SplitHostPort(wantAddr)
		ln, err = net.Listen("tcp", net.JoinHostPort(host, "0"))
		if err != nil {
			return nil, fmt.Errorf("listen: %w", err)
		}
		log.Printf("address %s unavailable, bound %s instead", wantAddr, ln.Addr())
	}
	s.cleanups = append(s.cleanups, func() { _ = ln.Close() })
	boundAddr := ln.Addr().String()

	conn, err := db.Open(bootCfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("opening db: %w", err)
	}
	s.cleanups = append(s.cleanups, func() { _ = conn.Close() })
	q := sqlcgen.New(conn)

	fsDir, _ := filepath.Abs(filepath.Join(filepath.Dir(bootCfg.DBPath), "flaresolverr"))
	fsMgr := flaresolverr.NewManager(fsDir)
	s.cleanups = append(s.cleanups, fsMgr.Shutdown)

	store := config.NewStore(bootCfg, q, tomlPath, activeKeys)

	cfMgr, err := contentfilter.New(q)
	if err != nil {
		return nil, fmt.Errorf("content filter: %w", err)
	}
	store.OnChange("content_filter_level", func(context.Context) {
		cfMgr.SetLevel(contentfilter.ParseLevel(store.Config().ContentFilterLevel))
	})

	if err := store.Sync(context.Background()); err != nil {
		return nil, fmt.Errorf("syncing config: %w", err)
	}
	fsMgr.ApplyConfig(store.Config().CloudflareSolverMode, store.Config().CloudflareSolverURL)
	cfMgr.SetLevel(contentfilter.ParseLevel(store.Config().ContentFilterLevel))

	cfg := store.Config()
	if opts.APIToken != "" {
		cfg.APIToken = opts.APIToken
	}
	if opts.SandboxAddr != "" {
		cfg.SandboxAddr = opts.SandboxAddr
	}
	if cfg.PublicURL == "" || strings.HasPrefix(cfg.PublicURL, "http://localhost:6007") {
		host, port, _ := net.SplitHostPort(boundAddr)
		if host == "" || host == "::" || host == "0.0.0.0" {
			host = "127.0.0.1"
		}
		cfg.PublicURL = "http://" + net.JoinHostPort(host, port)
	}
	if absCacheDir, err := filepath.Abs(cfg.JarCacheDir); err == nil {
		cfg.JarCacheDir = absCacheDir
	}

	absMediaDir, err := filepath.Abs(cfg.MediaDir)
	if err != nil {
		return nil, fmt.Errorf("resolving media dir: %w", err)
	}
	if err := os.MkdirAll(absMediaDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating media dir: %w", err)
	}

	syncer := sync.New(conn, q, cfg.JarCacheDir, absMediaDir)
	if n, err := syncer.BackfillChapterNumbers(context.Background()); err != nil {
		log.Printf("chapter-number backfill: %v", err)
	} else if n > 0 {
		log.Printf("chapter-number backfill: recovered %d chapter numbers from titles", n)
	}
	trackerMgr := tracker.NewManager(q, cfg.AniListClientID, tracker.MALConfig{
		ClientID:     cfg.MALClientID,
		ClientSecret: cfg.MALClientSecret,
		CallbackURL:  strings.TrimRight(cfg.PublicURL, "/") + "/api/tracker/mal/callback",
	})
	metadataMgr := metadata.NewManager(conn, q)
	syncer.SetEnricher(metadataMgr)
	metadataMgr.SetRecomputer(cfMgr)
	syncer.SetRecomputer(cfMgr)

	supervised := sandbox.NewSupervised(sandbox.SupervisedConfig{
		JarPath:       cfg.SandboxJarPath,
		Port:          cfg.SandboxPort,
		ExtensionsDir: cfg.SandboxExtDir,
		StorageDir:    cfg.SandboxStorageDir,
		NovelEnabled:  cfg.NovelEnabled,
		Addr:          cfg.SandboxAddr,
		Token:         opts.SandboxToken,
		External:      opts.ExternalSandbox,
		IdleTimeout:   cfg.IdleTimeout(),
		HeapMB:        cfg.SandboxHeapMB,
		FlareSolverrURLFunc: func() string {
			if store.Config().CloudflareSolverMode == flaresolverr.ModeDisabled {
				return ""
			}
			return strings.TrimRight(cfg.PublicURL, "/") + "/internal/flaresolverr"
		},
	})
	s.cleanups = append(s.cleanups, supervised.Shutdown)

	applyCloudflare := func(ctx context.Context) {
		c := store.Config()
		fsMgr.ApplyConfig(c.CloudflareSolverMode, c.CloudflareSolverURL)
		// sandbox only reads SANDBOX_FLARESOLVERR_URL at spawn, so kick it
		supervised.Restart()
	}
	store.OnChange("cloudflare_solver_mode", applyCloudflare)
	store.OnChange("cloudflare_solver_url", applyCloudflare)
	applySpawnOptions := func(context.Context) {
		c := store.Config()
		supervised.SetSpawnOptions(c.NovelEnabled, c.SandboxHeapMB)
		supervised.Restart()
	}
	store.OnChange("novel_enabled", applySpawnOptions)
	store.OnChange("sandbox_heap_mb", applySpawnOptions)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		start := time.Now()
		if err := reloadInstalledExtensions(ctx, syncer, supervised); err != nil {
			log.Printf("warning: could not reload installed extensions on startup: %v (after %s)", err, time.Since(start).Round(time.Second))
		} else {
			log.Printf("reloaded installed extensions in %s", time.Since(start).Round(time.Second))
		}
	}()

	resolveDownloadsDir := func() string {
		d := strings.TrimSpace(store.Config().DownloadsDir)
		if d == "" {
			return absMediaDir
		}
		if abs, err := filepath.Abs(d); err == nil {
			return abs
		}
		return d
	}
	downloadMgr := download.New(q, supervised, absMediaDir, resolveDownloadsDir(), store.Config().MangaDownloadFormat)
	downloadMgr.Start()
	s.cleanups = append(s.cleanups, downloadMgr.Shutdown)
	store.OnChange("downloads_dir", func(context.Context) {
		downloadMgr.SetDownloadsDir(resolveDownloadsDir())
	})
	store.OnChange("manga_download_format", func(context.Context) {
		downloadMgr.SetImageFormat(store.Config().MangaDownloadFormat)
	})

	automationMgr := automation.New(q, downloadMgr, syncer, supervised)
	automationMgr.Start(context.Background())

	resolveLocalSourceDir := func() string {
		d := strings.TrimSpace(store.Config().LocalSourceDir)
		if d == "" {
			return ""
		}
		if abs, err := filepath.Abs(d); err == nil {
			return abs
		}
		return d
	}
	localScanner := localsource.New(q, absMediaDir)
	localScanner.SetEnricher(metadataMgr)
	localScanner.SetLocalDir(resolveLocalSourceDir())
	store.OnChange("local_source_dir", func(context.Context) {
		localScanner.SetLocalDir(resolveLocalSourceDir())
	})

	streamResolver := streamresolve.New(supervised)

	if cfg.MetadataBackfill {
		go metadataMgr.EnrichLibrary(context.Background())
	}
	go cfMgr.RecomputeAll(context.Background())

	if h := cfg.TrackerPollHours; h > 0 {
		go func() {
			t := time.NewTicker(time.Duration(h) * time.Hour)
			defer t.Stop()
			trackerMgr.PollAll(context.Background())
			for range t.C {
				trackerMgr.PollAll(context.Background())
			}
		}()
	}

	if h := cfg.BackupIntervalHours; h > 0 {
		backupDir := filepath.Join(cfg.DataDir, "backups")
		go func() {
			t := time.NewTicker(time.Duration(h) * time.Hour)
			defer t.Stop()
			for range t.C {
				if _, err := backup.CreateSnapshot(context.Background(), conn, backupDir); err != nil {
					log.Printf("scheduled backup: %v", err)
					continue
				}
				if err := backup.PruneSnapshots(backupDir, cfg.BackupRetentionCount); err != nil {
					log.Printf("scheduled backup: prune: %v", err)
				}
			}
		}()
	}

	mux := http.NewServeMux()

	mux.Handle("/content/", &rest.ContentHandler{Q: q, Sc: supervised, Sr: streamResolver})
	mux.Handle("/proxy/cover/", &rest.CoverProxyHandler{Q: q, CoverCacheDir: filepath.Join(absMediaDir, "covers"), Sc: supervised})
	remoteImg := &rest.RemoteCoverProxyHandler{CoverCacheDir: filepath.Join(absMediaDir, "covers", "remote")}
	mux.Handle("/proxy/cover/remote/", remoteImg)
	mux.Handle("/proxy/img/", remoteImg)
	mux.Handle("/proxy/icon/", &rest.IconProxyHandler{Q: q, IconCacheDir: filepath.Join(absMediaDir, "icons")})
	mux.Handle("/internal/flaresolverr/", fsMgr.SolveHandler("/internal/flaresolverr"))
	mux.Handle("/api/backups/import-file", &rest.BackupImportHandler{Q: q, Cfg: store})
	mux.HandleFunc("/api/tracker/mal/callback", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		info, err := trackerMgr.OAuthCallback(r.Context(), "mal", r.URL.Query())
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("<h2>MyAnimeList connection failed</h2><p>" + html.EscapeString(err.Error()) + "</p>"))
			return
		}
		_, _ = w.Write([]byte("<h2>Connected to MyAnimeList as " + html.EscapeString(info.Username) + "</h2><p>You can close this tab.</p>"))
	})
	authMgr := auth.New(q)
	registerRoutes(mux, authMgr)
	registerGraphQL(mux, &graph.Resolver{
		Sy: syncer, Sc: supervised, Dm: downloadMgr, Au: automationMgr, Ls: localScanner, Tk: trackerMgr,
		Md: metadataMgr, Sr: streamResolver, Q: q, DB: conn, Fs: fsMgr, Cfg: store, Cf: cfMgr, Am: authMgr,
		MediaDir: absMediaDir, Name: Name, Version: opts.Version, BuildTime: opts.BuildTime,
	})

	s.http = &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           corsMiddleware(cfg.AllowedOrigins, authMiddleware(cfg.APIToken, authMgr, logRequests(mux))),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	if cfg.APIToken != "" {
		log.Print("API token required (Authorization: Bearer … or ?token=…)")
	} else if ip := net.ParseIP(readyHostOf(boundAddr)); (ip == nil || !ip.IsLoopback()) && !authMgr.PasswordSet(context.Background()) {
		log.Printf("WARNING: listening on %s with no password or API token; anyone on the network can use this server", boundAddr)
	}

	if cfg.PprofAddr != "" {
		go func() {
			log.Printf("pprof on %s", cfg.PprofAddr)
			_ = http.ListenAndServe(cfg.PprofAddr, nil)
		}()
	}

	readyHost, readyPort, _ := net.SplitHostPort(boundAddr)
	if readyHost == "" || readyHost == "::" || readyHost == "0.0.0.0" {
		readyHost = "127.0.0.1"
	}
	s.URL = "http://" + net.JoinHostPort(readyHost, readyPort)

	go func() {
		log.Printf("tsunagu %s listening on %s", opts.Version, boundAddr)
		if err := s.http.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("server error: %v", err)
		}
	}()

	ok = true
	return s, nil
}

// Shutdown stops accepting requests, then stops the background services.
func (s *Server) Shutdown(ctx context.Context) error {
	err := s.http.Shutdown(ctx)
	s.runCleanups()
	return err
}

func (s *Server) runCleanups() {
	for i := len(s.cleanups) - 1; i >= 0; i-- {
		s.cleanups[i]()
	}
	s.cleanups = nil
}
