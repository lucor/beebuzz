package event

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository provides data access for the event domain.
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new event repository.
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// InsertEvent inserts a notification event and atomically increments the daily summary.
func (r *Repository) InsertEvent(ctx context.Context, ev *NotificationEvent) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Insert raw event.
	_, err = tx.ExecContext(ctx,
		`INSERT INTO notification_events
			(id, user_id, event_type, device_id, topic, source, encryption_mode,
			 attachment_bytes, fail_reason, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ev.ID, ev.UserID, ev.EventType, ev.DeviceID, ev.Topic, ev.Source,
		ev.EncryptionMode, ev.AttachmentBytes, ev.FailReason, ev.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert event: %w", err)
	}

	// Compute summary deltas from this single event.
	delta := r.eventToDelta(ev)
	now := time.Now().UTC().UnixMilli()

	// Upsert daily summary with incremental deltas.
	_, err = tx.ExecContext(ctx,
		`INSERT INTO daily_usage_summary (
			user_id, day_start_ms,
			notifications_total, notifications_server_trusted, notifications_e2e,
			notifications_delivered, notifications_failed,
			attachments_count, attachments_bytes_total,
			sources_cli, sources_webhook, sources_api, sources_internal,
			devices_lost, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, day_start_ms) DO UPDATE SET
			notifications_total      = notifications_total + excluded.notifications_total,
			notifications_server_trusted = notifications_server_trusted + excluded.notifications_server_trusted,
			notifications_e2e        = notifications_e2e + excluded.notifications_e2e,
			notifications_delivered  = notifications_delivered + excluded.notifications_delivered,
			notifications_failed     = notifications_failed + excluded.notifications_failed,
			attachments_count        = attachments_count + excluded.attachments_count,
			attachments_bytes_total  = attachments_bytes_total + excluded.attachments_bytes_total,
			sources_cli              = sources_cli + excluded.sources_cli,
			sources_webhook          = sources_webhook + excluded.sources_webhook,
			sources_api              = sources_api + excluded.sources_api,
			sources_internal         = sources_internal + excluded.sources_internal,
			devices_lost             = devices_lost + excluded.devices_lost,
			updated_at               = excluded.updated_at`,
		ev.UserID, DayStartMs(time.UnixMilli(ev.CreatedAt)),
		delta.NotificationsTotal, delta.NotificationsServerTrusted, delta.NotificationsE2E,
		delta.NotificationsDelivered, delta.NotificationsFailed,
		delta.AttachmentsCount, delta.AttachmentsBytesTotal,
		delta.SourcesCLI, delta.SourcesWebhook, delta.SourcesAPI, delta.SourcesInternal,
		delta.DevicesLost, now, now,
	)
	if err != nil {
		return fmt.Errorf("upsert daily summary: %w", err)
	}

	return tx.Commit()
}

// eventToDelta computes the incremental summary deltas for a single event.
func (r *Repository) eventToDelta(ev *NotificationEvent) DailyUsageSummary {
	var d DailyUsageSummary

	switch ev.EventType {
	case TypeNotificationCreated:
		d.NotificationsTotal = 1

		if ev.EncryptionMode != nil {
			switch *ev.EncryptionMode {
			case EncryptionServerTrusted:
				d.NotificationsServerTrusted = 1
			case EncryptionE2E:
				d.NotificationsE2E = 1
			}
		}

		if ev.AttachmentBytes != nil {
			d.AttachmentsCount = 1
			d.AttachmentsBytesTotal = *ev.AttachmentBytes
		}

		if ev.Source != nil {
			switch *ev.Source {
			case SourceCLI:
				d.SourcesCLI = 1
			case SourceWebhook:
				d.SourcesWebhook = 1
			case SourceAPI:
				d.SourcesAPI = 1
			case SourceInternal:
				d.SourcesInternal = 1
			}
		}

	case TypeNotificationDelivered:
		d.NotificationsDelivered = 1

	case TypeNotificationFailed:
		d.NotificationsFailed = 1
		if ev.FailReason != nil && *ev.FailReason == FailSubscriptionGone {
			d.DevicesLost = 1
		}
	}

	return d
}

// GetDailySummaries retrieves daily usage summaries for a user within a time range.
func (r *Repository) GetDailySummaries(ctx context.Context, userID string, fromMs, toMs int64) ([]DailyUsageSummary, error) {
	var summaries []DailyUsageSummary
	err := r.db.SelectContext(ctx, &summaries,
		`SELECT user_id, day_start_ms,
			notifications_total, notifications_server_trusted, notifications_e2e,
			notifications_delivered, notifications_failed,
			attachments_count, attachments_bytes_total,
			sources_cli, sources_webhook, sources_api, sources_internal,
			devices_lost, created_at, updated_at
		 FROM daily_usage_summary
		 WHERE user_id = ? AND day_start_ms >= ? AND day_start_ms <= ?
		 ORDER BY day_start_ms`,
		userID, fromMs, toMs,
	)
	if err != nil {
		return nil, fmt.Errorf("get daily summaries: %w", err)
	}
	return summaries, nil
}

// DeleteEventsOlderThan deletes events older than cutoffMs in batches.
// Returns the total number of rows deleted.
func (r *Repository) DeleteEventsOlderThan(ctx context.Context, cutoffMs int64, batchSize int) (int64, error) {
	var totalDeleted int64

	for {
		result, err := r.db.ExecContext(ctx,
			`DELETE FROM notification_events
			 WHERE id IN (
				SELECT id FROM notification_events
				WHERE created_at < ?
				ORDER BY created_at
				LIMIT ?
			 )`,
			cutoffMs, batchSize,
		)
		if err != nil {
			return totalDeleted, fmt.Errorf("delete old events: %w", err)
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return totalDeleted, fmt.Errorf("rows affected: %w", err)
		}

		totalDeleted += affected
		if affected == 0 {
			break
		}
	}

	return totalDeleted, nil
}

// PlatformSummaryRow holds the result of a platform-wide aggregation query.
type PlatformSummaryRow struct {
	NotificationsTotal         int   `db:"notifications_total"`
	NotificationsServerTrusted int   `db:"notifications_server_trusted"`
	NotificationsE2E           int   `db:"notifications_e2e"`
	NotificationsDelivered     int   `db:"notifications_delivered"`
	NotificationsFailed        int   `db:"notifications_failed"`
	AttachmentsCount           int   `db:"attachments_count"`
	AttachmentsBytesTotal      int64 `db:"attachments_bytes_total"`
	SourcesCLI                 int   `db:"sources_cli"`
	SourcesWebhook             int   `db:"sources_webhook"`
	SourcesAPI                 int   `db:"sources_api"`
	SourcesInternal            int   `db:"sources_internal"`
	DevicesLost                int   `db:"devices_lost"`
	ActiveUsers                int   `db:"active_users"`
}

// GetPlatformSummary aggregates daily_usage_summary across all users within a time range.
func (r *Repository) GetPlatformSummary(ctx context.Context, fromMs, toMs int64) (*PlatformSummaryRow, error) {
	var row PlatformSummaryRow
	err := r.db.GetContext(ctx, &row,
		`SELECT
			COALESCE(SUM(notifications_total), 0) AS notifications_total,
			COALESCE(SUM(notifications_server_trusted), 0) AS notifications_server_trusted,
			COALESCE(SUM(notifications_e2e), 0) AS notifications_e2e,
			COALESCE(SUM(notifications_delivered), 0) AS notifications_delivered,
			COALESCE(SUM(notifications_failed), 0) AS notifications_failed,
			COALESCE(SUM(attachments_count), 0) AS attachments_count,
			COALESCE(SUM(attachments_bytes_total), 0) AS attachments_bytes_total,
			COALESCE(SUM(sources_cli), 0) AS sources_cli,
			COALESCE(SUM(sources_webhook), 0) AS sources_webhook,
			COALESCE(SUM(sources_api), 0) AS sources_api,
			COALESCE(SUM(sources_internal), 0) AS sources_internal,
			COALESCE(SUM(devices_lost), 0) AS devices_lost,
			COUNT(DISTINCT user_id) AS active_users
		 FROM daily_usage_summary
		 WHERE day_start_ms >= ? AND day_start_ms <= ?`,
		fromMs, toMs,
	)
	if err != nil {
		return nil, fmt.Errorf("get platform summary: %w", err)
	}
	return &row, nil
}

// GetPlatformDailyBreakdown returns daily totals aggregated across all users.
func (r *Repository) GetPlatformDailyBreakdown(ctx context.Context, fromMs, toMs int64) ([]DailyUsageSummary, error) {
	var summaries []DailyUsageSummary
	err := r.db.SelectContext(ctx, &summaries,
		`SELECT
			'' AS user_id,
			day_start_ms,
			COALESCE(SUM(notifications_total), 0) AS notifications_total,
			COALESCE(SUM(notifications_server_trusted), 0) AS notifications_server_trusted,
			COALESCE(SUM(notifications_e2e), 0) AS notifications_e2e,
			COALESCE(SUM(notifications_delivered), 0) AS notifications_delivered,
			COALESCE(SUM(notifications_failed), 0) AS notifications_failed,
			COALESCE(SUM(attachments_count), 0) AS attachments_count,
			COALESCE(SUM(attachments_bytes_total), 0) AS attachments_bytes_total,
			COALESCE(SUM(sources_cli), 0) AS sources_cli,
			COALESCE(SUM(sources_webhook), 0) AS sources_webhook,
			COALESCE(SUM(sources_api), 0) AS sources_api,
			COALESCE(SUM(sources_internal), 0) AS sources_internal,
			COALESCE(SUM(devices_lost), 0) AS devices_lost,
			MIN(created_at) AS created_at,
			MAX(updated_at) AS updated_at
		 FROM daily_usage_summary
		 WHERE day_start_ms >= ? AND day_start_ms <= ?
		 GROUP BY day_start_ms
		 ORDER BY day_start_ms`,
		fromMs, toMs,
	)
	if err != nil {
		return nil, fmt.Errorf("get platform daily breakdown: %w", err)
	}
	return summaries, nil
}

// NewEventID generates a new UUIDv7 string for event IDs.
func NewEventID() string {
	return uuid.Must(uuid.NewV7()).String()
}
