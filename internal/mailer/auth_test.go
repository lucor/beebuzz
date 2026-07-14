package mailer

import (
	"context"
	"testing"

	"go.beebuzz.app/beebuzz/internal/config"
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

func TestSendHostedProductNotices(t *testing.T) {
	m, err := New(testConfig())
	if err != nil {
		t.Fatalf("New(): %v", err)
	}

	ctx := context.Background()
	if err := m.SendHostedActivated(ctx, "test@example.com"); err != nil {
		t.Fatalf("SendHostedActivated(): %v", err)
	}
	if err := m.SendHostedEnded(ctx, "test@example.com"); err != nil {
		t.Fatalf("SendHostedEnded(): %v", err)
	}
}

func TestSendAccountNotices(t *testing.T) {
	m, err := New(testConfig())
	if err != nil {
		t.Fatalf("New(): %v", err)
	}

	ctx := context.Background()
	if err := m.SendAccountBlocked(ctx, "test@example.com"); err != nil {
		t.Fatalf("SendAccountBlocked(): %v", err)
	}
	if err := m.SendAccountReactivated(ctx, "test@example.com"); err != nil {
		t.Fatalf("SendAccountReactivated(): %v", err)
	}
}

func testConfig() *config.Mailer {
	return &config.Mailer{
		Sender:  "test@example.com",
		ReplyTo: "reply@example.com",
		SiteURL: "https://dashboard.example.com",
	}
}
