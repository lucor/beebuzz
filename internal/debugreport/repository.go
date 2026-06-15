package debugreport

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// Repository handles debug report database operations.
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new debug report repository.
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// Save inserts a debug report into the database.
func (r *Repository) Save(ctx context.Context, report *DebugReport) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO debug_reports (report_id, device_id, created_at, payload_json)
		 VALUES (?, ?, ?, ?)`,
		report.ReportID, report.DeviceID, report.CreatedAt, report.PayloadJSON,
	)
	if err != nil {
		return fmt.Errorf("save debug report: %w", err)
	}
	return nil
}
