package user

import "context"

// Service provides user business logic.
type Service struct {
	repo *Repository
}

// NewService creates a new user service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GetMe retrieves the current user by ID.
func (s *Service) GetMe(ctx context.Context, userID string) (*User, error) {
	return s.repo.GetByID(ctx, userID)
}

