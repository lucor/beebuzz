package plan

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// Repository reads plan entitlement and usage aggregates.
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a plan repository.
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// GetEntitlement reads the runtime plan cache for a user.
func (r *Repository) GetEntitlement(ctx context.Context, userID string) (*Entitlement, error) {
	var entitlement Entitlement
	err := r.db.GetContext(ctx, &entitlement,
		`SELECT plan, plan_expires_at
		 FROM users
		 WHERE id = ?`,
		userID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get plan entitlement: %w", err)
	}
	return &entitlement, nil
}

// GetUsage sums message usage for the requested inclusive UTC-millis range.
func (r *Repository) GetUsage(ctx context.Context, userID string, fromMs, toMs int64) (*Usage, error) {
	var usage Usage
	err := r.db.GetContext(ctx, &usage,
		`SELECT COALESCE(SUM(notifications_total), 0) AS messages
		 FROM daily_usage_summary
		 WHERE user_id = ? AND day_start_ms >= ? AND day_start_ms <= ?`,
		userID, fromMs, toMs,
	)
	if err != nil {
		return nil, fmt.Errorf("get plan usage: %w", err)
	}
	return &usage, nil
}
