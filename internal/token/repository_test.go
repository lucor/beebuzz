package token

import (
	"context"
	"testing"
	"time"

	"lucor.dev/beebuzz/internal/auth"
	"lucor.dev/beebuzz/internal/secure"
	"lucor.dev/beebuzz/internal/testutil"
	"lucor.dev/beebuzz/internal/topic"
)

func TestValidateAPITokenForTopic_Valid(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	tokenRepo := NewRepository(db)

	user, _, err := authRepo.GetOrCreateUser(ctx, "valid@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	tp, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	rawToken, err := secure.NewAPIToken()
	if err != nil {
		t.Fatalf("secure.NewAPIToken: %v", err)
	}

	tokenID, err := tokenRepo.CreateAPIToken(ctx, user.ID, "test", secure.Hash(rawToken), "")
	if err != nil {
		t.Fatalf("CreateAPIToken: %v", err)
	}

	if err := tokenRepo.AddTopicToAPIToken(ctx, tokenID, tp.ID); err != nil {
		t.Fatalf("AddTopicToAPIToken: %v", err)
	}

	gotUserID, _, err := tokenRepo.ValidateAPITokenForTopic(ctx, secure.Hash(rawToken), "alerts")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotUserID != user.ID {
		t.Errorf("userID: got %q, want %q", gotUserID, user.ID)
	}
}

func TestValidateAPITokenForTopic_TopicNotLinked(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	tokenRepo := NewRepository(db)

	user, _, err := authRepo.GetOrCreateUser(ctx, "topicnotlinked@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	// Create topic but do NOT link it to the token.
	_, err = topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	rawToken, err := secure.NewAPIToken()
	if err != nil {
		t.Fatalf("secure.NewAPIToken: %v", err)
	}

	_, err = tokenRepo.CreateAPIToken(ctx, user.ID, "test", secure.Hash(rawToken), "")
	if err != nil {
		t.Fatalf("CreateAPIToken: %v", err)
	}

	_, _, err = tokenRepo.ValidateAPITokenForTopic(ctx, secure.Hash(rawToken), "alerts")
	if err == nil {
		t.Fatal("expected error for token not linked to topic, got nil")
	}
}

func TestValidateAPITokenForTopic_Revoked(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	tokenRepo := NewRepository(db)

	user, _, err := authRepo.GetOrCreateUser(ctx, "revoked@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	tp, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	rawToken, err := secure.NewAPIToken()
	if err != nil {
		t.Fatalf("secure.NewAPIToken: %v", err)
	}

	tokenID, err := tokenRepo.CreateAPIToken(ctx, user.ID, "test", secure.Hash(rawToken), "")
	if err != nil {
		t.Fatalf("CreateAPIToken: %v", err)
	}

	if err := tokenRepo.AddTopicToAPIToken(ctx, tokenID, tp.ID); err != nil {
		t.Fatalf("AddTopicToAPIToken: %v", err)
	}

	if err := tokenRepo.RevokeAPIToken(ctx, user.ID, tokenID); err != nil {
		t.Fatalf("RevokeAPIToken: %v", err)
	}

	_, _, err = tokenRepo.ValidateAPITokenForTopic(ctx, secure.Hash(rawToken), "alerts")
	if err == nil {
		t.Fatal("expected error for revoked token, got nil")
	}
}

func TestValidateAPITokenForTopic_Expired(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	tokenRepo := NewRepository(db)

	user, _, err := authRepo.GetOrCreateUser(ctx, "expired@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	tp, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	rawToken, err := secure.NewAPIToken()
	if err != nil {
		t.Fatalf("secure.NewAPIToken: %v", err)
	}

	tokenID, err := tokenRepo.CreateAPIToken(ctx, user.ID, "test", secure.Hash(rawToken), "")
	if err != nil {
		t.Fatalf("CreateAPIToken: %v", err)
	}

	if err := tokenRepo.AddTopicToAPIToken(ctx, tokenID, tp.ID); err != nil {
		t.Fatalf("AddTopicToAPIToken: %v", err)
	}

	// No public API to set expiry, so we update it directly at the repo level.
	pastMs := time.Now().Add(-time.Second).UnixMilli()
	if _, err := db.ExecContext(ctx, "UPDATE api_tokens SET expires_at = ? WHERE id = ?", pastMs, tokenID); err != nil {
		t.Fatalf("set expires_at: %v", err)
	}

	_, _, err = tokenRepo.ValidateAPITokenForTopic(ctx, secure.Hash(rawToken), "alerts")
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestValidateAPITokenForTopic_UnknownToken(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	tokenRepo := NewRepository(db)

	_, _, err := tokenRepo.ValidateAPITokenForTopic(ctx, "nonexistent-hash", "alerts")
	if err == nil {
		t.Fatal("expected error for unknown token, got nil")
	}
}

func TestValidateAPITokenForTopic_UpdatesLastUsedAt(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	tokenRepo := NewRepository(db)

	user, _, err := authRepo.GetOrCreateUser(ctx, "lastusedat@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	tp, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	rawToken, err := secure.NewAPIToken()
	if err != nil {
		t.Fatalf("secure.NewAPIToken: %v", err)
	}

	tokenID, err := tokenRepo.CreateAPIToken(ctx, user.ID, "test", secure.Hash(rawToken), "")
	if err != nil {
		t.Fatalf("CreateAPIToken: %v", err)
	}

	if err := tokenRepo.AddTopicToAPIToken(ctx, tokenID, tp.ID); err != nil {
		t.Fatalf("AddTopicToAPIToken: %v", err)
	}

	if _, _, err := tokenRepo.ValidateAPITokenForTopic(ctx, secure.Hash(rawToken), "alerts"); err != nil {
		t.Fatalf("ValidateAPITokenForTopic: %v", err)
	}

	tok, err := tokenRepo.GetAPITokenByID(ctx, tokenID, user.ID)
	if err != nil {
		t.Fatalf("GetAPITokenByID: %v", err)
	}
	if tok.LastUsedAt == nil {
		t.Error("last_used_at not updated after validation")
	}
}

func TestValidateAPIToken_BlockedUser(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	tokenRepo := NewRepository(db)

	user, _, err := authRepo.GetOrCreateUser(ctx, "blocked@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	rawToken, err := secure.NewAPIToken()
	if err != nil {
		t.Fatalf("secure.NewAPIToken: %v", err)
	}

	_, err = tokenRepo.CreateAPIToken(ctx, user.ID, "test", secure.Hash(rawToken), "")
	if err != nil {
		t.Fatalf("CreateAPIToken: %v", err)
	}

	if _, err := db.ExecContext(ctx,
		`UPDATE users SET account_status = 'blocked' WHERE id = ?`,
		user.ID,
	); err != nil {
		t.Fatalf("set blocked status: %v", err)
	}

	_, err = tokenRepo.ValidateAPIToken(ctx, secure.Hash(rawToken))
	if err == nil {
		t.Fatal("expected error for blocked user, got nil")
	}
}

func TestValidateAPITokenForTopic_BlockedUser(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	tokenRepo := NewRepository(db)

	user, _, err := authRepo.GetOrCreateUser(ctx, "blockedtopic@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	tp, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	rawToken, err := secure.NewAPIToken()
	if err != nil {
		t.Fatalf("secure.NewAPIToken: %v", err)
	}

	tokenID, err := tokenRepo.CreateAPIToken(ctx, user.ID, "test", secure.Hash(rawToken), "")
	if err != nil {
		t.Fatalf("CreateAPIToken: %v", err)
	}

	if err := tokenRepo.AddTopicToAPIToken(ctx, tokenID, tp.ID); err != nil {
		t.Fatalf("AddTopicToAPIToken: %v", err)
	}

	if _, err := db.ExecContext(ctx,
		`UPDATE users SET account_status = 'blocked' WHERE id = ?`,
		user.ID,
	); err != nil {
		t.Fatalf("set blocked status: %v", err)
	}

	_, _, err = tokenRepo.ValidateAPITokenForTopic(ctx, secure.Hash(rawToken), "alerts")
	if err == nil {
		t.Fatal("expected error for blocked user, got nil")
	}
}
