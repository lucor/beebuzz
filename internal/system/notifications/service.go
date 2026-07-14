package notifications

import (
	"context"
	"fmt"
	"log/slog"

	"go.beebuzz.app/beebuzz/internal/core"
)

const signupCreatedTitle = "New BeeBuzz signup"
const hostedSubscriptionStartedTitle = "Hosted subscription started"
const billingWebhookFailedTitle = "Billing webhook failed"

// Service owns system notification policy and dispatch decisions.
type Service struct {
	repo          *Repository
	topics        TopicProvider
	delivery      Delivery
	subscriptions DeviceSubscriptionChecker
	log           *slog.Logger
}

// NewService creates a system notifications service.
func NewService(repo *Repository, topics TopicProvider, delivery Delivery, subscriptions DeviceSubscriptionChecker, log *slog.Logger) *Service {
	return &Service{
		repo:          repo,
		topics:        topics,
		delivery:      delivery,
		subscriptions: subscriptions,
		log:           log,
	}
}

// GetSettings returns the current singleton settings, enriched with the
// best-effort RecipientHasActiveDeviceForTopic flag for the admin UI.
func (s *Service) GetSettings(ctx context.Context) (*Settings, error) {
	settings, err := s.repo.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	s.fillRecipientDeviceFlag(ctx, settings)
	return settings, nil
}

// UpdateSettings validates and stores settings for the current admin user.
func (s *Service) UpdateSettings(ctx context.Context, adminUserID string, input UpdateSettingsRequest) (*Settings, error) {
	if input.Enabled && input.TopicID == "" {
		return nil, ErrTopicRequired
	}

	if input.TopicID != "" {
		topic, err := s.topics.GetTopicByID(ctx, adminUserID, input.TopicID)
		if err != nil {
			return nil, fmt.Errorf("resolve notification topic: %w", err)
		}
		if topic == nil {
			return nil, ErrInvalidTopicSelection
		}
	}

	settings, err := s.repo.UpsertSettings(ctx, Settings{
		Enabled:         input.Enabled,
		RecipientUserID: adminUserID,
		TopicID:         input.TopicID,
		EventFlags:      EventsFromUpdateRequest(input),
	})
	if err != nil {
		return nil, err
	}
	s.fillRecipientDeviceFlag(ctx, settings)
	return settings, nil
}

// fillRecipientDeviceFlag sets RecipientHasActiveDeviceForTopic on the given
// settings. The check is skipped (flag stays false) when there is no topic
// configured. Lookup failures are logged but not propagated: this flag is a
// UI hint, never a gate.
func (s *Service) fillRecipientDeviceFlag(ctx context.Context, settings *Settings) {
	if settings == nil || s.subscriptions == nil {
		return
	}
	if settings.RecipientUserID == "" || settings.TopicID == "" {
		return
	}

	topic, err := s.topics.GetTopicByID(ctx, settings.RecipientUserID, settings.TopicID)
	if err != nil {
		s.log.Warn("failed to resolve topic for system notification device check", "error", err)
		return
	}
	if topic == nil {
		return
	}

	hasDevice, err := s.subscriptions.HasActiveDeviceForTopic(ctx, settings.RecipientUserID, topic.Name)
	if err != nil {
		s.log.Warn("failed to check active device for system notification topic", "error", err)
		return
	}
	settings.RecipientHasActiveDeviceForTopic = hasDevice
}

// NotifySignupCreated sends the configured notification for a newly created account.
func (s *Service) NotifySignupCreated(ctx context.Context, createdUserID string, accountStatus core.AccountStatus) {
	body := fmt.Sprintf("A BeeBuzz account was created with status %q.", accountStatus)
	s.notifyConfiguredEvent(ctx, EventSignupCreated, signupCreatedTitle, body)
}

func (s *Service) NotifyHostedSubscriptionStarted(ctx context.Context, userID string) {
	s.notifyConfiguredEvent(ctx, EventHostedSubscriptionStarted, hostedSubscriptionStartedTitle, fmt.Sprintf("User %s upgraded to Hosted.", userID))
}

func (s *Service) NotifyBillingWebhookFailed(ctx context.Context, eventType string) {
	body := "A billing webhook failed to process."
	if eventType != "" {
		body = fmt.Sprintf("A billing webhook failed to process. Event type: %s.", eventType)
	}
	s.notifyConfiguredEvent(ctx, EventBillingWebhookFailed, billingWebhookFailedTitle, body)
}

func (s *Service) notifyConfiguredEvent(ctx context.Context, event Events, title string, body string) {
	settings, err := s.repo.GetSettings(ctx)
	if err != nil {
		s.log.Error("failed to read system notification settings", "event", event.String(), "error", err)
		return
	}
	if settings == nil || !settings.Enabled || !settings.EventFlags.Has(event) {
		return
	}

	topic, err := s.topics.GetTopicByID(ctx, settings.RecipientUserID, settings.TopicID)
	if err != nil {
		s.log.Error("failed to resolve system notification topic", "event", event.String(), "error", err)
		return
	}
	if topic == nil {
		s.log.Warn("system notification topic no longer exists", "event", event.String())
		return
	}

	if err := s.delivery.SendSystemNotification(ctx, DeliveryInput{
		RecipientUserID: settings.RecipientUserID,
		TopicID:         topic.ID,
		TopicName:       topic.Name,
		Title:           title,
		Body:            body,
	}); err != nil {
		s.log.Error("failed to send system notification", "event", event.String(), "error", err)
		return
	}

	s.log.Info("system notification sent", "event", event.String())
}
