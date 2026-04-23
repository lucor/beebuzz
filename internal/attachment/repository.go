package attachment

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository provides data access for the attachment domain.
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new attachment repository.
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// Create creates a new attachment record.
func (r *Repository) Create(ctx context.Context, topicID, mimeType string, fileSizeBytes int, ttl time.Duration) (*DBAttachment, error) {
	id := uuid.NewString()
	token := uuid.NewString()
	now := time.Now().UTC().UnixMilli()
	expiresAt := time.Now().UTC().Add(ttl).UnixMilli()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO attachments (id, token, topic_id, mime_type, file_size_bytes, created_at, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, token, topicID, mimeType, fileSizeBytes, now, expiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create attachment: %w", err)
	}

	return &DBAttachment{
		ID:            id,
		Token:         token,
		TopicID:       topicID,
		MimeType:      mimeType,
		FileSizeBytes: fileSizeBytes,
		CreatedAt:     now,
		ExpiresAt:     expiresAt,
	}, nil
}

// GetByToken retrieves an attachment by token. Returns nil, nil if not found.
func (r *Repository) GetByToken(ctx context.Context, token string) (*DBAttachment, error) {
	var att DBAttachment
	err := r.db.GetContext(ctx, &att,
		`SELECT id, token, topic_id, mime_type, file_size_bytes, created_at, expires_at
		 FROM attachments WHERE token = ?`,
		token,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get attachment: %w", err)
	}

	return &att, nil
}

// ListExpired returns all attachment records where expires_at is in the past.
func (r *Repository) ListExpired(ctx context.Context) ([]DBAttachment, error) {
	var atts []DBAttachment
	err := r.db.SelectContext(ctx, &atts,
		`SELECT id, token, topic_id, mime_type, file_size_bytes, created_at, expires_at
		 FROM attachments WHERE expires_at < ?`,
		time.Now().UTC().UnixMilli(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list expired attachments: %w", err)
	}
	return atts, nil
}

// DeleteExpired deletes all expired attachments.
func (r *Repository) DeleteExpired(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM attachments WHERE expires_at < ?`,
		time.Now().UTC().UnixMilli(),
	)
	if err != nil {
		return fmt.Errorf("failed to delete expired attachments: %w", err)
	}
	return nil
}
