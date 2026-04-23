package topic

import (
	"context"
	"testing"
	"time"

	"lucor.dev/beebuzz/internal/testutil"
)

func TestCreate(t *testing.T) {
	db := testutil.NewDBWithUsers(t, topicTestUserIDs...)
	repo := NewRepository(db)
	ctx := context.Background()

	userID := "test-user-1"
	topic, err := repo.Create(ctx, userID, "alerts", "Notifications for alerts")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if topic.ID == "" {
		t.Error("Create() topic.ID should not be empty")
	}
	if topic.UserID != userID {
		t.Errorf("Create() userID = %v, want %v", topic.UserID, userID)
	}
	if topic.Name != "alerts" {
		t.Errorf("Create() name = %v, want alerts", topic.Name)
	}
	if topic.Description == nil || *topic.Description != "Notifications for alerts" {
		t.Errorf("Create() description = %v, want Notifications for alerts", topic.Description)
	}
	if topic.CreatedAt == 0 {
		t.Error("Create() createdAt should not be zero")
	}
	if topic.UpdatedAt == 0 {
		t.Error("Create() updatedAt should not be zero")
	}
}

func TestCreateWithEmptyDescription(t *testing.T) {
	db := testutil.NewDBWithUsers(t, topicTestUserIDs...)
	repo := NewRepository(db)
	ctx := context.Background()

	topic, err := repo.Create(ctx, "user-1", "general", "")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if topic.Description != nil {
		t.Errorf("Create() description = %v, want nil", topic.Description)
	}
}

func TestCreateDuplicate(t *testing.T) {
	db := testutil.NewDBWithUsers(t, topicTestUserIDs...)
	repo := NewRepository(db)
	ctx := context.Background()

	userID := "test-user-2"
	_, err := repo.Create(ctx, userID, "alerts", "desc")
	if err != nil {
		t.Fatalf("first Create() error = %v", err)
	}

	_, err = repo.Create(ctx, userID, "alerts", "desc")
	if err != ErrTopicNameConflict {
		t.Errorf("second Create() error = %v, want ErrTopicNameConflict", err)
	}
}

func TestGetByUser(t *testing.T) {
	db := testutil.NewDBWithUsers(t, topicTestUserIDs...)
	repo := NewRepository(db)
	ctx := context.Background()

	userID := "test-user-3"
	_, _ = repo.Create(ctx, userID, "general", "default")
	_, _ = repo.Create(ctx, userID, "alerts", "alerts desc")
	_, _ = repo.Create(ctx, userID, "news", "news desc")

	topics, err := repo.GetByUser(ctx, userID)
	if err != nil {
		t.Fatalf("GetByUser() error = %v", err)
	}

	if len(topics) != 3 {
		t.Fatalf("GetByUser() len = %v, want 3", len(topics))
	}

	if topics[0].Name != "general" {
		t.Errorf("first topic name = %v, want general", topics[0].Name)
	}
	if topics[1].Name != "alerts" {
		t.Errorf("second topic name = %v, want alerts", topics[1].Name)
	}
	if topics[2].Name != "news" {
		t.Errorf("third topic name = %v, want news", topics[2].Name)
	}
}

func TestGetByUserEmpty(t *testing.T) {
	db := testutil.NewDBWithUsers(t, topicTestUserIDs...)
	repo := NewRepository(db)
	ctx := context.Background()

	topics, err := repo.GetByUser(ctx, "non-existent-user")
	if err != nil {
		t.Fatalf("GetByUser() error = %v", err)
	}

	if len(topics) != 0 {
		t.Errorf("GetByUser() len = %v, want 0", len(topics))
	}
}

