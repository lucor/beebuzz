package attachment

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"lucor.dev/beebuzz/internal/core"
)

const (
	// MaxAttachmentSize is the maximum allowed plaintext size for an attachment in bytes (1 MB).
	MaxAttachmentSize = 1024 * 1024
	// AttachmentTTL is how long attachments are retained before expiry.
	// The 6-hour window matches BeeBuzz's real-time alerting model and limits token exposure.
	AttachmentTTL = 6 * time.Hour
)

// Service provides attachment business logic.
type Service struct {
	repo *Repository
	dir  string
	log  *slog.Logger
}

// NewService creates a new attachment service.
func NewService(repo *Repository, attachmentsDir string, log *slog.Logger) *Service {
	return &Service{repo: repo, dir: attachmentsDir, log: log}
}

// Store persists already-encrypted ciphertext for a topic attachment and returns the access token.
// originalSize is the plaintext byte count, stored for UI display.
func (s *Service) Store(ctx context.Context, topicID, mimeType string, originalSize int, data []byte) (string, error) {
	att, err := s.repo.Create(ctx, topicID, mimeType, originalSize, AttachmentTTL)
	if err != nil {
		return "", fmt.Errorf("attachment: create record: %w", err)
	}

	path := filepath.Join(s.dir, att.ID)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return "", fmt.Errorf("attachment: write file: %w", err)
	}

	s.log.Info("attachment stored", "id", att.ID, "topic_id", topicID, "size", len(data))
	return att.Token, nil
}

// GetByToken retrieves the raw (ciphertext) bytes and metadata for an attachment token.
// Returns core.ErrNotFound if the token does not exist, ErrAttachmentExpired if expired.
func (s *Service) GetByToken(ctx context.Context, token string) ([]byte, *DBAttachment, error) {
	att, err := s.repo.GetByToken(ctx, token)
	if err != nil {
		return nil, nil, fmt.Errorf("attachment: get by token: %w", err)
	}
	if att == nil {
		return nil, nil, core.ErrNotFound
	}

	if time.Now().UTC().After(time.UnixMilli(att.ExpiresAt)) {
		return nil, nil, ErrAttachmentExpired
	}

	path := filepath.Join(s.dir, att.ID)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil, core.ErrNotFound
	}
	if err != nil {
		return nil, nil, fmt.Errorf("attachment: read file: %w", err)
	}

	return data, att, nil
}

// CleanupExpired removes expired attachment files from disk and their DB records.
func (s *Service) CleanupExpired(ctx context.Context) error {
	expired, err := s.repo.ListExpired(ctx)
	if err != nil {
		return fmt.Errorf("attachment: list expired: %w", err)
	}

	for _, att := range expired {
		path := filepath.Join(s.dir, att.ID)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			s.log.Warn("failed to remove attachment file", "id", att.ID, "error", err)
		}
	}

	if err := s.repo.DeleteExpired(ctx); err != nil {
		return fmt.Errorf("attachment: delete expired records: %w", err)
	}

	s.log.Info("attachment cleanup complete", "count", len(expired))
	return nil
}
