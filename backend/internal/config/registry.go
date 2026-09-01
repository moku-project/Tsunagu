package config

import (
	"fmt"
	"strconv"
	"strings"
)

// Kind is an internal detail: BOOTSTRAP settings are consumed before the DB /
// Store exists, so a stored value can never apply and the file is always
// authoritative. RUNTIME settings can be layered from app_settings on top of
// the file. Whether the UI may write a setting is a separate flag, Editable.
type Kind string

const (
	KindBootstrap Kind = "BOOTSTRAP"
	KindRuntime   Kind = "RUNTIME"
)

type ValType string

const (
	TypeBool   ValType = "BOOL"
	TypeInt    ValType = "INT"
	TypeString ValType = "STRING"
)

type Scope string

const (
	ScopeLive           Scope = "LIVE"
	ScopeSandboxRestart Scope = "SANDBOX_RESTART"
	ScopeFullRestart    Scope = "FULL_RESTART"
)

type Setting struct {
	Key      string
	Kind     Kind
	Type     ValType
	Scope    Scope
	Editable bool
	Desc     string
	get      func(*Config) string
	set      func(*Config, string) error
}

func (s Setting) Get(c *Config) string          { return s.get(c) }
func (s Setting) Set(c *Config, v string) error { return s.set(c, v) }

func Settings() []Setting { return settings }

func Lookup(key string) (Setting, bool) {
	s, ok := byKey[key]
	return s, ok
}

func DefaultString(key string) string {
	d := Defaults()
	if s, ok := byKey[key]; ok {
		return s.get(&d)
	}
	return ""
}

func strGet(f func(*Config) *string) func(*Config) string {
	return func(c *Config) string { return *f(c) }
}
func strSet(f func(*Config) *string) func(*Config, string) error {
	return func(c *Config, v string) error { *f(c) = strings.TrimSpace(v); return nil }
}
func boolGet(f func(*Config) *bool) func(*Config) string {
	return func(c *Config) string {
		if *f(c) {
			return "true"
		}
		return "false"
	}
}
func boolSet(f func(*Config) *bool) func(*Config, string) error {
	return func(c *Config, v string) error {
		b, err := strconv.ParseBool(strings.TrimSpace(v))
		if err != nil {
			return fmt.Errorf("expected true/false, got %q", v)
		}
		*f(c) = b
		return nil
	}
}
func intGet(f func(*Config) *int) func(*Config) string {
	return func(c *Config) string { return strconv.Itoa(*f(c)) }
}
func intSet(min int, f func(*Config) *int) func(*Config, string) error {
	return func(c *Config, v string) error {
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return fmt.Errorf("expected an integer, got %q", v)
		}
		if n < min {
			return fmt.Errorf("must be >= %d", min)
		}
		*f(c) = n
		return nil
	}
}
func enumSet(f func(*Config) *string, allowed ...string) func(*Config, string) error {
	return func(c *Config, v string) error {
		v = strings.ToLower(strings.TrimSpace(v))
		for _, a := range allowed {
			if v == a {
				*f(c) = v
				return nil
			}
		}
		return fmt.Errorf("expected one of %s, got %q", strings.Join(allowed, "/"), v)
	}
}

const (
	editable    = true
	notEditable = false
)

