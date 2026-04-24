package topic

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
)

const (
	defaultTopicName        = "general"
	defaultTopicDescription = "Default topic for all notifications"
)

// Service provides topic business logic.
type Service struct {
	repo *Repository
	log  *slog.Logger
}

// NewService creates a new topic service.
func NewService(repo *Repository, logger *slog.Logger) *Service {
	return &Service{repo: repo, log: logger}
}

// GetTopics retrieves all topics for a user.
func (s *Service) GetTopics(ctx context.Context, userID string) ([]Topic, error) {
	return s.repo.GetByUser(ctx, userID)
}

// GetTopicByID retrieves a topic owned by the user.
func (s *Service) GetTopicByID(ctx context.Context, userID, topicID string) (*Topic, error) {
	return s.repo.GetByID(ctx, userID, topicID)
}

// CreateTopic creates a new topic after validating the name.
func (s *Service) CreateTopic(ctx context.Context, userID, name, description string) (*Topic, error) {
	if name == defaultTopicName {
		return nil, ErrTopicNameReserved
	}
	return s.repo.Create(ctx, userID, name, description)
}

// UpdateTopic updates a topic's description.
func (s *Service) UpdateTopic(ctx context.Context, userID, topicID, description string) error {
	return s.repo.Update(ctx, userID, topicID, description)
}

// DeleteTopic deletes a topic, enforcing protection of the "general" topic.
func (s *Service) DeleteTopic(ctx context.Context, userID, topicID string) error {
	topic, err := s.repo.GetByID(ctx, userID, topicID)
	if err != nil {
		return err
	}
	if topic == nil {
		return ErrTopicNotFound
	}
	if topic.Name == defaultTopicName {
		return ErrTopicProtected
	}
	return s.repo.Delete(ctx, userID, topicID)
}

// CreateDefaultTopic creates the default "general" topic for a new user.
func (s *Service) CreateDefaultTopic(ctx context.Context, userID string) error {
	logger := s.log.With("user_id", userID)

	_, err := s.repo.Create(ctx, userID, defaultTopicName, defaultTopicDescription)
	if err != nil {
		logger.Error("failed to create default topic", "error", err)
		return fmt.Errorf("create default topic: %w", err)
	}
	logger.Info("default topic created")
	return nil
}

// GetByName retrieves a topic by name.
func (s *Service) GetByName(ctx context.Context, userID, name string) (*Topic, error) {
	return s.repo.GetByName(ctx, userID, name)
}

// ValidateTopicIDs verifies that all provided topic IDs belong to the given user.
func (s *Service) ValidateTopicIDs(ctx context.Context, userID string, topicIDs []string) error {
	uniqueTopicIDs := slices.Clone(topicIDs)
	slices.Sort(uniqueTopicIDs)
	uniqueTopicIDs = slices.Compact(uniqueTopicIDs)
	if len(uniqueTopicIDs) != len(topicIDs) {
		return ErrDuplicateTopicIDs
	}

	count, err := s.repo.CountOwnedByUser(ctx, userID, uniqueTopicIDs)
	if err != nil {
		return fmt.Errorf("validate topic ownership: %w", err)
	}
	if count != len(uniqueTopicIDs) {
		return ErrTopicNotFound
	}

	return nil
}
