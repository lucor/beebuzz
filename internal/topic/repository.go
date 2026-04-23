package topic

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"lucor.dev/beebuzz/internal/database"
)

// Repository provides data access for the topic domain.
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new topic repository.
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// Create creates a new topic.
func (r *Repository) Create(ctx context.Context, userID, name, description string) (*Topic, error) {
	id := uuid.NewString()
	now := time.Now().UTC().UnixMilli()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO topics (id, user_id, name, description, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		id, userID, name, description, now, now,
	)
	if err != nil {
		if database.IsUniqueConstraint(err) {
			return nil, ErrTopicNameConflict
		}
		return nil, fmt.Errorf("failed to create topic: %w", err)
	}

	var desc *string
	if description != "" {
		desc = &description
	}

	return &Topic{
		ID:          id,
		UserID:      userID,
		Name:        name,
		Description: desc,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// GetByUser retrieves all topics for a user.
func (r *Repository) GetByUser(ctx context.Context, userID string) ([]Topic, error) {
	var topics []Topic
	err := r.db.SelectContext(ctx, &topics,
		`SELECT id, user_id, name, description, created_at, updated_at
		 FROM topics WHERE user_id = ?
		 ORDER BY CASE WHEN name = 'general' THEN 0 ELSE 1 END, name ASC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get topics: %w", err)
	}
	return topics, nil
}

// GetByID retrieves a topic by ID and user.
func (r *Repository) GetByID(ctx context.Context, userID, topicID string) (*Topic, error) {
	var topic Topic
	err := r.db.GetContext(ctx, &topic,
		`SELECT id, user_id, name, description, created_at, updated_at
		 FROM topics WHERE id = ? AND user_id = ?`,
		topicID, userID,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get topic: %w", err)
	}
	return &topic, nil
}

// GetByName retrieves a topic by user and name.
func (r *Repository) GetByName(ctx context.Context, userID, name string) (*Topic, error) {
	var topic Topic
	err := r.db.GetContext(ctx, &topic,
		`SELECT id, user_id, name, description, created_at, updated_at
		 FROM topics WHERE user_id = ? AND name = ?`,
		userID, name,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get topic: %w", err)
	}
	return &topic, nil
}

// Update updates a topic's description.
func (r *Repository) Update(ctx context.Context, userID, topicID, description string) error {
	now := time.Now().UnixMilli()

	result, err := r.db.ExecContext(ctx,
		`UPDATE topics SET description = ?, updated_at = ? WHERE id = ? AND user_id = ?`,
		description, now, topicID, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to update topic: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read updated topic row count: %w", err)
	}
	if rows == 0 {
		return ErrTopicNotFound
	}

	return nil
}

// Delete deletes a topic by ID and user.
func (r *Repository) Delete(ctx context.Context, userID, topicID string) error {
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM topics WHERE id = ? AND user_id = ?",
		topicID, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete topic: %w", err)
	}
	return nil
}

// CountOwnedByUser returns how many of the provided topic IDs belong to the user.
func (r *Repository) CountOwnedByUser(ctx context.Context, userID string, topicIDs []string) (int, error) {
	if len(topicIDs) == 0 {
		return 0, nil
	}

	placeholders := make([]string, len(topicIDs))
	args := make([]any, 0, len(topicIDs)+1)
	args = append(args, userID)

	for i, topicID := range topicIDs {
		placeholders[i] = "?"
		args = append(args, topicID)
	}

	query := `SELECT COUNT(*) FROM topics WHERE user_id = ? AND id IN (` + strings.Join(placeholders, ",") + `)`

	var count int
	if err := r.db.GetContext(ctx, &count, query, args...); err != nil {
		return 0, fmt.Errorf("failed to count owned topics: %w", err)
	}

	return count, nil
}
