package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOriginAllowed(t *testing.T) {
	extra := []string{"https://moku.example.com"}
	cases := map[string]bool{
		"tauri://localhost":        true,
		"http://tauri.localhost":   true,
		"http://localhost:1420":    true,
		"http://127.0.0.1:5173":    true,
		"http://[::1]:3000":        true,
		"http://192.168.1.5:6007":  true, // same host as the request
		"https://moku.example.com": true,
		"https://evil.example":     false,
		"null":                     false,
		"http://localhost.evil.io": false,
	}
	for origin, want := range cases {
		if got := originAllowed(origin, "192.168.1.5:6007", extra); got != want {
			t.Errorf("originAllowed(%q) = %v, want %v", origin, got, want)
		}
	}
}

func TestCORSBlocksCrossSitePosts(t *testing.T) {
	reached := false
	h := corsMiddleware("", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached = true }))

	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:6007/api/graphql", strings.NewReader("x"))
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden || reached {
		t.Fatalf("cross-site POST: code %d, reached %v", rec.Code, reached)
	}

	req = httptest.NewRequest(http.MethodPost, "http://127.0.0.1:6007/api/graphql", strings.NewReader("x"))
	req.Header.Set("Origin", "tauri://localhost")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if !reached || rec.Header().Get("Access-Control-Allow-Origin") != "tauri://localhost" {
		t.Fatalf("app POST blocked: code %d", rec.Code)
	}
}
