package token

import (
	"context"
	"errors"

	"lucor.dev/beebuzz/internal/secure"
)

// Service provides token business logic.
type Service struct {
	repo           *Repository
	topicValidator TopicValidator
}

// NewService creates a new token service.
func NewService(repo *Repository, topicValidator TopicValidator) *Service {
	return &Service{repo: repo, topicValidator: topicValidator}
}

// CreateAPIToken creates a new API token for the user and associates the given topics.
func (s *Service) CreateAPIToken(ctx context.Context, userID, name, description string, topicIDs []string) (string, string, error) {
	if len(topicIDs) == 0 {
		return "", "", ErrAtLeastOneTopic
	}
	if err := s.topicValidator.ValidateTopicIDs(ctx, userID, topicIDs); err != nil {
		if errors.Is(err, ErrInvalidTopicSelection) {
			return "", "", err
		}
		return "", "", ErrInvalidTopicSelection
	}

	token, err := secure.NewAPIToken()
	if err != nil {
		return "", "", err
	}

	tokenHash := secure.Hash(token)
	tokenID, err := s.repo.CreateAPITokenWithTopics(ctx, userID, name, tokenHash, description, topicIDs)
	if err != nil {
		return "", "", err
	}

	return token, tokenID, nil
}

// ListAPITokens lists all active API tokens for the user.
func (s *Service) ListAPITokens(ctx context.Context, userID string) ([]APITokenResponse, error) {
	tokens, err := s.repo.ListAPITokens(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]APITokenResponse, len(tokens))
	for i := range tokens {
		topicIDs, err := s.repo.GetAPITokenTopicIDs(ctx, tokens[i].ID)
		if err != nil {
			return nil, err
		}
		result[i] = ToAPITokenResponse(&tokens[i], topicIDs)
	}

	return result, nil
}

// UpdateAPIToken updates an API token's name, description, and topic associations.
func (s *Service) UpdateAPIToken(ctx context.Context, userID, tokenID, name, description string, topicIDs []string) error {
	if len(topicIDs) == 0 {
		return ErrAtLeastOneTopic
	}
	if err := s.topicValidator.ValidateTopicIDs(ctx, userID, topicIDs); err != nil {
		if errors.Is(err, ErrInvalidTopicSelection) {
			return err
		}
		return ErrInvalidTopicSelection
	}

	t, err := s.repo.GetAPITokenByID(ctx, tokenID, userID)
	if err != nil {
		return err
	}
	if t == nil {
		return ErrTokenNotFound
	}

	if err := s.repo.UpdateAPITokenWithTopics(ctx, userID, tokenID, name, description, topicIDs); err != nil {
		return err
	}

	return nil
}

// RevokeAPIToken revokes an API token.
func (s *Service) RevokeAPIToken(ctx context.Context, userID, tokenID string) error {
	t, err := s.repo.GetAPITokenByID(ctx, tokenID, userID)
	if err != nil {
		return err
	}
	if t == nil {
		return ErrTokenNotFound
	}
	return s.repo.RevokeAPIToken(ctx, userID, tokenID)
}

// ValidateAPIToken validates an API token and returns the user ID.
func (s *Service) ValidateAPIToken(ctx context.Context, token string) (string, error) {
	return s.repo.ValidateAPIToken(ctx, secure.Hash(token))
}

// ValidateAPITokenForTopic validates an API token and checks authorization for a specific topic.
// Returns the user ID and topic ID on success.
func (s *Service) ValidateAPITokenForTopic(ctx context.Context, token, topicName string) (userID, topicID string, err error) {
	return s.repo.ValidateAPITokenForTopic(ctx, secure.Hash(token), topicName)
}
