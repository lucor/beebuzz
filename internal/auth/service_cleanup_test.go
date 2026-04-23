package auth

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"lucor.dev/beebuzz/internal/secure"
	"lucor.dev/beebuzz/internal/testutil"
)

func newCleanupTestService(t *testing.T) (*Repository, *Service, *User, context.Context) {
	t.Helper()

	db := testutil.NewDB(t)
	repo := NewRepository(db)
	svc := NewService(repo, nil, "", nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	ctx := context.Background()
	user, _, err := repo.GetOrCreateUser(ctx, "cleanup@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	return repo, svc, user, ctx
}

func insertAuthChallengeRow(
	t *testing.T,
	repo *Repository,
	ctx context.Context,
	id string,
	userID string,
	state string,
	otpHash string,
	expiresAt int64,
	usedAt *int64,
) {
	t.Helper()

	const insertChallengeQuery = `
		INSERT INTO auth_challenges (id, user_id, state, otp_hash, expires_at, used_at, attempt_count, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	createdAt := time.Now().UTC().UnixMilli()
	if _, err := repo.db.ExecContext(
		ctx,
		insertChallengeQuery,
		id,
		userID,
		state,
		otpHash,
		expiresAt,
		usedAt,
		0,
		createdAt,
	); err != nil {
		t.Fatalf("insert auth challenge row: %v", err)
	}
}

func TestCleanupExpiredSessions(t *testing.T) {
	t.Run("deletes expired sessions", func(t *testing.T) {
		repo, svc, user, ctx := newCleanupTestService(t)

		expiredAt := time.Now().UTC().Add(-time.Hour).UnixMilli()
		if err := repo.CreateSession(ctx, secure.Hash("expired-session"), user.ID, expiredAt); err != nil {
			t.Fatalf("CreateSession: %v", err)
		}

		if err := svc.CleanupExpired(ctx); err != nil {
			t.Fatalf("CleanupExpired: %v", err)
		}

		session, err := repo.GetSession(ctx, secure.Hash("expired-session"))
		if err != nil {
			t.Fatalf("GetSession: %v", err)
		}
		if session != nil {
			t.Fatal("expected expired session to be deleted")
		}
	})

	t.Run("deletes idle sessions", func(t *testing.T) {
		repo, svc, user, ctx := newCleanupTestService(t)

		activeAt := time.Now().UTC().Add(time.Hour).UnixMilli()
		rawTokenHash := secure.Hash("idle-session")
		if err := repo.CreateSession(ctx, rawTokenHash, user.ID, activeAt); err != nil {
			t.Fatalf("CreateSession: %v", err)
		}

		idleBefore := time.Now().UTC().Add(-(sessionIdleTimeout + time.Minute)).UnixMilli()
		if _, err := repo.db.ExecContext(ctx, `UPDATE sessions SET last_seen_at = ? WHERE token_hash = ?`, idleBefore, rawTokenHash); err != nil {
			t.Fatalf("seed idle session: %v", err)
		}

		if err := svc.CleanupExpired(ctx); err != nil {
			t.Fatalf("CleanupExpired: %v", err)
		}

		session, err := repo.GetSession(ctx, rawTokenHash)
		if err != nil {
			t.Fatalf("GetSession: %v", err)
		}
		if session != nil {
			t.Fatal("expected idle session to be deleted")
		}
	})

	t.Run("keeps active sessions", func(t *testing.T) {
		repo, svc, user, ctx := newCleanupTestService(t)

		activeAt := time.Now().UTC().Add(time.Hour).UnixMilli()
		if err := repo.CreateSession(ctx, secure.Hash("active-session"), user.ID, activeAt); err != nil {
			t.Fatalf("CreateSession: %v", err)
		}

		if err := svc.CleanupExpired(ctx); err != nil {
			t.Fatalf("CleanupExpired: %v", err)
		}

		session, err := repo.GetSession(ctx, secure.Hash("active-session"))
		if err != nil {
			t.Fatalf("GetSession: %v", err)
		}
		if session == nil {
			t.Fatal("expected active session to remain")
		}
	})
}

func TestCleanupExpiredChallenges(t *testing.T) {
	t.Run("deletes expired challenges", func(t *testing.T) {
		repo, svc, user, ctx := newCleanupTestService(t)

		expiredAt := time.Now().UTC().Add(-time.Hour).UnixMilli()
		insertAuthChallengeRow(t, repo, ctx, "expired-challenge", user.ID, "expired-state", "hash-1", expiredAt, nil)

		if err := svc.CleanupExpired(ctx); err != nil {
			t.Fatalf("CleanupExpired: %v", err)
		}

		challenge, err := repo.GetAuthChallengeByState(ctx, "expired-state")
		if err != nil {
			t.Fatalf("GetAuthChallengeByState: %v", err)
		}
		if challenge != nil {
			t.Fatal("expected expired challenge to be deleted")
		}
	})

	t.Run("deletes used challenges", func(t *testing.T) {
		repo, svc, user, ctx := newCleanupTestService(t)

		now := time.Now().UTC()
		activeAt := now.Add(time.Hour).UnixMilli()
		usedAt := now.UnixMilli()
		insertAuthChallengeRow(t, repo, ctx, "used-challenge", user.ID, "used-state", "hash-2", activeAt, &usedAt)

		if err := svc.CleanupExpired(ctx); err != nil {
			t.Fatalf("CleanupExpired: %v", err)
		}

		challenge, err := repo.GetAuthChallengeByState(ctx, "used-state")
		if err != nil {
			t.Fatalf("GetAuthChallengeByState: %v", err)
		}
		if challenge != nil {
			t.Fatal("expected used challenge to be deleted")
		}
	})

	t.Run("keeps active challenges", func(t *testing.T) {
		repo, svc, user, ctx := newCleanupTestService(t)

		activeAt := time.Now().UTC().Add(time.Hour).UnixMilli()
		insertAuthChallengeRow(t, repo, ctx, "active-challenge", user.ID, "active-state", "hash-3", activeAt, nil)

		if err := svc.CleanupExpired(ctx); err != nil {
			t.Fatalf("CleanupExpired: %v", err)
		}

		challenge, err := repo.GetAuthChallengeByState(ctx, "active-state")
		if err != nil {
			t.Fatalf("GetAuthChallengeByState: %v", err)
		}
		if challenge == nil {
			t.Fatal("expected active challenge to remain")
		}
	})
}
