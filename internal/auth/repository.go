package auth

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"lucor.dev/beebuzz/internal/core"
)

// AuthChallenge represents an OTP challenge stored in the database.
type AuthChallenge struct {
	ID           string `db:"id"`
	UserID       string `db:"user_id"`
	State        string `db:"state"`
	OTPHash      string `db:"otp_hash"`
	AttemptCount int    `db:"attempt_count"`
	ExpiresAt    int64  `db:"expires_at"`
	UsedAt       *int64 `db:"used_at"`
	CreatedAt    int64  `db:"created_at"`
}

// DBSession represents a user session stored in the database.
type DBSession struct {
	TokenHash  string `db:"token_hash"`
	UserID     string `db:"user_id"`
	CreatedAt  int64  `db:"created_at"`
	ExpiresAt  int64  `db:"expires_at"`
	LastSeenAt int64  `db:"last_seen_at"`
}

// Repository provides data access for the auth domain.
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new auth repository.
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// CreateAuthChallenge invalidates any active challenges for the user and creates a new one.
// Delete and insert are executed within a single transaction to prevent race conditions.
func (r *Repository) CreateAuthChallenge(ctx context.Context, userID string, state string, otpHash string, expiresAt int64) (string, error) {
	challengeID, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("failed to create challenge ID: %w", err)
	}
	now := time.Now().UnixMilli()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx,
		`DELETE FROM auth_challenges WHERE user_id = ?`,
		userID,
	)
	if err != nil {
		return "", fmt.Errorf("failed to delete existing challenges: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO auth_challenges
         (id, user_id, state, otp_hash, expires_at, attempt_count, created_at)
         VALUES (?, ?, ?, ?, ?, ?, ?)`,
		challengeID, userID, state, otpHash, expiresAt, 0, now,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create auth challenge: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	return challengeID.String(), nil
}

// GetAuthChallengeByState retrieves an auth challenge by state.
func (r *Repository) GetAuthChallengeByState(ctx context.Context, state string) (*AuthChallenge, error) {
	var challenge AuthChallenge
	err := r.db.GetContext(ctx, &challenge,
		`SELECT id, user_id, state, otp_hash, expires_at, used_at, attempt_count, created_at
		 FROM auth_challenges
		 WHERE state = ?`,
		state,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get auth challenge by state: %w", err)
	}
	return &challenge, nil
}

// MarkAuthChallengeAsUsed marks a challenge as used.
func (r *Repository) MarkAuthChallengeAsUsed(ctx context.Context, challengeID string) error {
	now := time.Now().UnixMilli()
	_, err := r.db.ExecContext(ctx,
		"UPDATE auth_challenges SET used_at = ? WHERE id = ?",
		now, challengeID,
	)
	if err != nil {
		return fmt.Errorf("failed to mark auth challenge as used: %w", err)
	}
	return nil
}

// IncrementAuthChallengeAttempts increments the attempt counter for a challenge.
func (r *Repository) IncrementAuthChallengeAttempts(ctx context.Context, challengeID string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE auth_challenges SET attempt_count = attempt_count + 1 WHERE id = ?",
		challengeID,
	)
	if err != nil {
		return fmt.Errorf("failed to increment auth challenge attempts: %w", err)
	}
	return nil
}

// CreateSession creates a new session row for a hashed token.
func (r *Repository) CreateSession(ctx context.Context, tokenHash string, userID string, expiresAt int64) error {
	now := time.Now().UnixMilli()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO sessions (token_hash, user_id, created_at, expires_at, last_seen_at)
		 VALUES (?, ?, ?, ?, ?)`,
		tokenHash, userID, now, expiresAt, now,
	)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}

// GetSession retrieves a session by hashed token.
func (r *Repository) GetSession(ctx context.Context, tokenHash string) (*DBSession, error) {
	var session DBSession
	err := r.db.GetContext(ctx, &session,
		`SELECT token_hash, user_id, created_at, expires_at, last_seen_at
		 FROM sessions
		 WHERE token_hash = ?`,
		tokenHash,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	return &session, nil
}

// DeleteSession revokes a session only when the hashed token belongs to the given user.
func (r *Repository) DeleteSession(ctx context.Context, userID string, tokenHash string) (int64, error) {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM sessions WHERE user_id = ? AND token_hash = ?`,
		userID, tokenHash,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to delete session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to read deleted session count: %w", err)
	}

	return rowsAffected, nil
}

// DeleteStaleSessions removes expired or idle sessions and returns the number of deleted rows.
func (r *Repository) DeleteStaleSessions(ctx context.Context, expiredBefore int64, idleBefore int64) (int64, error) {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM sessions WHERE expires_at < ? OR last_seen_at < ?`,
		expiredBefore, idleBefore,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to delete stale sessions: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to read deleted sessions count: %w", err)
	}

	return rowsAffected, nil
}

