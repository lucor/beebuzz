package admin

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"lucor.dev/beebuzz/internal/core"
)

// DBUser represents a user record from the database.
type DBUser struct {
	ID             string             `db:"id"`
	Email          string             `db:"email"`
	IsAdmin        bool               `db:"is_admin"`
	AccountStatus  core.AccountStatus `db:"account_status"`
	SignupReason   *string            `db:"signup_reason"`
	TrialStartedAt *int64             `db:"trial_started_at"`
	CreatedAt      int64              `db:"created_at"`
	UpdatedAt      int64              `db:"updated_at"`
}

// Repository provides data access for the admin domain.
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new admin repository.
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// GetUserByID retrieves a user by ID.
func (r *Repository) GetUserByID(ctx context.Context, userID string) (*DBUser, error) {
	var user DBUser
	err := r.db.GetContext(ctx, &user,
		`SELECT id, email, is_admin, account_status, signup_reason, trial_started_at, created_at, updated_at
		 FROM users WHERE id = ?`,
		userID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// UpdateAccountStatus updates a user's account_status with optimistic locking.
// Returns (true, nil) if RowsAffected == 1, (false, nil) if RowsAffected == 0.
func (r *Repository) UpdateAccountStatus(ctx context.Context, userID string, fromStatus core.AccountStatus, toStatus core.AccountStatus) (bool, error) {
	now := time.Now().UnixMilli()
	result, err := r.db.ExecContext(ctx,
		`UPDATE users SET account_status = ?, updated_at = ? WHERE id = ? AND account_status = ?`,
		toStatus, now, userID, fromStatus,
	)
	if err != nil {
		return false, fmt.Errorf("failed to update account status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to read rows affected: %w", err)
	}
	return rowsAffected > 0, nil
}

// GetAllUsers retrieves all users ordered by creation date.
func (r *Repository) GetAllUsers(ctx context.Context) ([]DBUser, error) {
	var users []DBUser
	err := r.db.SelectContext(ctx, &users, `
		SELECT id, email, is_admin, account_status, signup_reason, trial_started_at, created_at, updated_at
		FROM users
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to get all users: %w", err)
	}
	return users, nil
}
