package config

import (
	"os"
	"strconv"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	HTTPAddr          string `toml:"http_addr"`
	DBPath            string `toml:"db_path"`
	JarCacheDir       string `toml:"jar_cache_dir"`

	SandboxJarPath    string `toml:"sandbox_jar_path"`
	SandboxAddr       string `toml:"sandbox_addr"`
	SandboxPort       int    `toml:"sandbox_port"`
	SandboxExtDir     string `toml:"sandbox_extensions_dir"`
	NovelEnabled      bool   `toml:"novel_enabled"`
	IdleReapEnabled   bool   `toml:"idle_reap_enabled"`
	IdleTimeoutMin    int    `toml:"idle_timeout_minutes"`
}

func (c *Config) IdleTimeout() time.Duration {
	if !c.IdleReapEnabled {
		return 0
	}
	return time.Duration(c.IdleTimeoutMin) * time.Minute
}

func defaults() Config {
	return Config{
		HTTPAddr:          ":8080",
		DBPath:            "tsunagu.db",
		JarCacheDir:       "./jar-cache",
		SandboxJarPath:    "sandbox.jar",
		SandboxAddr:       "localhost:50051",
		SandboxPort:       50051,
		SandboxExtDir:     "sandbox/extensions",
		NovelEnabled:      false,
		IdleReapEnabled:   true,
		IdleTimeoutMin:    15,
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
	return &cfg, nil
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
}
