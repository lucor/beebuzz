package notification

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"go.beebuzz.app/beebuzz/internal/testutil"
)

func seedOutboxOwner(t *testing.T, ctx context.Context) (*OutboxRepository, string, string, string) {
	t.Helper()

	db := testutil.NewDB(t)
	userID := "user-outbox"
	topicID := "topic-outbox"
	deviceID := "device-outbox"
	now := time.Now().UTC().UnixMilli()

	testutil.InsertUsers(t, ctx, db, userID)
	if _, err := db.ExecContext(ctx,
		`INSERT INTO topics (id, user_id, name, description, created_at, updated_at)
		 VALUES (?, ?, ?, '', ?, ?)`,
		topicID, userID, "alerts", now, now,
	); err != nil {
		t.Fatalf("insert topic: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO devices (id, user_id, name, description, is_active, pairing_status, created_at, updated_at)
		 VALUES (?, ?, 'phone', '', 1, 'paired', ?, ?)`,
		deviceID, userID, now, now,
	); err != nil {
		t.Fatalf("insert device: %v", err)
	}

	return NewOutboxRepository(db), userID, topicID, deviceID
}

func TestOutboxListForDeviceFiltersByDeviceCursorAndExpiry(t *testing.T) {
	ctx := context.Background()
	repo, userID, topicID, deviceID := seedOutboxOwner(t, ctx)

	firstID := uuid.Must(uuid.NewV7()).String()
	time.Sleep(time.Millisecond)
	secondID := uuid.Must(uuid.NewV7()).String()
	now := time.Now().UTC()

	for _, record := range []OutboxRecord{
		{
			ID:           firstID,
			UserID:       userID,
			TopicID:      topicID,
			Topic:        "alerts",
			DeliveryMode: DeliveryModeServerTrusted,
			PayloadJSON:  `{"id":"` + firstID + `","title":"one","sent_at":"` + now.Format(time.RFC3339) + `"}`,
			ExpiresAt:    now.Add(time.Hour).UnixMilli(),
		},
		{
			ID:           secondID,
			UserID:       userID,
			TopicID:      topicID,
			Topic:        "alerts",
			DeliveryMode: DeliveryModeServerTrusted,
			PayloadJSON:  `{"id":"` + secondID + `","title":"two","sent_at":"` + now.Format(time.RFC3339) + `"}`,
			ExpiresAt:    now.Add(time.Hour).UnixMilli(),
		},
	} {
		if err := repo.Store(ctx, record, []string{deviceID}); err != nil {
			t.Fatalf("Store(%s): %v", record.ID, err)
		}
	}

	records, err := repo.ListForDevice(ctx, deviceID, firstID, now.UnixMilli(), 50)
	if err != nil {
		t.Fatalf("ListForDevice: %v", err)
	}
	if len(records) != 1 || records[0].ID != secondID {
		t.Fatalf("records = %#v, want only %s", records, secondID)
	}
}

func TestSyncDeviceNotificationsReportsOldCursorGap(t *testing.T) {
	ctx := context.Background()
	repo, _, _, deviceID := seedOutboxOwner(t, ctx)
	svc := NewService(nil, nil, nil, nil, "", nil)
	svc.SetOutbox(repo)

	resp, err := svc.SyncDeviceNotifications(ctx, deviceID, "00000000-0000-7000-8000-000000000000", 50)
	if err != nil {
		t.Fatalf("SyncDeviceNotifications: %v", err)
	}
	if !resp.Gap {
		t.Fatal("gap = false, want true")
	}
	if len(resp.Notifications) != 0 {
		t.Fatalf("notifications = %d, want 0", len(resp.Notifications))
	}
}

func TestSyncDeviceNotificationsReturnsRawPayload(t *testing.T) {
	ctx := context.Background()
	repo, userID, topicID, deviceID := seedOutboxOwner(t, ctx)
	svc := NewService(nil, nil, nil, nil, "", nil)
	svc.SetOutbox(repo)

	id := uuid.Must(uuid.NewV7()).String()
	sentAt := time.Now().UTC()
	payload := map[string]string{
		"id":      id,
		"title":   "Hello",
		"sent_at": sentAt.Format(time.RFC3339),
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	if err := repo.Store(ctx, OutboxRecord{
		ID:           id,
		UserID:       userID,
		TopicID:      topicID,
		Topic:        "alerts",
		DeliveryMode: DeliveryModeServerTrusted,
		PayloadJSON:  string(payloadBytes),
		ExpiresAt:    sentAt.Add(time.Hour).UnixMilli(),
	}, []string{deviceID}); err != nil {
		t.Fatalf("Store: %v", err)
	}

	resp, err := svc.SyncDeviceNotifications(ctx, deviceID, "", 50)
	if err != nil {
		t.Fatalf("SyncDeviceNotifications: %v", err)
	}
	if len(resp.Notifications) != 1 {
		t.Fatalf("notifications = %d, want 1", len(resp.Notifications))
	}
	if string(resp.Notifications[0].Payload) != string(payloadBytes) {
		t.Fatalf("payload = %s, want %s", resp.Notifications[0].Payload, payloadBytes)
	}
}
