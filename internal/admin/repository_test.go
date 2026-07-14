package admin

import (
	"context"
	"testing"
	"time"

	"go.beebuzz.app/beebuzz/internal/auth"
	"go.beebuzz.app/beebuzz/internal/billing"
	"go.beebuzz.app/beebuzz/internal/core"
	"go.beebuzz.app/beebuzz/internal/testutil"
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

func TestGetUserByID_IncludesLatestSubscriptionStatus(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	repo := NewRepository(db)

	user, _, err := auth.NewRepository(db).GetOrCreateUser(ctx, "hosted@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	periodEnd := int64(1800000000000)
	if _, err := db.ExecContext(ctx, `
		INSERT INTO billing_customers (id, user_id, provider, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`,
		"billing-customer-1", user.ID, billing.ProviderCreem, int64(1700000000000), int64(1700000000000),
	); err != nil {
		t.Fatalf("insert billing customer: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO billing_subscriptions
			(id, user_id, customer_id, provider, provider_subscription_id, plan, status, current_period_end, cancel_at_period_end, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"billing-subscription-1", user.ID, "billing-customer-1", billing.ProviderCreem, "sub_1", core.PlanHosted,
		billing.SubscriptionStatusActive, periodEnd, 0, int64(1700000000000), int64(1700000000000),
	); err != nil {
		t.Fatalf("insert billing subscription: %v", err)
	}

	got, err := repo.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if got == nil {
		t.Fatal("GetUserByID returned nil for existing user")
	}
	if got.SubscriptionStatus == nil {
		t.Fatal("subscription_status = nil, want active")
	}
	if *got.SubscriptionStatus != billing.SubscriptionStatusActive {
		t.Errorf("subscription_status = %q, want %q", *got.SubscriptionStatus, billing.SubscriptionStatusActive)
	}
}

func TestGetAllUsersIncludesCurrentMonthUsage(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	repo := NewRepository(db)

	activeUser, _, err := auth.NewRepository(db).GetOrCreateUser(ctx, "active@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser active: %v", err)
	}
	inactiveUser, _, err := auth.NewRepository(db).GetOrCreateUser(ctx, "inactive@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser inactive: %v", err)
	}

	monthStart, _ := currentMonthMillis()
	previousMonth := time.UnixMilli(monthStart).UTC().AddDate(0, -1, 0).UnixMilli()
	if _, err := db.ExecContext(ctx, `
		INSERT INTO daily_usage_summary
			(user_id, day_start_ms, notifications_total, created_at, updated_at)
		VALUES
			(?, ?, ?, ?, ?),
			(?, ?, ?, ?, ?)`,
		activeUser.ID, monthStart, 7, monthStart, monthStart,
		activeUser.ID, previousMonth, 99, previousMonth, previousMonth,
	); err != nil {
		t.Fatalf("insert usage summary: %v", err)
	}

	users, err := repo.GetAllUsers(ctx)
	if err != nil {
		t.Fatalf("GetAllUsers: %v", err)
	}

	usageByUser := make(map[string]int)
	for _, u := range users {
		usageByUser[u.ID] = u.UsageThisMonth
	}

	if usageByUser[activeUser.ID] != 7 {
		t.Fatalf("active user usage = %d, want 7", usageByUser[activeUser.ID])
	}
	if usageByUser[inactiveUser.ID] != 0 {
		t.Fatalf("inactive user usage = %d, want 0", usageByUser[inactiveUser.ID])
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

func TestUpdateAccountStatus_WrongFromStatus(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	repo := NewRepository(db)

	user, _, err := auth.NewRepository(db).GetOrCreateUser(ctx, "invalid@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	updated, err := repo.UpdateAccountStatus(ctx, user.ID, core.AccountStatusBlocked, core.AccountStatusActive)
	if err != nil {
		t.Fatalf("UpdateAccountStatus: %v", err)
	}
	if updated {
		t.Error("UpdateAccountStatus returned true for wrong from status, want false")
	}
}

func TestGetAllUsers_ReturnsActiveAndBlocked(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	repo := NewRepository(db)

	activeUser, _, _ := auth.NewRepository(db).GetOrCreateUser(ctx, "active3@example.com")
	blockedUser, _, _ := auth.NewRepository(db).GetOrCreateUser(ctx, "blocked3@example.com")

	if _, err := db.ExecContext(ctx,
		`UPDATE users SET account_status = 'blocked' WHERE id = ?`,
		blockedUser.ID,
	); err != nil {
		t.Fatalf("set blocked status: %v", err)
	}

	users, err := repo.GetAllUsers(ctx)
	if err != nil {
		t.Fatalf("GetAllUsers: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("GetAllUsers len = %d, want 2", len(users))
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
}
