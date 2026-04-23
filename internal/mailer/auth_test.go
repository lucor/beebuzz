package mailer

import (
	"context"
	"testing"

	"lucor.dev/beebuzz/internal/config"
)

func TestLookupTemplates(t *testing.T) {
	t.Run("sends auth request email successfully", func(t *testing.T) {
		m, err := New(testConfig())
		if err != nil {
			t.Fatalf("New(): %v", err)
		}

		ctx := context.Background()
		err = m.SendRequestAuth(ctx, "test@example.com", "123456")
		if err != nil {
			t.Fatalf("SendRequestAuth(): %v", err)
		}
	})
}

func TestSendRequestAuth(t *testing.T) {
	t.Run("renders templates with otp", func(t *testing.T) {
		m, err := New(testConfig())
		if err != nil {
			t.Fatalf("New(): %v", err)
		}

		ctx := context.Background()
		otp := "123456"

		err = m.SendRequestAuth(ctx, "test@example.com", otp)
		if err != nil {
			t.Fatalf("SendRequestAuth(): %v", err)
		}
	})
}

func testConfig() *config.Mailer {
	return &config.Mailer{
		Sender:  "test@example.com",
		ReplyTo: "reply@example.com",
	}
}
