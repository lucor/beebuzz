package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"filippo.io/age"
	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"

	"lucor.dev/beebuzz/internal/core"
	"lucor.dev/beebuzz/internal/httpfetch"
	"lucor.dev/beebuzz/internal/push"
)

// maxAttachmentBytes is the maximum plaintext attachment size this service will process.
// Must match attachment.MaxAttachmentSize.
const maxAttachmentBytes = 1024 * 1024

// pushTTL is how long the push service keeps an undelivered message queued
// before discarding it. Aligned with attachment.AttachmentTTL so that
// notifications don't outlive their downloadable attachments.
const pushTTL = 6 * 60 * 60 // 6 hours, in seconds

// DeviceProvider defines the interface for device operations needed by notification.
type DeviceProvider interface {
	GetSubscribedDevices(ctx context.Context, userID, topicName string) ([]PushSub, error)
	MarkSubscriptionGone(ctx context.Context, deviceID string) error
}

// Service provides notification business logic.
type Service struct {
	device     DeviceProvider
	attachment AttachmentStorer
	tracker    EventTracker // optional; nil disables analytics tracking
	vapidKeys  *VAPIDKeys
	subject    string // VAPID subject per RFC 8292 (https://... or mailto:...)
	log        *slog.Logger
}

// NewService creates a new notification service.
// tracker may be nil to disable event tracking.
func NewService(device DeviceProvider, attachment AttachmentStorer, tracker EventTracker, vapidKeys *VAPIDKeys, subject string, log *slog.Logger) *Service {
	return &Service{
		device:     device,
		attachment: attachment,
		tracker:    tracker,
		vapidKeys:  vapidKeys,
		subject:    subject,
		log:        log,
	}
}

// Send sends a push notification to all subscribed devices for a topic.
// userID and topicID must already be validated by the caller (e.g., via token authorization).
// Returns a SendReport with per-device results.
func (s *Service) Send(ctx context.Context, userID, topicID string, input SendInput, log *slog.Logger) (*SendReport, error) {
	subscriptions, err := s.device.GetSubscribedDevices(ctx, userID, input.TopicName)
	if err != nil {
		return nil, err
	}

	if input.DeliveryMode == DeliveryModeE2E {
		return s.sendE2E(ctx, userID, topicID, input, subscriptions, log)
	}

	payload := NotificationPayload{
		ID:       uuid.Must(uuid.NewV7()).String(),
		Title:    input.Title,
		Body:     input.Body,
		TopicID:  topicID,
		Topic:    input.TopicName,
		Priority: input.Priority,
		SentAt:   time.Now().UTC().Format(time.RFC3339),
	}

	var attachmentBytes int64
	if input.Attachment != nil && s.attachment != nil {
		plainSize, err := s.buildAttachment(ctx, topicID, input.Attachment, subscriptions, &payload)
		if err != nil {
			if errors.Is(err, core.ErrPayloadTooLarge) {
				return nil, err
			}
			return nil, fmt.Errorf("%w: %v", ErrAttachmentProcessingFailed, err)
		} else {
			attachmentBytes = plainSize
		}
	}

	report := &SendReport{
		DeviceResults: make([]DeviceResult, 0, len(subscriptions)),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal notification payload for logging: %w", err)
	}

	for _, sub := range subscriptions {
		endpointHost := pushEndpointHost(sub.Endpoint)
		log.Info(
			"sending push notification",
			"device_id", sub.DeviceID,
			"push_host", endpointHost,
			"delivery_mode", DeliveryModeServerTrusted,
			"payload_bytes", len(payloadBytes),
		)

		statusCode, err := s.sendPush(ctx, sub, payload, input.Priority)

		result := DeviceResult{
			DeviceID:   sub.DeviceID,
			StatusCode: statusCode,
			Err:        err,
		}

		if err != nil {
			if statusCode == 410 || statusCode == 404 {
				log.Warn(
					"removing invalid subscription",
					"device_id", sub.DeviceID,
					"push_host", endpointHost,
					"status", statusCode,
				)
				if delErr := s.device.MarkSubscriptionGone(ctx, sub.DeviceID); delErr != nil {
					log.Error("failed to delete subscription", "device_id", sub.DeviceID, "error", delErr)
				} else {
					result.SubscriptionGone = true
				}
			} else {
				log.Error(
					"error sending notification",
					"device_id", sub.DeviceID,
					"push_host", endpointHost,
					"status", statusCode,
					"error", err,
				)
				// Capture push failures in Sentry (non-410/404 are unexpected)
				sentry.WithScope(func(scope *sentry.Scope) {
					scope.SetTag("device_id", sub.DeviceID)
					scope.SetTag("push_host", endpointHost)
					scope.SetTag("status_code", strconv.Itoa(statusCode))
					sentry.CaptureException(err)
				})
			}
		} else {
			log.Info(
				"push notification sent",
				"device_id", sub.DeviceID,
				"push_host", endpointHost,
				"status", statusCode,
				"delivery_mode", DeliveryModeServerTrusted,
			)
			report.TotalSent++
		}

		report.DeviceResults = append(report.DeviceResults, result)
	}

	report.TotalFailed = len(report.DeviceResults) - report.TotalSent

	// Track events if a tracker is configured.
	if s.tracker != nil {
		deliveryMode := input.DeliveryMode
		if deliveryMode == "" {
			deliveryMode = DeliveryModeServerTrusted
		}
		s.tracker.NotificationCreated(ctx, userID, input.TopicName, input.Source, deliveryMode, attachmentBytes)

		for _, result := range report.DeviceResults {
			if result.Err == nil {
				s.tracker.DeviceDelivered(ctx, userID, result.DeviceID)
				continue
			}
			s.tracker.DeviceFailed(ctx, userID, result)
		}
	}

	return report, nil
}