// TouchSession updates the session last-seen timestamp to keep active sessions alive.
func (r *Repository) TouchSession(ctx context.Context, tokenHash string) error {
	now := time.Now().UTC().UnixMilli()
	_, err := r.db.ExecContext(ctx,
		`UPDATE sessions SET last_seen_at = ? WHERE token_hash = ?`,
		now, tokenHash,
	)
	if err != nil {
		return fmt.Errorf("failed to update session last seen: %w", err)
	}
	return nil
}

// DeleteSessionsByUserID removes all sessions for a user.
func (r *Repository) DeleteSessionsByUserID(ctx context.Context, userID string) (int64, error) {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM sessions WHERE user_id = ?`,
		userID,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to delete sessions by user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to read deleted sessions by user count: %w", err)
	}

	return rowsAffected, nil
}

// DeleteStaleAuthChallenges removes challenges that are expired or already used.
func (r *Repository) DeleteStaleAuthChallenges(ctx context.Context) (int64, error) {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM auth_challenges WHERE expires_at < ? OR used_at IS NOT NULL`,
		time.Now().UTC().UnixMilli(),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to delete stale auth challenges: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to read deleted challenges count: %w", err)
	}

	return rowsAffected, nil
}

// GetOrCreateUser retrieves a user by email or creates a new one if it doesn't exist.
func (r *Repository) GetOrCreateUser(ctx context.Context, email string, opts ...CreateUserOptions) (*User, bool, error) {
	opt := CreateUserOptions{AccountStatus: core.AccountStatusActive}
	if len(opts) > 0 {
		opt = opts[0]
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()

	id, err := uuid.NewV7()
	if err != nil {
		return nil, false, fmt.Errorf("create UUID: %w", err)
	}

	now := time.Now().UnixMilli()

	var trialStartedAt *int64
	if opt.StartTrialOnCreate {
		trialStartedAt = &now
	}

	result, err := tx.ExecContext(ctx, `
        INSERT OR IGNORE INTO users (id, email, account_status, signup_reason, trial_started_at, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?)
    `, id.String(), email, opt.AccountStatus, opt.SignupReason, trialStartedAt, now, now)
	if err != nil {
		return nil, false, err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, false, err
	}
	created := rows > 0

	var user User
	err = tx.GetContext(ctx, &user, `
    SELECT id, email, is_admin, account_status, trial_started_at, created_at, updated_at
    FROM users WHERE email = ?
		`, email)
	if err != nil {
		return nil, created, err
	}

	return &user, created, tx.Commit()
}

// EnsureUserAdmin promotes a user to admin if they are not already one.
// The method is idempotent so bootstrap logic can safely call it after OTP verification.
func (r *Repository) EnsureUserAdmin(ctx context.Context, userID string) (bool, error) {
	result, err := r.db.ExecContext(ctx,
		`UPDATE users SET is_admin = 1, updated_at = ? WHERE id = ? AND is_admin = 0`,
		time.Now().UnixMilli(),
		userID,
	)
	if err != nil {
		return false, fmt.Errorf("failed to ensure user admin: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to read ensure user admin row count: %w", err)
	}

	return rowsAffected > 0, nil
}

// GetUserByID retrieves a user by ID.
func (r *Repository) GetUserByID(ctx context.Context, userID string) (*User, error) {
	var user User
	err := r.db.GetContext(ctx, &user, "SELECT id, email, is_admin, account_status, trial_started_at, created_at, updated_at FROM users WHERE id = ?", userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// SetTrialStartedAt sets the trial_started_at timestamp for a user if not already set.
// Returns nil if RowsAffected == 0 (already set - idempotent).
func (r *Repository) SetTrialStartedAt(ctx context.Context, userID string, trialStart int64) error {
	result, err := r.db.ExecContext(ctx,
		"UPDATE users SET trial_started_at = ?, updated_at = ? WHERE id = ? AND trial_started_at IS NULL",
		trialStart, time.Now().UnixMilli(), userID,
	)
	if err != nil {
		return fmt.Errorf("failed to set trial started at: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return nil
	}
	return nil
}
