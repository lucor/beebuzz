package topic

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"lucor.dev/beebuzz/internal/testutil"
)

func newTestTopicService(repo *Repository) *Service {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewService(repo, logger)
}

func TestGetTopics(t *testing.T) {
	db := testutil.NewDBWithUsers(t, topicTestUserIDs...)
	repo := NewRepository(db)
	svc := newTestTopicService(repo)
	ctx := context.Background()

	userID := "test-user-1"
	if err := svc.CreateDefaultTopic(ctx, userID); err != nil {
		t.Fatalf("CreateDefaultTopic() error = %v", err)
	}

	topics, err := svc.GetTopics(ctx, userID)
	if err != nil {
		t.Fatalf("GetTopics() error = %v", err)
	}

	if len(topics) != 1 {
		t.Fatalf("GetTopics() len = %v, want 1", len(topics))
	}

	if topics[0].Name != "general" {
		t.Errorf("GetTopics() first topic = %v, want general", topics[0].Name)
	}
}

func TestGetTopicsDoesNotCreateDefault(t *testing.T) {
	db := testutil.NewDBWithUsers(t, topicTestUserIDs...)
	repo := NewRepository(db)
	svc := newTestTopicService(repo)
	ctx := context.Background()

	userID := "test-user-2"
	_, _ = svc.GetTopics(ctx, userID)

	found, _ := repo.GetByName(ctx, userID, "general")
	if found != nil {
		t.Error("GetTopics() should not create default topic")
	}
}

func TestGetTopicsPreservesExisting(t *testing.T) {
	db := testutil.NewDBWithUsers(t, topicTestUserIDs...)
	repo := NewRepository(db)
	svc := newTestTopicService(repo)
	ctx := context.Background()

	if err := svc.CreateDefaultTopic(ctx, "user-3"); err != nil {
		t.Fatalf("CreateDefaultTopic() error = %v", err)
	}
	repo.Create(ctx, "user-3", "custom", "custom desc")

	topics, _ := svc.GetTopics(ctx, "user-3")

	if len(topics) != 2 {
		t.Fatalf("GetTopics() len = %v, want 2", len(topics))
	}
	if topics[0].Name != "general" || topics[1].Name != "custom" {
		t.Errorf("GetTopics() topics not in expected order")
	}
}

func TestCreateTopicReservedName(t *testing.T) {
	db := testutil.NewDBWithUsers(t, topicTestUserIDs...)
	repo := NewRepository(db)
	svc := newTestTopicService(repo)
	ctx := context.Background()

	_, err := svc.CreateTopic(ctx, "user-1", "general", "desc")
	if !errors.Is(err, ErrTopicNameReserved) {
		t.Errorf("CreateTopic() error = %v, want ErrTopicNameReserved", err)
	}
}

func TestCreateTopicCustom(t *testing.T) {
	db := testutil.NewDBWithUsers(t, topicTestUserIDs...)
	repo := NewRepository(db)
	svc := newTestTopicService(repo)
	ctx := context.Background()

	topic, err := svc.CreateTopic(ctx, "user-1", "alerts", "alerts desc")
	if err != nil {
		t.Fatalf("CreateTopic() error = %v", err)
	}

	if topic.Name != "alerts" {
		t.Errorf("CreateTopic() name = %v, want alerts", topic.Name)
	}
}

func TestCreateTopicConflict(t *testing.T) {
	db := testutil.NewDBWithUsers(t, topicTestUserIDs...)
	repo := NewRepository(db)
	svc := newTestTopicService(repo)
	ctx := context.Background()

	_, _ = svc.CreateTopic(ctx, "user-1", "alerts", "desc")

	_, err := svc.CreateTopic(ctx, "user-1", "alerts", "desc")
	if !errors.Is(err, ErrTopicNameConflict) {
		t.Errorf("CreateTopic() error = %v, want ErrTopicNameConflict", err)
	}
}

func TestUpdateTopic(t *testing.T) {
	db := testutil.NewDBWithUsers(t, topicTestUserIDs...)
	repo := NewRepository(db)
	svc := newTestTopicService(repo)
	ctx := context.Background()

	created, _ := svc.CreateTopic(ctx, "user-1", "alerts", "old")

	err := svc.UpdateTopic(ctx, "user-1", created.ID, "new desc")
	if err != nil {
		t.Fatalf("UpdateTopic() error = %v", err)
	}

	updated, _ := repo.GetByID(ctx, "user-1", created.ID)
	if updated.Description == nil || *updated.Description != "new desc" {
		t.Errorf("UpdateTopic() description = %v, want new desc", updated.Description)
	}
}