// buildAttachment fetches/reads the attachment bytes, encrypts them for all age recipients,
// stores the ciphertext, and populates payload.Attachment.
func (s *Service) buildAttachment(ctx context.Context, topicID string, input *AttachmentInput, subs []PushSub, payload *NotificationPayload) (int64, error) {
	// Collect age recipients.
	var ageRecipients []string
	for _, sub := range subs {
		if sub.AgeRecipient != "" {
			ageRecipients = append(ageRecipients, sub.AgeRecipient)
		}
	}
	if len(ageRecipients) == 0 {
		s.log.Warn("no devices with age keys — skipping attachment")
		return 0, nil
	}

	// Resolve plaintext bytes.
	var rawBytes []byte
	var mimeType string

	switch {
	case input.URL != "":
		data, ct, err := httpfetch.Fetch(ctx, input.URL, maxAttachmentBytes)
		if err != nil {
			return 0, fmt.Errorf("fetch attachment URL: %w", err)
		}
		rawBytes = data
		mimeType = ct
	case input.Data != nil:
		data, err := io.ReadAll(io.LimitReader(input.Data, maxAttachmentBytes+1))
		if err != nil {
			return 0, fmt.Errorf("read attachment data: %w", err)
		}
		if int64(len(data)) > maxAttachmentBytes {
			return 0, core.ErrPayloadTooLarge
		}
		rawBytes = data
		mimeType = input.MimeType
	default:
		return 0, fmt.Errorf("attachment has neither URL nor Data")
	}

	// Detect MIME if not already set by the caller.
	if mimeType == "" {
		sniff := rawBytes
		if len(sniff) > 512 {
			sniff = sniff[:512]
		}
		mimeType = http.DetectContentType(sniff)
	}

	// Encrypt for all age recipients.
	ciphertext, err := encryptForRecipients(rawBytes, ageRecipients)
	if err != nil {
		return 0, fmt.Errorf("encrypt attachment: %w", err)
	}

	// Store ciphertext; originalSize is the plaintext byte count.
	token, err := s.attachment.Store(ctx, topicID, mimeType, len(rawBytes), ciphertext)
	if err != nil {
		return 0, fmt.Errorf("store attachment: %w", err)
	}

	payload.Attachment = &NotificationAttachment{
		Token:    token,
		MIME:     mimeType,
		Filename: input.Filename,
	}
	return int64(len(rawBytes)), nil
}

