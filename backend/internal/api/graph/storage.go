package graph

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"tsunagu/backend/internal/api/graph/model"
	"tsunagu/backend/internal/image"
)

func (r *Resolver) storageInfoModel() (*model.StorageInfo, error) {
	c := r.Cfg.Config()
	cats := r.storageCats()
	out := make([]*model.StorageCategory, 0, len(cats))
	var used int64
	for _, cat := range cats {
		var b int64
		var n int
		var err error
		if cat.key == "downloads" {
			b, n, err = r.downloadsSizeCount()
		} else {
			b, n, err = pathSizeCount(cat.key, cat.path)
		}
		if err != nil {
			return nil, fmt.Errorf("sizing %s: %w", cat.key, err)
		}
		used += b
		out = append(out, &model.StorageCategory{
			Key: cat.key, Label: cat.label, Path: cat.path,
			Bytes: float64(b), FileCount: int32(n), Clearable: cat.clearable,
		})
	}
	total, free, err := diskStats(r.MediaDir)
	if err != nil {
		return nil, fmt.Errorf("disk stats: %w", err)
	}
	return &model.StorageInfo{
		UsedBytes:    float64(used),
		TotalBytes:   float64(total),
		FreeBytes:    float64(free),
		DataDir:      c.DataDir,
		MediaDir:     r.MediaDir,
		DatabasePath: c.DBPath,
		Categories:   out,
	}, nil
}

func toDatabaseBackup(b dbBackup) *model.DatabaseBackup {
	return &model.DatabaseBackup{
		Name:      b.name,
		Path:      b.path,
		Bytes:     float64(b.bytes),
		CreatedAt: b.createdAt.UTC().Format(time.RFC3339),
	}
}

func dirSize(root string) (int64, error) {
	total, _, err := dirSizeCount(root)
	return total, err
}

func dirSizeCount(root string) (int64, int, error) {
	var total int64
	var count int
	err := filepath.Walk(root, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			total += info.Size()
			count++
		}
		return nil
	})
	if os.IsNotExist(err) {
		return 0, 0, nil
	}
	return total, count, err
}

type storageCat struct {
	key       string
	label     string
	path      string
	clearable bool
}

func (r *Resolver) storageCats() []storageCat {
	c := r.Cfg.Config()
	media := r.MediaDir
	return []storageCat{
		{"database", "Database", c.DBPath, false},
		{"covers", "Cover cache", filepath.Join(media, "covers"), true},
		{"icons", "Extension icons", filepath.Join(media, "icons"), true},
		{"localSource", "Local source", filepath.Join(media, "local"), false},
		{"downloads", "Downloads", media, false},
		{"jarCache", "Extension jar cache", c.JarCacheDir, true},
		{"extensions", "Installed extensions", c.SandboxExtDir, false},
		{"pluginStorage", "Extension storage", c.SandboxStorageDir, false},
		{"logs", "Logs", filepath.Join(c.DataDir, "tsunagu.log"), true},
	}
}

// downloadsBytes is the media dir minus the sub-caches that get their own row.
func (r *Resolver) downloadsSizeCount() (int64, int, error) {
	total, count, err := dirSizeCount(r.MediaDir)
	if err != nil {
		return 0, 0, err
	}
	for _, sub := range []string{"covers", "icons", "local"} {
		st, sc, _ := dirSizeCount(filepath.Join(r.MediaDir, sub))
		total -= st
		count -= sc
	}
	if total < 0 {
		total = 0
	}
	if count < 0 {
		count = 0
	}
	return total, count, nil
}

func pathSizeCount(key, path string) (int64, int, error) {
	if key == "logs" {
		var total int64
		var count int
		for _, p := range []string{path, path + ".old"} {
			if info, err := os.Stat(p); err == nil && !info.IsDir() {
				total += info.Size()
				count++
			}
		}
		return total, count, nil
	}
	if key == "database" {
		var total int64
		var count int
		for _, p := range []string{path, path + "-wal", path + "-shm"} {
			if info, err := os.Stat(p); err == nil {
				total += info.Size()
				count++
			}
		}
		return total, count, nil
	}
	return dirSizeCount(path)
}

func (r *Resolver) backupDir() string {
	return filepath.Join(r.Cfg.Config().DataDir, "backups")
}

type dbBackup struct {
	name      string
	path      string
	bytes     int64
	createdAt time.Time
}

func (r *Resolver) listBackups() ([]dbBackup, error) {
	dir := r.backupDir()
	ents, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []dbBackup
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".db") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, dbBackup{
			name:      e.Name(),
			path:      filepath.Join(dir, e.Name()),
			bytes:     info.Size(),
			createdAt: info.ModTime(),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].createdAt.After(out[j].createdAt) })
	return out, nil
}

func (r *Resolver) createBackup(ctx context.Context) (dbBackup, error) {
	dir := r.backupDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return dbBackup{}, err
	}
	name := "tsunagu-" + time.Now().Format("20060102-150405") + ".db"
	dest := filepath.Join(dir, name)
	if _, err := r.DB.ExecContext(ctx, "VACUUM INTO ?", dest); err != nil {
		return dbBackup{}, fmt.Errorf("vacuum into %s: %w", dest, err)
	}
	info, err := os.Stat(dest)
	if err != nil {
		return dbBackup{}, err
	}
	return dbBackup{name: name, path: dest, bytes: info.Size(), createdAt: info.ModTime()}, nil
}

func (r *Resolver) clearCategory(ctx context.Context, key string) error {
	cats := r.storageCats()
	var cat *storageCat
	for i := range cats {
		if cats[i].key == key {
			cat = &cats[i]
			break
		}
	}
	if cat == nil {
		return fmt.Errorf("unknown storage category %q", key)
	}
	if !cat.clearable {
		return fmt.Errorf("storage category %q cannot be cleared", key)
	}
	switch key {
	case "logs":
		for _, p := range []string{cat.path + ".old"} {
			_ = os.Remove(p)
		}
		f, err := os.OpenFile(cat.path, os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if f != nil {
			_ = f.Close()
		}
		return nil
	default:
		if err := image.ClearDir(cat.path); err != nil {
			return err
		}
	}
	switch key {
	case "covers":
		_ = image.ClearDir(filepath.Join(r.MediaDir, "covers", "remote"))
		if err := r.Q.ClearMediaCoverPaths(ctx); err != nil {
			return err
		}
	case "icons":
		if err := r.Q.ClearExtensionIconPaths(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (r *Resolver) deleteBackup(name string) error {
	if name == "" || strings.ContainsAny(name, "/\\") || !strings.HasSuffix(name, ".db") {
		return fmt.Errorf("invalid backup name")
	}
	return os.Remove(filepath.Join(r.backupDir(), name))
}
