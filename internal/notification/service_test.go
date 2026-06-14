package notification

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/jmoiron/sqlx"

	"go.beebuzz.app/beebuzz/internal/testutil"
)

func TestMapPriorityToUrgency(t *testing.T) {
	tests := []struct {
		priority string
		want     webpush.Urgency
	}{
		{"high", webpush.UrgencyHigh},
		{"normal", webpush.UrgencyNormal},
		{"", webpush.UrgencyNormal},
		{"unknown", webpush.UrgencyNormal},
	}

	for _, tt := range tests {
		t.Run(tt.priority, func(t *testing.T) {
			got := mapPriorityToUrgency(tt.priority)
			if got != tt.want {
				t.Errorf("mapPriorityToUrgency(%q) = %q, want %q", tt.priority, got, tt.want)
			}
		})
	}
}

func TestE2EEnvelopeJSONIncludesIDTokenAndSentAt(t *testing.T) {
	envelope := E2EEnvelope{
		BeeBuzz: E2EEnvelopeToken{
			ID:     "0195f5d4-c0de-7000-8000-000000000000",
			Token:  "attachment-token",
			SentAt: "2026-04-17T12:00:00Z",
		},
	}

	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var decoded map[string]map[string]string
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if decoded["beebuzz"]["id"] != envelope.BeeBuzz.ID {
		t.Fatalf("id: got %q, want %q", decoded["beebuzz"]["id"], envelope.BeeBuzz.ID)
	}
	if decoded["beebuzz"]["token"] != envelope.BeeBuzz.Token {
		t.Fatalf("token: got %q, want %q", decoded["beebuzz"]["token"], envelope.BeeBuzz.Token)
	}
	if decoded["beebuzz"]["sent_at"] != envelope.BeeBuzz.SentAt {
		t.Fatalf("sent_at: got %q, want %q", decoded["beebuzz"]["sent_at"], envelope.BeeBuzz.SentAt)
	}
}

type stubDeviceProvider struct {
	subs []PushSub
}

func (p *stubDeviceProvider) GetSubscribedDevices(_ context.Context, _, _ string) ([]PushSub, error) {
	return p.subs, nil
}

func (p *stubDeviceProvider) MarkSubscriptionGone(_ context.Context, _ string) error {
	return nil
}

type stubAttachmentStorer struct{}

func (s *stubAttachmentStorer) Store(_ context.Context, _, _ string, _ int, _ []byte) (string, error) {
	return "attachment-token", nil
}

type failingAttachmentStorer struct{}

func (s *failingAttachmentStorer) Store(_ context.Context, _, _ string, _ int, _ []byte) (string, error) {
	return "", errors.New("store failed")
}

