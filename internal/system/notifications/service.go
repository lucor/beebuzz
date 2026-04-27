package notifications

import (
	"context"
	"fmt"
	"log/slog"

	"lucor.dev/beebuzz/internal/core"
)

const signupCreatedTitle = "New BeeBuzz signup"

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
		Enabled:              input.Enabled,
		RecipientUserID:      adminUserID,
		TopicID:              input.TopicID,
		SignupCreatedEnabled: input.SignupCreatedEnabled,
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
	settings, err := s.repo.GetSettings(ctx)
	if err != nil {
		s.log.Error("failed to read system notification settings", "error", err)
		return
	}
	if settings == nil || !settings.Enabled || !settings.SignupCreatedEnabled {
		return
	}

	topic, err := s.topics.GetTopicByID(ctx, settings.RecipientUserID, settings.TopicID)
	if err != nil {
		s.log.Error("failed to resolve system notification topic", "error", err)
		return
	}
	if topic == nil {
		s.log.Warn("system notification topic no longer exists")
		return
	}

	body := fmt.Sprintf("A new account was created with status %q.", accountStatus)
	if err := s.delivery.SendSystemNotification(ctx, DeliveryInput{
		RecipientUserID: settings.RecipientUserID,
		TopicID:         topic.ID,
		TopicName:       topic.Name,
		Title:           signupCreatedTitle,
		Body:            body,
	}); err != nil {
		s.log.Error("failed to send signup system notification", "created_user_id", createdUserID, "error", err)
		return
	}

	s.log.Info("signup system notification sent", "created_user_id", createdUserID)
}
