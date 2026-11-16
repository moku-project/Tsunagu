package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/graphql/handler"

	"tsunagu/backend/internal/db"
	"tsunagu/backend/internal/db/sqlcgen"
)

func TestReadingHistoryThroughGraphQL(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	for _, q := range []string{
		`INSERT INTO repositories (index_url, name) VALUES ('https://example.com/index.json', 'Example')`,
		`INSERT INTO extensions (repository_id, package_name, name, version, content_type, lang, apk_url, installed) VALUES (1, 'example.source', 'Example', '1', 'manga', 'en', 'x.apk', 1)`,
		`INSERT INTO media (id, extension_id, external_id, content_type, title) VALUES (1, 1, '/a', 'manga', 'The Lantern Keeper')`,
		`INSERT INTO chapters (id, media_id, external_id, title, number) VALUES (10, 1, '/a/1', 'Chapter 1', 1)`,
	} {
		if _, err := conn.ExecContext(context.Background(), q); err != nil {
			t.Fatal(err)
		}
	}
	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{DB: conn, Q: sqlcgen.New(conn)}}))
	gql := func(q string) string {
		body, _ := json.Marshal(map[string]string{"query": q})
		req := httptest.NewRequest(http.MethodPost, "/api/graphql", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		return rec.Body.String()
	}
	out := gql(`mutation { recordReadingSession(input: {id: "s1", mediaId: "1", startChapterId: "10", endChapterId: "10",
		startPage: 1, endPage: 14, startedAt: "2026-11-10T20:00:00Z", endedAt: "2026-11-10T20:25:00Z", durationMs: 1500000, chapters: 1})
		{ id media { title } endChapter { title } endPage durationMs } }`)
	if !strings.Contains(out, `"title":"The Lantern Keeper"`) || !strings.Contains(out, `"endPage":14`) {
		t.Fatalf("record: %s", out)
	}
	out = gql(`mutation { importReadingHistory(source: "moku-1", days: [{date: "2026-11-09", seconds: 600, chapters: 2}],
		sessions: [{id: "old", mediaId: "77", startPage: 0, endPage: 0, startedAt: "2026-11-09T10:00:00Z", endedAt: "2026-11-09T10:10:00Z", durationMs: 600000, chapters: 2}])
		{ imported skipped } }`)
	if !strings.Contains(out, `{"imported":0,"skipped":1}`) {
		t.Fatalf("import: %s", out)
	}
	out = gql(`{ readingSessions(limit: 5) { id } readingStats(timeZone: "UTC") { totalMinutesRead longestStreakDays days { date seconds chapters } } }`)
	if !strings.Contains(out, `"readingSessions":[{"id":"s1"}]`) || !strings.Contains(out, `"totalMinutesRead":35`) || !strings.Contains(out, `"longestStreakDays":2`) {
		t.Fatalf("history and stats: %s", out)
	}
	if out := gql(`{ readingStats(timeZone: "Mars/Olympus") { totalMinutesRead } }`); !strings.Contains(out, "unknown time zone") {
		t.Fatalf("bad time zone: %s", out)
	}
}
