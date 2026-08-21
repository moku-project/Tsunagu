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

	"tsunagu/backend/internal/config"
	"tsunagu/backend/internal/db"
	"tsunagu/backend/internal/db/sqlcgen"
	"tsunagu/backend/internal/httputil"
	"tsunagu/backend/internal/sandbox"
	sandboxv1 "tsunagu/backend/internal/sandbox/gen/sandbox/v1"
	"tsunagu/backend/internal/sync"
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
		NovelEnabled:  cfg.NovelEnabled,
		Addr:          cfg.SandboxAddr,
		IdleTimeout:   cfg.IdleTimeout(),
	})
	defer supervised.Shutdown()

	if err := reloadInstalledExtensions(context.Background(), syncer, supervised); err != nil {
		log.Printf("warning: could not reload installed extensions on startup: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, supervised, syncer)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           logRequests(mux),
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

func registerRoutes(mux *http.ServeMux, sc *sandbox.SupervisedClient, sy *sync.Syncer) {
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
			_ = r.ParseForm()
			repo, err := sy.AddRepository(r.Context(), params["index_url"], r.Form.Get("name"))
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
		case http.MethodDelete:
			params, ok := httputil.RequireParams(w, r, "repository_id")
			if !ok {
				return
			}
			repoID, err := strconv.ParseInt(params["repository_id"], 10, 64)
			if err != nil {
				httputil.Error(w, "invalid repository_id", http.StatusBadRequest)
				return
			}
			if err := sy.DeleteRepository(r.Context(), repoID); err != nil {
				httputil.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			httputil.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/repositories/rename", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httputil.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		params, ok := httputil.RequireParams(w, r, "repository_id", "name")
		if !ok {
			return
		}
		repoID, err := strconv.ParseInt(params["repository_id"], 10, 64)
		if err != nil {
			httputil.Error(w, "invalid repository_id", http.StatusBadRequest)
			return
		}
		repo, err := sy.RenameRepository(r.Context(), repoID, params["name"])
		if err != nil {
			httputil.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		httputil.JSON(w, repo)
	})

	mux.HandleFunc("/library", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			params, ok := httputil.RequireParams(w, r, "extension_id", "source_entry_id")
			if !ok {
				return
			}
			client, err := sc.Ensure(r.Context())
			if err != nil {
				httputil.Error(w, "sandbox unavailable: "+err.Error(), http.StatusServiceUnavailable)
				return
			}
			entry, err := sy.AddToLibrary(r.Context(), client, params["extension_id"], params["source_entry_id"])
			if err != nil {
				httputil.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			httputil.JSON(w, entry)
		case http.MethodGet:
			entries, err := sy.ListLibraryEntries(r.Context(), r.URL.Query().Get("content_type"))
			if err != nil {
				httputil.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			httputil.JSON(w, entries)
		default:
			httputil.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/library/reading-status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httputil.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		params, ok := httputil.RequireParams(w, r, "library_entry_id", "system_key")
		if !ok {
			return
		}
		entryID, err := strconv.ParseInt(params["library_entry_id"], 10, 64)
		if err != nil {
			httputil.Error(w, "invalid library_entry_id", http.StatusBadRequest)
			return
		}
		if err := sy.SetReadingStatus(r.Context(), entryID, params["system_key"]); err != nil {
			httputil.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("/library/chapters/sync", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httputil.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		params, ok := httputil.RequireParams(w, r, "library_entry_id")
		if !ok {
			return
		}
		entryID, err := strconv.ParseInt(params["library_entry_id"], 10, 64)
		if err != nil {
			httputil.Error(w, "invalid library_entry_id", http.StatusBadRequest)
			return
		}
		client, err := sc.Ensure(r.Context())
		if err != nil {
			httputil.Error(w, "sandbox unavailable: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		chapters, err := sy.SyncChapters(r.Context(), client, entryID)
		if err != nil {
			httputil.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		httputil.JSON(w, chapters)
	})

	mux.HandleFunc("/reading-progress", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			params, ok := httputil.RequireParams(w, r, "library_entry_id", "chapter_id", "progress")
			if !ok {
				return
			}
			entryID, err := strconv.ParseInt(params["library_entry_id"], 10, 64)
			if err != nil {
				httputil.Error(w, "invalid library_entry_id", http.StatusBadRequest)
				return
			}
			chapterID, err := strconv.ParseInt(params["chapter_id"], 10, 64)
			if err != nil {
				httputil.Error(w, "invalid chapter_id", http.StatusBadRequest)
				return
			}
			progress, err := strconv.ParseFloat(params["progress"], 64)
			if err != nil {
				httputil.Error(w, "invalid progress", http.StatusBadRequest)
				return
			}
			completed := r.URL.Query().Get("completed") == "true" || r.Form.Get("completed") == "true"
			var positionSeconds, durationSeconds *float64
			if v := r.Form.Get("position_seconds"); v != "" {
				parsed, err := strconv.ParseFloat(v, 64)
				if err != nil {
					httputil.Error(w, "invalid position_seconds", http.StatusBadRequest)
					return
				}
				positionSeconds = &parsed
			}
			if v := r.Form.Get("duration_seconds"); v != "" {
				parsed, err := strconv.ParseFloat(v, 64)
				if err != nil {
					httputil.Error(w, "invalid duration_seconds", http.StatusBadRequest)
					return
				}
				durationSeconds = &parsed
			}
			result, err := sy.RecordProgress(r.Context(), entryID, chapterID, progress, completed, positionSeconds, durationSeconds)
			if err != nil {
				httputil.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			httputil.JSON(w, result)
		case http.MethodGet:
			params, ok := httputil.RequireParams(w, r, "library_entry_id")
			if !ok {
				return
			}
			entryID, err := strconv.ParseInt(params["library_entry_id"], 10, 64)
			if err != nil {
				httputil.Error(w, "invalid library_entry_id", http.StatusBadRequest)
				return
			}
			list, err := sy.ListReadingProgress(r.Context(), entryID)
			if err != nil {
				httputil.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			httputil.JSON(w, list)
		default:
			httputil.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/reading-progress/mark-read", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httputil.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		params, ok := httputil.RequireParams(w, r, "library_entry_id", "chapter_id")
		if !ok {
			return
		}
		entryID, err := strconv.ParseInt(params["library_entry_id"], 10, 64)
		if err != nil {
			httputil.Error(w, "invalid library_entry_id", http.StatusBadRequest)
			return
		}
		chapterID, err := strconv.ParseInt(params["chapter_id"], 10, 64)
		if err != nil {
			httputil.Error(w, "invalid chapter_id", http.StatusBadRequest)
			return
		}
		result, err := sy.MarkChapterRead(r.Context(), entryID, chapterID)
		if err != nil {
			httputil.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		httputil.JSON(w, result)
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

		client, err := sc.Ensure(r.Context())
		if err != nil {
			httputil.Error(w, "sandbox unavailable: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		loaded, err := client.LoadExtensions(r.Context(), []*sandboxv1.ExtensionToLoad{{
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
			client, err := sc.Ensure(r.Context())
			if err != nil {
				httputil.Error(w, "sandbox unavailable: "+err.Error(), http.StatusServiceUnavailable)
				return
			}
			if _, err := client.UnloadExtension(r.Context(), params["package_name"]); err != nil {
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

		client, err := sc.Ensure(r.Context())
		if err != nil {
			httputil.Error(w, "sandbox unavailable: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		if _, err := client.UnloadExtension(r.Context(), pkg); err != nil {
			httputil.GRPCError(w, err)
			return
		}
		ext, err := sy.UpdateExtension(r.Context(), pkg)
		if err != nil {
			httputil.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		loaded, err := client.LoadExtensions(r.Context(), []*sandboxv1.ExtensionToLoad{{
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
		client, err := sc.Ensure(r.Context())
		if err != nil {
			httputil.Error(w, "sandbox unavailable: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		resp, err := client.Search(r.Context(), params["extension_id"], params["q"], int32(page))
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
		client, err := sc.Ensure(r.Context())
		if err != nil {
			httputil.Error(w, "sandbox unavailable: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		resp, err := client.GetDetails(r.Context(), params["extension_id"], params["source_entry_id"])
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
		client, err := sc.Ensure(r.Context())
		if err != nil {
			httputil.Error(w, "sandbox unavailable: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		resp, err := client.GetChapters(r.Context(), params["extension_id"], params["source_entry_id"])
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
		client, err := sc.Ensure(r.Context())
		if err != nil {
			httputil.Error(w, "sandbox unavailable: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		resp, err := client.GetPages(r.Context(), params["extension_id"], params["source_entry_id"], params["source_chapter_id"])
		if err != nil {
			httputil.GRPCError(w, err)
			return
		}
		httputil.JSON(w, resp)
	})

	mux.HandleFunc("/chapter-text", func(w http.ResponseWriter, r *http.Request) {
		params, ok := httputil.RequireParams(w, r, "extension_id", "source_entry_id", "source_chapter_id")
		if !ok {
			return
		}
		client, err := sc.Ensure(r.Context())
		if err != nil {
			httputil.Error(w, "sandbox unavailable: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		resp, err := client.GetChapterText(r.Context(), params["extension_id"], params["source_entry_id"], params["source_chapter_id"])
		if err != nil {
			httputil.GRPCError(w, err)
			return
		}
		httputil.JSON(w, resp)
	})

	mux.HandleFunc("/episodes", func(w http.ResponseWriter, r *http.Request) {
		params, ok := httputil.RequireParams(w, r, "extension_id", "source_entry_id")
		if !ok {
			return
		}
		client, err := sc.Ensure(r.Context())
		if err != nil {
			httputil.Error(w, "sandbox unavailable: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		resp, err := client.GetEpisodes(r.Context(), params["extension_id"], params["source_entry_id"])
		if err != nil {
			httputil.GRPCError(w, err)
			return
		}
		httputil.JSON(w, resp)
	})

	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		params, ok := httputil.RequireParams(w, r, "extension_id", "source_entry_id", "source_episode_id")
		if !ok {
			return
		}
		client, err := sc.Ensure(r.Context())
		if err != nil {
			httputil.Error(w, "sandbox unavailable: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		resp, err := client.GetVideoStream(r.Context(), params["extension_id"], params["source_entry_id"], params["source_episode_id"])
		if err != nil {
			httputil.GRPCError(w, err)
			return
		}
		httputil.JSON(w, resp)
	})
}