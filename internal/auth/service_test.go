package auth

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"

	"lucor.dev/beebuzz/internal/config"
	"lucor.dev/beebuzz/internal/core"
	"lucor.dev/beebuzz/internal/mailer"
	"lucor.dev/beebuzz/internal/secure"
	"lucor.dev/beebuzz/internal/testutil"
	"lucor.dev/beebuzz/internal/topic"
)

// TestVerifyOTPExpiredChallenge verifies expired challenges return the expired error.
func TestVerifyOTPExpiredChallenge(t *testing.T) {
	db := testutil.NewDB(t)
	repo := NewRepository(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(repo, nil, "", nil, logger)
	ctx := context.Background()

	userID := insertTestUser(t, ctx, db, "expired@example.com")
	now := time.Now().UnixMilli()
	_, err := db.ExecContext(ctx, `
		INSERT INTO auth_challenges (id, user_id, state, otp_hash, expires_at, used_at, attempt_count, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, "challenge-expired", userID, "state-expired", secure.Hash("123456"), now-1, nil, 0, now)
	if err != nil {
		t.Fatalf("failed to insert expired challenge: %v", err)
	}

	_, err = svc.VerifyOTP(ctx, "123456", "state-expired")
	if !errors.Is(err, ErrOTPExpired) {
		t.Fatalf("VerifyOTP() error = %v, want ErrOTPExpired", err)
	}
}

// TestVerifyOTPUsedChallenge verifies used challenges return the used error.
func TestVerifyOTPUsedChallenge(t *testing.T) {
	db := testutil.NewDB(t)
	repo := NewRepository(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(repo, nil, "", nil, logger)
	ctx := context.Background()

	userID := insertTestUser(t, ctx, db, "used@example.com")
	now := time.Now().UnixMilli()
	usedAt := now - 1000
	_, err := db.ExecContext(ctx, `
		INSERT INTO auth_challenges (id, user_id, state, otp_hash, expires_at, used_at, attempt_count, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, "challenge-used", userID, "state-used", secure.Hash("123456"), now+60000, usedAt, 0, now)
	if err != nil {
		t.Fatalf("failed to insert used challenge: %v", err)
	}

	_, err = svc.VerifyOTP(ctx, "123456", "state-used")
	if !errors.Is(err, ErrOTPUsed) {
		t.Fatalf("VerifyOTP() error = %v, want ErrOTPUsed", err)
	}
}

// TestValidateSessionMissingUser verifies stale sessions are treated as unauthorized.
func TestValidateSessionMissingUser(t *testing.T) {
	db := testutil.NewDB(t)
	repo := NewRepository(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(repo, nil, "", nil, logger)
	ctx := context.Background()

	userID := insertTestUser(t, ctx, db, "stale-session@example.com")
	if err := repo.CreateSession(ctx, secure.Hash("stale-session-token"), userID, time.Now().Add(time.Hour).UnixMilli()); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if _, err := db.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		t.Fatalf("failed to disable foreign keys: %v", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, userID); err != nil {
		t.Fatalf("failed to delete user: %v", err)
	}
	if _, err := db.ExecContext(ctx, `PRAGMA foreign_keys = ON`); err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
	}

	_, err := svc.ValidateSession(ctx, "stale-session-token")
	if !errors.Is(err, core.ErrUnauthorized) {
		t.Fatalf("ValidateSession() error = %v, want ErrUnauthorized", err)
	}
}

// TestValidateSessionIdleTimeout verifies stale sessions are rejected even if they are not expired yet.
func TestValidateSessionIdleTimeout(t *testing.T) {
	db := testutil.NewDB(t)
	repo := NewRepository(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(repo, nil, "", nil, logger)
	ctx := context.Background()

	userID := insertTestUser(t, ctx, db, "idle-session@example.com")
	rawToken := "idle-session-token"
	if err := repo.CreateSession(ctx, secure.Hash(rawToken), userID, time.Now().Add(time.Hour).UnixMilli()); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	idleBefore := time.Now().UTC().Add(-(sessionIdleTimeout + time.Minute)).UnixMilli()
	if _, err := db.ExecContext(ctx, `UPDATE sessions SET last_seen_at = ? WHERE token_hash = ?`, idleBefore, secure.Hash(rawToken)); err != nil {
		t.Fatalf("failed to seed idle session: %v", err)
	}

	_, err := svc.ValidateSession(ctx, rawToken)
	if !errors.Is(err, core.ErrUnauthorized) {
		t.Fatalf("ValidateSession() error = %v, want ErrUnauthorized", err)
	}
}

// TestValidateSessionRefreshesLastSeenAt verifies active sessions get their last seen timestamp refreshed.
func TestValidateSessionRefreshesLastSeenAt(t *testing.T) {
	db := testutil.NewDB(t)
	repo := NewRepository(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(repo, nil, "", nil, logger)
	ctx := context.Background()

	userID := insertTestUser(t, ctx, db, "refresh-session@example.com")
	rawToken := "refresh-session-token"
	if err := repo.CreateSession(ctx, secure.Hash(rawToken), userID, time.Now().Add(time.Hour).UnixMilli()); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	oldLastSeen := time.Now().UTC().Add(-(sessionRefreshInterval + time.Minute)).UnixMilli()
	if _, err := db.ExecContext(ctx, `UPDATE sessions SET last_seen_at = ? WHERE token_hash = ?`, oldLastSeen, secure.Hash(rawToken)); err != nil {
		t.Fatalf("failed to seed refresh session: %v", err)
	}

	user, err := svc.ValidateSession(ctx, rawToken)
	if err != nil {
		t.Fatalf("ValidateSession() error = %v", err)
	}
	if user == nil {
		t.Fatal("ValidateSession() user = nil, want non-nil")
	}

	session, err := repo.GetSession(ctx, secure.Hash(rawToken))
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if session == nil {
		t.Fatal("GetSession() session = nil, want non-nil")
	}
	if session.LastSeenAt <= oldLastSeen {
		t.Fatalf("session last_seen_at = %d, want refreshed value above %d", session.LastSeenAt, oldLastSeen)
	}
}

// TestRevokeAllSessions removes every session belonging to the user.
func TestRevokeAllSessions(t *testing.T) {
	db := testutil.NewDB(t)
	repo := NewRepository(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(repo, nil, "", nil, logger)
	ctx := context.Background()

	userID := insertTestUser(t, ctx, db, "revoke-all@example.com")
	firstTokenHash := secure.Hash("first-session-token")
	secondTokenHash := secure.Hash("second-session-token")
	if err := repo.CreateSession(ctx, firstTokenHash, userID, time.Now().Add(time.Hour).UnixMilli()); err != nil {
		t.Fatalf("failed to create first session: %v", err)
	}
	if err := repo.CreateSession(ctx, secondTokenHash, userID, time.Now().Add(time.Hour).UnixMilli()); err != nil {
		t.Fatalf("failed to create second session: %v", err)
	}

	if err := svc.RevokeAllSessions(ctx, userID); err != nil {
		t.Fatalf("RevokeAllSessions() error = %v", err)
	}

	firstSession, err := repo.GetSession(ctx, firstTokenHash)
	if err != nil {
		t.Fatalf("GetSession(first) error = %v", err)
	}
	if firstSession != nil {
		t.Fatal("first session should be revoked")
	}

	secondSession, err := repo.GetSession(ctx, secondTokenHash)
	if err != nil {
		t.Fatalf("GetSession(second) error = %v", err)
	}
	if secondSession != nil {
		t.Fatal("second session should be revoked")
	}
}

func TestCreateSessionStoresTokenHashOnly(t *testing.T) {
	db := testutil.NewDB(t)
	repo := NewRepository(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(repo, nil, "", nil, logger)
	ctx := context.Background()

	userID := insertTestUser(t, ctx, db, "session-hash@example.com")
	session, err := svc.CreateSession(ctx, userID)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	var storedTokenHash string
	if err := db.GetContext(ctx, &storedTokenHash, `SELECT token_hash FROM sessions LIMIT 1`); err != nil {
		t.Fatalf("select token_hash: %v", err)
	}
	if storedTokenHash == session.Token {
		t.Fatal("stored token hash must not equal raw session token")
	}
	if storedTokenHash != secure.Hash(session.Token) {
		t.Fatalf("stored token hash = %q, want %q", storedTokenHash, secure.Hash(session.Token))
	}
}

func TestRevokeSessionRequiresMatchingUser(t *testing.T) {
	db := testutil.NewDB(t)
	repo := NewRepository(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(repo, nil, "", nil, logger)
	ctx := context.Background()

	ownerID := insertTestUser(t, ctx, db, "owner@example.com")
	otherID := insertTestUser(t, ctx, db, "other@example.com")
	rawToken := "owned-session-token"
	tokenHash := secure.Hash(rawToken)
	if err := repo.CreateSession(ctx, tokenHash, ownerID, time.Now().Add(time.Hour).UnixMilli()); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	if err := svc.RevokeSession(ctx, otherID, rawToken); err != nil {
		t.Fatalf("RevokeSession(other user) error = %v", err)
	}

	session, err := repo.GetSession(ctx, tokenHash)
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if session == nil {
		t.Fatal("session should remain when revoked by a different user")
	}

	if err := svc.RevokeSession(ctx, ownerID, rawToken); err != nil {
		t.Fatalf("RevokeSession(owner) error = %v", err)
	}

	session, err = repo.GetSession(ctx, tokenHash)
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if session != nil {
		t.Fatal("session should be deleted when revoked by the owning user")
	}
}

func TestRequestAuthNormalizesEmailBeforeLookup(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	svc := newRequestAuthService(t, db, false)

	approved, err := svc.RequestAuth(ctx, "  USER@Example.com  ", "state-1", nil)
	if err != nil {
		t.Fatalf("RequestAuth first call: %v", err)
	}
	if !approved {
		t.Fatal("RequestAuth first call approved = false, want true")
	}

	approved, err = svc.RequestAuth(ctx, "user@example.com", "state-2", nil)
	if err != nil {
		t.Fatalf("RequestAuth second call: %v", err)
	}
	if !approved {
		t.Fatal("RequestAuth second call approved = false, want true")
	}

	var userCount int
	if err := db.GetContext(ctx, &userCount, `SELECT COUNT(*) FROM users`); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if userCount != 1 {
		t.Fatalf("user count = %d, want 1", userCount)
	}

	var storedEmail string
	if err := db.GetContext(ctx, &storedEmail, `SELECT email FROM users LIMIT 1`); err != nil {
		t.Fatalf("select email: %v", err)
	}
	if storedEmail != "user@example.com" {
		t.Fatalf("stored email = %q, want %q", storedEmail, "user@example.com")
	}
}

func TestRequestAuthPrivateBetaWaitlistsNewUser(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	svc := newRequestAuthService(t, db, true)

	approved, err := svc.RequestAuth(ctx, "beta-user@example.com", "state-beta", nil)
	if err != nil {
		t.Fatalf("RequestAuth() error = %v", err)
	}
	if approved {
		t.Fatal("RequestAuth() approved = true, want false")
	}

	var accountStatus string
	if err := db.GetContext(ctx, &accountStatus, `SELECT account_status FROM users WHERE email = ?`, "beta-user@example.com"); err != nil {
		t.Fatalf("select account_status: %v", err)
	}
	if accountStatus != "pending" {
		t.Fatalf("account_status = %q, want %q", accountStatus, "pending")
	}

	var challengeCount int
	if err := db.GetContext(ctx, &challengeCount, `SELECT COUNT(*) FROM auth_challenges`); err != nil {
		t.Fatalf("count auth_challenges: %v", err)
	}
	if challengeCount != 0 {
		t.Fatalf("auth challenge count = %d, want 0", challengeCount)
	}
}

func TestRequestAuthPrivateBetaBypassesWaitlistForBootstrapAdminEmail(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	svc := newRequestAuthService(t, db, true)
	svc.SetBootstrapAdminEmail("BOOTSTRAP@example.com")

	approved, err := svc.RequestAuth(ctx, "bootstrap@example.com", "state-bootstrap", nil)
	if err != nil {
		t.Fatalf("RequestAuth() error = %v", err)
	}
	if !approved {
		t.Fatal("RequestAuth() approved = false, want true")
	}

	var accountStatus string
	if err := db.GetContext(ctx, &accountStatus, `SELECT account_status FROM users WHERE email = ?`, "bootstrap@example.com"); err != nil {
		t.Fatalf("select account_status: %v", err)
	}
	if accountStatus != "active" {
		t.Fatalf("account_status = %q, want %q", accountStatus, "active")
	}

	var challengeCount int
	if err := db.GetContext(ctx, &challengeCount, `SELECT COUNT(*) FROM auth_challenges`); err != nil {
		t.Fatalf("count auth_challenges: %v", err)
	}
	if challengeCount != 1 {
		t.Fatalf("auth challenge count = %d, want 1", challengeCount)
	}
}

func TestRequestAuthPublicModeSkipsWaitlist(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	svc := newRequestAuthService(t, db, false)

	approved, err := svc.RequestAuth(ctx, "public-user@example.com", "state-public", nil)
	if err != nil {
		t.Fatalf("RequestAuth() error = %v", err)
	}
	if !approved {
		t.Fatal("RequestAuth() approved = false, want true")
	}

	var accountStatus string
	if err := db.GetContext(ctx, &accountStatus, `SELECT account_status FROM users WHERE email = ?`, "public-user@example.com"); err != nil {
		t.Fatalf("select account_status: %v", err)
	}
	if accountStatus != "active" {
		t.Fatalf("account_status = %q, want %q", accountStatus, "active")
	}

	var challengeCount int
	if err := db.GetContext(ctx, &challengeCount, `SELECT COUNT(*) FROM auth_challenges`); err != nil {
		t.Fatalf("count auth_challenges: %v", err)
	}
	if challengeCount != 1 {
		t.Fatalf("auth challenge count = %d, want 1", challengeCount)
	}
}

func TestRequestAuthSilentlyThrottlesEmail(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	svc := newRequestAuthService(t, db, false)
	now := time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)
	current := now

	throttle := NewEmailThrottle(3, 15*time.Minute, time.Minute)
	throttle.now = func() time.Time {
		return current
	}
	svc.SetEmailThrottle(throttle)

	approved, err := svc.RequestAuth(ctx, "throttle@example.com", "state-1", nil)
	if err != nil {
		t.Fatalf("RequestAuth first call: %v", err)
	}
	if !approved {
		t.Fatal("RequestAuth first call approved = false, want true")
	}

	current = now.Add(30 * time.Second)
	approved, err = svc.RequestAuth(ctx, "throttle@example.com", "state-2", nil)
	if err != nil {
		t.Fatalf("RequestAuth second call: %v", err)
	}
	if !approved {
		t.Fatal("RequestAuth second call approved = false, want true")
	}

	var challengeCount int
	if err := db.GetContext(ctx, &challengeCount, `SELECT COUNT(*) FROM auth_challenges`); err != nil {
		t.Fatalf("count auth_challenges: %v", err)
	}
	if challengeCount != 1 {
		t.Fatalf("auth challenge count = %d, want 1", challengeCount)
	}
}

func TestRequestAuthReturnsGlobalRateLimit(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	svc := newRequestAuthService(t, db, false)
	now := time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)
	current := now

	globalThrottle := NewGlobalAuthThrottle(1, time.Minute)
	globalThrottle.now = func() time.Time {
		return current
	}
	svc.SetGlobalThrottle(globalThrottle)

	approved, err := svc.RequestAuth(ctx, "global@example.com", "state-1", nil)
	if err != nil {
		t.Fatalf("RequestAuth first call: %v", err)
	}
	if !approved {
		t.Fatal("RequestAuth first call approved = false, want true")
	}

	current = now.Add(30 * time.Second)
	approved, err = svc.RequestAuth(ctx, "other@example.com", "state-2", nil)
	if !errors.Is(err, ErrGlobalRateLimit) {
		t.Fatalf("RequestAuth second call error = %v, want ErrGlobalRateLimit", err)
	}
	if approved {
		t.Fatal("RequestAuth second call approved = true, want false")
	}

	var challengeCount int
	if err := db.GetContext(ctx, &challengeCount, `SELECT COUNT(*) FROM auth_challenges`); err != nil {
		t.Fatalf("count auth_challenges: %v", err)
	}
	if challengeCount != 1 {
		t.Fatalf("auth challenge count = %d, want 1", challengeCount)
	}
}

func TestCreateSessionBootstrapAdmin(t *testing.T) {
	t.Run("promotes configured bootstrap email", func(t *testing.T) {
		db := testutil.NewDB(t)
		ctx := context.Background()
		svc := newRequestAuthService(t, db, false)
		svc.SetBootstrapAdminEmail("bootstrap@example.com")

		userID := insertTestUser(t, ctx, db, "bootstrap@example.com")
		session, err := svc.CreateSession(ctx, userID)
		if err != nil {
			t.Fatalf("CreateSession() error = %v", err)
		}
		if session == nil {
			t.Fatal("CreateSession() session = nil, want non-nil")
		}

		if !userIsAdmin(t, ctx, db, userID) {
			t.Fatal("bootstrap user should be promoted to admin after OTP verification")
		}
	})

	t.Run("does not promote when bootstrap email is not configured", func(t *testing.T) {
		db := testutil.NewDB(t)
		ctx := context.Background()
		svc := newRequestAuthService(t, db, false)

		userID := insertTestUser(t, ctx, db, "bootstrap@example.com")
		session, err := svc.CreateSession(ctx, userID)
		if err != nil {
			t.Fatalf("CreateSession() error = %v", err)
		}
		if session == nil {
			t.Fatal("CreateSession() session = nil, want non-nil")
		}

		if userIsAdmin(t, ctx, db, userID) {
			t.Fatal("user should not be promoted without bootstrap config")
		}
	})

	t.Run("does not promote a different email", func(t *testing.T) {
		db := testutil.NewDB(t)
		ctx := context.Background()
		svc := newRequestAuthService(t, db, false)
		svc.SetBootstrapAdminEmail("bootstrap@example.com")

		userID := insertTestUser(t, ctx, db, "other@example.com")
		session, err := svc.CreateSession(ctx, userID)
		if err != nil {
			t.Fatalf("CreateSession() error = %v", err)
		}
		if session == nil {
			t.Fatal("CreateSession() session = nil, want non-nil")
		}

		if userIsAdmin(t, ctx, db, userID) {
			t.Fatal("non-bootstrap email should not be promoted to admin")
		}
	})

	t.Run("leaves existing admin unchanged", func(t *testing.T) {
		db := testutil.NewDB(t)
		ctx := context.Background()
		svc := newRequestAuthService(t, db, false)
		svc.SetBootstrapAdminEmail("bootstrap@example.com")

		userID := insertTestUser(t, ctx, db, "bootstrap@example.com")
		if _, err := db.ExecContext(ctx, `UPDATE users SET is_admin = 1 WHERE id = ?`, userID); err != nil {
			t.Fatalf("seed admin user: %v", err)
		}

		session, err := svc.CreateSession(ctx, userID)
		if err != nil {
			t.Fatalf("CreateSession() error = %v", err)
		}
		if session == nil {
			t.Fatal("CreateSession() session = nil, want non-nil")
		}

		if !userIsAdmin(t, ctx, db, userID) {
			t.Fatal("existing admin should remain admin")
		}
	})
}

func userIsAdmin(t *testing.T, ctx context.Context, db *sqlx.DB, userID string) bool {
	t.Helper()

	var isAdmin bool
	if err := db.GetContext(ctx, &isAdmin, `SELECT is_admin FROM users WHERE id = ?`, userID); err != nil {
		t.Fatalf("select is_admin: %v", err)
	}

	return isAdmin
}

func newRequestAuthService(t *testing.T, db *sqlx.DB, privateBeta bool) *Service {
	t.Helper()

	authRepo := NewRepository(db)
	topicRepo := topic.NewRepository(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	topicSvc := topic.NewService(topicRepo, logger)
	testMailer, err := mailer.New(&config.Mailer{
		Sender:  "noreply@example.com",
		ReplyTo: "support@example.com",
	})
	if err != nil {
		t.Fatalf("mailer.New: %v", err)
	}

	svc := NewService(authRepo, testMailer, "", topicSvc, logger)
	svc.UsePrivateBeta(privateBeta)
	return svc
}
