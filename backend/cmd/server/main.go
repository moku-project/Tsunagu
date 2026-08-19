// main.go
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	"tsunagu/backend/internal/sandbox"
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

	sandboxClient, err := sandbox.NewClient(sandboxAddr)
	if err != nil {
		log.Fatalf("connecting to sandbox: %v", err)
	}
	defer sandboxClient.Close()

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/repositories", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			indexURL := r.URL.Query().Get("index_url")
			if indexURL == "" {
				http.Error(w, "index_url is required", http.StatusBadRequest)
				return
			}
			resp, err := sandboxClient.AddRepository(r.Context(), indexURL)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		case http.MethodGet:
			resp, err := sandboxClient.ListRepositories(r.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/repositories/extensions", func(w http.ResponseWriter, r *http.Request) {
		repositoryID := r.URL.Query().Get("repository_id")
		if repositoryID == "" {
			http.Error(w, "repository_id is required", http.StatusBadRequest)
			return
		}

		resp, err := sandboxClient.ListAvailableExtensions(r.Context(), repositoryID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/extensions/install", func(w http.ResponseWriter, r *http.Request) {
		repositoryID := r.URL.Query().Get("repository_id")
		extensionID := r.URL.Query().Get("extension_id")
		if repositoryID == "" || extensionID == "" {
			http.Error(w, "repository_id and extension_id are required", http.StatusBadRequest)
			return
		}

		resp, err := sandboxClient.InstallExtension(r.Context(), repositoryID, extensionID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/extensions", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			resp, err := sandboxClient.ListInstalledExtensions(r.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		case http.MethodDelete:
			extensionID := r.URL.Query().Get("extension_id")
			if extensionID == "" {
				http.Error(w, "extension_id is required", http.StatusBadRequest)
				return
			}
			resp, err := sandboxClient.UninstallExtension(r.Context(), extensionID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		extensionID := r.URL.Query().Get("extension_id")
		query := r.URL.Query().Get("q")
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page == 0 {
			page = 1
		}

		if extensionID == "" || query == "" {
			http.Error(w, "extension_id and q are required", http.StatusBadRequest)
			return
		}

		resp, err := sandboxClient.Search(r.Context(), extensionID, query, int32(page))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/details", func(w http.ResponseWriter, r *http.Request) {
		extensionID := r.URL.Query().Get("extension_id")
		sourceEntryID := r.URL.Query().Get("source_entry_id")

		if extensionID == "" || sourceEntryID == "" {
			http.Error(w, "extension_id and source_entry_id are required", http.StatusBadRequest)
			return
		}

		resp, err := sandboxClient.GetDetails(r.Context(), extensionID, sourceEntryID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/chapters", func(w http.ResponseWriter, r *http.Request) {
		extensionID := r.URL.Query().Get("extension_id")
		sourceEntryID := r.URL.Query().Get("source_entry_id")

		if extensionID == "" || sourceEntryID == "" {
			http.Error(w, "extension_id and source_entry_id are required", http.StatusBadRequest)
			return
		}

		resp, err := sandboxClient.GetChapters(r.Context(), extensionID, sourceEntryID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/pages", func(w http.ResponseWriter, r *http.Request) {
		extensionID := r.URL.Query().Get("extension_id")
		sourceEntryID := r.URL.Query().Get("source_entry_id")
		sourceChapterID := r.URL.Query().Get("source_chapter_id")

		if extensionID == "" || sourceEntryID == "" || sourceChapterID == "" {
			http.Error(w, "extension_id, source_entry_id, and source_chapter_id are required", http.StatusBadRequest)
			return
		}

		resp, err := sandboxClient.GetPages(r.Context(), extensionID, sourceEntryID, sourceChapterID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	log.Printf("tsunagu backend listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}