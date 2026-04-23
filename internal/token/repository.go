package token

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository provides data access for the token domain.
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new token repository.
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// CreateAPIToken creates a new API token.
func (r *Repository) CreateAPIToken(ctx context.Context, userID, name, tokenHash, description string) (string, error) {
	tokenID := uuid.NewString()
	now := time.Now().UTC().UnixMilli()

	var desc *string
	if description != "" {
		desc = &description
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO api_tokens (id, user_id, token_hash, name, description, created_at, is_active)
		 VALUES (?, ?, ?, ?, ?, ?, 1)`,
		tokenID, userID, tokenHash, name, desc, now,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create API token: %w", err)
	}

	return tokenID, nil
}

// CreateAPITokenWithTopics atomically creates an API token and its topic associations.
func (r *Repository) CreateAPITokenWithTopics(ctx context.Context, userID, name, tokenHash, description string, topicIDs []string) (string, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to begin API token create transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	tokenID := uuid.NewString()
	now := time.Now().UTC().UnixMilli()

	var desc *string
	if description != "" {
		desc = &description
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO api_tokens (id, user_id, token_hash, name, description, created_at, is_active)
		 VALUES (?, ?, ?, ?, ?, ?, 1)`,
		tokenID, userID, tokenHash, name, desc, now,
	); err != nil {
		return "", fmt.Errorf("failed to create API token: %w", err)
	}

	for _, topicID := range topicIDs {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO api_token_topics (api_token_id, topic_id, created_at) VALUES (?, ?, ?)`,
			tokenID, topicID, now,
		); err != nil {
			return "", fmt.Errorf("failed to associate API token with topic: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit API token create transaction: %w", err)
	}

	return tokenID, nil
}

// GetAPITokenByID retrieves an API token by ID and user.
func (r *Repository) GetAPITokenByID(ctx context.Context, tokenID, userID string) (*APIToken, error) {
	var token APIToken
	err := r.db.GetContext(ctx, &token,
		`SELECT id, user_id, token_hash, name, description, expires_at, revoked_at, created_at, last_used_at, is_active
		 FROM api_tokens WHERE id = ? AND user_id = ?`,
		tokenID, userID,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get API token: %w", err)
	}
	return &token, nil
}

// ListAPITokens retrieves all active API tokens for a user.
func (r *Repository) ListAPITokens(ctx context.Context, userID string) ([]APIToken, error) {
	var tokens []APIToken
	err := r.db.SelectContext(ctx, &tokens,
		`SELECT id, user_id, token_hash, name, description, expires_at, revoked_at, created_at, last_used_at, is_active
		 FROM api_tokens WHERE user_id = ? AND is_active = 1
		 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get API tokens: %w", err)
	}
	return tokens, nil
}

// GetAPITokenTopicNames retrieves topic names associated with an API token.
func (r *Repository) GetAPITokenTopicNames(ctx context.Context, tokenID string) ([]string, error) {
	var names []string
	err := r.db.SelectContext(ctx, &names,
		`SELECT t.name FROM topics t
		 JOIN api_token_topics att ON t.id = att.topic_id
		 WHERE att.api_token_id = ?
		 ORDER BY t.name`,
		tokenID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get API token topic names: %w", err)
	}
	return names, nil
}

// GetAPITokenTopicIDs retrieves topic IDs associated with an API token.
func (r *Repository) GetAPITokenTopicIDs(ctx context.Context, tokenID string) ([]string, error) {
	var ids []string
	err := r.db.SelectContext(ctx, &ids,
		`SELECT topic_id FROM api_token_topics WHERE api_token_id = ? ORDER BY created_at DESC`,
		tokenID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get API token topic IDs: %w", err)
	}
	return ids, nil
}

// AddTopicToAPIToken associates a topic with an API token.
func (r *Repository) AddTopicToAPIToken(ctx context.Context, tokenID, topicID string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO api_token_topics (api_token_id, topic_id, created_at) VALUES (?, ?, ?)`,
		tokenID, topicID, time.Now().UTC().UnixMilli(),
	)
	if err != nil {
		return fmt.Errorf("failed to associate API token with topic: %w", err)
	}
	return nil
}

// DeleteTopicFromAPIToken removes a topic association from an API token.
func (r *Repository) DeleteTopicFromAPIToken(ctx context.Context, tokenID, topicID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM api_token_topics WHERE api_token_id = ? AND topic_id = ?`,
		tokenID, topicID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete API token topic association: %w", err)
	}
	return nil
}

// RevokeAPIToken revokes an API token.
func (r *Repository) RevokeAPIToken(ctx context.Context, userID, tokenID string) error {
	now := time.Now().UTC().UnixMilli()
	_, err := r.db.ExecContext(ctx,
		"UPDATE api_tokens SET is_active = 0, revoked_at = ? WHERE id = ? AND user_id = ?",
		now, tokenID, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to revoke API token: %w", err)
	}
	return nil
}

