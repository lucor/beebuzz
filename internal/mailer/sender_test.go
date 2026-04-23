package mailer

import (
	"testing"

	"lucor.dev/beebuzz/internal/config"
)

func TestNewTransport(t *testing.T) {
	t.Run("creates SMTP transport when address is set", func(t *testing.T) {
		cfg := &config.Mailer{
			SMTPAddress:  "smtp.example.com:587",
			SMTPUser:     "user",
			SMTPPassword: "pass",
			Sender:       "from@example.com",
			ReplyTo:      "reply@example.com",
		}

		transport, err := newTransport(cfg)
		if err != nil {
			t.Fatalf("newTransport(): %v", err)
		}

		_, ok := transport.(*SMTPSender)
		if !ok {
			t.Errorf("expected SMTPSender, got %T", transport)
		}
	})

	t.Run("creates Resend transport when API key is set and no SMTP", func(t *testing.T) {
		cfg := &config.Mailer{
			SMTPAddress:  "",
			ResendAPIKey: "re_abc123xyz",
			Sender:       "from@example.com",
			ReplyTo:      "reply@example.com",
		}

		transport, err := newTransport(cfg)
		if err != nil {
			t.Fatalf("newTransport(): %v", err)
		}

		_, ok := transport.(*ResendSender)
		if !ok {
			t.Errorf("expected ResendSender, got %T", transport)
		}
	})

	t.Run("creates NoOp transport when neither SMTP nor Resend is configured", func(t *testing.T) {
		cfg := &config.Mailer{
			SMTPAddress:  "",
			ResendAPIKey: "",
			Sender:       "from@example.com",
			ReplyTo:      "reply@example.com",
		}

		transport, err := newTransport(cfg)
		if err != nil {
			t.Fatalf("newTransport(): %v", err)
		}

		_, ok := transport.(*NoOpTransport)
		if !ok {
			t.Errorf("expected NoOpTransport, got %T", transport)
		}
	})

	t.Run("prioritizes SMTP over Resend", func(t *testing.T) {
		cfg := &config.Mailer{
			SMTPAddress:  "smtp.example.com:587",
			SMTPUser:     "user",
			SMTPPassword: "pass",
			ResendAPIKey: "re_abc123xyz",
			Sender:       "from@example.com",
			ReplyTo:      "reply@example.com",
		}

		transport, err := newTransport(cfg)
		if err != nil {
			t.Fatalf("newTransport(): %v", err)
		}

		_, ok := transport.(*SMTPSender)
		if !ok {
			t.Error("expected SMTP to be prioritized over Resend")
		}
	})
}
