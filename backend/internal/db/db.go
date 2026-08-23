package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	_ "modernc.org/sqlite"
)

//go:embed migrations/0001_init.sql
var migration0001 string

//go:embed migrations/0002_extension_updates.sql
var migration0002 string

//go:embed migrations/0003_extension_removal.sql
var migration0003 string

//go:embed migrations/0004_library_organization.sql
var migration0004 string

//go:embed migrations/0005_chapter_uploaded_at_epoch.sql
var migration0005 string

//go:embed migrations/0006_reading_progress_position.sql
var migration0006 string

//go:embed migrations/0007_download_bytes.sql
var migration0007 string

//go:embed migrations/0008_download_extras.sql
var migration0008 string

var migrations = []struct {
	name string
	sql  string
}{
	{"0001_init.sql", migration0001},
	{"0002_extension_updates.sql", migration0002},
	{"0003_extension_removal.sql", migration0003},
	{"0004_library_organization.sql", migration0004},
	{"0005_chapter_uploaded_at_epoch.sql", migration0005},
	{"0006_reading_progress_position.sql", migration0006},
	{"0007_download_bytes.sql", migration0007},
	{"0008_download_extras.sql", migration0008},
}

func Open(path string) (*sql.DB, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create db dir: %w", err)
		}
	}

	conn, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if err := migrate(conn); err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

func migrate(conn *sql.DB) error {
	if _, err := conn.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name TEXT PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	applied := map[string]bool{}
	rows, err := conn.Query(`SELECT name FROM schema_migrations`)
	if err != nil {
		return fmt.Errorf("read schema_migrations: %w", err)
	}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return fmt.Errorf("scan schema_migrations: %w", err)
		}
		applied[name] = true
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close schema_migrations rows: %w", err)
	}

	ordered := append([]struct {
		name string
		sql  string
	}{}, migrations...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].name < ordered[j].name })

	for _, m := range ordered {
		if applied[m.name] {
			continue
		}
		tx, err := conn.Begin()
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", m.name, err)
		}
		if _, err := tx.Exec(m.sql); err != nil {
			tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", m.name, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (name) VALUES (?)`, m.name); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %s: %w", m.name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", m.name, err)
		}
	}

	return nil
}
