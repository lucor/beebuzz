package main

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"lucor.dev/beebuzz/internal/auth"
	"lucor.dev/beebuzz/internal/core"
	"lucor.dev/beebuzz/internal/secure"
	"lucor.dev/beebuzz/internal/testutil"
)

// TestSessionValidatorAdapterStaleSession verifies stale sessions do not panic in the adapter.
func TestSessionValidatorAdapterStaleSession(t *testing.T) {
	db := testutil.NewDB(t)
	repo := auth.NewRepository(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	authSvc := auth.NewService(repo, nil, "", nil, logger)
	adapter := &sessionValidatorAdapter{authSvc: authSvc}
	ctx := context.Background()

	userID := insertServerTestUser(t, ctx, db, "adapter-stale@example.com")
	if err := repo.CreateSession(ctx, secure.Hash("adapter-stale-session"), userID, time.Now().Add(time.Hour).UnixMilli()); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if _, err := db.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		t.Fatalf("failed to disable foreign keys: %v", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, userID); err != nil {
		t.Fatalf("failed to delete user: %v", err)
	}
	if _, err := db.ExecContext(ctx, `PRAGMA foreign_keys = ON`); err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
	}

	sessionUser, err := adapter.ValidateSession(ctx, "adapter-stale-session")
	if sessionUser != nil {
		t.Fatalf("ValidateSession() user = %#v, want nil", sessionUser)
	}
	if !errors.Is(err, core.ErrUnauthorized) {
		t.Fatalf("ValidateSession() error = %v, want ErrUnauthorized", err)
	}
}

// insertServerTestUser inserts a user row and returns its ID.
func insertServerTestUser(t *testing.T, ctx context.Context, db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, email string) string {
	t.Helper()

	id := "test-user-" + email
	now := time.Now().UnixMilli()
	if _, err := db.ExecContext(ctx, `INSERT INTO users (id, email, created_at, updated_at) VALUES (?, ?, ?, ?)`, id, email, now, now); err != nil {
		t.Fatalf("insertServerTestUser: %v", err)
	}

	return id
}
