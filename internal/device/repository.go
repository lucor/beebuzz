package device

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Repository handles all device-related DB queries.
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new device Repository.
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// CreateDevice inserts a new device record.
func (r *Repository) CreateDevice(ctx context.Context, id, userID, name, description string) error {
	now := time.Now().UnixMilli()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO devices (id, user_id, name, description, is_active, pairing_status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 1, ?, ?, ?)`,
		id, userID, name, description, PairingStatusPending, now, now,
	)
	return err
}

// CreateDeviceWithTopicsAndPairingCode atomically creates a device, its topic associations, and its pairing code.
func (r *Repository) CreateDeviceWithTopicsAndPairingCode(ctx context.Context, id, userID, name, description string, topicIDs []string, codeHash string, expiresAt int64) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin device create transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	now := time.Now().UnixMilli()
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO devices (id, user_id, name, description, is_active, pairing_status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 1, ?, ?, ?)`,
		id, userID, name, description, PairingStatusPending, now, now,
	); err != nil {
		return fmt.Errorf("failed to create device: %w", err)
	}

	for _, topicID := range topicIDs {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO device_topics (device_id, topic_id, created_at) VALUES (?, ?, ?)`,
			id, topicID, now,
		); err != nil {
			return fmt.Errorf("failed to associate device with topic: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO device_pairing_codes (code_hash, device_id, expires_at, attempt_count, created_at)
		 VALUES (?, ?, ?, 0, ?)`,
		codeHash, id, expiresAt, now,
	); err != nil {
		return fmt.Errorf("failed to create pairing code: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit device create transaction: %w", err)
	}

	return nil
}

