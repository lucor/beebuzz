// Package main contains adapter implementations that bridge domain services to handlers and middleware.
package main

import (
	"context"
	"errors"
	"log/slog"

	"lucor.dev/beebuzz/internal/admin"
	"lucor.dev/beebuzz/internal/attachment"
	"lucor.dev/beebuzz/internal/auth"
	"lucor.dev/beebuzz/internal/core"
	"lucor.dev/beebuzz/internal/device"
	"lucor.dev/beebuzz/internal/event"
	"lucor.dev/beebuzz/internal/middleware"
	"lucor.dev/beebuzz/internal/notification"
	systemnotifications "lucor.dev/beebuzz/internal/system/notifications"
	"lucor.dev/beebuzz/internal/token"
	"lucor.dev/beebuzz/internal/topic"
	"lucor.dev/beebuzz/internal/webhook"
)

// sessionValidatorAdapter adapts auth.Service to middleware.SessionValidator.
type sessionValidatorAdapter struct {
	authSvc *auth.Service
}

// ValidateSession validates a session token and returns a SessionUser.
func (a *sessionValidatorAdapter) ValidateSession(ctx context.Context, sessionToken string) (*middleware.SessionUser, error) {
	u, err := a.authSvc.ValidateSession(ctx, sessionToken)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, core.ErrUnauthorized
	}
	return &middleware.SessionUser{
		ID:      u.ID,
		IsAdmin: u.IsAdmin,
	}, nil
}

// notificationDeviceAdapter adapts device.Service to notification.DeviceProvider.
type notificationDeviceAdapter struct {
	deviceSvc *device.Service
}

// GetSubscribedDevices returns push subscriptions for a user and topic.
func (a *notificationDeviceAdapter) GetSubscribedDevices(ctx context.Context, userID, topicName string) ([]notification.PushSub, error) {
	subs, err := a.deviceSvc.GetSubscribedDevices(ctx, userID, topicName)
	if err != nil {
		return nil, err
	}
	result := make([]notification.PushSub, len(subs))
	for i, s := range subs {
		result[i] = notification.PushSub{
			DeviceID:     s.DeviceID,
			Endpoint:     s.Endpoint,
			P256dh:       s.P256dh,
			Auth:         s.Auth,
			AgeRecipient: s.AgeRecipient,
		}
	}
	return result, nil
}

// MarkSubscriptionGone removes a push subscription invalidated by the push provider.
func (a *notificationDeviceAdapter) MarkSubscriptionGone(ctx context.Context, deviceID string) error {
	return a.deviceSvc.MarkSubscriptionGone(ctx, deviceID)
}

// authTopicInitializerAdapter adapts topic.Service to auth.TopicInitializer.
type authTopicInitializerAdapter struct {
	topicSvc *topic.Service
}

// CreateDefaultTopic creates the default topic for a new user.
func (a *authTopicInitializerAdapter) CreateDefaultTopic(ctx context.Context, userID string) error {
	return a.topicSvc.CreateDefaultTopic(ctx, userID)
}

// deviceTopicValidatorAdapter adapts topic.Service to device.TopicValidator.
type deviceTopicValidatorAdapter struct {
	topicSvc *topic.Service
}

// ValidateTopicIDs verifies that topic IDs belong to the given user.
func (a *deviceTopicValidatorAdapter) ValidateTopicIDs(ctx context.Context, userID string, topicIDs []string) error {
	err := a.topicSvc.ValidateTopicIDs(ctx, userID, topicIDs)
	if err == nil {
		return nil
	}
	if errors.Is(err, topic.ErrTopicNotFound) || errors.Is(err, topic.ErrDuplicateTopicIDs) {
		return device.ErrInvalidTopicSelection
	}
	return err
}

// tokenTopicValidatorAdapter adapts topic.Service to token.TopicValidator.
type tokenTopicValidatorAdapter struct {
	topicSvc *topic.Service
}

// ValidateTopicIDs verifies that topic IDs belong to the given user.
func (a *tokenTopicValidatorAdapter) ValidateTopicIDs(ctx context.Context, userID string, topicIDs []string) error {
	err := a.topicSvc.ValidateTopicIDs(ctx, userID, topicIDs)
	if err == nil {
		return nil
	}
	if errors.Is(err, topic.ErrTopicNotFound) || errors.Is(err, topic.ErrDuplicateTopicIDs) {
		return token.ErrInvalidTopicSelection
	}
	return err
}

