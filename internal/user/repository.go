package user

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// Repository provides data access for the user domain.
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new user repository.
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// GetByID retrieves a user by ID.
func (r *Repository) GetByID(ctx context.Context, userID string) (*User, error) {
	var user User
	err := r.db.GetContext(ctx, &user,
		`SELECT users.id, users.email, users.is_admin, users.account_status, users.plan, users.plan_expires_at,
		        (
		            SELECT billing_subscriptions.status
		            FROM billing_subscriptions
		            WHERE billing_subscriptions.user_id = users.id
		            ORDER BY billing_subscriptions.provider_event_at DESC, billing_subscriptions.updated_at DESC
		            LIMIT 1
		        ) AS subscription_status,
		        users.created_at, users.updated_at
		 FROM users WHERE users.id = ?`,
		userID,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}
