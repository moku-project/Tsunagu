package server

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"tsunagu/backend/internal/api/graph"
	"tsunagu/backend/internal/auth"
	"tsunagu/backend/internal/db/sqlcgen"
	"tsunagu/backend/internal/sandbox"
	sandboxv1 "tsunagu/backend/internal/sandbox/gen/sandbox/v1"
	"tsunagu/backend/internal/sync"
)

func reloadInstalledExtensions(ctx context.Context, sy *sync.Syncer, sc *sandbox.SupervisedClient) error {
	installed, err := sy.ListInstalledExtensions(ctx)
	if err != nil {
		return err
	}
	if len(installed) == 0 {
		return nil
	}

	toLoad := make([]*sandboxv1.ExtensionToLoad, 0, len(installed))
	sandboxOnly := 0
	for _, ext := range installed {
		// Sideloads live only in the sandbox's extensions dir.
		if !ext.JarPath.Valid || ext.JarPath.String == "" {
			sandboxOnly++
			continue
		}
		toLoad = append(toLoad, &sandboxv1.ExtensionToLoad{
			ExtensionId: ext.PackageName,
			JarPath:     ext.JarPath.String,
			ContentType: sandbox.ContentTypeToProto(ext.ContentType),
			Lang:        ext.Lang,
		})
	}
	if sandboxOnly > 0 {
		log.Printf("%d extensions without a cached jar load from the sandbox extensions dir", sandboxOnly)
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
		fromQuery := false
		if got == "" {
			got = r.URL.Query().Get("token")
			fromQuery = got != ""
		}
		if got != "" {
			ok := token != "" && subtle.ConstantTimeCompare([]byte(got), []byte(token)) == 1
			if !ok && passwordSet {
				ok = am.VerifySession(r.Context(), got)
			}
			if ok {
				if fromQuery {
					// Proxied streams forward their query upstream.
					q := r.URL.Query()
					q.Del("token")
					r.URL.RawQuery = q.Encode()
				}
				next.ServeHTTP(w, r)
				return
			}
		}
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
}

func registerGraphQL(mux *http.ServeMux, resolver *graph.Resolver) {
	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))
	srv.Use(extension.FixedComplexityLimit(8000))
	srv.SetErrorPresenter(sandboxErrorPresenter)
	mux.Handle("/api/graphql", withLoaders(resolver.Q, srv))
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