// GetDeviceByIDAndUser returns a device by ID scoped to a user, or nil if not found.
func (r *Repository) GetDeviceByIDAndUser(ctx context.Context, deviceID, userID string) (*Device, error) {
	var d Device
	err := r.db.GetContext(ctx, &d,
		`SELECT d.id, d.user_id, d.name, d.description, d.is_active, d.pairing_status, d.created_at, d.updated_at,
		        ps.created_at AS sub_created_at, ps.age_recipient AS sub_age_recipient
		 FROM devices d
		 LEFT JOIN push_subscriptions ps ON ps.device_id = d.id
		 WHERE d.id = ? AND d.user_id = ? AND d.is_active = 1`,
		deviceID, userID,
	)
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

// ListDevicesByUser returns all active devices for a user, ordered by created_at DESC.
func (r *Repository) ListDevicesByUser(ctx context.Context, userID string) ([]Device, error) {
	var devices []Device
	err := r.db.SelectContext(ctx, &devices,
		`SELECT d.id, d.user_id, d.name, d.description, d.is_active, d.pairing_status, d.created_at, d.updated_at,
		        ps.created_at AS sub_created_at, ps.age_recipient AS sub_age_recipient
		 FROM devices d
		 LEFT JOIN push_subscriptions ps ON ps.device_id = d.id
		 WHERE d.user_id = ? AND d.is_active = 1 ORDER BY d.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	return devices, nil
}

// UpdateDevice updates the name, description, and updated_at of a device.
func (r *Repository) UpdateDevice(ctx context.Context, deviceID, name, description string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE devices SET name = ?, description = ?, updated_at = ? WHERE id = ?`,
		name, description, time.Now().UnixMilli(), deviceID,
	)
	return err
}

// UpdateDeviceWithTopics atomically updates device fields and replaces topic associations.
func (r *Repository) UpdateDeviceWithTopics(ctx context.Context, userID, deviceID, name, description string, topicIDs []string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin device update transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	result, err := tx.ExecContext(ctx,
		`UPDATE devices SET name = ?, description = ?, updated_at = ? WHERE id = ? AND user_id = ? AND is_active = 1`,
		name, description, time.Now().UnixMilli(), deviceID, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to update device: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read updated device row count: %w", err)
	}
	if rows == 0 {
		return ErrDeviceNotFound
	}

	var existingTopicIDs []string
	if err := tx.SelectContext(ctx, &existingTopicIDs,
		`SELECT topic_id FROM device_topics WHERE device_id = ?`,
		deviceID,
	); err != nil {
		return fmt.Errorf("failed to load existing device topics: %w", err)
	}

	existingMap := make(map[string]struct{}, len(existingTopicIDs))
	for _, topicID := range existingTopicIDs {
		existingMap[topicID] = struct{}{}
	}

	desiredMap := make(map[string]struct{}, len(topicIDs))
	for _, topicID := range topicIDs {
		desiredMap[topicID] = struct{}{}
	}

	now := time.Now().UnixMilli()
	for _, topicID := range topicIDs {
		if _, exists := existingMap[topicID]; exists {
			continue
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO device_topics (device_id, topic_id, created_at) VALUES (?, ?, ?)`,
			deviceID, topicID, now,
		); err != nil {
			return fmt.Errorf("failed to add device topic association: %w", err)
		}
	}

	for _, topicID := range existingTopicIDs {
		if _, exists := desiredMap[topicID]; exists {
			continue
		}
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM device_topics WHERE device_id = ? AND topic_id = ?`,
			deviceID, topicID,
		); err != nil {
			return fmt.Errorf("failed to delete device topic association: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit device update transaction: %w", err)
	}

	return nil
}

// DeleteDevice soft-deletes a device by setting is_active = 0.
func (r *Repository) DeleteDevice(ctx context.Context, deviceID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE devices SET is_active = 0, updated_at = ? WHERE id = ?`,
		time.Now().UnixMilli(), deviceID,
	)
	return err
}

// CreatePairingCode inserts a new pairing code record.
func (r *Repository) CreatePairingCode(ctx context.Context, codeHash, deviceID string, expiresAt int64) error {
	now := time.Now().UnixMilli()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO device_pairing_codes (code_hash, device_id, expires_at, attempt_count, created_at)
		 VALUES (?, ?, ?, 0, ?)`,
		codeHash, deviceID, expiresAt, now,
	)
	return err
}

// InvalidatePairingCodes marks all unused pairing codes for a device as used.
func (r *Repository) InvalidatePairingCodes(ctx context.Context, deviceID string) error {
	now := time.Now().UnixMilli()
	_, err := r.db.ExecContext(ctx,
		`UPDATE device_pairing_codes SET used_at = ? WHERE device_id = ? AND used_at IS NULL`,
		now, deviceID,
	)
	return err
}

// GetActivePairingCode returns the active (unused, not expired) pairing code matching the hash.
// Returns nil, nil if not found.
func (r *Repository) GetActivePairingCode(ctx context.Context, codeHash string) (*PairingCode, error) {
	var pc PairingCode
	now := time.Now().UnixMilli()
	err := r.db.GetContext(ctx, &pc,
		`SELECT code_hash, device_id, expires_at, used_at, attempt_count, created_at
		 FROM device_pairing_codes
		 WHERE code_hash = ? AND used_at IS NULL AND expires_at > ?`,
		codeHash, now,
	)
	if isNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &pc, nil
}

// IncrementAndGetAttempts atomically increments the attempt counter and returns the new count.
func (r *Repository) IncrementAndGetAttempts(ctx context.Context, codeHash string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`UPDATE device_pairing_codes SET attempt_count = attempt_count + 1
		 WHERE code_hash = ? RETURNING attempt_count`,
		codeHash,
	).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// ConsumePairingCode atomically marks a code as used, stores the push subscription, and sets paired_at.
// All operations run inside a single transaction to prevent burning the code without completing pairing.
func (r *Repository) ConsumePairingCode(ctx context.Context, codeHash string, sub PushSubscription, deviceTokenHash string) (string, error) {
	now := time.Now().UnixMilli()

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	// Mark code as used
	result, err := tx.ExecContext(ctx,
		`UPDATE device_pairing_codes SET used_at = ?
		 WHERE code_hash = ? AND used_at IS NULL AND expires_at > ?`,
		now, codeHash, now,
	)
	if err != nil {
		return "", err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return "", err
	}
	if rows == 0 {
		return "", nil
	}

	// Get the device_id
	var deviceID string
	err = tx.QueryRowxContext(ctx,
		`SELECT device_id FROM device_pairing_codes WHERE code_hash = ?`,
		codeHash,
	).Scan(&deviceID)
	if err != nil {
		return "", err
	}

	// Delete any existing subscription for this device (handles reinstall/repair case)
	_, err = tx.ExecContext(ctx,
		`DELETE FROM push_subscriptions WHERE device_id = ?`,
		deviceID,
	)
	if err != nil {
		return "", err
	}

	// Upsert push subscription
	_, err = tx.ExecContext(ctx,
		`INSERT INTO push_subscriptions (device_id, endpoint, p256dh, auth, age_recipient, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(device_id) DO UPDATE SET
		   endpoint = excluded.endpoint,
		   p256dh = excluded.p256dh,
		   auth = excluded.auth,
		   age_recipient = excluded.age_recipient,
		   updated_at = excluded.updated_at`,
		deviceID, sub.Endpoint, sub.P256dh, sub.Auth, sub.AgeRecipient, now, now,
	)
	if err != nil {
		return "", err
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE devices SET pairing_status = ?, device_token_hash = ?, updated_at = ? WHERE id = ?`,
		PairingStatusPaired, deviceTokenHash, now, deviceID,
	)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return deviceID, nil
}

// ClearPushSubscriptionWithStatus deletes the subscription and updates pairing_status atomically.
func (r *Repository) ClearPushSubscriptionWithStatus(ctx context.Context, deviceID string, status PairingStatus) error {
	now := time.Now().UnixMilli()

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM push_subscriptions WHERE device_id = ?`,
		deviceID,
	); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE devices SET pairing_status = ?, updated_at = ? WHERE id = ?`,
		status, now, deviceID,
	); err != nil {
		return err
	}

	return tx.Commit()
}

// GetDeviceByID returns an active device by ID, or nil if not found.
func (r *Repository) GetDeviceByID(ctx context.Context, deviceID string) (*Device, error) {
	var d Device
	err := r.db.GetContext(ctx, &d,
		`SELECT d.id, d.user_id, d.name, d.description, d.is_active, d.pairing_status, d.created_at, d.updated_at,
		        ps.created_at AS sub_created_at, ps.age_recipient AS sub_age_recipient
		 FROM devices d
		 LEFT JOIN push_subscriptions ps ON ps.device_id = d.id
		 WHERE d.id = ? AND d.is_active = 1`,
		deviceID,
	)
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

// GetDeviceTopicIDs returns all topic IDs associated with a device.
func (r *Repository) GetDeviceTopicIDs(ctx context.Context, deviceID string) ([]string, error) {
	var topicIDs []string
	err := r.db.SelectContext(ctx, &topicIDs,
		`SELECT topic_id FROM device_topics WHERE device_id = ?`,
		deviceID,
	)
	if err != nil {
		return nil, err
	}
	return topicIDs, nil
}

// AddTopicToDevice inserts a device-topic association.
func (r *Repository) AddTopicToDevice(ctx context.Context, deviceID, topicID string) error {
	now := time.Now().UnixMilli()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO device_topics (device_id, topic_id, created_at) VALUES (?, ?, ?)`,
		deviceID, topicID, now,
	)
	return err
}

