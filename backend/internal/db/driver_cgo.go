//go:build android || cgosqlite

package db

import (
	"database/sql"

	sqlite3 "github.com/mattn/go-sqlite3"
)

// Android's x86_64 seccomp filter kills the process on the legacy syscalls
// modernc's translated libc makes, so Android links SQLite through cgo.
const driverName = "sqlite3_tsunagu"

// Take the write lock at BEGIN; upgrading a read transaction fails with SQLITE_BUSY without waiting.
const dsnPragmas = "?_txlock=immediate&_busy_timeout=5000"

func init() {
	sql.Register(driverName, &sqlite3.SQLiteDriver{
		ConnectHook: func(c *sqlite3.SQLiteConn) error {
			for _, p := range []string{
				"foreign_keys = ON",
				"journal_mode = WAL",
				"synchronous = NORMAL",
				"cache_size = -20000",
				"temp_store = MEMORY",
			} {
				if _, err := c.Exec("PRAGMA "+p, nil); err != nil {
					return err
				}
			}
			return nil
		},
	})
}