func TestSendFailsWhenAttachmentProcessingFails(t *testing.T) {
	svc := NewService(
		&stubDeviceProvider{
			subs: []PushSub{
				{
					DeviceID:     "device-1",
					AgeRecipient: testAgeRecipient,
				},
			},
		},
		&failingAttachmentStorer{},
		nil,
		&VAPIDKeys{PublicKey: "public", PrivateKey: "private"},
		"mailto:test@example.com",
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	_, err := svc.Send(context.Background(), "user-1", "topic-1", SendInput{
		TopicName: "alerts",
		Title:     "Title",
		Body:      "Body",
		Attachment: &AttachmentInput{
			Data:     strings.NewReader("hello"),
			MimeType: "text/plain",
			Filename: "hello.txt",
		},
	}, slog.Default())
	if !errors.Is(err, ErrAttachmentProcessingFailed) {
		t.Fatalf("Send() error = %v, want %v", err, ErrAttachmentProcessingFailed)
	}
}

func TestSendE2EStoresEnvelopeInOutbox(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewDB(t)
	userID := "user-send-e2e-outbox"
	topicID := "topic-send-e2e-outbox"
	deviceID := "device-send-e2e-outbox"
	seedNotificationOutboxRows(t, ctx, db, userID, topicID, deviceID)

	svc := NewService(
		&stubDeviceProvider{
			subs: []PushSub{
				{
					DeviceID:     deviceID,
					Endpoint:     "https://fcm.googleapis.com/fcm/send/e2e-outbox",
					P256dh:       "p256dh",
					Auth:         "auth",
					AgeRecipient: testAgeRecipient,
				},
			},
		},
		&stubAttachmentStorer{},
		nil,
		&VAPIDKeys{PublicKey: "public", PrivateKey: "private"},
		"mailto:test@example.com",
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	svc.SetOutbox(NewOutboxRepository(db))
	svc.SetPushStubBroker(NewPushStubBroker(slog.New(slog.NewTextHandler(io.Discard, nil))))

	report, err := svc.Send(ctx, userID, topicID, SendInput{
		TopicName:    "alerts",
		DeliveryMode: DeliveryModeE2E,
		OpaqueBlob:   []byte("encrypted-payload"),
	}, slog.Default())
	if err != nil {
		t.Fatalf("Send E2E: %v", err)
	}
	if report.TotalSent != 1 {
		t.Fatalf("TotalSent = %d, want 1", report.TotalSent)
	}

	var envelopeJSON string
	if err := db.GetContext(ctx, &envelopeJSON, `SELECT payload_json FROM notification_outbox WHERE id = (SELECT notification_id FROM notification_outbox_recipients WHERE device_id = ?)`, deviceID); err != nil {
		t.Fatalf("query outbox: %v", err)
	}
	if envelopeJSON == "" {
		t.Fatal("outbox envelope is empty")
	}

	var envelope E2EEnvelope
	if err := json.Unmarshal([]byte(envelopeJSON), &envelope); err != nil {
		t.Fatalf("unmarshal outbox envelope: %v", err)
	}
	if envelope.BeeBuzz.ID == "" {
		t.Fatal("envelope missing id")
	}
	if envelope.BeeBuzz.Token == "" {
		t.Fatal("envelope missing token")
	}
	if envelope.BeeBuzz.SentAt == "" {
		t.Fatal("envelope missing sent_at")
	}
}

func TestSendStoresTrustedPayloadInOutbox(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewDB(t)
	userID := "user-send-outbox"
	topicID := "topic-send-outbox"
	deviceID := "device-send-outbox"
	seedNotificationOutboxRows(t, ctx, db, userID, topicID, deviceID)

	svc := NewService(
		&stubDeviceProvider{
			subs: []PushSub{
				{
					DeviceID: deviceID,
					Endpoint: "https://fcm.googleapis.com/fcm/send/outbox",
					P256dh:   "p256dh",
					Auth:     "auth",
				},
			},
		},
		nil,
		nil,
		&VAPIDKeys{PublicKey: "public", PrivateKey: "private"},
		"mailto:test@example.com",
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	svc.SetOutbox(NewOutboxRepository(db))
	svc.SetPushStubBroker(NewPushStubBroker(slog.New(slog.NewTextHandler(io.Discard, nil))))

	report, err := svc.Send(ctx, userID, topicID, SendInput{
		TopicName:    "alerts",
		Title:        "Title",
		Body:         "Body",
		DeliveryMode: DeliveryModeServerTrusted,
	}, slog.Default())
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if report.TotalSent != 1 {
		t.Fatalf("TotalSent = %d, want 1", report.TotalSent)
	}

	var count int
	if err := db.GetContext(ctx, &count, `SELECT COUNT(*) FROM notification_outbox_recipients WHERE device_id = ?`, deviceID); err != nil {
		t.Fatalf("count recipients: %v", err)
	}
	if count != 1 {
		t.Fatalf("recipient count = %d, want 1", count)
	}
}

func seedNotificationOutboxRows(t *testing.T, ctx context.Context, db *sqlx.DB, userID, topicID, deviceID string) {
	t.Helper()

	now := int64(1000)
	testutil.InsertUsers(t, ctx, db, userID)
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("exec seed query: %v", err)
		}
	}
	exec(
		`INSERT INTO topics (id, user_id, name, description, created_at, updated_at)
		 VALUES (?, ?, 'alerts', '', ?, ?)`,
		topicID,
		userID,
		now,
		now,
	)
	exec(
		`INSERT INTO devices (id, user_id, name, description, is_active, pairing_status, created_at, updated_at)
		 VALUES (?, ?, 'phone', '', 1, 'paired', ?, ?)`,
		deviceID,
		userID,
		now,
		now,
	)
}
