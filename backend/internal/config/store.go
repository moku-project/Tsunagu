package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"tsunagu/backend/internal/db/sqlcgen"
)

type Source string

const (
	SourceFile    Source = "FILE"
	SourceDB      Source = "DB"
	SourceDefault Source = "DEFAULT"
)

type EffectiveSetting struct {
	Key         string
	Value       string
	Default     string
	Type        ValType
	Kind        Kind
	Scope       Scope
	Source      Source
	Editable    bool
	Description string
}

func (e EffectiveSetting) RestartRequired() bool { return e.Scope != ScopeLive }

type Store struct {
	mu     sync.RWMutex
	cfg    *Config
	q      *sqlcgen.Queries
	path   string
	active map[string]bool
	hooks  map[string]func(context.Context)
}

func NewStore(cfg *Config, q *sqlcgen.Queries, tomlPath string, active map[string]bool) *Store {
	if active == nil {
		active = map[string]bool{}
	}
	return &Store{cfg: cfg, q: q, path: tomlPath, active: active, hooks: map[string]func(context.Context){}}
}

// OnChange registers a callback fired (outside the lock) after a runtime setting
// is written through Set. Register before Sync.
func (s *Store) OnChange(key string, fn func(context.Context)) {
	s.mu.Lock()
	s.hooks[key] = fn
	s.mu.Unlock()
}

// Sync pushes active file keys into the DB, pulls the rest from the DB, then
// rewrites tsunagu.toml so it mirrors the effective config.
func (s *Store) Sync(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, st := range settings {
		if st.Kind != KindRuntime {
			continue
		}
		if s.active[st.Key] {
			if err := s.q.SetSetting(ctx, sqlcgen.SetSettingParams{Key: st.Key, Value: st.get(s.cfg)}); err != nil {
				return err
			}
			continue
		}
		if v, err := s.q.GetSetting(ctx, st.Key); err == nil && v != "" {
			if err := st.set(s.cfg, v); err != nil {
				return fmt.Errorf("stored setting %s=%q is invalid: %w", st.Key, v, err)
			}
		}
	}
	return s.regenerateLocked()
}

func (s *Store) Config() *Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c := *s.cfg
	return &c
}

func (s *Store) List(ctx context.Context) []EffectiveSetting {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]EffectiveSetting, 0, len(settings))
	for _, st := range settings {
		out = append(out, s.effectiveLocked(ctx, st))
	}
	return out
}

func (s *Store) Get(ctx context.Context, key string) (EffectiveSetting, error) {
	st, ok := Lookup(key)
	if !ok {
		return EffectiveSetting{}, fmt.Errorf("unknown setting %q", key)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.effectiveLocked(ctx, st), nil
}

func (s *Store) Set(ctx context.Context, key, value string) (EffectiveSetting, error) {
	st, ok := Lookup(key)
	if !ok {
		return EffectiveSetting{}, fmt.Errorf("unknown setting %q", key)
	}
	if !st.Editable {
		return EffectiveSetting{}, fmt.Errorf("%q can only be changed in %s (then restart)", key, filepath.Base(s.path))
	}

	s.mu.Lock()
	if err := st.set(s.cfg, value); err != nil {
		s.mu.Unlock()
		return EffectiveSetting{}, err
	}
	norm := st.get(s.cfg)
	if err := s.q.SetSetting(ctx, sqlcgen.SetSettingParams{Key: key, Value: norm}); err != nil {
		s.mu.Unlock()
		return EffectiveSetting{}, err
	}
	s.active[key] = true
	if err := s.regenerateLocked(); err != nil {
		s.mu.Unlock()
		return EffectiveSetting{}, err
	}
	hook := s.hooks[key]
	eff := s.effectiveLocked(ctx, st)
	s.mu.Unlock()

	if hook != nil {
		hook(ctx)
	}
	return eff, nil
}

func (s *Store) Unset(ctx context.Context, key string) (EffectiveSetting, error) {
	st, ok := Lookup(key)
	if !ok {
		return EffectiveSetting{}, fmt.Errorf("unknown setting %q", key)
	}
	if !st.Editable {
		return EffectiveSetting{}, fmt.Errorf("%q can only be changed in %s (then restart)", key, filepath.Base(s.path))
	}
	s.mu.Lock()
	_ = s.q.SetSetting(ctx, sqlcgen.SetSettingParams{Key: key, Value: ""})
	delete(s.active, key)
	d := Defaults()
	_ = st.set(s.cfg, st.get(&d))
	if err := s.regenerateLocked(); err != nil {
		s.mu.Unlock()
		return EffectiveSetting{}, err
	}
	hook := s.hooks[key]
	eff := s.effectiveLocked(ctx, st)
	s.mu.Unlock()
	if hook != nil {
		hook(ctx)
	}
	return eff, nil
}

func (s *Store) effectiveLocked(ctx context.Context, st Setting) EffectiveSetting {
	val := st.get(s.cfg)
	def := DefaultString(st.Key)
	src := SourceDefault
	switch {
	case s.active[st.Key]:
		src = SourceFile
	case st.Kind == KindRuntime:
		if v, err := s.q.GetSetting(ctx, st.Key); err == nil && v != "" {
			src = SourceDB
		}
	}
	return EffectiveSetting{
		Key:         st.Key,
		Value:       val,
		Default:     def,
		Type:        st.Type,
		Kind:        st.Kind,
		Scope:       st.Scope,
		Source:      src,
		Editable:    st.Editable,
		Description: st.Desc,
	}
}

func (s *Store) regenerateLocked() error {
	var b strings.Builder
	b.WriteString("# Tsunagu configuration -- regenerated on every save.\n")
	b.WriteString("# Active (uncommented) lines are authoritative: on boot they overwrite\n")
	b.WriteString("# whatever the app last set. Commented lines show the default; the app\n")
	b.WriteString("# controls those. Edit and restart to override.\n\n")

	b.WriteString("# ---- not editable in the app: change here, then restart ----\n\n")
	for _, st := range settings {
		if st.Editable {
			continue
		}
		writeLine(&b, st, s.cfg, true)
	}

	b.WriteString("\n# ---- editable here or in the app ----\n\n")
	for _, st := range settings {
		if !st.Editable {
			continue
		}
		def := DefaultString(st.Key)
		activeLine := s.active[st.Key] || st.get(s.cfg) != def
		writeLine(&b, st, s.cfg, activeLine)
	}

	tmp := s.path + ".tmp"
	if dir := filepath.Dir(s.path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	if err := os.WriteFile(tmp, []byte(b.String()), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func writeLine(b *strings.Builder, st Setting, c *Config, active bool) {
	fmt.Fprintf(b, "# %s", st.Desc)
	if st.Scope != ScopeLive {
		fmt.Fprintf(b, "  [%s]", st.Scope)
	}
	b.WriteString("\n")
	prefix := ""
	if !active {
		prefix = "# "
	}
	fmt.Fprintf(b, "%s%s = %s\n\n", prefix, st.Key, tomlValue(st, c))
}

func tomlValue(st Setting, c *Config) string {
	v := st.get(c)
	switch st.Type {
	case TypeBool, TypeInt:
		return v
	default:
		return fmt.Sprintf("%q", v)
	}
}
