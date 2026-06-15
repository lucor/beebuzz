package debugreport

import (
	"context"
	"log/slog"
)

// Service handles debug report business logic.
type Service struct {
	repo *Repository
	log  *slog.Logger
}

// NewService creates a new debug report service.
func NewService(repo *Repository, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

// Save persists a validated debug report.
func (s *Service) Save(ctx context.Context, report *DebugReport) error {
	return s.repo.Save(ctx, report)
}
