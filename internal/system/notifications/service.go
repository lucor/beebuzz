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
	repo     *Repository
	topics   TopicProvider
	delivery Delivery
	log      *slog.Logger
}

// NewService creates a system notifications service.
func NewService(repo *Repository, topics TopicProvider, delivery Delivery, log *slog.Logger) *Service {
	return &Service{
		repo:     repo,
		topics:   topics,
		delivery: delivery,
		log:      log,
	}
}

// GetSettings returns the current singleton settings.
func (s *Service) GetSettings(ctx context.Context) (*Settings, error) {
	return s.repo.GetSettings(ctx)
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

	return s.repo.UpsertSettings(ctx, Settings{
		Enabled:              input.Enabled,
		RecipientUserID:      adminUserID,
		TopicID:              input.TopicID,
		SignupCreatedEnabled: input.SignupCreatedEnabled,
	})
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
