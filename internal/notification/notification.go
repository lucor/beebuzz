// Package notification handles dispatching push notifications to subscribed devices via Web Push.
package notification

import (
	"context"
	"errors"
	"io"
	"log/slog"

	"lucor.dev/beebuzz/internal/device"
	"lucor.dev/beebuzz/internal/push"
	"lucor.dev/beebuzz/internal/validator"
)

const (
	DeliveryModeServerTrusted = "server_trusted"
	DeliveryModeE2E           = "e2e"
	MaxNotificationTitleLen   = 64
	MaxNotificationBodyLen    = 256
)

var (
	// ErrAttachmentProcessingFailed is returned when an attachment cannot be fetched, encrypted, or stored.
	ErrAttachmentProcessingFailed = errors.New("attachment processing failed")
)

// VAPIDKeys holds VAPID keys for Web Push API.
type VAPIDKeys struct {
	PublicKey  string
	PrivateKey string
}

// PushSub represents a push subscription used by the notification domain.
type PushSub struct {
	DeviceID     string
	Endpoint     string
	P256dh       string
	Auth         string
	AgeRecipient string
}

// NotificationPayload represents the JSON payload sent to devices.
type NotificationPayload struct {
	ID         string                  `json:"id"`
	Title      string                  `json:"title"`
	Body       string                  `json:"body"`
	TopicID    string                  `json:"topic_id,omitempty"`
	Topic      string                  `json:"topic,omitempty"`
	Priority   string                  `json:"priority,omitempty"`
	SentAt     string                  `json:"sent_at"`
	Attachment *NotificationAttachment `json:"attachment,omitempty"`
}

// NotificationAttachment represents an attachment in a notification.
type NotificationAttachment struct {
	Token    string `json:"token,omitempty"`
	MIME     string `json:"mime,omitempty"`
	Filename string `json:"filename,omitempty"`
}

// AttachmentInput carries the raw attachment source for a send request.
// Either Data or URL is set, never both.
type AttachmentInput struct {
	Data     io.Reader // raw file bytes (multipart mode)
	URL      string    // external URL to download (JSON mode)
	MimeType string    // detected or declared MIME type
	Filename string    // sanitized original filename (empty = unknown)
}

// SendInput carries all parameters for a push notification send operation.
type SendInput struct {
	TopicName    string
	Title        string
	Body         string
	Priority     string
	Source       string           // opaque source tag for analytics (e.g., "api", "webhook")
	DeliveryMode string           // delivery mode (e.g., "server_trusted", "e2e")
	Attachment   *AttachmentInput // nil when no attachment
	OpaqueBlob   []byte           // raw E2E-encrypted payload (octet-stream mode)
}

// SendRequest represents a JSON notification send request body.
type SendRequest struct {
	Title         string `json:"title"`
	Body          string `json:"body"`
	Priority      string `json:"priority,omitempty"`
	AttachmentURL string `json:"attachment_url,omitempty"`
}

// Validate validates the send request fields.
func (r *SendRequest) Validate() []error {
	errs := validator.Validate(
		validator.NotBlank("title", r.Title),
		validator.MaxLen("title", r.Title, MaxNotificationTitleLen),
		validator.MaxLen("body", r.Body, MaxNotificationBodyLen),
		validator.OneOf("priority", r.Priority, push.ValidPriorities),
	)
	if r.AttachmentURL != "" {
		if urlErr := validator.HTTPSURL("attachment_url", r.AttachmentURL); urlErr != nil {
			errs = append(errs, urlErr)
		}
	}
	return errs
}

// SendResponse represents a notification send response.
type SendResponse struct {
	Status      SendStatus                   `json:"status"`
	SentCount   int                          `json:"sent_count"`
	TotalCount  int                          `json:"total_count"`
	FailedCount int                          `json:"failed_count"`
	DeviceKeys  []device.DeviceKeyDescriptor `json:"device_keys,omitempty"` // current paired device keys for CLI auto-sync
}

// SendStatus represents the aggregate outcome of a push send request.
type SendStatus string

const (
	SendStatusDelivered SendStatus = "delivered"
	SendStatusPartial   SendStatus = "partial"
	SendStatusFailed    SendStatus = "failed"
)

// DeviceResult holds the result of sending to a single device.
type DeviceResult struct {
	DeviceID         string
	StatusCode       int
	Err              error
	SubscriptionGone bool // true if device was deleted due to 410/404
}

// SendReport holds detailed results from a send operation.
type SendReport struct {
	DeviceResults []DeviceResult
	TotalSent     int
	TotalFailed   int
}

// VAPIDPublicKeyResponse represents the VAPID public key response.
type VAPIDPublicKeyResponse struct {
	Key string `json:"key"`
}

// AttachmentStorer is a cross-domain interface for storing encrypted attachments.
type AttachmentStorer interface {
	Store(ctx context.Context, topicID, mimeType string, originalSize int, data []byte) (token string, err error)
}

// EventTracker reports notification send outcomes for analytics.
// Implementations bridge to the event/analytics layer via adapters.
type EventTracker interface {
	// NotificationCreated is called once per Send operation after the push loop completes.
	NotificationCreated(ctx context.Context, userID, topic, source, deliveryMode string, attachmentBytes int64)
	// DeviceDelivered is called for each successful push delivery.
	DeviceDelivered(ctx context.Context, userID, deviceID string)
	// DeviceFailed is called for each failed push delivery.
	DeviceFailed(ctx context.Context, userID string, result DeviceResult)
}

// PushAuthorizer validates API tokens for push operations.
type PushAuthorizer interface {
	ValidateAPITokenForTopic(ctx context.Context, token, topicName string) (userID, topicID string, err error)
	ValidateAPIToken(ctx context.Context, token string) (userID string, err error)
}

// KeyProvider returns age public keys for a user's devices.
type KeyProvider interface {
	GetDeviceKeys(ctx context.Context, userID string) ([]device.DeviceKeyDescriptor, error)
}

// KeysResponse is the response for GET /v1/push/keys.
type KeysResponse struct {
	Data []device.DeviceKeyDescriptor `json:"data"`
}

// E2EEnvelope points a paired device to an opaque encrypted blob stored on the server.
type E2EEnvelope struct {
	BeeBuzz E2EEnvelopeToken `json:"beebuzz"`
}

// E2EEnvelopeToken carries the attachment token for an encrypted payload blob.
type E2EEnvelopeToken struct {
	ID     string `json:"id"`
	Token  string `json:"token"`
	SentAt string `json:"sent_at"`
}

// Sender is the interface used by Handler to dispatch push notifications.
// It is satisfied by *Service and by any tracking wrapper.
type Sender interface {
	Send(ctx context.Context, userID, topicID string, input SendInput, log *slog.Logger) (*SendReport, error)
	VAPIDPublicKey() string
}
