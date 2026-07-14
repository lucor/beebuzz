package billing

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.beebuzz.app/beebuzz/internal/core"
	"go.beebuzz.app/beebuzz/internal/testutil"
)

func TestRepositoryEnsureCustomer(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewDBWithUsers(t, "user_1")
	repo := NewRepository(db)
	repo.now = fixedNow

	customer, err := repo.EnsureCustomer(ctx, "user_1", ProviderCreem)
	if err != nil {
		t.Fatalf("EnsureCustomer() error = %v", err)
	}
	if customer == nil {
		t.Fatal("EnsureCustomer() returned nil")
	}
	if customer.ProviderCustomerID != nil {
		t.Fatal("ProviderCustomerID should be nil before provider checkout")
	}

	again, err := repo.EnsureCustomer(ctx, "user_1", ProviderCreem)
	if err != nil {
		t.Fatalf("EnsureCustomer() second call error = %v", err)
	}
	if again.ID != customer.ID {
		t.Fatalf("EnsureCustomer() ID = %q, want %q", again.ID, customer.ID)
	}
}

func TestRepositoryUpsertSubscriptionSyncsUserPlan(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewDBWithUsers(t, "user_1")
	repo := NewRepository(db)
	repo.now = fixedNow
	periodEnd := fixedNow().Add(30 * 24 * time.Hour).UnixMilli()
	providerCustomerID := "cus_123"

	subscription, err := repo.UpsertSubscription(ctx, UpsertSubscriptionParams{
		UserID:                 "user_1",
		Provider:               ProviderCreem,
		ProviderCustomerID:     &providerCustomerID,
		ProviderSubscriptionID: "sub_123",
		Plan:                   core.PlanHosted,
		Status:                 SubscriptionStatusActive,
		CurrentPeriodEnd:       &periodEnd,
	})
	if err != nil {
		t.Fatalf("UpsertSubscription() error = %v", err)
	}
	if subscription.Status != SubscriptionStatusActive {
		t.Fatalf("subscription status = %q, want active", subscription.Status)
	}
	if subscription.CurrentPeriodEnd == nil || *subscription.CurrentPeriodEnd != periodEnd {
		t.Fatalf("subscription period end = %v, want %d", subscription.CurrentPeriodEnd, periodEnd)
	}

	var userPlan core.Plan
	var planExpiresAt *int64
	if err := db.QueryRowContext(ctx,
		`SELECT plan, plan_expires_at FROM users WHERE id = ?`,
		"user_1",
	).Scan(&userPlan, &planExpiresAt); err != nil {
		t.Fatalf("read user plan: %v", err)
	}
	if userPlan != core.PlanHosted {
		t.Fatalf("user plan = %q, want hosted", userPlan)
	}
	if planExpiresAt == nil || *planExpiresAt != periodEnd {
		t.Fatalf("user plan_expires_at = %v, want %d", planExpiresAt, periodEnd)
	}

	_, err = repo.UpsertSubscription(ctx, UpsertSubscriptionParams{
		UserID:                 "user_1",
		Provider:               ProviderCreem,
		ProviderCustomerID:     &providerCustomerID,
		ProviderSubscriptionID: "sub_123",
		Plan:                   core.PlanHosted,
		Status:                 SubscriptionStatusCanceled,
		CurrentPeriodEnd:       &periodEnd,
		CancelAtPeriodEnd:      true,
	})
	if err != nil {
		t.Fatalf("UpsertSubscription() canceled error = %v", err)
	}

	if err := db.QueryRowContext(ctx,
		`SELECT plan, plan_expires_at FROM users WHERE id = ?`,
		"user_1",
	).Scan(&userPlan, &planExpiresAt); err != nil {
		t.Fatalf("read canceled user plan: %v", err)
	}
	if userPlan != core.PlanFree {
		t.Fatalf("canceled user plan = %q, want free", userPlan)
	}
	if planExpiresAt != nil {
		t.Fatalf("canceled plan_expires_at = %v, want nil", planExpiresAt)
	}
}

