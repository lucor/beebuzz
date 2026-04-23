package mailer

import (
	"strings"
	"testing"
)

func TestNewSMTPSender(t *testing.T) {
	t.Run("returns error on empty address", func(t *testing.T) {
		_, err := NewSMTPSender("", "user", "pass", "from@example.com", "reply@example.com")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "address is required") {
			t.Errorf("error = %q, want to contain %q", err.Error(), "address is required")
		}
	})

	t.Run("returns error on invalid address format", func(t *testing.T) {
		_, err := NewSMTPSender("smtp.example.com", "user", "pass", "from@example.com", "reply@example.com")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "invalid address") {
			t.Errorf("error = %q, want to contain %q", err.Error(), "invalid address")
		}
	})

	t.Run("creates sender successfully with valid config", func(t *testing.T) {
		sender, err := NewSMTPSender("smtp.example.com:587", "user", "pass", "from@example.com", "reply@example.com")
		if err != nil {
			t.Fatalf("NewSMTPSender(): %v", err)
		}
		if sender.address != "smtp.example.com:587" {
			t.Errorf("sender.address = %q, want %q", sender.address, "smtp.example.com:587")
		}
		if sender.user != "user" {
			t.Errorf("sender.user = %q, want %q", sender.user, "user")
		}
		if sender.password != "pass" {
			t.Errorf("sender.password = %q, want %q", sender.password, "pass")
		}
		if sender.sender != "from@example.com" {
			t.Errorf("sender.sender = %q, want %q", sender.sender, "from@example.com")
		}
		if sender.replyTo != "reply@example.com" {
			t.Errorf("sender.replyTo = %q, want %q", sender.replyTo, "reply@example.com")
		}
	})
}

func TestSmtpMessage(t *testing.T) {
	boundary := "testboundary"

	newTestMessage := func() smtpMessage {
		return smtpMessage{
			from:     "from@example.com",
			to:       "to@example.com",
			replyTo:  "reply@example.com",
			subject:  "Test Subject",
			text:     "Text body",
			html:     "<p>HTML body</p>",
			boundary: boundary,
		}
	}

	t.Run("message contains required headers", func(t *testing.T) {
		msg := newTestMessage().String()

		if !strings.Contains(msg, "From: from@example.com") {
			t.Error("message should contain 'From: from@example.com'")
		}
		if !strings.Contains(msg, "To: to@example.com") {
			t.Error("message should contain 'To: to@example.com'")
		}
		if !strings.Contains(msg, "Reply-To: reply@example.com") {
			t.Error("message should contain 'Reply-To: reply@example.com'")
		}
		if !strings.Contains(msg, "Subject: Test Subject") {
			t.Error("message should contain 'Subject: Test Subject'")
		}
	})

	t.Run("message has multipart/alternative content type", func(t *testing.T) {
		msg := newTestMessage().String()

		if !strings.Contains(msg, "Content-Type: multipart/alternative; boundary="+boundary) {
			t.Error("message should contain multipart/alternative content type")
		}
	})

	t.Run("message contains boundary markers", func(t *testing.T) {
		msg := newTestMessage().String()

		if !strings.Contains(msg, "--"+boundary) {
			t.Error("message should contain boundary markers")
		}
		count := strings.Count(msg, "--"+boundary)
		if count != 3 {
			t.Errorf("count = %d, want 3", count)
		}
	})

	t.Run("message contains text part with correct content type", func(t *testing.T) {
		msg := newTestMessage().String()

		if !strings.Contains(msg, "Content-Type: text/plain; charset=UTF-8") {
			t.Error("message should contain text/plain content type")
		}
		if !strings.Contains(msg, "Text body") {
			t.Error("message should contain 'Text body'")
		}
	})

	t.Run("message contains html part with correct content type", func(t *testing.T) {
		msg := newTestMessage().String()

		if !strings.Contains(msg, "Content-Type: text/html; charset=UTF-8") {
			t.Error("message should contain text/html content type")
		}
		if !strings.Contains(msg, "<p>HTML body</p>") {
			t.Error("message should contain '<p>HTML body</p>'")
		}
	})

	t.Run("text part appears before html part", func(t *testing.T) {
		msg := newTestMessage().String()

		textIdx := strings.Index(msg, "Text body")
		htmlIdx := strings.Index(msg, "<p>HTML body</p>")

		if textIdx == -1 {
			t.Error("text body not found")
		}
		if htmlIdx == -1 {
			t.Error("html body not found")
		}
		if textIdx >= htmlIdx {
			t.Error("text part should appear before html part")
		}
	})

	t.Run("message ends with closing boundary", func(t *testing.T) {
		msg := newTestMessage().String()

		if !strings.HasSuffix(msg, "--"+boundary+"--\r\n") {
			t.Error("message should end with closing boundary")
		}
	})

	t.Run("omits Reply-To header when empty", func(t *testing.T) {
		m := newTestMessage()
		m.replyTo = ""
		msg := m.String()

		if strings.Contains(msg, "Reply-To:") {
			t.Error("message should not contain 'Reply-To:' when empty")
		}
	})
}
