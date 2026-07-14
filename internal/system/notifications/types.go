// Package notifications manages system-generated notification policy.
package notifications

import (
	"context"
	"errors"
	"time"

	"go.beebuzz.app/beebuzz/internal/core"
)

var (
	ErrInvalidTopicSelection = errors.New("invalid topic selection")
	ErrTopicRequired         = errors.New("topic_id is required when system notifications are enabled")
)

// Topic is the minimal topic view needed by system notifications.
type Topic struct {
	ID     string
	UserID string
	Name   string
}

// TopicProvider validates and resolves user-owned topics.
type TopicProvider interface {
	GetTopicByID(ctx context.Context, userID, topicID string) (*Topic, error)
}

// Delivery sends a system notification through the delivery engine.
type Delivery interface {
	SendSystemNotification(ctx context.Context, input DeliveryInput) error
}

// DeviceSubscriptionChecker reports whether a user has at least one active
// paired device subscribed to the given topic. It is used to surface a
// configuration warning when system notifications are enabled but would have
// no destination at delivery time.
type DeviceSubscriptionChecker interface {
	HasActiveDeviceForTopic(ctx context.Context, userID, topicName string) (bool, error)
}

// DeliveryInput carries a resolved system notification delivery request.
type DeliveryInput struct {
	RecipientUserID string
	TopicID         string
	TopicName       string
	Title           string
	Body            string
}

type Events int64

const (
	EventSignupCreated Events = 1 << iota
	EventHostedSubscriptionStarted
	EventBillingWebhookFailed
)

func (e *Events) Enable(event Events) {
	*e |= event
}

func (e *Events) Disable(event Events) {
	*e &^= event
}

func (e Events) Has(event Events) bool {
	return e&event != 0
}

func (e Events) String() string {
	switch e {
	case EventSignupCreated:
		return "signup_created"
	case EventHostedSubscriptionStarted:
		return "hosted_subscription_started"
	case EventBillingWebhookFailed:
		return "billing_webhook_failed"
	default:
		return "unknown"
	}
}

// Settings is the persisted system notifications configuration.
//
// RecipientHasActiveDeviceForTopic is a derived (non-persisted) flag computed
// at read time so the admin UI can warn when delivery would have no
// destination. It is best-effort: callers may leave it false when the check
// is not applicable or fails.
type Settings struct {
	Enabled         bool   `db:"enabled"`
	RecipientUserID string `db:"recipient_user_id"`
	TopicID         string `db:"topic_id"`
	EventFlags      Events `db:"event_flags"`
	CreatedAt       int64  `db:"created_at"`
	UpdatedAt       int64  `db:"updated_at"`

	RecipientHasActiveDeviceForTopic bool `db:"-"`
}

// SettingsResponse is the admin API response for system notification settings.
type SettingsResponse struct {
	Enabled                          bool      `json:"enabled"`
	RecipientUserID                  string    `json:"recipient_user_id,omitempty"`
	TopicID                          string    `json:"topic_id,omitempty"`
	SignupCreatedEnabled             bool      `json:"signup_created_enabled"`
	HostedSubscriptionStartedEnabled bool      `json:"hosted_subscription_started_enabled"`
	BillingWebhookFailedEnabled      bool      `json:"billing_webhook_failed_enabled"`
	RecipientHasActiveDeviceForTopic bool      `json:"recipient_has_active_device_for_topic"`
	CreatedAt                        time.Time `json:"created_at,omitempty"`
	UpdatedAt                        time.Time `json:"updated_at,omitempty"`
}

// UpdateSettingsRequest is the admin API request for system notification settings.
type UpdateSettingsRequest struct {
	Enabled                          bool   `json:"enabled"`
	TopicID                          string `json:"topic_id"`
	SignupCreatedEnabled             bool   `json:"signup_created_enabled"`
	HostedSubscriptionStartedEnabled bool   `json:"hosted_subscription_started_enabled"`
	BillingWebhookFailedEnabled      bool   `json:"billing_webhook_failed_enabled"`
}

func EventsFromUpdateRequest(input UpdateSettingsRequest) Events {
	var events Events
	if input.SignupCreatedEnabled {
		events.Enable(EventSignupCreated)
	}
	if input.HostedSubscriptionStartedEnabled {
		events.Enable(EventHostedSubscriptionStarted)
	}
	if input.BillingWebhookFailedEnabled {
		events.Enable(EventBillingWebhookFailed)
	}
	return events
}

// ToSettingsResponse converts persisted settings to the admin API shape.
func ToSettingsResponse(settings *Settings) SettingsResponse {
	if settings == nil {
		return SettingsResponse{}
	}

	return SettingsResponse{
		Enabled:                          settings.Enabled,
		RecipientUserID:                  settings.RecipientUserID,
		TopicID:                          settings.TopicID,
		SignupCreatedEnabled:             settings.EventFlags.Has(EventSignupCreated),
		HostedSubscriptionStartedEnabled: settings.EventFlags.Has(EventHostedSubscriptionStarted),
		BillingWebhookFailedEnabled:      settings.EventFlags.Has(EventBillingWebhookFailed),
		RecipientHasActiveDeviceForTopic: settings.RecipientHasActiveDeviceForTopic,
		CreatedAt:                        time.UnixMilli(settings.CreatedAt).UTC(),
		UpdatedAt:                        time.UnixMilli(settings.UpdatedAt).UTC(),
	}
}

// SignupEvent carries the facts needed for a signup-created notification.
type SignupEvent struct {
	CreatedUserID string
	AccountStatus core.AccountStatus
}