func TestGetByID(t *testing.T) {
	db := testutil.NewDBWithUsers(t, topicTestUserIDs...)
	repo := NewRepository(db)
	ctx := context.Background()

	userID := "test-user-4"
	created, _ := repo.Create(ctx, userID, "alerts", "desc")

	found, err := repo.GetByID(ctx, userID, created.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if found == nil {
		t.Fatal("GetByID() got nil, want topic")
	}
	if found.ID != created.ID {
		t.Errorf("GetByID() id = %v, want %v", found.ID, created.ID)
	}
}

func TestGetByIDNotFound(t *testing.T) {
	db := testutil.NewDBWithUsers(t, topicTestUserIDs...)
	repo := NewRepository(db)
	ctx := context.Background()

	found, err := repo.GetByID(ctx, "user-1", "non-existent-id")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if found != nil {
		t.Errorf("GetByID() got %v, want nil", found)
	}
}

func TestGetByIDWrongUser(t *testing.T) {
	db := testutil.NewDBWithUsers(t, topicTestUserIDs...)
	repo := NewRepository(db)
	ctx := context.Background()

	userID := "test-user-5"
	created, _ := repo.Create(ctx, userID, "alerts", "desc")

	found, err := repo.GetByID(ctx, "other-user", created.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if found != nil {
		t.Errorf("GetByID() got %v, want nil", found)
	}
}

func TestGetByName(t *testing.T) {
	db := testutil.NewDBWithUsers(t, topicTestUserIDs...)
	repo := NewRepository(db)
	ctx := context.Background()

	userID := "test-user-6"
	_, _ = repo.Create(ctx, userID, "alerts", "desc")

	found, err := repo.GetByName(ctx, userID, "alerts")
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

func TestGetByNameNotFound(t *testing.T) {
	db := testutil.NewDBWithUsers(t, topicTestUserIDs...)
	repo := NewRepository(db)
	ctx := context.Background()

	found, err := repo.GetByName(ctx, "user-1", "non-existent")
	if err != nil {
		t.Fatalf("GetByName() error = %v", err)
	}

	if found != nil {
		t.Errorf("GetByName() got %v, want nil", found)
	}
}

func TestUpdate(t *testing.T) {
	db := testutil.NewDBWithUsers(t, topicTestUserIDs...)
	repo := NewRepository(db)
	ctx := context.Background()

	userID := "test-user-7"
	created, _ := repo.Create(ctx, userID, "alerts", "old desc")

	err := repo.Update(ctx, userID, created.ID, "new desc")
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	updated, _ := repo.GetByID(ctx, userID, created.ID)
	if updated.Description == nil || *updated.Description != "new desc" {
		t.Errorf("Update() description = %v, want new desc", updated.Description)
	}
}

func TestUpdateNotFound(t *testing.T) {
	db := testutil.NewDBWithUsers(t, topicTestUserIDs...)
	repo := NewRepository(db)
	ctx := context.Background()

	err := repo.Update(ctx, "user-1", "non-existent-id", "desc")
	if err != ErrTopicNotFound {
		t.Errorf("Update() error = %v, want ErrTopicNotFound", err)
	}
}

func TestDelete(t *testing.T) {
	db := testutil.NewDBWithUsers(t, topicTestUserIDs...)
	repo := NewRepository(db)
	ctx := context.Background()

	userID := "test-user-8"
	created, _ := repo.Create(ctx, userID, "alerts", "desc")

	err := repo.Delete(ctx, userID, created.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	found, _ := repo.GetByID(ctx, userID, created.ID)
	if found != nil {
		t.Error("Delete() topic should not exist after delete")
	}
}

func TestDeleteNotFound(t *testing.T) {
	db := testutil.NewDBWithUsers(t, topicTestUserIDs...)
	repo := NewRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, "user-1", "non-existent-id")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}

func TestDeleteOtherUserTopic(t *testing.T) {
	db := testutil.NewDBWithUsers(t, topicTestUserIDs...)
	repo := NewRepository(db)
	ctx := context.Background()

	userID := "test-user-9"
	created, _ := repo.Create(ctx, userID, "alerts", "desc")

	err := repo.Delete(ctx, "other-user", created.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	found, _ := repo.GetByID(ctx, userID, created.ID)
	if found == nil {
		t.Error("Delete() topic should still exist for owner")
	}
}

func TestTopicTimestamps(t *testing.T) {
	db := testutil.NewDBWithUsers(t, topicTestUserIDs...)
	repo := NewRepository(db)
	ctx := context.Background()

	before := time.Now().UTC().UnixMilli()
	topic, _ := repo.Create(ctx, "user-1", "test", "desc")
	after := time.Now().UTC().UnixMilli()

	if topic.CreatedAt < before || topic.CreatedAt > after {
		t.Errorf("Create() createdAt outside expected range: %v", topic.CreatedAt)
	}
	if topic.UpdatedAt < before || topic.UpdatedAt > after {
		t.Errorf("Create() updatedAt outside expected range: %v", topic.UpdatedAt)
	}
}

func TestMultipleUsersIsolation(t *testing.T) {
	db := testutil.NewDBWithUsers(t, topicTestUserIDs...)
	repo := NewRepository(db)
	ctx := context.Background()

	_, _ = repo.Create(ctx, "user-a", "alerts", "desc a")
	_, _ = repo.Create(ctx, "user-b", "alerts", "desc b")

	topicsA, _ := repo.GetByUser(ctx, "user-a")
	topicsB, _ := repo.GetByUser(ctx, "user-b")

	if len(topicsA) != 1 || len(topicsB) != 1 {
		t.Fatalf("GetByUser() should return 1 topic each, got %d and %d", len(topicsA), len(topicsB))
	}

	if topicsA[0].UserID == topicsB[0].UserID {
		t.Error("Topics should be isolated between users")
	}
}
