package creem

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	SignatureHeader = "creem-signature"
	metadataUserID  = "beebuzz_user_id"

	eventTypeCheckoutCompleted     = "checkout.completed"
	eventTypeSubscriptionActive    = "subscription.active"
	eventTypeSubscriptionPaid      = "subscription.paid"
	eventTypeSubscriptionScheduled = "subscription.scheduled_cancel"
	eventTypeSubscriptionPastDue   = "subscription.past_due"
	eventTypeSubscriptionUnpaid    = "subscription.unpaid"
	eventTypeSubscriptionExpired   = "subscription.expired"
	eventTypeSubscriptionCanceled  = "subscription.canceled"
	eventTypeSubscriptionPaused    = "subscription.paused"
	eventTypeSubscriptionUpdate    = "subscription.update"

	providerStatusActive     = "active"
	providerStatusScheduled  = "scheduled_cancel"
	providerStatusPastDue    = "past_due"
	providerStatusUnpaid     = "unpaid"
	providerStatusExpired    = "expired"
	providerStatusCanceled   = "canceled"
	providerStatusIncomplete = "incomplete"
)

// VerifySignature verifies the Creem HMAC-SHA256 signature over the raw body.
func VerifySignature(rawBody []byte, signature string, secret string) error {
	if secret == "" {
		return fmt.Errorf("creem webhook secret is required")
	}
	signature = strings.TrimSpace(signature)
	if signature == "" {
		return fmt.Errorf("creem webhook signature is required")
	}

	mac := hmac.New(sha256.New, []byte(secret))
	if _, err := mac.Write(rawBody); err != nil {
		return fmt.Errorf("hash creem webhook body: %w", err)
	}
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return fmt.Errorf("creem webhook signature mismatch")
	}
	return nil
}

// WebhookEvent is the subset of Creem webhook data BeeBuzz persists.
type WebhookEvent struct {
	ID              string
	EventType       string
	OccurredAt      int64
	ProductID       string
	UserID          string
	CustomerID      *string
	SubscriptionID  string
	Status          string
	PeriodEndMillis *int64
}

type rawWebhookEvent struct {
	ID        string          `json:"id"`
	EventType string          `json:"eventType"`
	CreatedAt int64           `json:"created_at"`
	Object    json.RawMessage `json:"object"`
}

// ParseWebhookEventType returns the verified event type before object parsing.
// Callers use it to acknowledge event types that do not affect BeeBuzz access.
func ParseWebhookEventType(rawBody []byte) (string, error) {
	var raw rawWebhookEvent
	if err := json.Unmarshal(rawBody, &raw); err != nil {
		return "", fmt.Errorf("parse creem webhook event: %w", err)
	}
	if raw.ID == "" {
		return "", fmt.Errorf("creem webhook event id is required")
	}
	if raw.EventType == "" {
		return "", fmt.Errorf("creem webhook event type is required")
	}
	return raw.EventType, nil
}

type rawSubscription struct {
	ID                   string            `json:"id"`
	Status               string            `json:"status"`
	Product              providerID        `json:"product"`
	Customer             providerID        `json:"customer"`
	Metadata             map[string]string `json:"metadata"`
	CurrentPeriodEndDate string            `json:"current_period_end_date"`
}

type rawCheckout struct {
	Metadata     map[string]string `json:"metadata"`
	Product      providerID        `json:"product"`
	Customer     providerID        `json:"customer"`
	Subscription rawSubscription   `json:"subscription"`
}

type providerID struct {
	ID string
}

func (id *providerID) UnmarshalJSON(data []byte) error {
	var rawString string
	if err := json.Unmarshal(data, &rawString); err == nil {
		id.ID = rawString
		return nil
	}

	var rawObject struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &rawObject); err != nil {
		return err
	}
	id.ID = rawObject.ID
	return nil
}

// ParseWebhookEvent extracts the BeeBuzz-relevant fields from a Creem webhook.
func ParseWebhookEvent(rawBody []byte) (*WebhookEvent, error) {
	var raw rawWebhookEvent
	if err := json.Unmarshal(rawBody, &raw); err != nil {
		return nil, fmt.Errorf("parse creem webhook event: %w", err)
	}
	if _, err := ParseWebhookEventType(rawBody); err != nil {
		return nil, err
	}
	if raw.CreatedAt <= 0 {
		return nil, fmt.Errorf("creem webhook event created_at is required")
	}

	switch raw.EventType {
	case eventTypeCheckoutCompleted:
		return parseCheckoutEvent(raw)
	default:
		return parseSubscriptionEvent(raw)
	}
}

func parseCheckoutEvent(raw rawWebhookEvent) (*WebhookEvent, error) {
	var checkout rawCheckout
	if err := json.Unmarshal(raw.Object, &checkout); err != nil {
		return nil, fmt.Errorf("parse creem checkout object: %w", err)
	}

	userID := checkout.Metadata[metadataUserID]
	if userID == "" {
		userID = checkout.Subscription.Metadata[metadataUserID]
	}
	if userID == "" {
		return nil, fmt.Errorf("creem webhook metadata %q is required", metadataUserID)
	}
	if checkout.Subscription.ID == "" {
		return nil, fmt.Errorf("creem subscription id is required")
	}
	productID := checkout.Product.ID
	if productID == "" {
		productID = checkout.Subscription.Product.ID
	}
	if productID == "" {
		return nil, fmt.Errorf("creem product id is required")
	}

	customerID := checkout.Customer.ID
	if customerID == "" {
		customerID = checkout.Subscription.Customer.ID
	}
	periodEndMillis, err := parsePeriodEndMillis(checkout.Subscription.CurrentPeriodEndDate)
	if err != nil {
		return nil, err
	}

	return &WebhookEvent{
		ID:              raw.ID,
		EventType:       raw.EventType,
		OccurredAt:      raw.CreatedAt,
		ProductID:       productID,
		UserID:          userID,
		CustomerID:      stringPtrOrNil(customerID),
		SubscriptionID:  checkout.Subscription.ID,
		Status:          checkout.Subscription.Status,
		PeriodEndMillis: periodEndMillis,
	}, nil
}

func parseSubscriptionEvent(raw rawWebhookEvent) (*WebhookEvent, error) {
	var subscription rawSubscription
	if err := json.Unmarshal(raw.Object, &subscription); err != nil {
		return nil, fmt.Errorf("parse creem subscription object: %w", err)
	}
	if subscription.ID == "" {
		return nil, fmt.Errorf("creem subscription id is required")
	}
	if subscription.Product.ID == "" {
		return nil, fmt.Errorf("creem product id is required")
	}

	userID := subscription.Metadata[metadataUserID]
	if userID == "" {
		return nil, fmt.Errorf("creem webhook metadata %q is required", metadataUserID)
	}

	var periodEndMillis *int64
	periodEndMillis, err := parsePeriodEndMillis(subscription.CurrentPeriodEndDate)
	if err != nil {
		return nil, err
	}

	customerID := subscription.Customer.ID

	return &WebhookEvent{
		ID:              raw.ID,
		EventType:       raw.EventType,
		OccurredAt:      raw.CreatedAt,
		ProductID:       subscription.Product.ID,
		UserID:          userID,
		CustomerID:      stringPtrOrNil(customerID),
		SubscriptionID:  subscription.ID,
		Status:          subscription.Status,
		PeriodEndMillis: periodEndMillis,
	}, nil
}

func parsePeriodEndMillis(value string) (*int64, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, fmt.Errorf("parse creem subscription period end: %w", err)
	}
	millis := parsed.UTC().UnixMilli()
	return &millis, nil
}

func stringPtrOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
