package main

import (
	"context"
	"flag"
	"fmt"
	"html"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/playground"

	"tsunagu/backend/internal/api/graph"
	"tsunagu/backend/internal/api/rest"
	"tsunagu/backend/internal/config"
	"tsunagu/backend/internal/db"
	"tsunagu/backend/internal/db/sqlcgen"
	"tsunagu/backend/internal/download"
	"tsunagu/backend/internal/localsource"
	"tsunagu/backend/internal/metadata"
	"tsunagu/backend/internal/sandbox"
	sandboxv1 "tsunagu/backend/internal/sandbox/gen/sandbox/v1"
	"tsunagu/backend/internal/streamresolve"
	"tsunagu/backend/internal/sync"
	"tsunagu/backend/internal/tracker"
)

const serverName = "Tsunagu"

var (
	serverVersion   = "dev"
	serverBuildTime = "unknown"
)

func main() {
	dataDir := flag.String("data-dir", "", "directory for the DB, caches, extensions and downloads")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Printf("%s %s (%s)\n", serverName, serverVersion, serverBuildTime)
		return
	}
	if *dataDir != "" {
		_ = os.Setenv("TSUNAGU_DATA_DIR", *dataDir)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}

	wantAddr := cfg.HTTPAddr
	if !strings.Contains(wantAddr, ":") {
		wantAddr = ":" + wantAddr
	}
	ln, err := net.Listen("tcp", wantAddr)
	if err != nil {
		host, _, _ := net.SplitHostPort(wantAddr)
		ln, err = net.Listen("tcp", net.JoinHostPort(host, "0"))
		if err != nil {
			log.Fatalf("listen: %v", err)
		}
		log.Printf("address %s unavailable, bound %s instead", wantAddr, ln.Addr())
	}
	boundAddr := ln.Addr().String()
	if cfg.PublicURL == "" || strings.HasPrefix(cfg.PublicURL, "http://localhost:6007") {
		host, port, _ := net.SplitHostPort(boundAddr)
		if host == "" || host == "::" || host == "0.0.0.0" {
			host = "127.0.0.1"
		}
		cfg.PublicURL = "http://" + net.JoinHostPort(host, port)
	}

	absCacheDir, err := filepath.Abs(cfg.JarCacheDir)
	if err != nil {
		log.Fatalf("resolving cache dir: %v", err)
	}
	cfg.JarCacheDir = absCacheDir

	conn, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("opening db: %v", err)
	}
	defer conn.Close()

	q := sqlcgen.New(conn)
	absMediaDir, err := filepath.Abs(cfg.MediaDir)
	if err != nil {
		log.Fatalf("resolving media dir: %v", err)
	}
	if err := os.MkdirAll(absMediaDir, 0o755); err != nil {
		log.Fatalf("creating media dir: %v", err)
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

	supervised := sandbox.NewSupervised(sandbox.SupervisedConfig{
		JarPath:       cfg.SandboxJarPath,
		Port:          cfg.SandboxPort,
		ExtensionsDir: cfg.SandboxExtDir,
		StorageDir:    cfg.SandboxStorageDir,
		NovelEnabled:  cfg.NovelEnabled,
		Addr:          cfg.SandboxAddr,
		IdleTimeout:   cfg.IdleTimeout(),
		HeapMB:        cfg.SandboxHeapMB,
	})
	defer supervised.Shutdown()

	if err := reloadInstalledExtensions(context.Background(), syncer, supervised); err != nil {
		log.Printf("warning: could not reload installed extensions on startup: %v", err)

	}
	downloadMgr := download.New(q, supervised, absMediaDir)
	downloadMgr.Start()
	defer downloadMgr.Shutdown()

	streamResolver := streamresolve.New(supervised)

	if cfg.MetadataBackfill {
		go metadataMgr.EnrichLibrary(context.Background())
	}

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

	globalMediaDir = absMediaDir

	mux := http.NewServeMux()

	mux.Handle("/content/", &rest.ContentHandler{Q: q, Sc: supervised, Sr: streamResolver})
	mux.Handle("/proxy/cover/", &rest.CoverProxyHandler{Q: q, CoverCacheDir: filepath.Join(absMediaDir, "covers")})
	remoteImg := &rest.RemoteCoverProxyHandler{CoverCacheDir: filepath.Join(absMediaDir, "covers", "remote")}
	mux.Handle("/proxy/cover/remote/", remoteImg)
	mux.Handle("/proxy/img/", remoteImg)
	mux.Handle("/proxy/icon/", &rest.IconProxyHandler{Q: q, IconCacheDir: filepath.Join(absMediaDir, "icons")})
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
	registerRoutes(mux)
	registerGraphQL(mux, supervised, syncer, downloadMgr, trackerMgr, metadataMgr, streamResolver, q)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           corsMiddleware(authMiddleware(cfg.APIToken, logRequests(mux))),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	if cfg.APIToken != "" {
		log.Print("API token required (Authorization: Bearer … or ?token=…)")
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
	fmt.Printf("TSUNAGU_READY url=http://%s version=%s\n", net.JoinHostPort(readyHost, readyPort), serverVersion)
	_ = os.Stdout.Sync()

	go func() {
		log.Printf("tsunagu %s listening on %s", serverVersion, boundAddr)
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Print("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}

func reloadInstalledExtensions(ctx context.Context, sy *sync.Syncer, sc *sandbox.SupervisedClient) error {
	installed, err := sy.ListInstalledExtensions(ctx)
	if err != nil {
		return err
	}
	if len(installed) == 0 {
		return nil
	}

	toLoad := make([]*sandboxv1.ExtensionToLoad, 0, len(installed))
	for _, ext := range installed {
		if !ext.JarPath.Valid || ext.JarPath.String == "" {
			log.Printf("skipping reload of %s: no cached jar_path", ext.PackageName)
			continue
		}
		toLoad = append(toLoad, &sandboxv1.ExtensionToLoad{
			ExtensionId: ext.PackageName,
			JarPath:     ext.JarPath.String,
			ContentType: sandbox.ContentTypeToProto(ext.ContentType),
			Lang:        ext.Lang,
		})
	}
	if len(toLoad) == 0 {
		return nil
	}

	c, err := sc.Ensure(ctx)
	if err != nil {
		return err
	}
	_, err = c.LoadExtensions(ctx, toLoad)
	return err
}

type statusRecorder struct {
	http.ResponseWriter
	code int
}

func (s *statusRecorder) WriteHeader(c int) { s.code = c; s.ResponseWriter.WriteHeader(c) }
func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.code == 0 {
		s.code = 200
	}
	return s.ResponseWriter.Write(b)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sr := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(sr, r)
		p := r.URL.Path
		noisy := strings.HasPrefix(p, "/content/") || strings.HasPrefix(p, "/proxy/")
		if !noisy || sr.code >= 400 {
			log.Printf("%s %s %d %s", r.Method, p, sr.code, time.Since(start))
		}
	})
}

func authMiddleware(token string, next http.Handler) http.Handler {
	if token == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" || strings.HasPrefix(r.URL.Path, "/api/tracker/") || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if got == "" {
			got = r.URL.Query().Get("token")
		}
		if got != token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func registerGraphQL(mux *http.ServeMux, sc *sandbox.SupervisedClient, sy *sync.Syncer, dm *download.Manager, tk *tracker.Manager, md *metadata.Manager, sr *streamresolve.Resolver, q *sqlcgen.Queries) {
	resolver := &graph.Resolver{Sy: sy, Sc: sc, Dm: dm, Ls: localsource.New(q, globalMediaDir), Tk: tk, Md: md, Sr: sr, Q: q, MediaDir: globalMediaDir, Name: serverName, Version: serverVersion, BuildTime: serverBuildTime}
	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))
	srv.Use(extension.FixedComplexityLimit(8000))
	mux.Handle("/api/graphql", withLoaders(q, srv))
	mux.Handle("/api/graphql/playground", playground.Handler("Tsunagu GraphQL", "/api/graphql"))
}

func withLoaders(q *sqlcgen.Queries, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := graph.WithLoaders(r.Context(), q)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}

var globalMediaDir string
