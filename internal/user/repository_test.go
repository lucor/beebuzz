package user

import (
	"context"
	"testing"

	"go.beebuzz.app/beebuzz/internal/auth"
	"go.beebuzz.app/beebuzz/internal/billing"
	"go.beebuzz.app/beebuzz/internal/core"
	"go.beebuzz.app/beebuzz/internal/testutil"
)

func TestRepositoryGetByIDIncludesLatestSubscriptionStatus(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	repo := NewRepository(db)

	u, _, err := auth.NewRepository(db).GetOrCreateUser(ctx, "hosted@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO billing_customers (id, user_id, provider, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`,
		"billing-customer-1", u.ID, billing.ProviderCreem, int64(1700000000000), int64(1700000000000),
	); err != nil {
		t.Fatalf("insert billing customer: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO billing_subscriptions
			(id, user_id, customer_id, provider, provider_subscription_id, plan, status, current_period_end, cancel_at_period_end, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"billing-subscription-1", u.ID, "billing-customer-1", billing.ProviderCreem, "sub_1", core.PlanHosted,
		billing.SubscriptionStatusPastDue, int64(1800000000000), 0, int64(1700000000000), int64(1700000000000),
	); err != nil {
		t.Fatalf("insert billing subscription: %v", err)
	}

	got, err := repo.GetByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil {
		t.Fatal("GetByID returned nil for existing user")
	}
	if got.SubscriptionStatus == nil {
		t.Fatal("subscription_status = nil, want past_due")
	}
	if *got.SubscriptionStatus != billing.SubscriptionStatusPastDue {
		t.Fatalf("subscription_status = %q, want %q", *got.SubscriptionStatus, billing.SubscriptionStatusPastDue)
	}
}
