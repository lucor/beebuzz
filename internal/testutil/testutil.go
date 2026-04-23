// Package testutil provides shared test helpers for all domain packages.
package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	"lucor.dev/beebuzz/internal/middleware"
	"lucor.dev/beebuzz/internal/migrations"
)

// NewDB opens an in-memory SQLite database and runs all migrations.
// The database is closed automatically when the test ends.
func NewDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("testutil.NewDB: failed to open in-memory database: %v", err)
	}

	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatalf("testutil.NewDB: failed to enable foreign keys: %v", err)
	}

	if err := migrations.Run(db); err != nil {
		t.Fatalf("testutil.NewDB: failed to run migrations: %v", err)
	}

	t.Cleanup(func() { db.Close() })

	return db
}

// NewDBWithUsers opens an in-memory SQLite database, runs migrations, and seeds users.
func NewDBWithUsers(t *testing.T, userIDs ...string) *sqlx.DB {
	t.Helper()

	db := NewDB(t)
	InsertUsers(t, context.Background(), db, userIDs...)
	return db
}

// InsertUsers seeds users with deterministic emails derived from their IDs.
func InsertUsers(t *testing.T, ctx context.Context, db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, userIDs ...string) {
	t.Helper()

	now := time.Now().UnixMilli()
	for _, userID := range userIDs {
		email := userID + "@example.com"
		if _, err := db.ExecContext(ctx,
			`INSERT OR IGNORE INTO users (id, email, created_at, updated_at) VALUES (?, ?, ?, ?)`,
			userID, email, now, now,
		); err != nil {
			t.Fatalf("InsertUsers(%q): %v", userID, err)
		}
	}
}

// WithUserContext attaches an authenticated user to the context for handler tests.
func WithUserContext(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, middleware.CtxKeyUser, &middleware.CtxUser{
		ID:      userID,
		IsAdmin: false,
	})
}

// WithRouteParams attaches chi route params to the context for handler tests.
func WithRouteParams(ctx context.Context, params map[string]string) context.Context {
	rctx := chi.NewRouteContext()
	for key, value := range params {
		rctx.URLParams.Add(key, value)
	}
	return context.WithValue(ctx, chi.RouteCtxKey, rctx)
}
