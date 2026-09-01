package creem

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"go.beebuzz.app/beebuzz/internal/billing"
	"go.beebuzz.app/beebuzz/internal/core"
	"go.beebuzz.app/beebuzz/internal/testutil"
)

const lifecycleWebhookSecret = "lifecycle-test-secret"

func TestBillingLifecyclePaymentRecovery(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewDBWithUsers(t, "user_1")
	repo := billing.NewRepository(db)
	service := newLifecycleService(t, repo)
	pastDueEnd := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	recoveryEnd := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	sendLifecycleWebhook(t, service, "evt_past_due", "subscription.past_due", "past_due", pastDueEnd, 100)
	sendLifecycleWebhook(t, service, "evt_recovered", "subscription.paid", "active", recoveryEnd, 200)

	subscription, err := repo.GetSubscriptionByProviderID(ctx, billing.ProviderCreem, "sub_lifecycle")
	if err != nil {
		t.Fatalf("GetSubscriptionByProviderID() error = %v", err)
	}
	if subscription.Status != billing.SubscriptionStatusActive {
		t.Fatalf("subscription status = %q, want active", subscription.Status)
	}
	if subscription.CurrentPeriodEnd == nil || *subscription.CurrentPeriodEnd != recoveryEnd.UnixMilli() {
		t.Fatalf("subscription period end = %v, want %d", subscription.CurrentPeriodEnd, recoveryEnd.UnixMilli())
	}
	plan, err := repo.GetUserPlan(ctx, "user_1")
	if err != nil {
		t.Fatalf("GetUserPlan() error = %v", err)
	}
	if plan != core.PlanHosted {
		t.Fatalf("user plan = %q, want hosted", plan)
	}
	var expiresAt *int64
	if err := db.QueryRowContext(ctx, `SELECT plan_expires_at FROM users WHERE id = ?`, "user_1").Scan(&expiresAt); err != nil {
		t.Fatalf("read plan expiry: %v", err)
	}
	if expiresAt == nil || *expiresAt != recoveryEnd.UnixMilli() {
		t.Fatalf("plan expiry = %v, want %d", expiresAt, recoveryEnd.UnixMilli())
	}
}

func TestBillingLifecycleImmediateCancellation(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewDBWithUsers(t, "user_1")
	repo := billing.NewRepository(db)
	service := newLifecycleService(t, repo)
	periodEnd := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	sendLifecycleWebhook(t, service, "evt_active", "subscription.active", "active", periodEnd, 100)
	sendLifecycleWebhook(t, service, "evt_canceled", "subscription.canceled", "canceled", periodEnd, 200)

	plan, err := repo.GetUserPlan(ctx, "user_1")
	if err != nil {
		t.Fatalf("GetUserPlan() error = %v", err)
	}
	if plan != core.PlanFree {
		t.Fatalf("user plan = %q, want free after immediate cancellation", plan)
	}
	var expiresAt *int64
	if err := db.QueryRowContext(ctx, `SELECT plan_expires_at FROM users WHERE id = ?`, "user_1").Scan(&expiresAt); err != nil {
		t.Fatalf("read plan expiry: %v", err)
	}
	if expiresAt != nil {
		t.Fatalf("plan expiry = %v, want nil after immediate cancellation", expiresAt)
	}
	subscription, err := repo.GetSubscriptionByProviderID(ctx, billing.ProviderCreem, "sub_lifecycle")
	if err != nil {
		t.Fatalf("GetSubscriptionByProviderID() error = %v", err)
	}
	if subscription.Status != billing.SubscriptionStatusCanceled {
		t.Fatalf("subscription status = %q, want canceled", subscription.Status)
	}
	if subscription.CancelAtPeriodEnd {
		t.Fatal("canceled subscription should not remain scheduled for period end")
	}
}

