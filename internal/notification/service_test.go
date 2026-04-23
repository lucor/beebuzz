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