var settings = []Setting{
	// ---- not UI-editable: edit tsunagu.toml and restart ----
	{"data_dir", KindBootstrap, TypeString, ScopeFullRestart, notEditable, "Root directory for the DB, caches, extensions and downloads. Set via --data-dir.", strGet(func(c *Config) *string { return &c.DataDir }), strSet(func(c *Config) *string { return &c.DataDir })},
	{"db_path", KindBootstrap, TypeString, ScopeFullRestart, notEditable, "SQLite database file.", strGet(func(c *Config) *string { return &c.DBPath }), strSet(func(c *Config) *string { return &c.DBPath })},
	{"http_addr", KindBootstrap, TypeString, ScopeFullRestart, notEditable, "Address the HTTP API binds to.", strGet(func(c *Config) *string { return &c.HTTPAddr }), strSet(func(c *Config) *string { return &c.HTTPAddr })},
	{"api_token", KindRuntime, TypeString, ScopeFullRestart, notEditable, "Require this bearer token on API requests when set.", strGet(func(c *Config) *string { return &c.APIToken }), strSet(func(c *Config) *string { return &c.APIToken })},
	{"sandbox_jar_path", KindRuntime, TypeString, ScopeFullRestart, notEditable, "Path to sandbox.jar. A bundled launcher may override this at runtime.", strGet(func(c *Config) *string { return &c.SandboxJarPath }), strSet(func(c *Config) *string { return &c.SandboxJarPath })},

	// ---- UI-editable ----
	{"public_url", KindRuntime, TypeString, ScopeFullRestart, editable, "Externally reachable base URL (OAuth callbacks, image links).", strGet(func(c *Config) *string { return &c.PublicURL }), strSet(func(c *Config) *string { return &c.PublicURL })},
	{"media_dir", KindRuntime, TypeString, ScopeFullRestart, editable, "Where covers, downloads and other media are stored. Move existing files yourself before changing.", strGet(func(c *Config) *string { return &c.MediaDir }), strSet(func(c *Config) *string { return &c.MediaDir })},
	{"jar_cache_dir", KindRuntime, TypeString, ScopeFullRestart, editable, "Where downloaded extension jars are cached.", strGet(func(c *Config) *string { return &c.JarCacheDir }), strSet(func(c *Config) *string { return &c.JarCacheDir })},
	{"novel_enabled", KindRuntime, TypeBool, ScopeSandboxRestart, editable, "Load light-novel extensions in the sandbox.", boolGet(func(c *Config) *bool { return &c.NovelEnabled }), boolSet(func(c *Config) *bool { return &c.NovelEnabled })},
	{"metadata_backfill", KindRuntime, TypeBool, ScopeFullRestart, editable, "Auto-match library items against AniList on startup.", boolGet(func(c *Config) *bool { return &c.MetadataBackfill }), boolSet(func(c *Config) *bool { return &c.MetadataBackfill })},
	{"idle_timeout_minutes", KindRuntime, TypeInt, ScopeFullRestart, editable, "Stop the sandbox after this many idle minutes (0 = never).", intGet(func(c *Config) *int { return &c.IdleTimeoutMin }), intSet(0, func(c *Config) *int { return &c.IdleTimeoutMin })},
	{"tracker_poll_hours", KindRuntime, TypeInt, ScopeFullRestart, editable, "How often background tracker sync runs, in hours (0 = never).", intGet(func(c *Config) *int { return &c.TrackerPollHours }), intSet(0, func(c *Config) *int { return &c.TrackerPollHours })},
	{"anilist_client_id", KindRuntime, TypeString, ScopeFullRestart, editable, "AniList OAuth client id.", strGet(func(c *Config) *string { return &c.AniListClientID }), strSet(func(c *Config) *string { return &c.AniListClientID })},
	{"mal_client_id", KindRuntime, TypeString, ScopeFullRestart, editable, "MyAnimeList OAuth client id.", strGet(func(c *Config) *string { return &c.MALClientID }), strSet(func(c *Config) *string { return &c.MALClientID })},
	{"mal_client_secret", KindRuntime, TypeString, ScopeFullRestart, editable, "MyAnimeList OAuth client secret (PKCE flow leaves this empty).", strGet(func(c *Config) *string { return &c.MALClientSecret }), strSet(func(c *Config) *string { return &c.MALClientSecret })},
	{"sandbox_addr", KindRuntime, TypeString, ScopeFullRestart, editable, "Loopback address the extension sandbox listens on.", strGet(func(c *Config) *string { return &c.SandboxAddr }), strSet(func(c *Config) *string { return &c.SandboxAddr })},
	{"sandbox_port", KindRuntime, TypeInt, ScopeFullRestart, editable, "Loopback port the extension sandbox listens on.", intGet(func(c *Config) *int { return &c.SandboxPort }), intSet(1, func(c *Config) *int { return &c.SandboxPort })},
	{"sandbox_extensions_dir", KindRuntime, TypeString, ScopeFullRestart, editable, "Directory holding installed extension jars. Move existing files yourself before changing.", strGet(func(c *Config) *string { return &c.SandboxExtDir }), strSet(func(c *Config) *string { return &c.SandboxExtDir })},
	{"sandbox_storage_dir", KindRuntime, TypeString, ScopeFullRestart, editable, "Per-extension key/value storage directory. Move existing files yourself before changing.", strGet(func(c *Config) *string { return &c.SandboxStorageDir }), strSet(func(c *Config) *string { return &c.SandboxStorageDir })},
	{"sandbox_heap_mb", KindRuntime, TypeInt, ScopeSandboxRestart, editable, "JVM heap for the sandbox in MB (0 = JVM default).", intGet(func(c *Config) *int { return &c.SandboxHeapMB }), intSet(0, func(c *Config) *int { return &c.SandboxHeapMB })},
	{"pprof_addr", KindRuntime, TypeString, ScopeFullRestart, editable, "Expose net/http/pprof on this address when set, e.g. localhost:6060.", strGet(func(c *Config) *string { return &c.PprofAddr }), strSet(func(c *Config) *string { return &c.PprofAddr })},
	{"content_filter_level", KindRuntime, TypeString, ScopeLive, editable, "Hide explicit content from lists: strict (all) / moderate (allows gore) / unrestricted.", strGet(func(c *Config) *string { return &c.ContentFilterLevel }), enumSet(func(c *Config) *string { return &c.ContentFilterLevel }, "strict", "moderate", "unrestricted")},
	{"cloudflare_solver_mode", KindRuntime, TypeString, ScopeLive, editable, "Cloudflare bypass: disabled / external / managed.", strGet(func(c *Config) *string { return &c.CloudflareSolverMode }), enumSet(func(c *Config) *string { return &c.CloudflareSolverMode }, "disabled", "external", "managed")},
	{"cloudflare_solver_url", KindRuntime, TypeString, ScopeLive, editable, "FlareSolverr URL for external mode, e.g. http://127.0.0.1:8191.", strGet(func(c *Config) *string { return &c.CloudflareSolverURL }), strSet(func(c *Config) *string { return &c.CloudflareSolverURL })},
}

var byKey = func() map[string]Setting {
	m := make(map[string]Setting, len(settings))
	for _, s := range settings {
		m[s.Key] = s
	}
	return m
}()
