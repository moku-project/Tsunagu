//go:build !android && !cgosqlite

package db

import _ "modernc.org/sqlite"

const driverName = "sqlite"

const dsnPragmas = "?_pragma=foreign_keys(1)" +
	"&_pragma=journal_mode(WAL)" +
	"&_pragma=synchronous(NORMAL)" +
	"&_pragma=busy_timeout(5000)" +
	"&_pragma=cache_size(-20000)" +
	"&_pragma=temp_store(MEMORY)" +
	// Take the write lock at BEGIN; upgrading a read transaction fails with SQLITE_BUSY without waiting.
	"&_txlock=immediate"