// webhookTopicValidatorAdapter adapts topic.Service to webhook.TopicValidator.
type webhookTopicValidatorAdapter struct {
	topicSvc *topic.Service
}

// ValidateTopicIDs verifies that topic IDs belong to the given user.
func (a *webhookTopicValidatorAdapter) ValidateTopicIDs(ctx context.Context, userID string, topicIDs []string) error {
	err := a.topicSvc.ValidateTopicIDs(ctx, userID, topicIDs)
	if err == nil {
		return nil
	}
	if errors.Is(err, topic.ErrTopicNotFound) || errors.Is(err, topic.ErrDuplicateTopicIDs) {
		return webhook.ErrInvalidTopicSelection
	}
	return err
}

var _ device.TopicValidator = (*deviceTopicValidatorAdapter)(nil)
var _ token.TopicValidator = (*tokenTopicValidatorAdapter)(nil)
var _ webhook.TopicValidator = (*webhookTopicValidatorAdapter)(nil)

// systemNotificationTopicProviderAdapter adapts topic.Service to system notifications.
type systemNotificationTopicProviderAdapter struct {
	topicSvc *topic.Service
}

// GetTopicByID resolves a user-owned topic for system notification settings and dispatch.
func (a *systemNotificationTopicProviderAdapter) GetTopicByID(ctx context.Context, userID, topicID string) (*systemnotifications.Topic, error) {
	t, err := a.topicSvc.GetTopicByID(ctx, userID, topicID)
	if err != nil || t == nil {
		return nil, err
	}
	return &systemnotifications.Topic{
		ID:     t.ID,
		UserID: t.UserID,
		Name:   t.Name,
	}, nil
}

var _ systemnotifications.TopicProvider = (*systemNotificationTopicProviderAdapter)(nil)

// systemNotificationDeviceSubscriptionAdapter adapts device.Service to the
// system notifications device subscription checker. It is used by the admin
// settings response to flag the misconfiguration where notifications are
// enabled but the recipient has no paired device on the chosen topic.
type systemNotificationDeviceSubscriptionAdapter struct {
	deviceSvc *device.Service
}

// HasActiveDeviceForTopic reports whether the user has at least one paired
// device subscribed to topicName.
func (a *systemNotificationDeviceSubscriptionAdapter) HasActiveDeviceForTopic(ctx context.Context, userID, topicName string) (bool, error) {
	subs, err := a.deviceSvc.GetSubscribedDevices(ctx, userID, topicName)
	if err != nil {
		return false, err
	}
	return len(subs) > 0, nil
}

var _ systemnotifications.DeviceSubscriptionChecker = (*systemNotificationDeviceSubscriptionAdapter)(nil)

// pushAuthorizerAdapter adapts token.Service to notification.PushAuthorizer.
type pushAuthorizerAdapter struct {
	tokenSvc *token.Service
}

// ValidateAPITokenForTopic validates an API token for a specific topic.
// Returns the user ID and topic ID on success.
func (a *pushAuthorizerAdapter) ValidateAPITokenForTopic(ctx context.Context, rawToken, topicName string) (string, string, error) {
	return a.tokenSvc.ValidateAPITokenForTopic(ctx, rawToken, topicName)
}

// ValidateAPIToken validates an API token without topic authorization.
// Returns the user ID on success.
func (a *pushAuthorizerAdapter) ValidateAPIToken(ctx context.Context, rawToken string) (string, error) {
	return a.tokenSvc.ValidateAPIToken(ctx, rawToken)
}

// keyProviderAdapter adapts device.Service to notification.KeyProvider.
type keyProviderAdapter struct {
	deviceSvc *device.Service
}

// GetDeviceKeys returns paired device descriptors for a user's current age keys.
func (a *keyProviderAdapter) GetDeviceKeys(ctx context.Context, userID string) ([]device.DeviceKeyDescriptor, error) {
	return a.deviceSvc.GetDeviceKeysByUser(ctx, userID)
}

// notificationAttachmentAdapter adapts attachment.Service to notification.AttachmentStorer.
type notificationAttachmentAdapter struct {
	attachmentSvc *attachment.Service
}

// Store delegates to the real attachment service.
func (a *notificationAttachmentAdapter) Store(ctx context.Context, topicID, mimeType string, originalSize int, data []byte) (string, error) {
	return a.attachmentSvc.Store(ctx, topicID, mimeType, originalSize, data)
}

// notificationEventTrackerAdapter adapts event.Service to notification.EventTracker.
type notificationEventTrackerAdapter struct {
	eventSvc *event.Service
}

