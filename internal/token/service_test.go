package token

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"lucor.dev/beebuzz/internal/auth"
	"lucor.dev/beebuzz/internal/testutil"
	"lucor.dev/beebuzz/internal/topic"
)

func TestCreateAPITokenRollsBackOnTopicAssociationFailure(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	tokenRepo := NewRepository(db)
	topicRepo := topic.NewRepository(db)
	topicSvc := topic.NewService(topicRepo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	tokenSvc := NewService(tokenRepo, topicSvc)

	user, _, err := authRepo.GetOrCreateUser(ctx, "token-create-rollback@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	_, _, err = tokenSvc.CreateAPIToken(ctx, user.ID, "test-token", "", []string{"missing-topic"})
	if !errors.Is(err, ErrInvalidTopicSelection) {
		t.Fatalf("CreateAPIToken() error = %v, want %v", err, ErrInvalidTopicSelection)
	}

	tokens, err := tokenRepo.ListAPITokens(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListAPITokens: %v", err)
	}
	if len(tokens) != 0 {
		t.Fatalf("ListAPITokens() len = %d, want 0", len(tokens))
	}
}

func TestUpdateAPITokenRollsBackOnTopicAssociationFailure(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	tokenRepo := NewRepository(db)
	topicSvc := topic.NewService(topicRepo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	tokenSvc := NewService(tokenRepo, topicSvc)

	user, _, err := authRepo.GetOrCreateUser(ctx, "token-update-rollback@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	originalTopic, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	_, tokenID, err := tokenSvc.CreateAPIToken(ctx, user.ID, "token-name", "desc", []string{originalTopic.ID})
	if err != nil {
		t.Fatalf("CreateAPIToken: %v", err)
	}

	err = tokenSvc.UpdateAPIToken(ctx, user.ID, tokenID, "updated-name", "updated-desc", []string{"missing-topic"})
	if !errors.Is(err, ErrInvalidTopicSelection) {
		t.Fatalf("UpdateAPIToken() error = %v, want %v", err, ErrInvalidTopicSelection)
	}

	storedToken, err := tokenRepo.GetAPITokenByID(ctx, tokenID, user.ID)
	if err != nil {
		t.Fatalf("GetAPITokenByID: %v", err)
	}
	if storedToken.Name != "token-name" {
		t.Fatalf("token name = %q, want %q", storedToken.Name, "token-name")
	}
	if storedToken.Description == nil || *storedToken.Description != "desc" {
		t.Fatalf("token description = %v, want desc", storedToken.Description)
	}

	topicIDs, err := tokenRepo.GetAPITokenTopicIDs(ctx, tokenID)
	if err != nil {
		t.Fatalf("GetAPITokenTopicIDs: %v", err)
	}
	if len(topicIDs) != 1 || topicIDs[0] != originalTopic.ID {
		t.Fatalf("token topicIDs = %#v, want [%q]", topicIDs, originalTopic.ID)
	}
}

func TestCreateAPITokenRejectsTopicOwnedByAnotherUser(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	tokenRepo := NewRepository(db)
	topicSvc := topic.NewService(topicRepo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	tokenSvc := NewService(tokenRepo, topicSvc)

	owner, _, err := authRepo.GetOrCreateUser(ctx, "token-owner@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser owner: %v", err)
	}
	other, _, err := authRepo.GetOrCreateUser(ctx, "token-other@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser other: %v", err)
	}

	otherTopic, err := topicRepo.Create(ctx, other.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	_, _, err = tokenSvc.CreateAPIToken(ctx, owner.ID, "token-name", "", []string{otherTopic.ID})
	if !errors.Is(err, ErrInvalidTopicSelection) {
		t.Fatalf("CreateAPIToken() error = %v, want %v", err, ErrInvalidTopicSelection)
	}
}

func TestCreateAPITokenSuccess(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	tokenRepo := NewRepository(db)
	topicSvc := topic.NewService(topicRepo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	tokenSvc := NewService(tokenRepo, topicSvc)

	user, _, err := authRepo.GetOrCreateUser(ctx, "token-create-success@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	topicRow, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	rawToken, tokenID, err := tokenSvc.CreateAPIToken(ctx, user.ID, "test-token", "desc", []string{topicRow.ID})
	if err != nil {
		t.Fatalf("CreateAPIToken: %v", err)
	}
	if rawToken == "" {
		t.Fatal("rawToken is empty")
	}
	if tokenID == "" {
		t.Fatal("tokenID is empty")
	}

	tokens, err := tokenSvc.ListAPITokens(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListAPITokens: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("ListAPITokens() len = %d, want 1", len(tokens))
	}
	if tokens[0].ID != tokenID {
		t.Errorf("tokenID = %q, want %q", tokens[0].ID, tokenID)
	}
}

func TestListAPITokensEmpty(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	tokenRepo := NewRepository(db)
	topicSvc := topic.NewService(nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	tokenSvc := NewService(tokenRepo, topicSvc)

	user, _, err := authRepo.GetOrCreateUser(ctx, "token-list-empty@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	tokens, err := tokenSvc.ListAPITokens(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListAPITokens: %v", err)
	}
	if len(tokens) != 0 {
		t.Fatalf("ListAPITokens() len = %d, want 0", len(tokens))
	}
}

func TestUpdateAPITokenSuccess(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	tokenRepo := NewRepository(db)
	topicSvc := topic.NewService(topicRepo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	tokenSvc := NewService(tokenRepo, topicSvc)

	user, _, err := authRepo.GetOrCreateUser(ctx, "token-update-success@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	topicRow, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	_, tokenID, err := tokenSvc.CreateAPIToken(ctx, user.ID, "original-name", "original-desc", []string{topicRow.ID})
	if err != nil {
		t.Fatalf("CreateAPIToken: %v", err)
	}

	err = tokenSvc.UpdateAPIToken(ctx, user.ID, tokenID, "updated-name", "updated-desc", []string{topicRow.ID})
	if err != nil {
		t.Fatalf("UpdateAPIToken: %v", err)
	}

	storedToken, err := tokenRepo.GetAPITokenByID(ctx, tokenID, user.ID)
	if err != nil {
		t.Fatalf("GetAPITokenByID: %v", err)
	}
	if storedToken.Name != "updated-name" {
		t.Errorf("token name = %q, want %q", storedToken.Name, "updated-name")
	}
	if storedToken.Description == nil || *storedToken.Description != "updated-desc" {
		t.Errorf("token description = %v, want %q", storedToken.Description, "updated-desc")
	}
}

func TestRevokeAPITokenSuccess(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	tokenRepo := NewRepository(db)
	topicSvc := topic.NewService(topicRepo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	tokenSvc := NewService(tokenRepo, topicSvc)

	user, _, err := authRepo.GetOrCreateUser(ctx, "token-revoke@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	topicRow, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	_, tokenID, err := tokenSvc.CreateAPIToken(ctx, user.ID, "test-token", "", []string{topicRow.ID})
	if err != nil {
		t.Fatalf("CreateAPIToken: %v", err)
	}

	err = tokenSvc.RevokeAPIToken(ctx, user.ID, tokenID)
	if err != nil {
		t.Fatalf("RevokeAPIToken: %v", err)
	}

	tokens, err := tokenSvc.ListAPITokens(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListAPITokens: %v", err)
	}
	if len(tokens) != 0 {
		t.Fatalf("ListAPITokens() len = %d, want 0 after revocation", len(tokens))
	}
}

func TestRevokeAPITokenNotFound(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	tokenRepo := NewRepository(db)
	topicSvc := topic.NewService(nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	tokenSvc := NewService(tokenRepo, topicSvc)

	user, _, err := authRepo.GetOrCreateUser(ctx, "token-revoke-notfound@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	err = tokenSvc.RevokeAPIToken(ctx, user.ID, "non-existent-token-id")
	if !errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("RevokeAPIToken() error = %v, want %v", err, ErrTokenNotFound)
	}
}

func TestValidateAPITokenSuccess(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	tokenRepo := NewRepository(db)
	topicSvc := topic.NewService(topicRepo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	tokenSvc := NewService(tokenRepo, topicSvc)

	user, _, err := authRepo.GetOrCreateUser(ctx, "token-validate@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	topicRow, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	rawToken, _, err := tokenSvc.CreateAPIToken(ctx, user.ID, "test-token", "", []string{topicRow.ID})
	if err != nil {
		t.Fatalf("CreateAPIToken: %v", err)
	}

	validatedUserID, err := tokenSvc.ValidateAPIToken(ctx, rawToken)
	if err != nil {
		t.Fatalf("ValidateAPIToken: %v", err)
	}
	if validatedUserID != user.ID {
		t.Errorf("validatedUserID = %q, want %q", validatedUserID, user.ID)
	}
}

func TestValidateAPITokenInvalid(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	tokenRepo := NewRepository(db)
	topicSvc := topic.NewService(nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	tokenSvc := NewService(tokenRepo, topicSvc)

	_, err := tokenSvc.ValidateAPIToken(ctx, "invalid-token-that-does-not-exist")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestValidateAPITokenForTopicSuccess(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	tokenRepo := NewRepository(db)
	topicSvc := topic.NewService(topicRepo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	tokenSvc := NewService(tokenRepo, topicSvc)

	user, _, err := authRepo.GetOrCreateUser(ctx, "token-validate-topic@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	topicRow, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	rawToken, _, err := tokenSvc.CreateAPIToken(ctx, user.ID, "test-token", "", []string{topicRow.ID})
	if err != nil {
		t.Fatalf("CreateAPIToken: %v", err)
	}

	validatedUserID, validatedTopicID, err := tokenSvc.ValidateAPITokenForTopic(ctx, rawToken, "alerts")
	if err != nil {
		t.Fatalf("ValidateAPITokenForTopic: %v", err)
	}
	if validatedUserID != user.ID {
		t.Errorf("validatedUserID = %q, want %q", validatedUserID, user.ID)
	}
	if validatedTopicID != topicRow.ID {
		t.Errorf("validatedTopicID = %q, want %q", validatedTopicID, topicRow.ID)
	}
}

func TestValidateAPITokenForTopicUnauthorized(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	tokenRepo := NewRepository(db)
	topicSvc := topic.NewService(topicRepo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	tokenSvc := NewService(tokenRepo, topicSvc)

	user, _, err := authRepo.GetOrCreateUser(ctx, "token-validate-topic-unauth@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	topicRow, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	topicRow2, err := topicRepo.Create(ctx, user.ID, "other-topic", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	rawToken, _, err := tokenSvc.CreateAPIToken(ctx, user.ID, "test-token", "", []string{topicRow.ID})
	if err != nil {
		t.Fatalf("CreateAPIToken: %v", err)
	}

	_, _, err = tokenSvc.ValidateAPITokenForTopic(ctx, rawToken, topicRow2.Name)
	if err == nil {
		t.Fatal("expected error for unauthorized topic")
	}
}
