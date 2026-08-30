package config

import (
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	DataDir string `toml:"data_dir"`

	HTTPAddr    string `toml:"http_addr"`
	DBPath      string `toml:"db_path"`
	JarCacheDir string `toml:"jar_cache_dir"`
	MediaDir    string `toml:"media_dir"`

	SandboxJarPath    string `toml:"sandbox_jar_path"`
	SandboxAddr       string `toml:"sandbox_addr"`
	SandboxPort       int    `toml:"sandbox_port"`
	SandboxExtDir     string `toml:"sandbox_extensions_dir"`
	SandboxStorageDir string `toml:"sandbox_storage_dir"`
	NovelEnabled      bool   `toml:"novel_enabled"`
	IdleReapEnabled   bool   `toml:"idle_reap_enabled"`
	IdleTimeoutMin    int    `toml:"idle_timeout_minutes"`

	AniListClientID string `toml:"anilist_client_id"`

	PublicURL string `toml:"public_url"`

	MALClientID     string `toml:"mal_client_id"`
	MALClientSecret string `toml:"mal_client_secret"`

	TrackerPollHours int `toml:"tracker_poll_hours"`

	MetadataBackfill bool `toml:"metadata_backfill"`

	PprofAddr string `toml:"pprof_addr"`

	SandboxHeapMB int `toml:"sandbox_heap_mb"`

	APIToken string `toml:"api_token"`
}

func (c *Config) IdleTimeout() time.Duration {
	if !c.IdleReapEnabled {
		return 0
	}
	return time.Duration(c.IdleTimeoutMin) * time.Minute
}

func defaults() Config {
	return Config{
		HTTPAddr:          ":6007",
		DBPath:            "tsunagu.db",
		JarCacheDir:       "./jar-cache",
		MediaDir:          "./media",
		SandboxJarPath:    "sandbox.jar",
		SandboxAddr:       "localhost:50051",
		SandboxPort:       50051,
		SandboxExtDir:     "sandbox/extensions",
		SandboxStorageDir: "data/plugin-storage",
		NovelEnabled:      false,
		IdleReapEnabled:   true,
		IdleTimeoutMin:    15,
		TrackerPollHours:  6,
		MetadataBackfill:  true,

		AniListClientID: "49724",

		MALClientID: "611c821aee93c5e51411bfa86ca32597",
		PublicURL:   "http://localhost:6007",
	}
}

func Load() (*Config, error) {
	cfg := defaults()

	path := os.Getenv("TSUNAGU_CONFIG")
	if path == "" {
		path = "tsunagu.toml"
	}
	if _, err := os.Stat(path); err == nil {
		if _, err := toml.DecodeFile(path, &cfg); err != nil {
			return nil, err
		}
	}

	applyEnvOverrides(&cfg)
	cfg.applyDataDir()
	return &cfg, nil
}

func (c *Config) applyDataDir() {
	if c.DataDir == "" {
		return
	}
	set := func(dst *string, env, name string) {
		if os.Getenv(env) == "" {
			*dst = filepath.Join(c.DataDir, name)
		}
	}
	set(&c.DBPath, "TSUNAGU_DB_PATH", "tsunagu.db")
	set(&c.JarCacheDir, "TSUNAGU_JAR_CACHE_DIR", "jar-cache")
	set(&c.MediaDir, "TSUNAGU_MEDIA_DIR", "media")
	set(&c.SandboxExtDir, "SANDBOX_EXTENSIONS_DIR", "extensions")
	set(&c.SandboxStorageDir, "SANDBOX_STORAGE_DIR", "plugin-storage")
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("TSUNAGU_ADDR"); v != "" {
		cfg.HTTPAddr = v
	}
	if v := os.Getenv("TSUNAGU_DB_PATH"); v != "" {
		cfg.DBPath = v
	}
	if v := os.Getenv("TSUNAGU_JAR_CACHE_DIR"); v != "" {
		cfg.JarCacheDir = v
	}
	if v := os.Getenv("TSUNAGU_MEDIA_DIR"); v != "" {
		cfg.MediaDir = v
	}
	if v := os.Getenv("TSUNAGU_SANDBOX_ADDR"); v != "" {
		cfg.SandboxAddr = v
	}
	if v := os.Getenv("SANDBOX_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.SandboxPort = p
		}
	}
	if v := os.Getenv("SANDBOX_EXTENSIONS_DIR"); v != "" {
		cfg.SandboxExtDir = v
	}
	if v := os.Getenv("SANDBOX_STORAGE_DIR"); v != "" {
		cfg.SandboxStorageDir = v
	}
	if v := os.Getenv("SANDBOX_ENABLE_NOVEL"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.NovelEnabled = b
		}
	}
	if v := os.Getenv("TSUNAGU_SANDBOX_JAR"); v != "" {
		cfg.SandboxJarPath = v
	}
	if v := os.Getenv("TSUNAGU_SANDBOX_IDLE_MINUTES"); v != "" {
		if m, err := strconv.Atoi(v); err == nil {
			cfg.IdleTimeoutMin = m
		}
	}
	if v := os.Getenv("TSUNAGU_ANILIST_CLIENT_ID"); v != "" {
		cfg.AniListClientID = v
	}
	if v := os.Getenv("TSUNAGU_PUBLIC_URL"); v != "" {
		cfg.PublicURL = v
	}
	if v := os.Getenv("TSUNAGU_MAL_CLIENT_ID"); v != "" {
		cfg.MALClientID = v
	}
	if v := os.Getenv("TSUNAGU_MAL_CLIENT_SECRET"); v != "" {
		cfg.MALClientSecret = v
	}
	if v := os.Getenv("TSUNAGU_TRACKER_POLL_HOURS"); v != "" {
		if h, err := strconv.Atoi(v); err == nil {
			cfg.TrackerPollHours = h
		}
	}
	if v := os.Getenv("TSUNAGU_METADATA_BACKFILL"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.MetadataBackfill = b
		}
	}
	if v := os.Getenv("TSUNAGU_PPROF_ADDR"); v != "" {
		cfg.PprofAddr = v
	}
	if v := os.Getenv("TSUNAGU_API_TOKEN"); v != "" {
		cfg.APIToken = v
	}
	if v := os.Getenv("TSUNAGU_SANDBOX_HEAP_MB"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.SandboxHeapMB = n
		}
	}
	if v := os.Getenv("TSUNAGU_DATA_DIR"); v != "" {
		cfg.DataDir = v
	}
}
