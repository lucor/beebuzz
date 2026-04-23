package webhook

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository provides data access for the webhook domain.
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new webhook repository.
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new webhook. tokenHash must be the SHA-256 hash of the raw token.
func (r *Repository) Create(ctx context.Context, userID, name, description string, payloadType PayloadType, tokenHash, titlePath, bodyPath, priority string) (string, error) {
	webhookID := uuid.NewString()
	now := time.Now().UTC().UnixMilli()

	var desc *string
	if description != "" {
		desc = &description
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO webhooks (id, user_id, token_hash, name, description, payload_type, title_path, body_path, priority, created_at, is_active)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		webhookID, userID, tokenHash, name, desc, payloadType, titlePath, bodyPath, priority, now, true,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create webhook: %w", err)
	}

	return webhookID, nil
}

// CreateWithTopics atomically creates a webhook and its topic associations.
func (r *Repository) CreateWithTopics(ctx context.Context, userID, name, description string, payloadType PayloadType, tokenHash, titlePath, bodyPath, priority string, topicIDs []string) (string, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to begin webhook create transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	webhookID := uuid.NewString()
	now := time.Now().UTC().UnixMilli()

	var desc *string
	if description != "" {
		desc = &description
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO webhooks (id, user_id, token_hash, name, description, payload_type, title_path, body_path, priority, created_at, is_active)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		webhookID, userID, tokenHash, name, desc, payloadType, titlePath, bodyPath, priority, now, true,
	); err != nil {
		return "", fmt.Errorf("failed to create webhook: %w", err)
	}

	for _, topicID := range topicIDs {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO webhook_topics (webhook_id, topic_id, created_at) VALUES (?, ?, ?)`,
			webhookID, topicID, now,
		); err != nil {
			return "", fmt.Errorf("failed to associate webhook with topic: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit webhook create transaction: %w", err)
	}

	return webhookID, nil
}

// GetByUser retrieves all active webhooks for a user.
func (r *Repository) GetByUser(ctx context.Context, userID string) ([]Webhook, error) {
	var webhooks []Webhook
	err := r.db.SelectContext(ctx, &webhooks,
		`SELECT id, user_id, token_hash, name, description, payload_type, title_path, body_path, priority, is_active, revoked_at, last_used_at, created_at
		 FROM webhooks WHERE user_id = ? AND is_active = 1
		 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get webhooks: %w", err)
	}
	return webhooks, nil
}

// GetByID retrieves a webhook by ID and user. Returns nil, nil if not found.
func (r *Repository) GetByID(ctx context.Context, userID, webhookID string) (*Webhook, error) {
	var webhook Webhook
	err := r.db.GetContext(ctx, &webhook,
		`SELECT id, user_id, token_hash, name, description, payload_type, title_path, body_path, priority, is_active, revoked_at, last_used_at, created_at
		 FROM webhooks WHERE id = ? AND user_id = ?`,
		webhookID, userID,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook: %w", err)
	}
	return &webhook, nil
}

// GetByTokenHash looks up an active webhook by its token hash. Returns nil, nil if not found.
func (r *Repository) GetByTokenHash(ctx context.Context, tokenHash string) (*Webhook, error) {
	var webhook Webhook
	err := r.db.GetContext(ctx, &webhook,
		`SELECT w.id, w.user_id, w.token_hash, w.name, w.description, w.payload_type, w.title_path, w.body_path, w.priority, w.is_active, w.revoked_at, w.last_used_at, w.created_at
		 FROM webhooks w
		 JOIN users u ON w.user_id = u.id
		 WHERE w.token_hash = ?
		   AND w.is_active = 1
		   AND u.account_status = 'active'`,
		tokenHash,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook by token hash: %w", err)
	}
	return &webhook, nil
}

// TouchLastUsedAt sets last_used_at to the current time for the given webhook.
func (r *Repository) TouchLastUsedAt(ctx context.Context, webhookID string) error {
	now := time.Now().UTC().UnixMilli()
	_, err := r.db.ExecContext(ctx, "UPDATE webhooks SET last_used_at = ? WHERE id = ?", now, webhookID)
	if err != nil {
		return fmt.Errorf("failed to update webhook last_used_at: %w", err)
	}
	return nil
}

// Revoke marks a webhook as inactive.
func (r *Repository) Revoke(ctx context.Context, userID, webhookID string) error {
	now := time.Now().UTC().UnixMilli()
	_, err := r.db.ExecContext(ctx,
		"UPDATE webhooks SET is_active = 0, revoked_at = ? WHERE id = ? AND user_id = ?",
		now, webhookID, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to revoke webhook: %w", err)
	}
	return nil
}

