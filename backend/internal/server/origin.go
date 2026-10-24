package main

import (
	"net"
	"net/http"
	"net/url"
	"strings"
)

// originAllowed reports whether a browser page at origin may talk to this server.
// Form POSTs skip the CORS preflight, so the origin is checked here.
func originAllowed(origin, host string, extra []string) bool {
	switch origin {
	case "tauri://localhost", "http://tauri.localhost", "https://tauri.localhost":
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	if strings.EqualFold(u.Host, host) {
		return true
	}
	if ip := net.ParseIP(u.Hostname()); (ip != nil && ip.IsLoopback()) || strings.EqualFold(u.Hostname(), "localhost") {
		return true
	}
	for _, o := range extra {
		if strings.EqualFold(strings.TrimRight(o, "/"), origin) {
			return true
		}
	}
	return false
}

func splitOrigins(s string) []string {
	var out []string
	for _, o := range strings.Split(s, ",") {
		if o = strings.TrimSpace(o); o != "" {
			out = append(out, o)
		}
	}
	return out
}

func corsMiddleware(extraOrigins string, next http.Handler) http.Handler {
	extra := splitOrigins(extraOrigins)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if !originAllowed(origin, r.Host, extra) {
				if r.Method != http.MethodGet && r.Method != http.MethodHead {
					http.Error(w, "origin not allowed", http.StatusForbidden)
					return
				}
			} else {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Add("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			}
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func readyHostOf(addr string) string {
	host, _, _ := net.SplitHostPort(addr)
	return host
}