func TestUpdateTopicNotFound(t *testing.T) {
	db := testutil.NewDBWithUsers(t, topicTestUserIDs...)
	repo := NewRepository(db)
	svc := newTestTopicService(repo)
	ctx := context.Background()

	err := svc.UpdateTopic(ctx, "user-1", "non-existent", "desc")
	if !errors.Is(err, ErrTopicNotFound) {
		t.Errorf("UpdateTopic() error = %v, want ErrTopicNotFound", err)
	}
}

func TestDeleteTopic(t *testing.T) {
	db := testutil.NewDBWithUsers(t, topicTestUserIDs...)
	repo := NewRepository(db)
	svc := newTestTopicService(repo)
	ctx := context.Background()

	created, _ := svc.CreateTopic(ctx, "user-1", "alerts", "desc")

	err := svc.DeleteTopic(ctx, "user-1", created.ID)
	if err != nil {
		t.Fatalf("DeleteTopic() error = %v", err)
	}

	found, _ := repo.GetByID(ctx, "user-1", created.ID)
	if found != nil {
		t.Error("DeleteTopic() topic should be deleted")
	}
}

func TestDeleteTopicProtected(t *testing.T) {
	db := testutil.NewDBWithUsers(t, topicTestUserIDs...)
	repo := NewRepository(db)
	svc := newTestTopicService(repo)
	ctx := context.Background()

	userID := "test-user-4"
	if err := svc.CreateDefaultTopic(ctx, userID); err != nil {
		t.Fatalf("CreateDefaultTopic() error = %v", err)
	}
	topic, _ := repo.GetByName(ctx, userID, "general")

	err := svc.DeleteTopic(ctx, userID, topic.ID)
	if !errors.Is(err, ErrTopicProtected) {
		t.Errorf("DeleteTopic() error = %v, want ErrTopicProtected", err)
	}
}

func TestDeleteTopicNotFound(t *testing.T) {
	db := testutil.NewDBWithUsers(t, topicTestUserIDs...)
	repo := NewRepository(db)
	svc := newTestTopicService(repo)
	ctx := context.Background()

	err := svc.DeleteTopic(ctx, "user-1", "non-existent-id")
	if !errors.Is(err, ErrTopicNotFound) {
		t.Errorf("DeleteTopic() error = %v, want ErrTopicNotFound", err)
	}
}

func TestCreateDefaultTopic(t *testing.T) {
	db := testutil.NewDBWithUsers(t, topicTestUserIDs...)
	repo := NewRepository(db)
	svc := newTestTopicService(repo)
	ctx := context.Background()

	err := svc.CreateDefaultTopic(ctx, "user-1")
	if err != nil {
		t.Fatalf("CreateDefaultTopic() error = %v", err)
	}

	found, _ := repo.GetByName(ctx, "user-1", "general")
	if found == nil {
		t.Fatal("CreateDefaultTopic() topic not found")
	}
	if found.Name != "general" {
		t.Errorf("CreateDefaultTopic() name = %v, want general", found.Name)
	}
}

func TestCreateDefaultTopicIdempotent(t *testing.T) {
	db := testutil.NewDBWithUsers(t, topicTestUserIDs...)
	repo := NewRepository(db)
	svc := newTestTopicService(repo)
	ctx := context.Background()

	_ = svc.CreateDefaultTopic(ctx, "user-1")
	err := svc.CreateDefaultTopic(ctx, "user-1")

	if !errors.Is(err, ErrTopicNameConflict) {
		t.Errorf("CreateDefaultTopic() second call error = %v, want ErrTopicNameConflict", err)
	}
}

func TestServiceGetByName(t *testing.T) {
	db := testutil.NewDBWithUsers(t, topicTestUserIDs...)
	repo := NewRepository(db)
	svc := newTestTopicService(repo)
	ctx := context.Background()

	_, _ = svc.CreateTopic(ctx, "user-1", "alerts", "desc")

	found, err := svc.GetByName(ctx, "user-1", "alerts")
	if err != nil {
		t.Fatalf("GetByName() error = %v", err)
	}

	if found == nil {
		t.Fatal("GetByName() got nil, want topic")
	}
	if found.Name != "alerts" {
		t.Errorf("GetByName() name = %v, want alerts", found.Name)
	}
}

func TestValidateTopicIDsRejectsDuplicates(t *testing.T) {
	db := testutil.NewDBWithUsers(t, topicTestUserIDs...)
	repo := NewRepository(db)
	svc := newTestTopicService(repo)
	ctx := context.Background()

	created, err := svc.CreateTopic(ctx, "user-1", "alerts", "desc")
	if err != nil {
		t.Fatalf("CreateTopic() error = %v", err)
	}

	err = svc.ValidateTopicIDs(ctx, "user-1", []string{created.ID, created.ID})
	if !errors.Is(err, ErrDuplicateTopicIDs) {
		t.Fatalf("ValidateTopicIDs() error = %v, want ErrDuplicateTopicIDs", err)
	}
}