// AddTopic associates a webhook with a topic.
func (r *Repository) AddTopic(ctx context.Context, webhookID, topicID string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO webhook_topics (webhook_id, topic_id, created_at)
		 VALUES (?, ?, ?)`,
		webhookID, topicID, time.Now().UTC().UnixMilli(),
	)
	if err != nil {
		return fmt.Errorf("failed to associate webhook with topic: %w", err)
	}
	return nil
}

// GetTopics retrieves topic names for a webhook.
func (r *Repository) GetTopics(ctx context.Context, webhookID string) ([]string, error) {
	var topics []string
	err := r.db.SelectContext(ctx, &topics,
		`SELECT t.name FROM topics t
		 JOIN webhook_topics wt ON t.id = wt.topic_id
		 WHERE wt.webhook_id = ?
		 ORDER BY wt.created_at DESC`,
		webhookID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook topics: %w", err)
	}
	return topics, nil
}

// GetTopicIDs retrieves topic IDs for a webhook.
func (r *Repository) GetTopicIDs(ctx context.Context, webhookID string) ([]string, error) {
	var topicIDs []string
	err := r.db.SelectContext(ctx, &topicIDs,
		`SELECT topic_id FROM webhook_topics
		 WHERE webhook_id = ?
		 ORDER BY created_at DESC`,
		webhookID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook topic IDs: %w", err)
	}
	return topicIDs, nil
}

// GetTopicsWithIDs retrieves topic IDs and names for a webhook.
func (r *Repository) GetTopicsWithIDs(ctx context.Context, webhookID string) ([]WebhookTopic, error) {
	var topics []WebhookTopic
	err := r.db.SelectContext(ctx, &topics,
		`SELECT t.id, t.name FROM topics t
		 JOIN webhook_topics wt ON t.id = wt.topic_id
		 WHERE wt.webhook_id = ?
		 ORDER BY wt.created_at DESC`,
		webhookID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook topics with IDs: %w", err)
	}
	return topics, nil
}

// Update updates mutable fields of a webhook.
func (r *Repository) Update(ctx context.Context, userID, webhookID, name, description string, payloadType PayloadType, titlePath, bodyPath, priority string) error {
	var desc *string
	if description != "" {
		desc = &description
	}

	_, err := r.db.ExecContext(ctx,
		`UPDATE webhooks SET name = ?, description = ?, payload_type = ?, title_path = ?, body_path = ?, priority = ?
		 WHERE id = ? AND user_id = ?`,
		name, desc, payloadType, titlePath, bodyPath, priority, webhookID, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to update webhook: %w", err)
	}
	return nil
}

// UpdateWithTopics atomically updates a webhook and replaces topic associations.
func (r *Repository) UpdateWithTopics(ctx context.Context, userID, webhookID, name, description string, payloadType PayloadType, titlePath, bodyPath, priority string, topicIDs []string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin webhook update transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	var desc *string
	if description != "" {
		desc = &description
	}

	result, err := tx.ExecContext(ctx,
		`UPDATE webhooks SET name = ?, description = ?, payload_type = ?, title_path = ?, body_path = ?, priority = ?
		 WHERE id = ? AND user_id = ?`,
		name, desc, payloadType, titlePath, bodyPath, priority, webhookID, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to update webhook: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read updated webhook row count: %w", err)
	}
	if rows == 0 {
		return ErrWebhookNotFound
	}

	var existingTopicIDs []string
	if err := tx.SelectContext(ctx, &existingTopicIDs,
		`SELECT topic_id FROM webhook_topics WHERE webhook_id = ? ORDER BY created_at DESC`,
		webhookID,
	); err != nil {
		return fmt.Errorf("failed to load webhook topics: %w", err)
	}

	existingMap := make(map[string]struct{}, len(existingTopicIDs))
	for _, topicID := range existingTopicIDs {
		existingMap[topicID] = struct{}{}
	}

	desiredMap := make(map[string]struct{}, len(topicIDs))
	for _, topicID := range topicIDs {
		desiredMap[topicID] = struct{}{}
	}

	now := time.Now().UTC().UnixMilli()
	for _, topicID := range topicIDs {
		if _, exists := existingMap[topicID]; exists {
			continue
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO webhook_topics (webhook_id, topic_id, created_at) VALUES (?, ?, ?)`,
			webhookID, topicID, now,
		); err != nil {
			return fmt.Errorf("failed to add webhook topic association: %w", err)
		}
	}

	for _, topicID := range existingTopicIDs {
		if _, exists := desiredMap[topicID]; exists {
			continue
		}
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM webhook_topics WHERE webhook_id = ? AND topic_id = ?`,
			webhookID, topicID,
		); err != nil {
			return fmt.Errorf("failed to delete webhook topic association: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit webhook update transaction: %w", err)
	}

	return nil
}

// UpdateTokenHash replaces the token hash for a webhook owned by userID.
func (r *Repository) UpdateTokenHash(ctx context.Context, userID, webhookID, newTokenHash string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE webhooks SET token_hash = ? WHERE id = ? AND user_id = ?`,
		newTokenHash, webhookID, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to update webhook token hash: %w", err)
	}
	return nil
}

// DeleteTopic removes a topic association from a webhook.
func (r *Repository) DeleteTopic(ctx context.Context, webhookID, topicID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM webhook_topics WHERE webhook_id = ? AND topic_id = ?`,
		webhookID, topicID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete webhook topic association: %w", err)
	}
	return nil
}
