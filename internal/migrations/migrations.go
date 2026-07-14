// Package migrations provides SQLite database migrations for Beebuzz.
package migrations

import (
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
)

//go:embed *.sql
var migrationFiles embed.FS

// Run executes all pending migrations.
//
// NoTxWrap is required because migration 010 changes the users table CHECK
// constraint, which needs a table rebuild. SQLite's DROP TABLE on a parent
// table with FK references fails even with PRAGMA defer_foreign_keys = ON.
// The solution is PRAGMA foreign_keys = OFF outside a transaction — which
// is impossible when the driver wraps migrations in a transaction (as
// PRAGMA foreign_keys is a no-op inside a transaction).
//
// With NoTxWrap=true the driver runs each migration's SQL directly,
// so migration 010 can manage its own transaction control:
//
//	PRAGMA foreign_keys = OFF;  -- outside tx
//	BEGIN; ... rebuild ... COMMIT;
//	PRAGMA foreign_keys = ON;   -- outside tx
func Run(db *sqlx.DB) error {
	// Get underlying SQL database from sqlx
	sqlDB := db.DB
	if sqlDB == nil {
		return fmt.Errorf("failed to get underlying database connection")
	}

	driver, err := sqlite.WithInstance(sqlDB, &sqlite.Config{
		NoTxWrap: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	// Create source from embedded files
	source, err := iofs.New(migrationFiles, ".")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	// Create migrate instance
	m, err := migrate.NewWithInstance("iofs", source, "sqlite3", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	// Run migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
