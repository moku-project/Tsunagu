package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"tsunagu/backend/internal/db"
	"tsunagu/backend/internal/db/sqlcgen"
	"tsunagu/backend/internal/httputil"
	"tsunagu/backend/internal/sandbox"
	sandboxv1 "tsunagu/backend/internal/sandbox/gen/sandbox/v1"
	"tsunagu/backend/internal/sync"
)

func main() {
	addr := os.Getenv("TSUNAGU_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	sandboxAddr := os.Getenv("TSUNAGU_SANDBOX_ADDR")
	if sandboxAddr == "" {
		sandboxAddr = "localhost:50051"
	}

	dbPath := os.Getenv("TSUNAGU_DB_PATH")
	if dbPath == "" {
		dbPath = "tsunagu.db"
	}

	cacheDir := os.Getenv("TSUNAGU_JAR_CACHE_DIR")
	if cacheDir == "" {
		cacheDir = "./jar-cache"
	}

	absCacheDir, err := filepath.Abs(cacheDir)
	if err != nil {
		log.Fatalf("resolving cache dir: %v", err)
	}
	cacheDir = absCacheDir

	conn, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("opening db: %v", err)
	}
	defer conn.Close()

	q := sqlcgen.New(conn)
	syncer := sync.New(q, cacheDir)

	sandboxClient, err := sandbox.NewClient(sandboxAddr)
	if err != nil {
		log.Fatalf("connecting to sandbox: %v", err)
	}
	defer sandboxClient.Close()

	if err := reloadInstalledExtensions(context.Background(), syncer, sandboxClient); err != nil {
		log.Printf("warning: could not reload installed extensions on startup: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, sandboxClient, syncer)

	srv := &http.Server{
		Addr:              addr,
		Handler:           logRequests(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("tsunagu backend listening on %s", addr)
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

func reloadInstalledExtensions(ctx context.Context, sy *sync.Syncer, c *sandbox.Client) error {
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

func registerRoutes(mux *http.ServeMux, c *sandbox.Client, sy *sync.Syncer) {
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/repositories", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			params, ok := httputil.RequireParams(w, r, "index_url")
			if !ok {
				return
			}
			repo, err := sy.AddRepository(r.Context(), params["index_url"])
			if err != nil {
				httputil.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
			httputil.JSON(w, repo)
		case http.MethodGet:
			repos, err := sy.ListRepositories(r.Context())
			if err != nil {
				httputil.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			httputil.JSON(w, repos)
		default:
			httputil.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/repositories/extensions", func(w http.ResponseWriter, r *http.Request) {
		params, ok := httputil.RequireParams(w, r, "repository_id")
		if !ok {
			return
		}
		repoID, err := strconv.ParseInt(params["repository_id"], 10, 64)
		if err != nil {
			httputil.Error(w, "invalid repository_id", http.StatusBadRequest)
			return
		}
		exts, err := sy.ListAvailableExtensions(r.Context(), repoID)
		if err != nil {
			httputil.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		httputil.JSON(w, exts)
	})

	mux.HandleFunc("/extensions/install", func(w http.ResponseWriter, r *http.Request) {
		params, ok := httputil.RequireParams(w, r, "package_name")
		if !ok {
			return
		}
		ext, err := sy.InstallExtension(r.Context(), params["package_name"])
		if err != nil {
			httputil.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		loaded, err := c.LoadExtensions(r.Context(), []*sandboxv1.ExtensionToLoad{{
			ExtensionId: ext.PackageName,
			JarPath:     ext.JarPath.String,
			ContentType: sandbox.ContentTypeToProto(ext.ContentType),
			Lang:        ext.Lang,
		}})
		if err != nil {
			httputil.GRPCError(w, err)
			return
		}
		httputil.JSON(w, loaded)
	})

	mux.HandleFunc("/extensions", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			exts, err := sy.ListInstalledExtensions(r.Context())
			if err != nil {
				httputil.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			httputil.JSON(w, exts)
		case http.MethodDelete:
			params, ok := httputil.RequireParams(w, r, "package_name")
			if !ok {
				return
			}
			if _, err := c.UnloadExtension(r.Context(), params["package_name"]); err != nil {
				httputil.GRPCError(w, err)
				return
			}
			ext, err := sy.UninstallExtension(r.Context(), params["package_name"])
			if err != nil {
				httputil.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			httputil.JSON(w, ext)
		default:
			httputil.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/extensions/updates", func(w http.ResponseWriter, r *http.Request) {
		exts, err := sy.CheckForUpdates(r.Context())
		if err != nil {
			httputil.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		httputil.JSON(w, exts)
	})

	mux.HandleFunc("/extensions/update", func(w http.ResponseWriter, r *http.Request) {
		params, ok := httputil.RequireParams(w, r, "package_name")
		if !ok {
			return
		}
		pkg := params["package_name"]

		if _, err := c.UnloadExtension(r.Context(), pkg); err != nil {
			httputil.GRPCError(w, err)
			return
		}
		ext, err := sy.UpdateExtension(r.Context(), pkg)
		if err != nil {
			httputil.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		loaded, err := c.LoadExtensions(r.Context(), []*sandboxv1.ExtensionToLoad{{
			ExtensionId: ext.PackageName,
			JarPath:     ext.JarPath.String,
			ContentType: sandbox.ContentTypeToProto(ext.ContentType),
			Lang:        ext.Lang,
		}})
		if err != nil {
			httputil.GRPCError(w, err)
			return
		}
		httputil.JSON(w, loaded)
	})

	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		params, ok := httputil.RequireParams(w, r, "extension_id", "q")
		if !ok {
			return
		}
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page == 0 {
			page = 1
		}
		resp, err := c.Search(r.Context(), params["extension_id"], params["q"], int32(page))
		if err != nil {
			httputil.GRPCError(w, err)
			return
		}
		httputil.JSON(w, resp)
	})

	mux.HandleFunc("/details", func(w http.ResponseWriter, r *http.Request) {
		params, ok := httputil.RequireParams(w, r, "extension_id", "source_entry_id")
		if !ok {
			return
		}
		resp, err := c.GetDetails(r.Context(), params["extension_id"], params["source_entry_id"])
		if err != nil {
			httputil.GRPCError(w, err)
			return
		}
		httputil.JSON(w, resp)
	})

	mux.HandleFunc("/chapters", func(w http.ResponseWriter, r *http.Request) {
		params, ok := httputil.RequireParams(w, r, "extension_id", "source_entry_id")
		if !ok {
			return
		}
		resp, err := c.GetChapters(r.Context(), params["extension_id"], params["source_entry_id"])
		if err != nil {
			httputil.GRPCError(w, err)
			return
		}
		httputil.JSON(w, resp)
	})

	mux.HandleFunc("/pages", func(w http.ResponseWriter, r *http.Request) {
		params, ok := httputil.RequireParams(w, r, "extension_id", "source_entry_id", "source_chapter_id")
		if !ok {
			return
		}
		resp, err := c.GetPages(r.Context(), params["extension_id"], params["source_entry_id"], params["source_chapter_id"])
		if err != nil {
			httputil.GRPCError(w, err)
			return
		}
		httputil.JSON(w, resp)
	})
}