// ValidateAPIToken validates an API token and returns the associated user ID.
func (r *Repository) ValidateAPIToken(ctx context.Context, tokenHash string) (string, error) {
	now := time.Now().UTC().UnixMilli()

	var userID string
	err := r.db.QueryRowContext(ctx,
		`SELECT at.user_id FROM api_tokens at
		 JOIN users u ON at.user_id = u.id
		 WHERE at.token_hash = ? AND at.is_active = 1 AND (at.expires_at IS NULL OR at.expires_at > ?)
		   AND u.account_status = 'active'`,
		tokenHash, now,
	).Scan(&userID)

	if err == sql.ErrNoRows {
		return "", fmt.Errorf("invalid or expired API token")
	}
	if err != nil {
		return "", fmt.Errorf("failed to validate API token: %w", err)
	}

	_, err = r.db.ExecContext(ctx, "UPDATE api_tokens SET last_used_at = ? WHERE token_hash = ?", now, tokenHash)
	if err != nil {
		return "", fmt.Errorf("failed to update API token last_used_at: %w", err)
	}

	return userID, nil
}

// ValidateAPITokenForTopic validates an API token for a specific topic.
// Returns the user ID and topic ID if the token is valid, active, not expired, and authorized for the topic.
func (r *Repository) ValidateAPITokenForTopic(ctx context.Context, tokenHash, topicName string) (userID, topicID string, err error) {
	now := time.Now().UTC().UnixMilli()

	err = r.db.QueryRowContext(ctx,
		`SELECT at.user_id, t.id FROM api_tokens at
		 JOIN api_token_topics att ON at.id = att.api_token_id
		 JOIN topics t ON att.topic_id = t.id
		 JOIN users u ON at.user_id = u.id
		 WHERE at.token_hash = ?
		   AND at.is_active = 1
		   AND (at.expires_at IS NULL OR at.expires_at > ?)
		   AND t.name = ?
		   AND t.user_id = at.user_id
		   AND u.account_status = 'active'`,
		tokenHash, now, topicName,
	).Scan(&userID, &topicID)

	if err == sql.ErrNoRows {
		return "", "", fmt.Errorf("token not authorized for topic")
	}
	if err != nil {
		return "", "", fmt.Errorf("failed to validate API token for topic: %w", err)
	}

	_, err = r.db.ExecContext(ctx, "UPDATE api_tokens SET last_used_at = ? WHERE token_hash = ?", now, tokenHash)
	if err != nil {
		return "", "", fmt.Errorf("failed to update API token last_used_at: %w", err)
	}

	return userID, topicID, nil
}

// UpdateAPIToken updates an API token's name and description.
func (r *Repository) UpdateAPIToken(ctx context.Context, userID, tokenID, name, description string) error {
	var desc *string
	if description != "" {
		desc = &description
	}

	_, err := r.db.ExecContext(ctx,
		`UPDATE api_tokens SET name = ?, description = ? WHERE id = ? AND user_id = ?`,
		name, desc, tokenID, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to update API token: %w", err)
	}
	return nil
}

// UpdateAPITokenWithTopics atomically updates an API token and replaces topic associations.
func (r *Repository) UpdateAPITokenWithTopics(ctx context.Context, userID, tokenID, name, description string, topicIDs []string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin API token update transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	var desc *string
	if description != "" {
		desc = &description
	}

	result, err := tx.ExecContext(ctx,
		`UPDATE api_tokens SET name = ?, description = ? WHERE id = ? AND user_id = ?`,
		name, desc, tokenID, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to update API token: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read updated API token row count: %w", err)
	}
	if rows == 0 {
		return ErrTokenNotFound
	}

	var existingIDs []string
	if err := tx.SelectContext(ctx, &existingIDs,
		`SELECT topic_id FROM api_token_topics WHERE api_token_id = ? ORDER BY created_at DESC`,
		tokenID,
	); err != nil {
		return fmt.Errorf("failed to load API token topics: %w", err)
	}

	existingMap := make(map[string]struct{}, len(existingIDs))
	for _, topicID := range existingIDs {
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
			`INSERT INTO api_token_topics (api_token_id, topic_id, created_at) VALUES (?, ?, ?)`,
			tokenID, topicID, now,
		); err != nil {
			return fmt.Errorf("failed to add API token topic association: %w", err)
		}
	}

	for _, topicID := range existingIDs {
		if _, exists := desiredMap[topicID]; exists {
			continue
		}
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM api_token_topics WHERE api_token_id = ? AND topic_id = ?`,
			tokenID, topicID,
		); err != nil {
			return fmt.Errorf("failed to delete API token topic association: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit API token update transaction: %w", err)
	}

	return nil
}
