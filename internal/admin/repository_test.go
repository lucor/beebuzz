package admin

import (
	"context"
	"testing"

	"lucor.dev/beebuzz/internal/auth"
	"lucor.dev/beebuzz/internal/core"
	"lucor.dev/beebuzz/internal/testutil"
)

func TestGetUserByID_ActiveUser(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	repo := NewRepository(db)

	user, _, err := auth.NewRepository(db).GetOrCreateUser(ctx, "active@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	got, err := repo.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if got == nil {
		t.Fatal("GetUserByID returned nil for existing user")
	}
	if got.AccountStatus != core.AccountStatusActive {
		t.Errorf("account_status = %q, want %q", got.AccountStatus, core.AccountStatusActive)
	}
}

func TestGetUserByID_BlockedUser(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	repo := NewRepository(db)

	user, _, err := auth.NewRepository(db).GetOrCreateUser(ctx, "blocked@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	if _, err := db.ExecContext(ctx,
		`UPDATE users SET account_status = 'blocked' WHERE id = ?`,
		user.ID,
	); err != nil {
		t.Fatalf("set blocked status: %v", err)
	}

	got, err := repo.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if got == nil {
		t.Fatal("GetUserByID returned nil for existing blocked user")
	}
	if got.AccountStatus != core.AccountStatusBlocked {
		t.Errorf("account_status = %q, want %q", got.AccountStatus, core.AccountStatusBlocked)
	}
}

func TestGetUserByID_NotFound(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	repo := NewRepository(db)

	got, err := repo.GetUserByID(ctx, "nonexistent-id")
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if got != nil {
		t.Errorf("GetUserByID returned %v, want nil", got)
	}
}

func TestUpdateAccountStatus_PendingToActive(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	repo := NewRepository(db)

	user, _, err := auth.NewRepository(db).GetOrCreateUser(ctx, "pending@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	if _, err := db.ExecContext(ctx,
		`UPDATE users SET account_status = 'pending' WHERE id = ?`,
		user.ID,
	); err != nil {
		t.Fatalf("set pending status: %v", err)
	}

	updated, err := repo.UpdateAccountStatus(ctx, user.ID, core.AccountStatusPending, core.AccountStatusActive)
	if err != nil {
		t.Fatalf("UpdateAccountStatus: %v", err)
	}
	if !updated {
		t.Error("UpdateAccountStatus returned false, want true")
	}

	got, err := repo.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if got.AccountStatus != core.AccountStatusActive {
		t.Errorf("account_status = %q, want %q", got.AccountStatus, core.AccountStatusActive)
	}
}

func TestUpdateAccountStatus_ActiveToBlocked(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	repo := NewRepository(db)

	user, _, err := auth.NewRepository(db).GetOrCreateUser(ctx, "active2@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	updated, err := repo.UpdateAccountStatus(ctx, user.ID, core.AccountStatusActive, core.AccountStatusBlocked)
	if err != nil {
		t.Fatalf("UpdateAccountStatus: %v", err)
	}
	if !updated {
		t.Error("UpdateAccountStatus returned false, want true")
	}

	got, err := repo.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if got.AccountStatus != core.AccountStatusBlocked {
		t.Errorf("account_status = %q, want %q", got.AccountStatus, core.AccountStatusBlocked)
	}
}

func TestUpdateAccountStatus_BlockedToActive(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	repo := NewRepository(db)

	user, _, err := auth.NewRepository(db).GetOrCreateUser(ctx, "blocked2@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	if _, err := db.ExecContext(ctx,
		`UPDATE users SET account_status = 'blocked' WHERE id = ?`,
		user.ID,
	); err != nil {
		t.Fatalf("set blocked status: %v", err)
	}

	updated, err := repo.UpdateAccountStatus(ctx, user.ID, core.AccountStatusBlocked, core.AccountStatusActive)
	if err != nil {
		t.Fatalf("UpdateAccountStatus: %v", err)
	}
	if !updated {
		t.Error("UpdateAccountStatus returned false, want true")
	}

	got, err := repo.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if got.AccountStatus != core.AccountStatusActive {
		t.Errorf("account_status = %q, want %q", got.AccountStatus, core.AccountStatusActive)
	}
}

func TestUpdateAccountStatus_ConcurrentModification(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	repo := NewRepository(db)

	user, _, err := auth.NewRepository(db).GetOrCreateUser(ctx, "concurrent@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	updated, err := repo.UpdateAccountStatus(ctx, user.ID, core.AccountStatusActive, core.AccountStatusBlocked)
	if err != nil {
		t.Fatalf("UpdateAccountStatus: %v", err)
	}
	if !updated {
		t.Error("UpdateAccountStatus returned false, want true")
	}

	updated, err = repo.UpdateAccountStatus(ctx, user.ID, core.AccountStatusActive, core.AccountStatusBlocked)
	if err != nil {
		t.Fatalf("UpdateAccountStatus: %v", err)
	}
	if updated {
		t.Error("UpdateAccountStatus returned true on concurrent modification, want false")
	}
}

func TestUpdateAccountStatus_InvalidTransition(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	repo := NewRepository(db)

	user, _, err := auth.NewRepository(db).GetOrCreateUser(ctx, "invalid@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	updated, err := repo.UpdateAccountStatus(ctx, user.ID, core.AccountStatusPending, core.AccountStatusBlocked)
	if err != nil {
		t.Fatalf("UpdateAccountStatus: %v", err)
	}
	if updated {
		t.Error("UpdateAccountStatus returned true for invalid transition, want false")
	}
}

func TestGetAllUsers_ReturnsAllStatuses(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	repo := NewRepository(db)

	activeUser, _, _ := auth.NewRepository(db).GetOrCreateUser(ctx, "active3@example.com")
	blockedUser, _, _ := auth.NewRepository(db).GetOrCreateUser(ctx, "blocked3@example.com")
	pendingUser, _, _ := auth.NewRepository(db).GetOrCreateUser(ctx, "pending3@example.com")

	if _, err := db.ExecContext(ctx,
		`UPDATE users SET account_status = 'blocked' WHERE id = ?`,
		blockedUser.ID,
	); err != nil {
		t.Fatalf("set blocked status: %v", err)
	}

	if _, err := db.ExecContext(ctx,
		`UPDATE users SET account_status = 'pending' WHERE id = ?`,
		pendingUser.ID,
	); err != nil {
		t.Fatalf("set pending status: %v", err)
	}

	users, err := repo.GetAllUsers(ctx)
	if err != nil {
		t.Fatalf("GetAllUsers: %v", err)
	}
	if len(users) != 3 {
		t.Errorf("GetAllUsers len = %d, want 3", len(users))
	}

	statusMap := make(map[string]core.AccountStatus)
	for _, u := range users {
		statusMap[u.ID] = u.AccountStatus
	}

	if statusMap[activeUser.ID] != core.AccountStatusActive {
		t.Errorf("active user status = %q, want %q", statusMap[activeUser.ID], core.AccountStatusActive)
	}
	if statusMap[blockedUser.ID] != core.AccountStatusBlocked {
		t.Errorf("blocked user status = %q, want %q", statusMap[blockedUser.ID], core.AccountStatusBlocked)
	}
	if statusMap[pendingUser.ID] != core.AccountStatusPending {
		t.Errorf("pending user status = %q, want %q", statusMap[pendingUser.ID], core.AccountStatusPending)
	}
}