func TestRepositoryUpsertSubscriptionAppliesPastDueGracePeriod(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewDBWithUsers(t, "user_1")
	repo := NewRepository(db)
	repo.now = fixedNow
	periodEnd := time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	wantGraceEnd := time.Date(2027, 7, 8, 0, 0, 0, 0, time.UTC).UnixMilli()

	_, err := repo.UpsertSubscription(ctx, UpsertSubscriptionParams{
		UserID:                 "user_1",
		Provider:               ProviderCreem,
		ProviderSubscriptionID: "sub_123",
		Plan:                   core.PlanHosted,
		Status:                 SubscriptionStatusPastDue,
		CurrentPeriodEnd:       &periodEnd,
		GracePeriodDays:        7,
	})
	if err != nil {
		t.Fatalf("UpsertSubscription() error = %v", err)
	}

	var userPlan core.Plan
	var planExpiresAt *int64
	if err := db.QueryRowContext(ctx,
		`SELECT plan, plan_expires_at FROM users WHERE id = ?`,
		"user_1",
	).Scan(&userPlan, &planExpiresAt); err != nil {
		t.Fatalf("read user plan: %v", err)
	}
	if userPlan != core.PlanHosted {
		t.Fatalf("user plan = %q, want hosted", userPlan)
	}
	if planExpiresAt == nil || *planExpiresAt != wantGraceEnd {
		t.Fatalf("plan_expires_at = %v, want %d", planExpiresAt, wantGraceEnd)
	}
}

func TestRepositoryUpsertSubscriptionRequiresPeriodEndForHostedAccess(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewDBWithUsers(t, "user_1")
	repo := NewRepository(db)

	_, err := repo.UpsertSubscription(ctx, UpsertSubscriptionParams{
		UserID:                 "user_1",
		Provider:               ProviderCreem,
		ProviderSubscriptionID: "sub_123",
		Plan:                   core.PlanHosted,
		Status:                 SubscriptionStatusActive,
	})
	if err == nil {
		t.Fatal("UpsertSubscription() error = nil, want missing period end")
	}
}

func TestRepositoryRecordEventIsIdempotent(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewDB(t)
	repo := NewRepository(db)
	repo.now = fixedNow

	event := Event{
		Provider:        ProviderCreem,
		ProviderEventID: "evt_123",
		EventType:       "subscription.active",
		PayloadSHA256:   "abc123",
	}
	if err := repo.RecordEvent(ctx, event); err != nil {
		t.Fatalf("RecordEvent() error = %v", err)
	}
	if err := repo.RecordEvent(ctx, event); !errors.Is(err, ErrEventAlreadyProcessed) {
		t.Fatalf("RecordEvent() second error = %v, want %v", err, ErrEventAlreadyProcessed)
	}
}

func TestRepositoryApplySubscriptionEventIsAtomicAndIdempotent(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewDBWithUsers(t, "user_1")
	repo := NewRepository(db)
	repo.now = fixedNow
	periodEnd := fixedNow().Add(30 * 24 * time.Hour).UnixMilli()

	event := Event{
		Provider:        ProviderCreem,
		ProviderEventID: "evt_123",
		EventType:       "subscription.active",
		PayloadSHA256:   "abc123",
	}
	params := UpsertSubscriptionParams{
		UserID:                 "user_1",
		Provider:               ProviderCreem,
		ProviderSubscriptionID: "sub_123",
		Plan:                   core.PlanHosted,
		Status:                 SubscriptionStatusActive,
		CurrentPeriodEnd:       &periodEnd,
	}
	if _, err := repo.ApplySubscriptionEvent(ctx, event, params); err != nil {
		t.Fatalf("ApplySubscriptionEvent() error = %v", err)
	}

	var userPlan core.Plan
	if err := db.QueryRowContext(ctx, `SELECT plan FROM users WHERE id = ?`, "user_1").Scan(&userPlan); err != nil {
		t.Fatalf("read user plan: %v", err)
	}
	if userPlan != core.PlanHosted {
		t.Fatalf("user plan = %q, want hosted", userPlan)
	}

	params.Status = SubscriptionStatusCanceled
	if _, err := repo.ApplySubscriptionEvent(ctx, event, params); !errors.Is(err, ErrEventAlreadyProcessed) {
		t.Fatalf("ApplySubscriptionEvent() second error = %v, want %v", err, ErrEventAlreadyProcessed)
	}
	if err := db.QueryRowContext(ctx, `SELECT plan FROM users WHERE id = ?`, "user_1").Scan(&userPlan); err != nil {
		t.Fatalf("read user plan after duplicate: %v", err)
	}
	if userPlan != core.PlanHosted {
		t.Fatalf("user plan after duplicate = %q, want hosted", userPlan)
	}
}

