package event

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"lucor.dev/beebuzz/internal/testutil"
)

// seedUser inserts a minimal user row so FK constraints pass.
func seedUser(t *testing.T, db *sqlx.DB, userID string) {
	t.Helper()
	now := time.Now().UTC().UnixMilli()
	_, err := db.Exec(
		`INSERT INTO users (id, email, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		userID, userID+"@test.local", now, now,
	)
	if err != nil {
		t.Fatalf("seedUser: %v", err)
	}
}

func newNotificationCreatedEvent(userID string, createdAt int64, topic *string, source *string, enc *string) *NotificationEvent {
	return &NotificationEvent{
		ID:             NewEventID(),
		UserID:         userID,
		EventType:      TypeNotificationCreated,
		Topic:          topic,
		Source:         source,
		EncryptionMode: enc,
		CreatedAt:      createdAt,
	}
}

func TestInsertEventPersistsRawEvent(t *testing.T) {
	db := testutil.NewDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	userID := "user-1"
	seedUser(t, db, userID)

	source := SourceAPI
	enc := EncryptionServerTrusted
	topic := "alerts"
	ev := newNotificationCreatedEvent(userID, time.Now().UTC().UnixMilli(), &topic, &source, &enc)

	if err := repo.InsertEvent(ctx, ev); err != nil {
		t.Fatalf("InsertEvent: %v", err)
	}

	// Verify raw event was inserted.
	var count int
	if err := db.Get(&count, "SELECT COUNT(*) FROM notification_events WHERE user_id = ?", userID); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 event, got %d", count)
	}
}

func TestInsertEventUpdatesDailySummary(t *testing.T) {
	db := testutil.NewDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	userID := "user-1"
	seedUser(t, db, userID)

	source := SourceAPI
	enc := EncryptionServerTrusted
	topic := "alerts"
	now := time.Now().UTC().UnixMilli()
	ev := newNotificationCreatedEvent(userID, now, &topic, &source, &enc)

	if err := repo.InsertEvent(ctx, ev); err != nil {
		t.Fatalf("InsertEvent: %v", err)
	}

	// Verify summary was upserted.
	summaries, err := repo.GetDailySummaries(ctx, userID, 0, now+86400000)
	if err != nil {
		t.Fatalf("GetDailySummaries: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary row, got %d", len(summaries))
	}

	s := summaries[0]
	if s.NotificationsTotal != 1 {
		t.Errorf("notifications_total = %d, want 1", s.NotificationsTotal)
	}
	if s.NotificationsServerTrusted != 1 {
		t.Errorf("notifications_server_trusted = %d, want 1", s.NotificationsServerTrusted)
	}
	if s.SourcesAPI != 1 {
		t.Errorf("sources_api = %d, want 1", s.SourcesAPI)
	}
}

func TestInsertEvent_IncrementsSummary(t *testing.T) {
	db := testutil.NewDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	userID := "user-2"
	seedUser(t, db, userID)

	source := SourceWebhook
	enc := EncryptionE2E
	now := time.Now().UTC().UnixMilli()

	// Insert two notification_created events.
	for i := 0; i < 2; i++ {
		ev := newNotificationCreatedEvent(userID, now, nil, &source, &enc)
		if err := repo.InsertEvent(ctx, ev); err != nil {
			t.Fatalf("InsertEvent #%d: %v", i, err)
		}
	}

	// Insert one delivered event.
	deviceID := "dev-1"
	evDelivered := &NotificationEvent{
		ID:        NewEventID(),
		UserID:    userID,
		EventType: TypeNotificationDelivered,
		DeviceID:  &deviceID,
		CreatedAt: now,
	}
	if err := repo.InsertEvent(ctx, evDelivered); err != nil {
		t.Fatalf("InsertEvent delivered: %v", err)
	}

	summaries, err := repo.GetDailySummaries(ctx, userID, 0, now+86400000)
	if err != nil {
		t.Fatalf("GetDailySummaries: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}

	s := summaries[0]
	if s.NotificationsTotal != 2 {
		t.Errorf("notifications_total = %d, want 2", s.NotificationsTotal)
	}
	if s.NotificationsE2E != 2 {
		t.Errorf("notifications_e2e = %d, want 2", s.NotificationsE2E)
	}
	if s.NotificationsDelivered != 1 {
		t.Errorf("notifications_delivered = %d, want 1", s.NotificationsDelivered)
	}
	if s.SourcesWebhook != 2 {
		t.Errorf("sources_webhook = %d, want 2", s.SourcesWebhook)
	}
}

func TestInsertEvent_FailedWithSubscriptionGone(t *testing.T) {
	db := testutil.NewDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	userID := "user-3"
	seedUser(t, db, userID)

	deviceID := "dev-gone"
	failReason := FailSubscriptionGone
	ev := &NotificationEvent{
		ID:         NewEventID(),
		UserID:     userID,
		EventType:  TypeNotificationFailed,
		DeviceID:   &deviceID,
		FailReason: &failReason,
		CreatedAt:  time.Now().UTC().UnixMilli(),
	}
	if err := repo.InsertEvent(ctx, ev); err != nil {
		t.Fatalf("InsertEvent: %v", err)
	}

	summaries, err := repo.GetDailySummaries(ctx, userID, 0, time.Now().UTC().UnixMilli()+86400000)
	if err != nil {
		t.Fatalf("GetDailySummaries: %v", err)
	}

	s := summaries[0]
	if s.NotificationsFailed != 1 {
		t.Errorf("notifications_failed = %d, want 1", s.NotificationsFailed)
	}
	if s.DevicesLost != 1 {
		t.Errorf("devices_lost = %d, want 1", s.DevicesLost)
	}
}

func TestDeleteEventsOlderThan(t *testing.T) {
	db := testutil.NewDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	userID := "user-4"
	seedUser(t, db, userID)

	// Insert old event (60 days ago).
	oldTime := time.Now().UTC().AddDate(0, 0, -60).UnixMilli()
	evOld := &NotificationEvent{
		ID:        NewEventID(),
		UserID:    userID,
		EventType: TypeNotificationCreated,
		CreatedAt: oldTime,
	}
	if err := repo.InsertEvent(ctx, evOld); err != nil {
		t.Fatalf("InsertEvent old: %v", err)
	}

	// Insert recent event.
	evNew := &NotificationEvent{
		ID:        NewEventID(),
		UserID:    userID,
		EventType: TypeNotificationCreated,
		CreatedAt: time.Now().UTC().UnixMilli(),
	}
	if err := repo.InsertEvent(ctx, evNew); err != nil {
		t.Fatalf("InsertEvent new: %v", err)
	}

	cutoff := time.Now().UTC().AddDate(0, 0, -30).UnixMilli()
	deleted, err := repo.DeleteEventsOlderThan(ctx, cutoff, 100)
	if err != nil {
		t.Fatalf("DeleteEventsOlderThan: %v", err)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}

	// Verify only new event remains.
	var count int
	if err := db.Get(&count, "SELECT COUNT(*) FROM notification_events WHERE user_id = ?", userID); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if count != 1 {
		t.Errorf("remaining events = %d, want 1", count)
	}
}

func TestDayStartMs(t *testing.T) {
	ts := time.Date(2026, 4, 1, 15, 30, 45, 0, time.UTC)
	expected := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

	got := DayStartMs(ts)
	if got != expected {
		t.Errorf("DayStartMs = %d, want %d", got, expected)
	}
}

func TestToAccountUsageResponseIncludesSourceBreakdown(t *testing.T) {
	summaryDay := time.Date(2026, 4, 17, 0, 0, 0, 0, time.UTC)
	fromMs := summaryDay.UnixMilli()
	toMs := summaryDay.Add(24*time.Hour).UnixMilli()

	resp := ToAccountUsageResponse(
		[]DailyUsageSummary{
			{
				DayStartMs:                 fromMs,
				NotificationsTotal:         5,
				NotificationsDelivered:     11,
				NotificationsFailed:        2,
				AttachmentsCount:           3,
				AttachmentsBytesTotal:      2048,
				NotificationsServerTrusted: 4,
				NotificationsE2E:           1,
				SourcesCLI:                 2,
				SourcesWebhook:             1,
				SourcesAPI:                 2,
				DevicesLost:                1,
			},
		},
		fromMs,
		toMs,
	)

	if len(resp.Data) != 2 {
		t.Fatalf("response day count = %d, want 2", len(resp.Data))
	}

	firstDay := resp.Data[0]
	if firstDay.SourcesCLI != 2 {
		t.Errorf("sources_cli = %d, want 2", firstDay.SourcesCLI)
	}
	if firstDay.SourcesWebhook != 1 {
		t.Errorf("sources_webhook = %d, want 1", firstDay.SourcesWebhook)
	}
	if firstDay.SourcesAPI != 2 {
		t.Errorf("sources_api = %d, want 2", firstDay.SourcesAPI)
	}

	emptyDay := resp.Data[1]
	if emptyDay.SourcesCLI != 0 || emptyDay.SourcesWebhook != 0 || emptyDay.SourcesAPI != 0 {
		t.Errorf(
			"empty day sources = (%d, %d, %d), want zeros",
			emptyDay.SourcesCLI,
			emptyDay.SourcesWebhook,
			emptyDay.SourcesAPI,
		)
	}
}
