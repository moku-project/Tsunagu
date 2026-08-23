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

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"

	"tsunagu/backend/internal/api/graph"
	"tsunagu/backend/internal/config"
	"tsunagu/backend/internal/db"
	"tsunagu/backend/internal/db/sqlcgen"
	"tsunagu/backend/internal/download"
	"tsunagu/backend/internal/httputil"
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
	registerRoutes(mux, supervised, syncer, q, absMediaDir)
	registerGraphQL(mux, supervised, syncer, downloadMgr, q)

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

func registerGraphQL(mux *http.ServeMux, sc *sandbox.SupervisedClient, sy *sync.Syncer, dm *download.Manager, q *sqlcgen.Queries) {
	resolver := &graph.Resolver{Sy: sy, Sc: sc, Dm: dm, Q: q, MediaDir: globalMediaDir, Name: serverName, Version: serverVersion, BuildTime: serverBuildTime}
	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))
	mux.Handle("/api/graphql", srv)
	mux.Handle("/api/graphql/playground", playground.Handler("Tsunagu GraphQL", "/api/graphql"))
}

func registerRoutes(mux *http.ServeMux, sc *sandbox.SupervisedClient, sy *sync.Syncer, q *sqlcgen.Queries, mediaDir string) {
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

		if localURLs, found := localMangaPageURLs(r.Context(), q, params["extension_id"], params["source_entry_id"], params["source_chapter_id"]); found {
			httputil.JSON(w, map[string]any{"page_urls": localURLs})
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

		if content, format, found := localNovelChapterContent(r.Context(), q, params["extension_id"], params["source_entry_id"], params["source_chapter_id"]); found {
			httputil.JSON(w, map[string]any{"content": content, "format": format})
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
		if localURL, found := localAnimeEpisodeStreamURL(r.Context(), q, params["extension_id"], params["source_entry_id"], params["source_episode_id"]); found {
			httputil.JSON(w, map[string]any{"stream_url": localURL, "local": true})
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

func localMangaPageURLs(ctx context.Context, q *sqlcgen.Queries, extensionID, sourceEntryID, sourceChapterID string) ([]string, bool) {
	chapter, ok := resolveLocalChapter(ctx, q, extensionID, sourceEntryID, sourceChapterID)
	if !ok {
		return nil, false
	}
	pages, err := q.ListMangaPages(ctx, chapter.ID)
	if err != nil || len(pages) == 0 {
		return nil, false
	}
	urls := make([]string, 0, len(pages))
	for _, p := range pages {
		if !p.LocalPath.Valid || p.LocalPath.String == "" {
			return nil, false
		}
		urls = append(urls, localPathToMediaURL(p.LocalPath.String))
	}
	return urls, true
}

func localAnimeEpisodeStreamURL(ctx context.Context, q *sqlcgen.Queries, extensionID, sourceEntryID, sourceEpisodeID string) (string, bool) {
	chapter, ok := resolveLocalChapter(ctx, q, extensionID, sourceEntryID, sourceEpisodeID)
	if !ok {
		return "", false
	}
	row, err := q.GetAnimeEpisodeStream(ctx, chapter.ID)
	if err != nil || !row.LocalPath.Valid || row.LocalPath.String == "" {
		return "", false
	}
	return localPathToMediaURL(row.LocalPath.String), true
}

func localNovelChapterContent(ctx context.Context, q *sqlcgen.Queries, extensionID, sourceEntryID, sourceChapterID string) (string, string, bool) {
	chapter, ok := resolveLocalChapter(ctx, q, extensionID, sourceEntryID, sourceChapterID)
	if !ok {
		return "", "", false
	}
	row, err := q.GetNovelChapterContent(ctx, chapter.ID)
	if err != nil || !row.LocalPath.Valid || row.LocalPath.String == "" {
		return "", "", false
	}
	data, err := os.ReadFile(row.LocalPath.String)
	if err != nil {
		return "", "", false
	}
	format := "html"
	if ext := filepath.Ext(row.LocalPath.String); ext != "" {
		format = ext[1:]
	}
	return string(data), format, true
}

func resolveLocalChapter(ctx context.Context, q *sqlcgen.Queries, extensionID, sourceEntryID, sourceChapterID string) (sqlcgen.Chapter, bool) {
	entry, err := q.GetLibraryEntryByExtensionAndExternalID(ctx, sqlcgen.GetLibraryEntryByExtensionAndExternalIDParams{
		PackageName: extensionID,
		ExternalID:  sourceEntryID,
	})
	if err != nil {
		return sqlcgen.Chapter{}, false
	}
	chapter, err := q.GetChapterByLibraryEntryAndExternalID(ctx, sqlcgen.GetChapterByLibraryEntryAndExternalIDParams{
		LibraryEntryID: entry.ID,
		ExternalID:     sourceChapterID,
	})
	if err != nil {
		return sqlcgen.Chapter{}, false
	}
	return chapter, true
}

func localPathToMediaURL(absPath string) string {
	rel, err := filepath.Rel(globalMediaDir, absPath)
	if err != nil {
		return absPath
	}
	return "/media/" + filepath.ToSlash(rel)
}

var globalMediaDir string
