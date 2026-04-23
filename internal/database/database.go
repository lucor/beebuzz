// Package database provides a SQLite database connection for Beebuzz.
package database

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// Connection pool constants
const (
	maxOpenConns    = 1
	maxIdleConns    = 1
	connMaxLifetime = 0
)

// New creates or opens a SQLite database on disk at the given directory.
// It applies recommended pragmas for concurrency and integrity, and configures
// the connection pool. Returns a *DB or an error if initialization fails.
func New(dbDir string) (*sqlx.DB, error) {
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create DB directory: %w", err)
	}

	dbPath := filepath.Join(dbDir, "beebuzz.db")
	dsn := fmt.Sprintf("file:%s?mode=rwc&parseTime=true&_txlock=immediate", dbPath)

	return newWithDSN(dsn)
}

// newWithDSN opens the SQLite database using the provided DSN,
// applies recommended pragmas (WAL, synchronous, foreign keys, busy timeout),
// configures the connection pool, and pings the database to verify connectivity.
func newWithDSN(dsn string) (*sqlx.DB, error) {
	db, err := sqlx.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Apply recommended pragmas
	pragmas := []string{
		"PRAGMA journal_mode=WAL",   // allows concurrent reads/writes
		"PRAGMA synchronous=NORMAL", // balance between safety and performance
		"PRAGMA foreign_keys=ON",    // enforce referential integrity
		"PRAGMA busy_timeout=5000",  // wait up to 5s if DB is locked
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("failed to set pragma '%s': %w", pragma, err)
		}
	}

	// Configure connection pool using constants
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)

	// Verify connection
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