// NotificationCreated translates notification-domain facts to event-domain recording.
func (a *notificationEventTrackerAdapter) NotificationCreated(ctx context.Context, userID, topic, source, deliveryMode string, attachmentBytes int64) {
	var ab *int64
	if attachmentBytes > 0 {
		ab = &attachmentBytes
	}

	a.eventSvc.RecordNotificationCreated(ctx, userID, topic, source, deliveryMode, ab)
}

// DeviceDelivered records a successful push delivery.
func (a *notificationEventTrackerAdapter) DeviceDelivered(ctx context.Context, userID, deviceID string) {
	a.eventSvc.RecordNotificationDelivered(ctx, userID, deviceID)
}

// DeviceFailed maps a DeviceResult to an event fail reason and records the failure.
func (a *notificationEventTrackerAdapter) DeviceFailed(ctx context.Context, userID string, result notification.DeviceResult) {
	a.eventSvc.RecordNotificationFailed(ctx, userID, result.DeviceID, mapNotificationFailReason(result))
}

// mapNotificationFailReason maps a DeviceResult to an event fail reason constant.
func mapNotificationFailReason(r notification.DeviceResult) string {
	if r.SubscriptionGone {
		return event.FailSubscriptionGone
	}

	switch {
	case r.StatusCode == 429:
		return event.FailRateLimited
	case r.StatusCode >= 500:
		return event.FailServerError
	default:
		return event.FailUnknown
	}
}

// Ensure notificationEventTrackerAdapter satisfies notification.EventTracker at compile time.
var _ notification.EventTracker = (*notificationEventTrackerAdapter)(nil)

// systemNotificationDeliveryAdapter adapts notification.Service to system notification delivery.
type systemNotificationDeliveryAdapter struct {
	notifSvc *notification.Service
	log      *slog.Logger
}

// SendSystemNotification sends a server-trusted notification generated by BeeBuzz itself.
//
// A zero TotalSent is logged at warn level: the call succeeded but no paired
// device on the configured topic received the push, which usually means the
// admin enabled system notifications without pairing a device on that topic.
// Without this signal the misconfiguration is silent in production.
func (a *systemNotificationDeliveryAdapter) SendSystemNotification(ctx context.Context, input systemnotifications.DeliveryInput) error {
	report, err := a.notifSvc.Send(ctx, input.RecipientUserID, input.TopicID, notification.SendInput{
		TopicName:    input.TopicName,
		Title:        input.Title,
		Body:         input.Body,
		Source:       event.SourceInternal,
		DeliveryMode: notification.DeliveryModeServerTrusted,
	}, a.log)
	if err != nil {
		return err
	}
	if report != nil && report.TotalSent == 0 {
		a.log.Warn("system notification delivered to zero devices",
			"recipient_user_id", input.RecipientUserID,
			"topic_id", input.TopicID,
			"topic", input.TopicName,
		)
	}
	return nil
}

var _ systemnotifications.Delivery = (*systemNotificationDeliveryAdapter)(nil)

// webhookDispatcherAdapter adapts notification.Service to webhook.Dispatcher.
type webhookDispatcherAdapter struct {
	notifSvc *notification.Service
}

// Dispatch sends a push notification for the given user/topic.
func (a *webhookDispatcherAdapter) Dispatch(ctx context.Context, userID, topicID, topicName, title, body, priority string, log *slog.Logger) (*webhook.DispatchReport, error) {
	report, err := a.notifSvc.Send(ctx, userID, topicID, notification.SendInput{
		TopicName:    topicName,
		Title:        title,
		Body:         body,
		Priority:     priority,
		Source:       event.SourceWebhook,
		DeliveryMode: notification.DeliveryModeServerTrusted,
	}, log)
	if err != nil {
		return nil, err
	}

	return &webhook.DispatchReport{
		TotalSent: report.TotalSent,
	}, nil
}

// Ensure webhookDispatcherAdapter satisfies webhook.Dispatcher at compile time.
var _ webhook.Dispatcher = (*webhookDispatcherAdapter)(nil)

// adminSessionRevokerAdapter adapts auth.Service to admin.SessionRevoker.
type adminSessionRevokerAdapter struct {
	svc *auth.Service
}

func (a *adminSessionRevokerAdapter) RevokeAllSessions(ctx context.Context, userID string) error {
	return a.svc.RevokeAllSessions(ctx, userID)
}

// Ensure adminSessionRevokerAdapter satisfies admin.SessionRevoker at compile time.
var _ admin.SessionRevoker = (*adminSessionRevokerAdapter)(nil)
