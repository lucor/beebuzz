package auth

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"lucor.dev/beebuzz/internal/testutil"
)

func TestGetOrCreateUserDoesNotPromoteFirstUser(t *testing.T) {
	db := testutil.NewDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user, created, err := repo.GetOrCreateUser(ctx, "first@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser() error = %v", err)
	}
	if !created {
		t.Fatal("GetOrCreateUser() created = false, want true")
	}
	if user.IsAdmin {
		t.Fatal("first user should not be auto-promoted to admin")
	}
}

func TestEnsureUserAdmin(t *testing.T) {
	db := testutil.NewDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	userID := insertTestUser(t, ctx, db, "ensure-admin@example.com")

	changed, err := repo.EnsureUserAdmin(ctx, userID)
	if err != nil {
		t.Fatalf("EnsureUserAdmin() error = %v", err)
	}
	if !changed {
		t.Fatal("EnsureUserAdmin() changed = false, want true")
	}

	changed, err = repo.EnsureUserAdmin(ctx, userID)
	if err != nil {
		t.Fatalf("EnsureUserAdmin() second call error = %v", err)
	}
	if changed {
		t.Fatal("EnsureUserAdmin() second call changed = true, want false")
	}
}

// insertTestUser inserts a user row and returns its ID.
func insertTestUser(t *testing.T, ctx context.Context, db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, email string) string {
	t.Helper()
	id := "test-user-" + email
	now := time.Now().UnixMilli()
	if _, err := db.ExecContext(ctx, `INSERT INTO users (id, email, created_at, updated_at) VALUES (?, ?, ?, ?)`, id, email, now, now); err != nil {
		t.Fatalf("insertTestUser: %v", err)
	}
	return id
}
