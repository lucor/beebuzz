package webhook

import (
	"context"
	"testing"

	"lucor.dev/beebuzz/internal/auth"
	"lucor.dev/beebuzz/internal/push"
	"lucor.dev/beebuzz/internal/secure"
	"lucor.dev/beebuzz/internal/testutil"
	"lucor.dev/beebuzz/internal/topic"
)

func TestGetByTokenHash_ActiveUser(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	repo := NewRepository(db)

	user, _, err := authRepo.GetOrCreateUser(ctx, "active@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	rawToken, err := secure.NewWebhookToken()
	if err != nil {
		t.Fatalf("secure.NewWebhookToken: %v", err)
	}

	webhookID, err := repo.Create(ctx, user.ID, "test-webhook", "", PayloadTypeBeebuzz, secure.Hash(rawToken), "", "", push.PriorityNormal)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	wh, err := repo.GetByTokenHash(ctx, secure.Hash(rawToken))
	if err != nil {
		t.Fatalf("GetByTokenHash: %v", err)
	}
	if wh == nil {
		t.Fatal("GetByTokenHash returned nil for valid token")
	}
	if wh.ID != webhookID {
		t.Errorf("webhook ID = %q, want %q", wh.ID, webhookID)
	}
}

func TestGetByTokenHash_BlockedUser(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	repo := NewRepository(db)

	user, _, err := authRepo.GetOrCreateUser(ctx, "blocked@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	rawToken, err := secure.NewWebhookToken()
	if err != nil {
		t.Fatalf("secure.NewWebhookToken: %v", err)
	}

	_, err = repo.Create(ctx, user.ID, "test-webhook", "", PayloadTypeBeebuzz, secure.Hash(rawToken), "", "", push.PriorityNormal)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := db.ExecContext(ctx,
		`UPDATE users SET account_status = 'blocked' WHERE id = ?`,
		user.ID,
	); err != nil {
		t.Fatalf("set blocked status: %v", err)
	}

	wh, err := repo.GetByTokenHash(ctx, secure.Hash(rawToken))
	if err != nil {
		t.Fatalf("GetByTokenHash: %v", err)
	}
	if wh != nil {
		t.Error("GetByTokenHash should return nil for blocked user")
	}
}

func TestGetByTokenHash_NotFound(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	repo := NewRepository(db)

	wh, err := repo.GetByTokenHash(ctx, "nonexistent-hash")
	if err != nil {
		t.Fatalf("GetByTokenHash: %v", err)
	}
	if wh != nil {
		t.Errorf("GetByTokenHash returned %v, want nil", wh)
	}
}

func TestGetByTokenHash_InactiveWebhook(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	repo := NewRepository(db)

	user, _, err := authRepo.GetOrCreateUser(ctx, "inactive@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	rawToken, err := secure.NewWebhookToken()
	if err != nil {
		t.Fatalf("secure.NewWebhookToken: %v", err)
	}

	webhookID, err := repo.Create(ctx, user.ID, "test-webhook", "", PayloadTypeBeebuzz, secure.Hash(rawToken), "", "", push.PriorityNormal)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Revoke(ctx, user.ID, webhookID); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	wh, err := repo.GetByTokenHash(ctx, secure.Hash(rawToken))
	if err != nil {
		t.Fatalf("GetByTokenHash: %v", err)
	}
	if wh != nil {
		t.Error("GetByTokenHash should return nil for revoked webhook")
	}
}

func TestCreate_WithTopics(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	repo := NewRepository(db)

	user, _, err := authRepo.GetOrCreateUser(ctx, "withtopics@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	tp, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	rawToken, err := secure.NewWebhookToken()
	if err != nil {
		t.Fatalf("secure.NewWebhookToken: %v", err)
	}

	webhookID, err := repo.CreateWithTopics(ctx, user.ID, "test-wh", "", PayloadTypeBeebuzz, secure.Hash(rawToken), "", "", push.PriorityNormal, []string{tp.ID})
	if err != nil {
		t.Fatalf("CreateWithTopics: %v", err)
	}

	topicIDs, err := repo.GetTopicIDs(ctx, webhookID)
	if err != nil {
		t.Fatalf("GetTopicIDs: %v", err)
	}
	if len(topicIDs) != 1 {
		t.Errorf("GetTopicIDs len = %d, want 1", len(topicIDs))
	}
	if topicIDs[0] != tp.ID {
		t.Errorf("topic ID = %q, want %q", topicIDs[0], tp.ID)
	}
}

func TestRevoke(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	repo := NewRepository(db)

	user, _, err := authRepo.GetOrCreateUser(ctx, "revoke@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	rawToken, err := secure.NewWebhookToken()
	if err != nil {
		t.Fatalf("secure.NewWebhookToken: %v", err)
	}

	webhookID, err := repo.Create(ctx, user.ID, "test-webhook", "", PayloadTypeBeebuzz, secure.Hash(rawToken), "", "", push.PriorityNormal)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Revoke(ctx, user.ID, webhookID); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	wh, err := repo.GetByID(ctx, user.ID, webhookID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if wh.IsActive {
		t.Error("webhook should be inactive after revoke")
	}
}

func TestGetByUser(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	repo := NewRepository(db)

	user, _, err := authRepo.GetOrCreateUser(ctx, "list@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	for i := 0; i < 3; i++ {
		rawToken, err := secure.NewWebhookToken()
		if err != nil {
			t.Fatalf("secure.NewWebhookToken: %v", err)
		}
		if _, err := repo.Create(ctx, user.ID, "test-webhook", "", PayloadTypeBeebuzz, secure.Hash(rawToken), "", "", push.PriorityNormal); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	webhooks, err := repo.GetByUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetByUser: %v", err)
	}
	if len(webhooks) != 3 {
		t.Errorf("GetByUser len = %d, want 3", len(webhooks))
	}
}

func TestGetByID(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	repo := NewRepository(db)

	user, _, err := authRepo.GetOrCreateUser(ctx, "getbyid@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	webhookID, err := repo.Create(ctx, user.ID, "test-webhook", "", PayloadTypeBeebuzz, secure.Hash("token"), "", "", push.PriorityNormal)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	wh, err := repo.GetByID(ctx, user.ID, webhookID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if wh == nil {
		t.Fatal("GetByID returned nil")
	}
	if wh.Name != "test-webhook" {
		t.Errorf("webhook name = %q, want %q", wh.Name, "test-webhook")
	}
}

func TestGetByID_NotFound(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	repo := NewRepository(db)

	user, _, err := authRepo.GetOrCreateUser(ctx, "notfound@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	wh, err := repo.GetByID(ctx, user.ID, "nonexistent-id")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if wh != nil {
		t.Errorf("GetByID returned %v, want nil", wh)
	}
}

func TestUpdateTokenHash(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	repo := NewRepository(db)

	user, _, err := authRepo.GetOrCreateUser(ctx, "updatetoken@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	webhookID, err := repo.Create(ctx, user.ID, "test-webhook", "", PayloadTypeBeebuzz, secure.Hash("old-token"), "", "", push.PriorityNormal)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	newRawToken, err := secure.NewWebhookToken()
	if err != nil {
		t.Fatalf("secure.NewWebhookToken: %v", err)
	}

	if err := repo.UpdateTokenHash(ctx, user.ID, webhookID, secure.Hash(newRawToken)); err != nil {
		t.Fatalf("UpdateTokenHash: %v", err)
	}

	oldWh, err := repo.GetByTokenHash(ctx, secure.Hash("old-token"))
	if err != nil {
		t.Fatalf("GetByTokenHash (old): %v", err)
	}
	if oldWh != nil {
		t.Error("old token should not be valid after UpdateTokenHash")
	}

	newWh, err := repo.GetByTokenHash(ctx, secure.Hash(newRawToken))
	if err != nil {
		t.Fatalf("GetByTokenHash (new): %v", err)
	}
	if newWh == nil {
		t.Error("new token should be valid after UpdateTokenHash")
	}
}