// encryptForRecipients encrypts data for one or more age X25519 recipients.
func encryptForRecipients(data []byte, ageRecipients []string) ([]byte, error) {
	var recipients []age.Recipient
	for _, r := range ageRecipients {
		rec, err := age.ParseX25519Recipient(r)
		if err != nil {
			return nil, fmt.Errorf("parse age recipient %q: %w", r, err)
		}
		recipients = append(recipients, rec)
	}

	var out bytes.Buffer
	w, err := age.Encrypt(&out, recipients...)
	if err != nil {
		return nil, fmt.Errorf("age.Encrypt: %w", err)
	}

	if _, err := w.Write(data); err != nil {
		return nil, fmt.Errorf("age write: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("age close: %w", err)
	}

	return out.Bytes(), nil
}

// sendE2E handles the E2E delivery path: stores the opaque blob as an attachment,
// then sends a plain JSON envelope pointing to the attachment token.
func (s *Service) sendE2E(ctx context.Context, userID, topicID string, input SendInput, subs []PushSub, log *slog.Logger) (*SendReport, error) {
	if len(input.OpaqueBlob) == 0 {
		return nil, fmt.Errorf("empty E2E payload")
	}

	if len(input.OpaqueBlob) > maxAttachmentBytes {
		return nil, core.ErrPayloadTooLarge
	}

	if s.attachment == nil {
		return nil, fmt.Errorf("attachment storage not configured")
	}

	if len(subs) == 0 {
		return &SendReport{}, nil
	}

	notificationID := uuid.Must(uuid.NewV7()).String()
	sentAt := time.Now().UTC().Format(time.RFC3339)

	// Store the opaque blob. MIME is "application/octet-stream" since the server cannot inspect it.
	token, err := s.attachment.Store(ctx, topicID, "application/octet-stream", len(input.OpaqueBlob), input.OpaqueBlob)
	if err != nil {
		return nil, fmt.Errorf("store E2E blob: %w", err)
	}

	// Build the minimal envelope that the SW will use to fetch the blob.
	envelope, err := json.Marshal(E2EEnvelope{
		BeeBuzz: E2EEnvelopeToken{
			ID:     notificationID,
			Token:  token,
			SentAt: sentAt,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal E2E envelope: %w", err)
	}

	urgency := mapPriorityToUrgency(input.Priority)

	report := &SendReport{
		DeviceResults: make([]DeviceResult, 0, len(subs)),
	}

	for _, sub := range subs {
		result := DeviceResult{DeviceID: sub.DeviceID}
		endpointHost := pushEndpointHost(sub.Endpoint)

		log.Info(
			"sending e2e push envelope",
			"device_id", sub.DeviceID,
			"push_host", endpointHost,
			"delivery_mode", DeliveryModeE2E,
			"payload_bytes", len(envelope),
			"notification_id", notificationID,
		)

		statusCode, sendErr := s.sendRawPush(ctx, sub, envelope, urgency)
		result.StatusCode = statusCode
		result.Err = sendErr

		if sendErr != nil {
			if statusCode == http.StatusGone || statusCode == http.StatusNotFound {
				log.Warn(
					"removing invalid subscription",
					"device_id", sub.DeviceID,
					"push_host", endpointHost,
					"status", statusCode,
				)
				if delErr := s.device.MarkSubscriptionGone(ctx, sub.DeviceID); delErr != nil {
					log.Error("failed to delete subscription", "device_id", sub.DeviceID, "error", delErr)
				} else {
					result.SubscriptionGone = true
				}
			} else {
				log.Error(
					"error sending E2E notification",
					"device_id", sub.DeviceID,
					"push_host", endpointHost,
					"status", statusCode,
					"notification_id", notificationID,
					"error", sendErr,
				)
			}
		} else {
			log.Info(
				"e2e push envelope sent",
				"device_id", sub.DeviceID,
				"push_host", endpointHost,
				"status", statusCode,
				"delivery_mode", DeliveryModeE2E,
				"notification_id", notificationID,
			)
			report.TotalSent++
		}

		report.DeviceResults = append(report.DeviceResults, result)
	}

	report.TotalFailed = len(report.DeviceResults) - report.TotalSent

	if s.tracker != nil {
		s.tracker.NotificationCreated(ctx, userID, input.TopicName, input.Source, DeliveryModeE2E, int64(len(input.OpaqueBlob)))
		for _, result := range report.DeviceResults {
			if result.Err == nil {
				s.tracker.DeviceDelivered(ctx, userID, result.DeviceID)
				continue
			}
			s.tracker.DeviceFailed(ctx, userID, result)
		}
	}

	return report, nil
}

// sendPush sends a push notification to a single subscription.
func (s *Service) sendPush(ctx context.Context, sub PushSub, payload NotificationPayload, priority string) (int, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal payload: %w", err)
	}
	urgency := mapPriorityToUrgency(priority)
	return s.sendRawPush(ctx, sub, payloadJSON, urgency)
}

// sendRawPush sends raw bytes as a web push notification to a single subscription.
func (s *Service) sendRawPush(ctx context.Context, sub PushSub, data []byte, urgency webpush.Urgency) (int, error) {
	subscription := &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys: webpush.Keys{
			P256dh: sub.P256dh,
			Auth:   sub.Auth,
		},
	}

	resp, err := webpush.SendNotificationWithContext(ctx, data, subscription, &webpush.Options{
		Subscriber:      s.subject,
		VAPIDPublicKey:  s.vapidKeys.PublicKey,
		VAPIDPrivateKey: s.vapidKeys.PrivateKey,
		Urgency:         urgency,
		TTL:             pushTTL,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to send push: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode >= 400 {
		return resp.StatusCode, fmt.Errorf("push service returned %d", resp.StatusCode)
	}

	return resp.StatusCode, nil
}

// mapPriorityToUrgency maps a priority string to a webpush Urgency value.
func mapPriorityToUrgency(priority string) webpush.Urgency {
	switch priority {
	case push.PriorityHigh:
		return webpush.UrgencyHigh
	default:
		return webpush.UrgencyNormal
	}
}

// VAPIDPublicKey returns the VAPID public key.
func (s *Service) VAPIDPublicKey() string {
	return s.vapidKeys.PublicKey
}

// pushEndpointHost returns the host portion of a push endpoint for safe diagnostics.
func pushEndpointHost(endpoint string) string {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return ""
	}

	return parsed.Host
}