// DeleteTopicFromDevice removes a device-topic association.
func (r *Repository) DeleteTopicFromDevice(ctx context.Context, deviceID, topicID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM device_topics WHERE device_id = ? AND topic_id = ?`,
		deviceID, topicID,
	)
	return err
}

// GetPushSubscriptionsByUserAndTopic returns push subscriptions for all active devices
// subscribed to a given topic name for a given user.
func (r *Repository) GetPushSubscriptionsByUserAndTopic(ctx context.Context, userID, topicName string) ([]PushSubscription, error) {
	var subs []PushSubscription
	err := r.db.SelectContext(ctx, &subs,
		`SELECT ps.device_id, ps.endpoint, ps.p256dh, ps.auth, ps.age_recipient, ps.created_at, ps.updated_at
		 FROM push_subscriptions ps
		 JOIN device_topics dt ON dt.device_id = ps.device_id
		 JOIN topics t ON t.id = dt.topic_id
		 JOIN devices d ON d.id = ps.device_id
		 WHERE d.user_id = ? AND t.name = ? AND d.is_active = 1`,
		userID, topicName,
	)
	if err != nil {
		return nil, err
	}
	return subs, nil
}

// GetDeviceKeysByUser returns paired devices with their age public keys for a user.
func (r *Repository) GetDeviceKeysByUser(ctx context.Context, userID string) ([]Device, error) {
	var devices []Device
	err := r.db.SelectContext(ctx, &devices,
		`SELECT d.id, d.user_id, d.name, d.description, d.is_active, d.created_at, d.updated_at,
		        ps.created_at AS sub_created_at, ps.age_recipient AS sub_age_recipient
		 FROM devices d
		 JOIN push_subscriptions ps ON ps.device_id = d.id
		 WHERE d.user_id = ? AND d.is_active = 1 AND ps.age_recipient != ''
		 ORDER BY d.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	if devices == nil {
		devices = []Device{}
	}
	return devices, nil
}

// DeletePushSubscription deletes a push subscription by device ID.
func (r *Repository) DeletePushSubscription(ctx context.Context, deviceID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM push_subscriptions WHERE device_id = ?`,
		deviceID,
	)
	return err
}

// GetDeviceByIDAndTokenHash returns an active device by ID and token hash, or nil if not found.
func (r *Repository) GetDeviceByIDAndTokenHash(ctx context.Context, deviceID, tokenHash string) (*Device, error) {
	var d Device
	err := r.db.GetContext(ctx, &d,
		`SELECT d.id, d.user_id, d.name, d.description, d.is_active, d.pairing_status, d.device_token_hash, d.created_at, d.updated_at,
		        ps.created_at AS sub_created_at, ps.age_recipient AS sub_age_recipient
		 FROM devices d
		 LEFT JOIN push_subscriptions ps ON ps.device_id = d.id
		 WHERE d.id = ? AND d.device_token_hash = ? AND d.is_active = 1`,
		deviceID, tokenHash,
	)
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

// isNotFound returns true if the error is a not-found DB error.
func isNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
