package mailer

import (
	"strings"
	"testing"
)

func TestNewResendSender(t *testing.T) {
	t.Run("returns error on empty API key", func(t *testing.T) {
		sender, err := NewResendSender("", "from@example.com", "reply@example.com")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if sender != nil {
			t.Errorf("sender = %v, want nil", sender)
		}
		if !strings.Contains(err.Error(), "API key is required") {
			t.Errorf("error = %q, want to contain %q", err.Error(), "API key is required")
		}
	})

	t.Run("creates sender successfully with valid config", func(t *testing.T) {
		sender, err := NewResendSender("re_abc123xyz", "from@example.com", "reply@example.com")
		if err != nil {
			t.Fatalf("NewResendSender(): %v", err)
		}
		if sender == nil {
			t.Fatal("sender is nil")
		}
		if sender.apiKey != "re_abc123xyz" {
			t.Errorf("sender.apiKey = %q, want %q", sender.apiKey, "re_abc123xyz")
		}
		if sender.sender != "from@example.com" {
			t.Errorf("sender.sender = %q, want %q", sender.sender, "from@example.com")
		}
		if sender.replyTo != "reply@example.com" {
			t.Errorf("sender.replyTo = %q, want %q", sender.replyTo, "reply@example.com")
		}
		if sender.client == nil {
			t.Error("sender.client is nil")
		}
	})
}
