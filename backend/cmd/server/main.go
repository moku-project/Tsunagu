package main

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html"
	"io"
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

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"tsunagu/backend/internal/api/graph"
	"tsunagu/backend/internal/api/rest"
	"tsunagu/backend/internal/auth"
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
	log.Printf("=== %s %s (%s) starting, pid %d ===", serverName, serverVersion, serverBuildTime, os.Getpid())
}

func main() {
	dataDir := flag.String("data-dir", "", "root directory for the DB, caches, extensions and downloads")
	configPath := flag.String("config", "", "path to tsunagu.toml (defaults to <data-dir>/tsunagu.toml)")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Printf("%s %s (%s)\n", serverName, serverVersion, serverBuildTime)
		return
	}
	if flag.NArg() > 0 && flag.Arg(0) == "config" {
		runConfigCLI(flag.Args()[1:], config.Options{ConfigPath: *configPath, DataDir: *dataDir})
		return
	}

	setupFileLog(*dataDir)

	bootCfg, tomlPath, activeKeys, err := config.Load(config.Options{ConfigPath: *configPath, DataDir: *dataDir})
	if err != nil {
		log.Fatalf("loading config: %v", err)
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
			log.Fatalf("listen: %v", err)
		}
		log.Printf("address %s unavailable, bound %s instead", wantAddr, ln.Addr())
	}
	boundAddr := ln.Addr().String()

	conn, err := db.Open(bootCfg.DBPath)
	if err != nil {
		log.Fatalf("opening db: %v", err)
	}
	defer conn.Close()
	q := sqlcgen.New(conn)
	globalDB = conn

	fsDir, _ := filepath.Abs(filepath.Join(filepath.Dir(bootCfg.DBPath), "flaresolverr"))
	fsMgr := flaresolverr.NewManager(fsDir)
	defer fsMgr.Shutdown()
	globalFsMgr = fsMgr

	store := config.NewStore(bootCfg, q, tomlPath, activeKeys)

	cfMgr, err := contentfilter.New(q)
	if err != nil {
		log.Fatalf("content filter: %v", err)
	}
	store.OnChange("content_filter_level", func(context.Context) {
		cfMgr.SetLevel(contentfilter.ParseLevel(store.Config().ContentFilterLevel))
	})
	globalCf = cfMgr

	if err := store.Sync(context.Background()); err != nil {
		log.Fatalf("syncing config: %v", err)
	}
	fsMgr.ApplyConfig(store.Config().CloudflareSolverMode, store.Config().CloudflareSolverURL)
	cfMgr.SetLevel(contentfilter.ParseLevel(store.Config().ContentFilterLevel))
	globalStore = store

	cfg := store.Config()
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
	metadataMgr.SetRecomputer(cfMgr)
	syncer.SetRecomputer(cfMgr)

	supervised := sandbox.NewSupervised(sandbox.SupervisedConfig{
		JarPath:       cfg.SandboxJarPath,
		Port:          cfg.SandboxPort,
		ExtensionsDir: cfg.SandboxExtDir,
		StorageDir:    cfg.SandboxStorageDir,
		NovelEnabled:  cfg.NovelEnabled,
		Addr:          cfg.SandboxAddr,
		IdleTimeout:   cfg.IdleTimeout(),
		HeapMB:        cfg.SandboxHeapMB,
		FlareSolverrURLFunc: func() string {
			if store.Config().CloudflareSolverMode == flaresolverr.ModeDisabled {
				return ""
			}
			return strings.TrimRight(cfg.PublicURL, "/") + "/internal/flaresolverr"
		},
	})
	defer supervised.Shutdown()

	applyCloudflare := func(ctx context.Context) {
		c := store.Config()
		fsMgr.ApplyConfig(c.CloudflareSolverMode, c.CloudflareSolverURL)
		// sandbox only reads SANDBOX_FLARESOLVERR_URL at spawn, so kick it
		supervised.Restart()
	}
	store.OnChange("cloudflare_solver_mode", applyCloudflare)
	store.OnChange("cloudflare_solver_url", applyCloudflare)

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
	defer downloadMgr.Shutdown()
	store.OnChange("downloads_dir", func(context.Context) {
		downloadMgr.SetDownloadsDir(resolveDownloadsDir())
	})
	store.OnChange("manga_download_format", func(context.Context) {
		downloadMgr.SetImageFormat(store.Config().MangaDownloadFormat)
	})

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
				if _, err := backup.CreateSnapshot(context.Background(), globalDB, backupDir); err != nil {
					log.Printf("scheduled backup: %v", err)
					continue
				}
				if err := backup.PruneSnapshots(backupDir, cfg.BackupRetentionCount); err != nil {
					log.Printf("scheduled backup: prune: %v", err)
				}
			}
		}()
	}

	globalMediaDir = absMediaDir

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
	registerGraphQL(mux, supervised, syncer, downloadMgr, trackerMgr, metadataMgr, streamResolver, q, authMgr, localScanner)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           corsMiddleware(authMiddleware(cfg.APIToken, authMgr, logRequests(mux))),
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
	log.Printf("reloading %d installed extensions", len(toLoad))

	c, err := sc.Ensure(ctx)
	if err != nil {
		return err
	}
	resp, err := c.LoadExtensions(ctx, toLoad)
	if err != nil {
		return err
	}
	byPackage := make(map[string]sqlcgen.Extension, len(installed))
	for _, ext := range installed {
		byPackage[ext.PackageName] = ext
	}
	for _, loaded := range resp.GetExtensions() {
		if ext, ok := byPackage[loaded.GetId()]; ok {
			sy.PersistExtensionMeta(ctx, ext, loaded)
		}
	}
	return nil
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
		dur := time.Since(start)
		bulky := strings.HasPrefix(p, "/content/") || strings.HasPrefix(p, "/proxy/")
		chatty := p == "/api/graphql" || p == "/healthz"
		quiet := bulky || (chatty && dur < 500*time.Millisecond)
		if !quiet || sr.code >= 400 {
			log.Printf("%s %s %d %s", r.Method, p, sr.code, dur)
		}
	})
}

