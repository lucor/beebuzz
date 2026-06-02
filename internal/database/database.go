// Package database provides a SQLite database connection for Beebuzz.
package database

import (
	"errors"
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

// dbFileName is the on-disk name of the SQLite database. SQLite creates
// companion -journal, -wal, and -shm files alongside it.
const dbFileName = "beebuzz.db"

// New creates or opens a SQLite database on disk at the given directory.
// It applies recommended pragmas for concurrency and integrity, configures
// the connection pool, and tightens filesystem permissions on the DB
// directory and its files. Returns a *DB or an error if initialization fails.
func New(dbDir string) (*sqlx.DB, error) {
	if err := os.MkdirAll(dbDir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create DB directory: %w", err)
	}

	dbPath := filepath.Join(dbDir, dbFileName)
	dsn := fmt.Sprintf("file:%s?mode=rwc&parseTime=true&_txlock=immediate", dbPath)

	db, err := newWithDSN(dsn)
	if err != nil {
		return nil, err
	}

	if err := tightenPerms(dbDir); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

// tightenPerms restricts permissions on the SQLite directory and any
// of its journal/WAL/SHM companion files to owner-only access.
//
// SQLite honors the process umask when creating the WAL/SHM files,
// which on most Linux systems leaves them group/world readable. The
// database stores session, API-key, and webhook secret hashes plus
// user identifying data, so we set the modes explicitly on every
// startup to also tighten files left over from a previous looser
// configuration.
func tightenPerms(dbDir string) error {
	if err := os.Chmod(dbDir, 0o700); err != nil {
		return fmt.Errorf("chmod DB directory: %w", err)
	}
	for _, suffix := range []string{"", "-journal", "-wal", "-shm"} {
		path := filepath.Join(dbDir, dbFileName+suffix)
		if err := os.Chmod(path, 0o600); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("chmod %s: %w", filepath.Base(path), err)
		}
	}
	return nil
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
