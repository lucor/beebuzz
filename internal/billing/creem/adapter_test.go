package creem

import (
	"errors"
	"testing"

	"go.beebuzz.app/beebuzz/internal/billing"
)

func TestBillingAdapterParseRequiresConfiguredProduct(t *testing.T) {
	client, err := NewClient(Config{
		APIKey:    "creem_test_key",
		BaseURL:   "https://api.example.test",
		ProductID: "prod_hosted",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	raw := []byte(`{
		"id":"evt_123",
		"eventType":"subscription.paid",
		"created_at":1782907200000,
		"object":{
			"id":"sub_123",
			"status":"active",
			"product":{"id":"prod_hosted"},
			"metadata":{"beebuzz_user_id":"user_1"},
			"current_period_end_date":"2026-08-01T12:00:00Z"
		}
	}`)

	adapter := NewBillingAdapter(client, "secret", "prod_hosted")
	if _, err := adapter.Parse(raw); err != nil {
		t.Fatalf("Parse() matching product error = %v", err)
	}

	adapter = NewBillingAdapter(client, "secret", "prod_other")
	if _, err := adapter.Parse(raw); err == nil {
		t.Fatal("Parse() mismatched product error = nil")
	}
}

func TestNormalizeStatus(t *testing.T) {
	tests := []struct {
		eventType          string
		providerStatus     string
		wantStatus         billing.SubscriptionStatus
		wantCancelAtPeriod bool
	}{
		{
			eventType:      "subscription.paid",
			wantStatus:     billing.SubscriptionStatusActive,
			providerStatus: "active",
		},
		{
			eventType:          "subscription.scheduled_cancel",
			wantStatus:         billing.SubscriptionStatusScheduled,
			providerStatus:     "scheduled_cancel",
			wantCancelAtPeriod: true,
		},
		{
			eventType:      "subscription.past_due",
			wantStatus:     billing.SubscriptionStatusPastDue,
			providerStatus: "past_due",
		},
		{
			eventType:      "subscription.unpaid",
			wantStatus:     billing.SubscriptionStatusPastDue,
			providerStatus: "unpaid",
		},
		{
			eventType:      "subscription.expired",
			wantStatus:     billing.SubscriptionStatusPastDue,
			providerStatus: "expired",
		},
		{
			eventType:      "subscription.paused",
			wantStatus:     billing.SubscriptionStatusCanceled,
			providerStatus: "paused",
		},
		{
			eventType:      "subscription.update",
			wantStatus:     billing.SubscriptionStatusActive,
			providerStatus: "active",
		},
		{
			eventType:      "subscription.update",
			wantStatus:     billing.SubscriptionStatusPastDue,
			providerStatus: "unpaid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			status, cancelAtPeriod, err := normalizeStatus(tt.eventType, tt.providerStatus)
			if err != nil {
				t.Fatalf("normalizeStatus() error = %v", err)
			}
			if status != tt.wantStatus {
				t.Fatalf("status = %q, want %q", status, tt.wantStatus)
			}
			if cancelAtPeriod != tt.wantCancelAtPeriod {
				t.Fatalf("cancelAtPeriod = %v, want %v", cancelAtPeriod, tt.wantCancelAtPeriod)
			}
		})
	}
}

func TestBillingAdapterParseIgnoresNonEntitlementEvent(t *testing.T) {
	client, err := NewClient(Config{
		APIKey:    "creem_test_key",
		BaseURL:   "https://api.example.test",
		ProductID: "prod_hosted",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	adapter := NewBillingAdapter(client, "secret", "prod_hosted")
	_, err = adapter.Parse([]byte(`{"id":"evt_refund","eventType":"refund.created","created_at":1782907200000,"object":{}}`))
	if !errors.Is(err, billing.ErrWebhookEventIgnored) {
		t.Fatalf("Parse() error = %v, want %v", err, billing.ErrWebhookEventIgnored)
	}
}

func TestBillingAdapterParseNormalizesUnpaid(t *testing.T) {
	client, err := NewClient(Config{
		APIKey:    "creem_test_key",
		BaseURL:   "https://api.example.test",
		ProductID: "prod_hosted",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	raw := []byte(`{
		"id":"evt_unpaid",
		"eventType":"subscription.unpaid",
		"created_at":1782907200000,
		"object":{
			"id":"sub_123",
			"status":"unpaid",
			"product":{"id":"prod_hosted"},
			"metadata":{"beebuzz_user_id":"user_1"},
			"current_period_end_date":"2026-08-01T12:00:00Z"
		}
	}`)

	adapter := NewBillingAdapter(client, "secret", "prod_hosted")
	payload, err := adapter.Parse(raw)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if payload.Status != billing.SubscriptionStatusPastDue {
		t.Fatalf("status = %q, want %q", payload.Status, billing.SubscriptionStatusPastDue)
	}
}

func TestNormalizeStatusRejectsUnsupportedEvent(t *testing.T) {
	if _, _, err := normalizeStatus("refund.created", "active"); err == nil {
		t.Fatal("normalizeStatus() error = nil, want unsupported event")
	}
}