func authMiddleware(token string, am *auth.Manager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" || r.URL.Path == "/api/auth/status" || r.URL.Path == "/api/auth/login" ||
			strings.HasPrefix(r.URL.Path, "/api/tracker/") || strings.HasPrefix(r.URL.Path, "/internal/") || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		passwordSet := am.PasswordSet(r.Context())
		if token == "" && !passwordSet {
			next.ServeHTTP(w, r)
			return
		}
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if got == "" {
			got = r.URL.Query().Get("token")
		}
		if got != "" {
			if token != "" && subtle.ConstantTimeCompare([]byte(got), []byte(token)) == 1 {
				next.ServeHTTP(w, r)
				return
			}
			if passwordSet && am.VerifySession(r.Context(), got) {
				next.ServeHTTP(w, r)
				return
			}
		}
		http.Error(w, "unauthorized", http.StatusUnauthorized)
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

func registerGraphQL(mux *http.ServeMux, sc *sandbox.SupervisedClient, sy *sync.Syncer, dm *download.Manager, tk *tracker.Manager, md *metadata.Manager, sr *streamresolve.Resolver, q *sqlcgen.Queries, am *auth.Manager, ls *localsource.Scanner) {
	resolver := &graph.Resolver{Sy: sy, Sc: sc, Dm: dm, Ls: ls, Tk: tk, Md: md, Sr: sr, Q: q, DB: globalDB, Fs: globalFsMgr, Cfg: globalStore, Cf: globalCf, Am: am, MediaDir: globalMediaDir, Name: serverName, Version: serverVersion, BuildTime: serverBuildTime}
	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))
	srv.Use(extension.FixedComplexityLimit(8000))
	srv.SetErrorPresenter(sandboxErrorPresenter)
	mux.Handle("/api/graphql", withLoaders(q, srv))
	mux.Handle("/api/graphql/playground", playground.Handler("Tsunagu GraphQL", "/api/graphql"))
}

func sandboxErrorPresenter(ctx context.Context, e error) *gqlerror.Error {
	gqlErr := graphql.DefaultErrorPresenter(ctx, e)

	var st *status.Status
	for err := e; err != nil; err = errors.Unwrap(err) {
		if s, ok := status.FromError(err); ok && s.Code() != codes.OK {
			st = s
			break
		}
	}
	if st == nil {
		if errors.Is(e, context.DeadlineExceeded) || errors.Is(e, context.Canceled) {
			if gqlErr.Extensions == nil {
				gqlErr.Extensions = map[string]any{}
			}
			gqlErr.Extensions["code"] = "SOURCE_NETWORK"
			gqlErr.Message = "context deadline exceeded"
		}
		return gqlErr
	}

	code := "SOURCE_ERROR"
	msg := st.Message()
	if i := strings.Index(msg, ": "); i > 0 && (strings.HasPrefix(msg[:i], "SOURCE_") || msg[:i] == "INTERNAL") {
		code = msg[:i]
		msg = msg[i+2:]
	} else {
		switch st.Code() {
		case codes.NotFound:
			code = "SOURCE_NOT_FOUND"
		case codes.Unavailable:
			code = "SOURCE_UNAVAILABLE"
		case codes.DeadlineExceeded, codes.Canceled:
			code = "SOURCE_NETWORK"
		case codes.ResourceExhausted:
			code = "SOURCE_RATE_LIMITED"
		case codes.FailedPrecondition:
			code = "SOURCE_CLOUDFLARE"
		case codes.DataLoss:
			code = "SOURCE_PARSE"
		}
	}

	if gqlErr.Extensions == nil {
		gqlErr.Extensions = map[string]any{}
	}
	gqlErr.Extensions["code"] = code
	gqlErr.Extensions["grpc"] = st.Code().String()
	gqlErr.Message = msg
	return gqlErr
}

func withLoaders(q *sqlcgen.Queries, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := graph.WithLoaders(r.Context(), q)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func registerRoutes(mux *http.ServeMux, am *auth.Manager) {
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/api/auth/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]bool{"passwordRequired": am.PasswordSet(r.Context())})
	})

	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Password string `json:"password"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		token, exp, err := am.Login(r.Context(), req.Password)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"token": token, "expiresAt": exp.UTC().Format(time.RFC3339)})
	})
}

var globalMediaDir string
var globalDB *sql.DB
var globalFsMgr *flaresolverr.Manager
var globalStore *config.Store
var globalCf *contentfilter.Manager