func TestRepositoryApplySubscriptionEventIgnoresOlderProviderState(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewDBWithUsers(t, "user_1")
	repo := NewRepository(db)
	periodEnd := fixedNow().Add(30 * 24 * time.Hour).UnixMilli()

	apply := func(eventID string, eventAt int64, status SubscriptionStatus) error {
		_, err := repo.ApplySubscriptionEvent(ctx, Event{
			Provider:        ProviderCreem,
			ProviderEventID: eventID,
			EventType:       "subscription.changed",
			PayloadSHA256:   eventID,
		}, UpsertSubscriptionParams{
			UserID:                 "user_1",
			Provider:               ProviderCreem,
			ProviderSubscriptionID: "sub_123",
			Plan:                   core.PlanHosted,
			Status:                 status,
			CurrentPeriodEnd:       &periodEnd,
			ProviderEventAt:        eventAt,
		})
		return err
	}

	if err := apply("evt_new", 200, SubscriptionStatusActive); err != nil {
		t.Fatalf("apply newer event: %v", err)
	}
	if err := apply("evt_old", 100, SubscriptionStatusCanceled); !errors.Is(err, ErrStaleSubscriptionEvent) {
		t.Fatalf("apply older event error = %v, want %v", err, ErrStaleSubscriptionEvent)
	}

	subscription, err := repo.GetSubscriptionByProviderID(ctx, ProviderCreem, "sub_123")
	if err != nil {
		t.Fatalf("get subscription: %v", err)
	}
	if subscription.Status != SubscriptionStatusActive {
		t.Fatalf("subscription status = %q, want active", subscription.Status)
	}
	if subscription.ProviderEventAt != 200 {
		t.Fatalf("provider event time = %d, want 200", subscription.ProviderEventAt)
	}
	plan, err := repo.GetUserPlan(ctx, "user_1")
	if err != nil {
		t.Fatalf("get user plan: %v", err)
	}
	if plan != core.PlanHosted {
		t.Fatalf("plan = %q, want hosted", plan)
	}
}

func TestRepositorySubscriptionUpdateKeepsHostedWhenAnotherSubscriptionIsActive(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewDBWithUsers(t, "user_1")
	repo := NewRepository(db)
	periodEnd := fixedNow().Add(30 * 24 * time.Hour).UnixMilli()

	apply := func(eventID, subscriptionID string, eventAt int64, status SubscriptionStatus) error {
		_, err := repo.ApplySubscriptionEvent(ctx, Event{
			Provider:        ProviderCreem,
			ProviderEventID: eventID,
			EventType:       "subscription.changed",
			PayloadSHA256:   eventID,
		}, UpsertSubscriptionParams{
			UserID:                 "user_1",
			Provider:               ProviderCreem,
			ProviderSubscriptionID: subscriptionID,
			Plan:                   core.PlanHosted,
			Status:                 status,
			CurrentPeriodEnd:       &periodEnd,
			ProviderEventAt:        eventAt,
		})
		return err
	}

	if err := apply("evt_first_active", "sub_first", 100, SubscriptionStatusActive); err != nil {
		t.Fatalf("apply first active subscription: %v", err)
	}
	if err := apply("evt_second_active", "sub_second", 200, SubscriptionStatusActive); err != nil {
		t.Fatalf("apply second active subscription: %v", err)
	}
	if err := apply("evt_first_canceled", "sub_first", 300, SubscriptionStatusCanceled); err != nil {
		t.Fatalf("cancel first subscription: %v", err)
	}

	plan, err := repo.GetUserPlan(ctx, "user_1")
	if err != nil {
		t.Fatalf("get user plan: %v", err)
	}
	if plan != core.PlanHosted {
		t.Fatalf("plan = %q, want hosted while second subscription is active", plan)
	}
}

func fixedNow() time.Time {
	return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
}
