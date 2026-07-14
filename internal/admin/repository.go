package admin

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"go.beebuzz.app/beebuzz/internal/billing"
	"go.beebuzz.app/beebuzz/internal/core"
)

// DBUser represents a user record from the database.
type DBUser struct {
	ID                 string                      `db:"id"`
	Email              string                      `db:"email"`
	IsAdmin            bool                        `db:"is_admin"`
	AccountStatus      core.AccountStatus          `db:"account_status"`
	Plan               core.Plan                   `db:"plan"`
	PlanExpiresAt      *int64                      `db:"plan_expires_at"`
	SubscriptionStatus *billing.SubscriptionStatus `db:"subscription_status"`
	UsageThisMonth     int                         `db:"usage_this_month"`
	CreatedAt          int64                       `db:"created_at"`
	UpdatedAt          int64                       `db:"updated_at"`
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
	monthStart, nextMonthStart := currentMonthMillis()
	err := r.db.GetContext(ctx, &user,
		`SELECT users.id, users.email, users.is_admin, users.account_status, users.plan, users.plan_expires_at,
		        (
		            SELECT billing_subscriptions.status
		            FROM billing_subscriptions
		            WHERE billing_subscriptions.user_id = users.id
		            ORDER BY billing_subscriptions.provider_event_at DESC, billing_subscriptions.updated_at DESC
		            LIMIT 1
		        ) AS subscription_status,
		        COALESCE((
		            SELECT SUM(daily_usage_summary.notifications_total)
		            FROM daily_usage_summary
		            WHERE daily_usage_summary.user_id = users.id
		              AND daily_usage_summary.day_start_ms >= ?
		              AND daily_usage_summary.day_start_ms < ?
		        ), 0) AS usage_this_month,
		        users.created_at, users.updated_at
		 FROM users WHERE users.id = ?`,
		monthStart,
		nextMonthStart,
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
	monthStart, nextMonthStart := currentMonthMillis()
	err := r.db.SelectContext(ctx, &users, `
		SELECT users.id, users.email, users.is_admin, users.account_status, users.plan, users.plan_expires_at,
		       (
		           SELECT billing_subscriptions.status
		           FROM billing_subscriptions
		           WHERE billing_subscriptions.user_id = users.id
		           ORDER BY billing_subscriptions.provider_event_at DESC, billing_subscriptions.updated_at DESC
		           LIMIT 1
		       ) AS subscription_status,
		       COALESCE((
		           SELECT SUM(daily_usage_summary.notifications_total)
		           FROM daily_usage_summary
		           WHERE daily_usage_summary.user_id = users.id
		             AND daily_usage_summary.day_start_ms >= ?
		             AND daily_usage_summary.day_start_ms < ?
		       ), 0) AS usage_this_month,
		       users.created_at, users.updated_at
		FROM users
		ORDER BY users.created_at DESC`,
		monthStart,
		nextMonthStart,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get all users: %w", err)
	}
	return users, nil
}

func currentMonthMillis() (int64, int64) {
	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	nextMonthStart := monthStart.AddDate(0, 1, 0)
	return monthStart.UnixMilli(), nextMonthStart.UnixMilli()
}
