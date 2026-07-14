package billing

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"go.beebuzz.app/beebuzz/internal/core"
)

var (
	ErrEventAlreadyProcessed  = errors.New("billing event already processed")
	ErrStaleSubscriptionEvent = errors.New("stale billing subscription event")
)

// Repository provides data access for provider-neutral billing state.
type Repository struct {
	db  *sqlx.DB
	now func() time.Time
}

// NewRepository creates a billing repository.
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db, now: time.Now}
}

// EnsureCustomer returns the provider customer row for a user, creating it when needed.
func (r *Repository) EnsureCustomer(ctx context.Context, userID string, provider Provider) (*Customer, error) {
	if !provider.IsValid() {
		return nil, fmt.Errorf("invalid billing provider %q", provider)
	}

	now := r.now().UnixMilli()
	id := uuid.NewString()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO billing_customers (id, user_id, provider, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(user_id, provider) DO UPDATE SET updated_at = updated_at`,
		id, userID, provider, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("ensure billing customer: %w", err)
	}
	return r.GetCustomer(ctx, userID, provider)
}

// GetCustomer returns the provider customer row for a user.
func (r *Repository) GetCustomer(ctx context.Context, userID string, provider Provider) (*Customer, error) {
	var customer Customer
	err := r.db.GetContext(ctx, &customer,
		`SELECT id, user_id, provider, provider_customer_id, created_at, updated_at
		 FROM billing_customers
		 WHERE user_id = ? AND provider = ?`,
		userID, provider,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get billing customer: %w", err)
	}
	return &customer, nil
}

// HasHostedSubscription reports whether the user has a subscription that grants Hosted access.
func (r *Repository) HasHostedSubscription(ctx context.Context, userID string) (bool, error) {
	var exists bool
	if err := r.db.GetContext(ctx, &exists,
		`SELECT EXISTS(
			SELECT 1 FROM billing_subscriptions
			WHERE user_id = ? AND status IN (?, ?, ?)
		)`,
		userID,
		SubscriptionStatusActive,
		SubscriptionStatusScheduled,
		SubscriptionStatusPastDue,
	); err != nil {
		return false, fmt.Errorf("check hosted subscription: %w", err)
	}
	return exists, nil
}

// UpsertSubscriptionParams carries normalized provider subscription state.
type UpsertSubscriptionParams struct {
	UserID                 string
	Provider               Provider
	ProviderCustomerID     *string
	ProviderSubscriptionID string
	Plan                   core.Plan
	Status                 SubscriptionStatus
	CurrentPeriodEnd       *int64
	CancelAtPeriodEnd      bool
	GracePeriodDays        int
	ProviderEventAt        int64
}

// UpsertSubscription stores normalized subscription state and syncs the user's plan cache.
func (r *Repository) UpsertSubscription(ctx context.Context, params UpsertSubscriptionParams) (*Subscription, error) {
	if err := validateSubscriptionParams(params); err != nil {
		return nil, err
	}

	now := r.now().UnixMilli()
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin billing subscription upsert: %w", err)
	}
	defer tx.Rollback()

	if params.ProviderEventAt <= 0 {
		params.ProviderEventAt = now
	}
	updated, err := upsertSubscription(ctx, tx, params, now, false)
	if err != nil {
		return nil, err
	}
	if !updated {
		return nil, ErrStaleSubscriptionEvent
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit billing subscription upsert: %w", err)
	}

	return r.GetSubscriptionByProviderID(ctx, params.Provider, params.ProviderSubscriptionID)
}

// ApplySubscriptionEvent records a provider event and applies subscription state atomically.
func (r *Repository) ApplySubscriptionEvent(ctx context.Context, event Event, params UpsertSubscriptionParams) (*Subscription, error) {
	if event.Provider != params.Provider {
		return nil, fmt.Errorf("billing event provider %q does not match subscription provider %q", event.Provider, params.Provider)
	}
	if err := validateEvent(event); err != nil {
		return nil, err
	}
	if err := validateSubscriptionParams(params); err != nil {
		return nil, err
	}

	now := r.now().UnixMilli()
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin billing subscription event: %w", err)
	}
	defer tx.Rollback()

	inserted, err := recordEvent(ctx, tx, event, now)
	if err != nil {
		return nil, err
	}
	if !inserted {
		return nil, ErrEventAlreadyProcessed
	}

	updated, err := upsertSubscription(ctx, tx, params, now, true)
	if err != nil {
		return nil, err
	}
	if !updated {
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit stale billing subscription event: %w", err)
		}
		return nil, ErrStaleSubscriptionEvent
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit billing subscription event: %w", err)
	}

	return r.GetSubscriptionByProviderID(ctx, params.Provider, params.ProviderSubscriptionID)
}

func validateSubscriptionParams(params UpsertSubscriptionParams) error {
	if !params.Provider.IsValid() {
		return fmt.Errorf("invalid billing provider %q", params.Provider)
	}
	if params.Plan != core.PlanHosted {
		return fmt.Errorf("invalid billing subscription plan %q", params.Plan)
	}
	if !params.Status.IsValid() {
		return fmt.Errorf("invalid billing subscription status %q", params.Status)
	}
	if params.ProviderSubscriptionID == "" {
		return fmt.Errorf("provider subscription id is required")
	}
	if params.Status.GrantsHostedPlan() && params.CurrentPeriodEnd == nil {
		return fmt.Errorf("current period end is required for hosted subscription status %q", params.Status)
	}
	if params.GracePeriodDays < 0 {
		return fmt.Errorf("grace period days must be >= 0")
	}
	if params.ProviderEventAt < 0 {
		return fmt.Errorf("provider event time must be >= 0")
	}
	return nil
}

func upsertSubscription(ctx context.Context, tx *sqlx.Tx, params UpsertSubscriptionParams, now int64, enforceEventOrder bool) (bool, error) {
	customerID, err := upsertCustomer(ctx, tx, params, now)
	if err != nil {
		return false, err
	}

	subscriptionID := uuid.NewString()
	result, err := tx.ExecContext(ctx,
		`INSERT INTO billing_subscriptions
		 (id, user_id, customer_id, provider, provider_subscription_id, plan, status, current_period_end, cancel_at_period_end, provider_event_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(provider, provider_subscription_id) DO UPDATE SET
			user_id = excluded.user_id,
			customer_id = excluded.customer_id,
			plan = excluded.plan,
			status = excluded.status,
			current_period_end = excluded.current_period_end,
			cancel_at_period_end = excluded.cancel_at_period_end,
			provider_event_at = excluded.provider_event_at,
			updated_at = excluded.updated_at
		 WHERE excluded.provider_event_at > billing_subscriptions.provider_event_at OR ? = 0`,
		subscriptionID,
		params.UserID,
		customerID,
		params.Provider,
		params.ProviderSubscriptionID,
		params.Plan,
		params.Status,
		params.CurrentPeriodEnd,
		boolToInt(params.CancelAtPeriodEnd),
		params.ProviderEventAt,
		now,
		now,
		boolToInt(enforceEventOrder),
	)
	if err != nil {
		return false, fmt.Errorf("upsert billing subscription: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read billing subscription upsert count: %w", err)
	}
	if rowsAffected == 0 {
		return false, nil
	}
	if err := syncUserPlan(ctx, tx, params.UserID, params.GracePeriodDays, now); err != nil {
		return false, err
	}
	return true, nil
}

func syncUserPlan(ctx context.Context, tx *sqlx.Tx, userID string, gracePeriodDays int, now int64) error {
	var subscriptions []Subscription
	if err := tx.SelectContext(ctx, &subscriptions,
		`SELECT status, current_period_end FROM billing_subscriptions WHERE user_id = ?`, userID); err != nil {
		return fmt.Errorf("list user billing subscriptions: %w", err)
	}

	plan := core.PlanFree
	var planExpiresAt *int64
	for _, subscription := range subscriptions {
		candidatePlan, candidateExpiresAt := userPlanFromSubscription(subscription.Status, subscription.CurrentPeriodEnd, gracePeriodDays)
		if candidatePlan != core.PlanHosted {
			continue
		}
		plan = core.PlanHosted
		if candidateExpiresAt != nil && (planExpiresAt == nil || *candidateExpiresAt > *planExpiresAt) {
			expiresAt := *candidateExpiresAt
			planExpiresAt = &expiresAt
		}
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE users SET plan = ?, plan_expires_at = ?, updated_at = ? WHERE id = ?`,
		plan, planExpiresAt, now, userID,
	); err != nil {
		return fmt.Errorf("sync user plan from billing subscriptions: %w", err)
	}
	return nil
}

func userPlanFromSubscription(status SubscriptionStatus, currentPeriodEnd *int64, gracePeriodDays int) (core.Plan, *int64) {
	if !status.GrantsHostedPlan() {
		return core.PlanFree, nil
	}
	if status != SubscriptionStatusPastDue || currentPeriodEnd == nil || gracePeriodDays <= 0 {
		return core.PlanHosted, currentPeriodEnd
	}

	gracePeriodEnd := *currentPeriodEnd + int64(gracePeriodDays)*24*60*60*1000
	return core.PlanHosted, &gracePeriodEnd
}

func upsertCustomer(ctx context.Context, tx *sqlx.Tx, params UpsertSubscriptionParams, now int64) (*string, error) {
	id := uuid.NewString()
	_, err := tx.ExecContext(ctx,
		`INSERT INTO billing_customers (id, user_id, provider, provider_customer_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(user_id, provider) DO UPDATE SET
			provider_customer_id = COALESCE(excluded.provider_customer_id, billing_customers.provider_customer_id),
			updated_at = excluded.updated_at`,
		id,
		params.UserID,
		params.Provider,
		params.ProviderCustomerID,
		now,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert billing customer: %w", err)
	}

	var customerID string
	err = tx.GetContext(ctx, &customerID,
		`SELECT id FROM billing_customers WHERE user_id = ? AND provider = ?`,
		params.UserID, params.Provider,
	)
	if err != nil {
		return nil, fmt.Errorf("get billing customer id: %w", err)
	}
	return &customerID, nil
}

// GetSubscriptionByProviderID returns a normalized subscription by provider ID.
func (r *Repository) GetSubscriptionByProviderID(ctx context.Context, provider Provider, providerSubscriptionID string) (*Subscription, error) {
	var subscription Subscription
	err := r.db.GetContext(ctx, &subscription,
		`SELECT id, user_id, customer_id, provider, provider_subscription_id, plan, status,
		        current_period_end, cancel_at_period_end, provider_event_at, created_at, updated_at
		 FROM billing_subscriptions
		 WHERE provider = ? AND provider_subscription_id = ?`,
		provider, providerSubscriptionID,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get billing subscription: %w", err)
	}
	return &subscription, nil
}

// GetUserPlan returns the persisted entitlement projection for a user.
func (r *Repository) GetUserPlan(ctx context.Context, userID string) (core.Plan, error) {
	var plan core.Plan
	if err := r.db.GetContext(ctx, &plan, `SELECT plan FROM users WHERE id = ?`, userID); err != nil {
		return "", fmt.Errorf("get user plan: %w", err)
	}
	return plan, nil
}

// RecordEvent records a provider webhook event once for idempotent processing.
func (r *Repository) RecordEvent(ctx context.Context, event Event) error {
	if err := validateEvent(event); err != nil {
		return err
	}

	now := r.now().UnixMilli()
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin billing event record: %w", err)
	}
	defer tx.Rollback()

	inserted, err := recordEvent(ctx, tx, event, now)
	if err != nil {
		return err
	}
	if !inserted {
		return ErrEventAlreadyProcessed
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit billing event record: %w", err)
	}
	return nil
}

func validateEvent(event Event) error {
	if !event.Provider.IsValid() {
		return fmt.Errorf("invalid billing provider %q", event.Provider)
	}
	if event.ProviderEventID == "" {
		return fmt.Errorf("provider event id is required")
	}
	if event.EventType == "" {
		return fmt.Errorf("event type is required")
	}
	if event.PayloadSHA256 == "" {
		return fmt.Errorf("payload sha256 is required")
	}
	return nil
}

func recordEvent(ctx context.Context, tx *sqlx.Tx, event Event, now int64) (bool, error) {
	id := uuid.NewString()
	result, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO billing_events
		 (id, provider, provider_event_id, event_type, payload_sha256, processed_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, event.Provider, event.ProviderEventID, event.EventType, event.PayloadSHA256, now, now,
	)
	if err != nil {
		return false, fmt.Errorf("record billing event: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read billing event insert count: %w", err)
	}
	return rowsAffected > 0, nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