func TestBillingLifecycleSubscriptionUpdateProjectsStatusAndPeriod(t *testing.T) {
	tests := []struct {
		name, providerStatus string
		wantStatus           billing.SubscriptionStatus
		wantPlan             core.Plan
		wantCancelAtEnd      bool
		wantExpiry           func(time.Time) *int64
	}{
		{name: "active", providerStatus: "active", wantStatus: billing.SubscriptionStatusActive, wantPlan: core.PlanHosted, wantExpiry: func(end time.Time) *int64 { value := end.UnixMilli(); return &value }},
		{name: "scheduled cancel", providerStatus: "scheduled_cancel", wantStatus: billing.SubscriptionStatusScheduled, wantPlan: core.PlanHosted, wantCancelAtEnd: true, wantExpiry: func(end time.Time) *int64 { value := end.UnixMilli(); return &value }},
		{name: "past due", providerStatus: "past_due", wantStatus: billing.SubscriptionStatusPastDue, wantPlan: core.PlanHosted, wantExpiry: func(end time.Time) *int64 { value := end.Add(7 * 24 * time.Hour).UnixMilli(); return &value }},
		{name: "expired", providerStatus: "expired", wantStatus: billing.SubscriptionStatusPastDue, wantPlan: core.PlanHosted, wantExpiry: func(end time.Time) *int64 { value := end.Add(7 * 24 * time.Hour).UnixMilli(); return &value }},
		{name: "canceled", providerStatus: "canceled", wantStatus: billing.SubscriptionStatusCanceled, wantPlan: core.PlanFree, wantExpiry: func(time.Time) *int64 { return nil }},
		{name: "incomplete", providerStatus: "incomplete", wantStatus: billing.SubscriptionStatusIncomplete, wantPlan: core.PlanFree, wantExpiry: func(time.Time) *int64 { return nil }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			db := testutil.NewDBWithUsers(t, "user_1")
			repo := billing.NewRepository(db)
			service := newLifecycleService(t, repo)
			periodEnd := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
			sendLifecycleWebhook(t, service, "evt_update", "subscription.update", tt.providerStatus, periodEnd, 100)

			subscription, err := repo.GetSubscriptionByProviderID(ctx, billing.ProviderCreem, "sub_lifecycle")
			if err != nil {
				t.Fatalf("GetSubscriptionByProviderID() error = %v", err)
			}
			if subscription.Status != tt.wantStatus || subscription.CurrentPeriodEnd == nil || *subscription.CurrentPeriodEnd != periodEnd.UnixMilli() {
				t.Fatalf("subscription projection = status %q, period %v; want %q, %d", subscription.Status, subscription.CurrentPeriodEnd, tt.wantStatus, periodEnd.UnixMilli())
			}
			if subscription.CancelAtPeriodEnd != tt.wantCancelAtEnd {
				t.Fatalf("cancel at period end = %v, want %v", subscription.CancelAtPeriodEnd, tt.wantCancelAtEnd)
			}
			plan, err := repo.GetUserPlan(ctx, "user_1")
			if err != nil {
				t.Fatalf("GetUserPlan() error = %v", err)
			}
			if plan != tt.wantPlan {
				t.Fatalf("user plan = %q, want %q", plan, tt.wantPlan)
			}
			var expiresAt *int64
			if err := db.QueryRowContext(ctx, `SELECT plan_expires_at FROM users WHERE id = ?`, "user_1").Scan(&expiresAt); err != nil {
				t.Fatalf("read plan expiry: %v", err)
			}
			wantExpiry := tt.wantExpiry(periodEnd)
			if (expiresAt == nil) != (wantExpiry == nil) || expiresAt != nil && *expiresAt != *wantExpiry {
				t.Fatalf("plan expiry = %v, want %v", expiresAt, wantExpiry)
			}
		})
	}
}

func newLifecycleService(t *testing.T, repo *billing.Repository) *billing.Service {
	t.Helper()
	client, err := NewClient(Config{APIKey: "creem_test_key", BaseURL: "https://api.example.test", ProductID: "prod_hosted"})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return billing.NewService(repo, nil, nil, NewBillingAdapter(client, lifecycleWebhookSecret, "prod_hosted"), billing.ServiceConfig{GracePeriodDays: 7}, nil)
}

func sendLifecycleWebhook(t *testing.T, service *billing.Service, eventID, eventType, status string, periodEnd time.Time, occurredAt int64) {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"id": eventID, "eventType": eventType, "created_at": occurredAt,
		"object": map[string]any{
			"id": "sub_lifecycle", "status": status,
			"product": map[string]string{"id": "prod_hosted"}, "customer": "cust_lifecycle",
			"current_period_end_date": periodEnd.Format(time.RFC3339),
			"metadata":                map[string]string{"beebuzz_user_id": "user_1"},
		},
	})
	if err != nil {
		t.Fatalf("marshal webhook: %v", err)
	}
	mac := hmac.New(sha256.New, []byte(lifecycleWebhookSecret))
	_, _ = mac.Write(raw)
	if err := service.HandleWebhook(context.Background(), raw, hex.EncodeToString(mac.Sum(nil))); err != nil {
		t.Fatalf("HandleWebhook(%s): %v", eventType, err)
	}
}
