package notifications

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

const settingsID = 1

// Repository provides data access for system notification settings.
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new system notifications repository.
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// GetSettings retrieves the singleton settings row.
func (r *Repository) GetSettings(ctx context.Context) (*Settings, error) {
	var settings Settings
	err := r.db.GetContext(ctx, &settings,
		`SELECT enabled, recipient_user_id, topic_id, signup_created_enabled, created_at, updated_at
		 FROM system_notification_settings
		 WHERE id = ?`,
		settingsID,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get system notification settings: %w", err)
	}
	return &settings, nil
}

// UpsertSettings stores the singleton settings row.
func (r *Repository) UpsertSettings(ctx context.Context, settings Settings) (*Settings, error) {
	now := time.Now().UTC().UnixMilli()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO system_notification_settings
			(id, enabled, recipient_user_id, topic_id, signup_created_enabled, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
			enabled = excluded.enabled,
			recipient_user_id = excluded.recipient_user_id,
			topic_id = excluded.topic_id,
			signup_created_enabled = excluded.signup_created_enabled,
			updated_at = excluded.updated_at`,
		settingsID,
		settings.Enabled,
		settings.RecipientUserID,
		settings.TopicID,
		settings.SignupCreatedEnabled,
		now,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert system notification settings: %w", err)
	}

	return r.GetSettings(ctx)
}
