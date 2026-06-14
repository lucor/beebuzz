package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

const (
	defaultNotificationSyncLimit = 50
	maxNotificationSyncLimit     = 100
)

var ErrInvalidCursor = fmt.Errorf("invalid cursor")

// OutboxRepository stores short-lived notification payloads for Hive recovery.
type OutboxRepository struct {
	db *sqlx.DB
}

// NewOutboxRepository creates a notification outbox repository.
func NewOutboxRepository(db *sqlx.DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

// OutboxRecord is a short-lived notification payload stored for recovery.
// The id is a UUIDv7, which is time-sortable and carries the millisecond
// timestamp used for ordering, cursoring, and expiry.
type OutboxRecord struct {
	ID           string `db:"id"`
	UserID       string `db:"user_id"`
	TopicID      string `db:"topic_id"`
	Topic        string `db:"topic"`
	DeliveryMode string `db:"delivery_mode"`
	PayloadJSON  string `db:"payload_json"`
	ExpiresAt    int64  `db:"expires_at"`
}

// Store inserts one outbox record and its recipient snapshot.
func (r *OutboxRepository) Store(ctx context.Context, record OutboxRecord, deviceIDs []string) error {
	if len(deviceIDs) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("notification outbox: begin store transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO notification_outbox
			(id, user_id, topic_id, topic, delivery_mode, payload_json, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		record.ID,
		record.UserID,
		record.TopicID,
		record.Topic,
		record.DeliveryMode,
		record.PayloadJSON,
		record.ExpiresAt,
	); err != nil {
		return fmt.Errorf("notification outbox: insert record: %w", err)
	}

	now := time.Now().UTC().UnixMilli()
	for _, deviceID := range deviceIDs {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO notification_outbox_recipients (notification_id, device_id, created_at)
			 VALUES (?, ?, ?)`,
			record.ID,
			deviceID,
			now,
		); err != nil {
			return fmt.Errorf("notification outbox: insert recipient: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("notification outbox: commit store transaction: %w", err)
	}
	return nil
}

// ListForDevice returns unexpired notifications for one device after the optional
// cursor. UUIDv7 ids are time-sortable, so both ordering and the cursor use id
// directly. Returns ErrInvalidCursor if afterID is not a UUIDv7.
func (r *OutboxRepository) ListForDevice(ctx context.Context, deviceID, afterID string, nowMs int64, limit int) ([]OutboxRecord, error) {
	if limit <= 0 {
		limit = defaultNotificationSyncLimit
	}
	if limit > maxNotificationSyncLimit {
		limit = maxNotificationSyncLimit
	}

	args := []any{deviceID, nowMs}
	cursorClause := ""
	if afterID != "" {
		if _, err := uuidV7UnixMilli(afterID); err != nil {
			return nil, ErrInvalidCursor
		}
		cursorClause = ` AND o.id > ?`
		args = append(args, afterID)
	}
	args = append(args, limit)

	var records []OutboxRecord
	query := `SELECT o.id, o.user_id, o.topic_id, o.topic, o.delivery_mode, o.payload_json, o.expires_at
		FROM notification_outbox o
		JOIN notification_outbox_recipients r ON r.notification_id = o.id
		WHERE r.device_id = ? AND o.expires_at > ?` + cursorClause + `
		ORDER BY o.id ASC
		LIMIT ?`
	if err := r.db.SelectContext(ctx, &records, query, args...); err != nil {
		return nil, fmt.Errorf("notification outbox: list for device: %w", err)
	}
	if records == nil {
		records = []OutboxRecord{}
	}
	return records, nil
}

// DeleteExpired removes expired outbox rows. Recipients are cascade-deleted.
func (r *OutboxRepository) DeleteExpired(ctx context.Context) error {
	if _, err := r.db.ExecContext(ctx,
		`DELETE FROM notification_outbox WHERE expires_at < ?`,
		time.Now().UTC().UnixMilli(),
	); err != nil {
		return fmt.Errorf("notification outbox: delete expired: %w", err)
	}
	return nil
}

// uuidV7UnixMilli extracts the embedded millisecond Unix timestamp from a
// UUIDv7 string, validating the version nibble.
func uuidV7UnixMilli(id string) (int64, error) {
	u, err := uuid.Parse(id)
	if err != nil || u.Version() != 7 {
		return 0, ErrInvalidCursor
	}
	sec, nsec := u.Time().UnixTime()
	return sec*1000 + nsec/1_000_000, nil
}
