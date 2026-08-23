package main


import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"

	"tsunagu/backend/internal/api/graph"
	"tsunagu/backend/internal/config"
	"tsunagu/backend/internal/db"
	"tsunagu/backend/internal/db/sqlcgen"
	"tsunagu/backend/internal/download"
	"tsunagu/backend/internal/sandbox"
	sandboxv1 "tsunagu/backend/internal/sandbox/gen/sandbox/v1"
	"tsunagu/backend/internal/sync"
)

const (
	serverName      = "Tsunagu"
	serverVersion   = "0.1.0"
	serverBuildTime = "dev"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("loading config: %v", err)
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
	syncer := sync.New(conn, q, cfg.JarCacheDir)

	supervised := sandbox.NewSupervised(sandbox.SupervisedConfig{
		JarPath:       cfg.SandboxJarPath,
		Port:          cfg.SandboxPort,
		ExtensionsDir: cfg.SandboxExtDir,
		StorageDir:    cfg.SandboxStorageDir,
		NovelEnabled:  cfg.NovelEnabled,
		Addr:          cfg.SandboxAddr,
		IdleTimeout:   cfg.IdleTimeout(),
	})
	defer supervised.Shutdown()

	if err := reloadInstalledExtensions(context.Background(), syncer, supervised); err != nil {
		log.Printf("warning: could not reload installed extensions on startup: %v", err)
	}

	absMediaDir, err := filepath.Abs(cfg.MediaDir)
	if err != nil {
		log.Fatalf("resolving media dir: %v", err)
	}
	downloadMgr := download.New(q, supervised, absMediaDir)
	downloadMgr.Start()
	defer downloadMgr.Shutdown()

	globalMediaDir = absMediaDir

	mux := http.NewServeMux()
	mux.Handle("/media/", http.StripPrefix("/media/", http.FileServer(http.Dir(absMediaDir))))
	registerRoutes(mux)
	registerGraphQL(mux, supervised, syncer, downloadMgr, q)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           corsMiddleware(logRequests(mux)),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("tsunagu backend listening on %s", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
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

func registerGraphQL(mux *http.ServeMux, sc *sandbox.SupervisedClient, sy *sync.Syncer, dm *download.Manager, q *sqlcgen.Queries) {
	resolver := &graph.Resolver{Sy: sy, Sc: sc, Dm: dm, Q: q, MediaDir: globalMediaDir, Name: serverName, Version: serverVersion, BuildTime: serverBuildTime}
	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))
	mux.Handle("/api/graphql", srv)
	mux.Handle("/api/graphql/playground", playground.Handler("Tsunagu GraphQL", "/api/graphql"))
}

func registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}

var globalMediaDir string
