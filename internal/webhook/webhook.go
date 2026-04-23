// Package webhook manages webhooks that dispatch push notifications on incoming HTTP payloads.
package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"lucor.dev/beebuzz/internal/push"
	"lucor.dev/beebuzz/internal/validator"
)

// PayloadType identifies how a webhook payload should be parsed.
type PayloadType string

const (
	PayloadTypeBeebuzz PayloadType = "beebuzz"
	PayloadTypeCustom  PayloadType = "custom"
)

// validPayloadTypes lists accepted payload_type values.
var validPayloadTypes = []string{string(PayloadTypeBeebuzz), string(PayloadTypeCustom)}

// validWebhookPriorities lists accepted priority values at the webhook boundary (no empty string).
var validWebhookPriorities = []string{push.PriorityNormal, push.PriorityHigh}

// DispatchReport contains webhook-local delivery reporting needed for logging.
type DispatchReport struct {
	TotalSent int
}

// Dispatcher dispatches push notifications for a given user/topic.
type Dispatcher interface {
	Dispatch(ctx context.Context, userID, topicID, topicName, title, body, priority string, log *slog.Logger) (*DispatchReport, error)
}

// Webhook is the DB struct — db tags only.
type Webhook struct {
	ID          string      `db:"id"`
	UserID      string      `db:"user_id"`
	TokenHash   string      `db:"token_hash"`
	Name        string      `db:"name"`
	Description *string     `db:"description"`
	PayloadType PayloadType `db:"payload_type"`
	TitlePath   string      `db:"title_path"`
	BodyPath    string      `db:"body_path"`
	Priority    string      `db:"priority"`
	IsActive    bool        `db:"is_active"`
	RevokedAt   *int64      `db:"revoked_at"`
	LastUsedAt  *int64      `db:"last_used_at"`
	CreatedAt   int64       `db:"created_at"`
}

// WebhookTopic holds topic ID and name for dispatch — db tags only.
type WebhookTopic struct {
	ID   string `db:"id"`
	Name string `db:"name"`
}

// WebhookResponse is the HTTP response struct — json tags only.
type WebhookResponse struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description *string     `json:"description,omitempty"`
	PayloadType PayloadType `json:"payload_type"`
	TitlePath   string      `json:"title_path,omitempty"`
	BodyPath    string      `json:"body_path,omitempty"`
	Priority    string      `json:"priority"`
	IsActive    bool        `json:"is_active"`
	CreatedAt   time.Time   `json:"created_at"`
	LastUsedAt  *time.Time  `json:"last_used_at,omitempty"`
	TopicIDs    []string    `json:"topic_ids,omitempty"`
}

// WebhooksListResponse wraps a collection.
type WebhooksListResponse struct {
	Data []WebhookResponse `json:"data"`
}

// CreatedWebhookResponse is returned on POST /webhooks (one-time token reveal).
type CreatedWebhookResponse struct {
	ID    string `json:"id"`
	Token string `json:"token"`
	Name  string `json:"name"`
}

// ReceiveResponse is returned on successful webhook delivery.
type ReceiveResponse struct {
	Status      ReceiveStatus `json:"status"`
	SentCount   int           `json:"sent_count"`
	TotalCount  int           `json:"total_count"`
	FailedCount int           `json:"failed_count"`
}

// ReceiveStatus represents the outcome of a webhook delivery attempt.
type ReceiveStatus string

const (
	ReceiveStatusDelivered ReceiveStatus = "delivered"
	ReceiveStatusPartial   ReceiveStatus = "partial"
	ReceiveStatusFailed    ReceiveStatus = "failed"
)

// RegenerateTokenResponse is returned when a webhook token is regenerated.
type RegenerateTokenResponse struct {
	Token string `json:"token"`
}

// unixMilliPtr converts an optional int64 Unix-millis value to *time.Time.
func unixMilliPtr(v *int64) *time.Time {
	if v == nil {
		return nil
	}
	t := time.UnixMilli(*v).UTC()
	return &t
}

// toWebhookResponse converts a Webhook DB struct to its HTTP response representation.
func toWebhookResponse(wh Webhook, topicIDs []string) WebhookResponse {
	return WebhookResponse{
		ID:          wh.ID,
		Name:        wh.Name,
		Description: wh.Description,
		PayloadType: wh.PayloadType,
		TitlePath:   wh.TitlePath,
		BodyPath:    wh.BodyPath,
		Priority:    wh.Priority,
		IsActive:    wh.IsActive,
		CreatedAt:   time.UnixMilli(wh.CreatedAt).UTC(),
		LastUsedAt:  unixMilliPtr(wh.LastUsedAt),
		TopicIDs:    topicIDs,
	}
}

// CreateWebhookRequest is the request body for POST /webhooks.
type CreateWebhookRequest struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	PayloadType PayloadType `json:"payload_type"`
	TitlePath   string      `json:"title_path"`
	BodyPath    string      `json:"body_path"`
	Priority    string      `json:"priority"`
	Topics      []string    `json:"topics"`
}

// Validate validates the create webhook request fields.
func (r *CreateWebhookRequest) Validate() []error {
	if r.Priority == "" {
		r.Priority = push.PriorityNormal
	}
	errs := validator.Validate(
		validator.NotBlank("name", r.Name),
		validator.RequiredSlice("topics", r.Topics),
		validator.UniqueStrings("topics", r.Topics),
		validator.MaxLen("name", r.Name, validator.MaxDisplayNameLen),
		validator.MaxLen("description", r.Description, validator.MaxDescriptionLen),
		validator.OneOf("payload_type", string(r.PayloadType), validPayloadTypes),
		validator.OneOf("priority", r.Priority, validWebhookPriorities),
	)
	switch r.PayloadType {
	case PayloadTypeCustom:
		errs = append(errs, validator.Validate(
			validator.JSONPath("title_path", r.TitlePath),
			validator.JSONPath("body_path", r.BodyPath),
		)...)
	case PayloadTypeBeebuzz:
		errs = append(errs, validator.Validate(
			validator.Blank("title_path", r.TitlePath),
			validator.Blank("body_path", r.BodyPath),
		)...)
	}
	return errs
}

