// Package billing stores provider-neutral merchant-of-record state.
package billing

import "go.beebuzz.app/beebuzz/internal/core"

// Provider identifies the external merchant of record integration.
type Provider string

const (
	ProviderCreem Provider = "creem"
)

// IsValid reports whether p is a supported billing provider.
func (p Provider) IsValid() bool {
	switch p {
	case ProviderCreem:
		return true
	}
	return false
}

// SubscriptionStatus is the normalized subscription lifecycle state.
type SubscriptionStatus string

const (
	SubscriptionStatusIncomplete SubscriptionStatus = "incomplete"
	SubscriptionStatusActive     SubscriptionStatus = "active"
	SubscriptionStatusScheduled  SubscriptionStatus = "scheduled_cancel"
	SubscriptionStatusPastDue    SubscriptionStatus = "past_due"
	SubscriptionStatusCanceled   SubscriptionStatus = "canceled"
	SubscriptionStatusExpired    SubscriptionStatus = "expired"
)

// IsValid reports whether s is a recognized subscription status.
func (s SubscriptionStatus) IsValid() bool {
	switch s {
	case SubscriptionStatusIncomplete,
		SubscriptionStatusActive,
		SubscriptionStatusScheduled,
		SubscriptionStatusPastDue,
		SubscriptionStatusCanceled,
		SubscriptionStatusExpired:
		return true
	}
	return false
}

// GrantsHostedPlan reports whether this status should grant hosted entitlements.
func (s SubscriptionStatus) GrantsHostedPlan() bool {
	switch s {
	case SubscriptionStatusActive, SubscriptionStatusScheduled, SubscriptionStatusPastDue:
		return true
	}
	return false
}

// Customer links a BeeBuzz user to a provider customer without storing billing email.
type Customer struct {
	ID                 string   `db:"id"`
	UserID             string   `db:"user_id"`
	Provider           Provider `db:"provider"`
	ProviderCustomerID *string  `db:"provider_customer_id"`
	CreatedAt          int64    `db:"created_at"`
	UpdatedAt          int64    `db:"updated_at"`
}

// Subscription is the provider-neutral runtime billing record.
type Subscription struct {
	ID                     string             `db:"id"`
	UserID                 string             `db:"user_id"`
	CustomerID             *string            `db:"customer_id"`
	Provider               Provider           `db:"provider"`
	ProviderSubscriptionID string             `db:"provider_subscription_id"`
	Plan                   core.Plan          `db:"plan"`
	Status                 SubscriptionStatus `db:"status"`
	CurrentPeriodEnd       *int64             `db:"current_period_end"`
	CancelAtPeriodEnd      bool               `db:"cancel_at_period_end"`
	ProviderEventAt        int64              `db:"provider_event_at"`
	CreatedAt              int64              `db:"created_at"`
	UpdatedAt              int64              `db:"updated_at"`
}

// Event records a processed provider webhook for idempotency.
type Event struct {
	ID              string   `db:"id"`
	Provider        Provider `db:"provider"`
	ProviderEventID string   `db:"provider_event_id"`
	EventType       string   `db:"event_type"`
	PayloadSHA256   string   `db:"payload_sha256"`
	ProcessedAt     int64    `db:"processed_at"`
	CreatedAt       int64    `db:"created_at"`
}

// CheckoutResponse returns a provider checkout URL to the dashboard.
type CheckoutResponse struct {
	CheckoutURL string `json:"checkout_url"`
}

// CustomerPortalResponse returns a provider billing management URL.
type CustomerPortalResponse struct {
	PortalURL string `json:"portal_url"`
}