// UpdateWebhookRequest is the request body for PATCH /webhooks/{webhookID}.
type UpdateWebhookRequest struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	PayloadType PayloadType `json:"payload_type"`
	TitlePath   string      `json:"title_path"`
	BodyPath    string      `json:"body_path"`
	Priority    string      `json:"priority"`
	Topics      []string    `json:"topics"`
}

// Validate validates the update webhook request fields.
func (r *UpdateWebhookRequest) Validate() []error {
	if r.Priority == "" {
		r.Priority = push.PriorityNormal
	}
	errs := validator.Validate(
		validator.NotBlank("name", r.Name),
		validator.RequiredSlice("topics", r.Topics),
		validator.UniqueStrings("topics", r.Topics),
		validator.MaxLen("name", r.Name, validator.MaxDisplayNameLen),
		validator.MaxLen("description", r.Description, validator.MaxDescriptionLen),
		validator.OneOf("payload_type", string(r.PayloadType), validPayloadTypes),
		validator.OneOf("priority", r.Priority, validWebhookPriorities),
	)
	switch r.PayloadType {
	case PayloadTypeCustom:
		errs = append(errs, validator.Validate(
			validator.JSONPath("title_path", r.TitlePath),
			validator.JSONPath("body_path", r.BodyPath),
		)...)
	case PayloadTypeBeebuzz:
		errs = append(errs, validator.Validate(
			validator.Blank("title_path", r.TitlePath),
			validator.Blank("body_path", r.BodyPath),
		)...)
	}
	return errs
}

var (
	// ErrWebhookNotFound is returned when a webhook does not exist or does not belong to the user.
	ErrWebhookNotFound = errors.New("webhook not found")
	// ErrWebhookInactive is returned when a webhook token resolves to a revoked webhook.
	ErrWebhookInactive = errors.New("webhook is inactive")
	// ErrPayloadExtraction is returned when gjson cannot find the configured path in the payload.
	ErrPayloadExtraction = errors.New("failed to extract fields from payload")
	// ErrAtLeastOneTopic is returned when a webhook request contains no topics.
	ErrAtLeastOneTopic = errors.New("at least one topic is required")
	// ErrInvalidTopicSelection is returned when one or more topics are invalid for the user.
	ErrInvalidTopicSelection = errors.New("invalid topic selection")
	// ErrWebhookDeliveryFailed is returned when every dispatch attempt for a webhook fails.
	ErrWebhookDeliveryFailed = errors.New("webhook delivery failed")

	// ErrInspectSessionNotFound is returned when an inspect session does not exist or has expired.
	ErrInspectSessionNotFound = errors.New("inspect session not found")
	// ErrInspectNotWaiting is returned when trying to capture a payload on a non-waiting session.
	ErrInspectNotWaiting = errors.New("inspect session is not in waiting state")
	// ErrInspectNotCaptured is returned when trying to finalize without a captured payload.
	ErrInspectNotCaptured = errors.New("no payload has been captured")
	// ErrInspectSessionExpired is returned when an inspect session has expired.
	ErrInspectSessionExpired = errors.New("inspect session expired")
)

// TopicValidator verifies that topic IDs belong to the given user.
type TopicValidator interface {
	ValidateTopicIDs(ctx context.Context, userID string, topicIDs []string) error
}

// CreateInspectSessionRequest is the request body for POST /webhooks/inspect.
type CreateInspectSessionRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Topics      []string `json:"topics"`
	Priority    string   `json:"priority"`
}

// Validate validates the create inspect session request fields.
func (r *CreateInspectSessionRequest) Validate() []error {
	if r.Priority == "" {
		r.Priority = push.PriorityNormal
	}
	return validator.Validate(
		validator.NotBlank("name", r.Name),
		validator.RequiredSlice("topics", r.Topics),
		validator.UniqueStrings("topics", r.Topics),
		validator.MaxLen("name", r.Name, validator.MaxDisplayNameLen),
		validator.MaxLen("description", r.Description, validator.MaxDescriptionLen),
		validator.OneOf("priority", r.Priority, validWebhookPriorities),
	)
}

// InspectSessionResponse is the HTTP response for inspect session operations.
type InspectSessionResponse struct {
	Token     string        `json:"token"`
	URL       string        `json:"url"`
	Status    InspectStatus `json:"status"`
	ExpiresAt time.Time     `json:"expires_at"`
}

// InspectSessionStatusResponse is the HTTP response for GET /webhooks/inspect.
type InspectSessionStatusResponse struct {
	Status     InspectStatus   `json:"status"`
	Payload    json.RawMessage `json:"payload,omitempty"`
	CapturedAt *time.Time      `json:"captured_at,omitempty"`
	ExpiresAt  time.Time       `json:"expires_at"`
}

// FinalizeInspectRequest is the request body for POST /webhooks/inspect/finalize.
type FinalizeInspectRequest struct {
	TitlePath string `json:"title_path"`
	BodyPath  string `json:"body_path"`
}

// Validate validates the finalize inspect request fields.
func (r *FinalizeInspectRequest) Validate() []error {
	return validator.Validate(
		validator.JSONPath("title_path", r.TitlePath),
		validator.JSONPath("body_path", r.BodyPath),
	)
}
