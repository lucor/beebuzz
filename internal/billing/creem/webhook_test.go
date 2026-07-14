package creem

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"
)

func TestVerifySignature(t *testing.T) {
	body := []byte(`{"id":"evt_123"}`)
	secret := "secret"
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	signature := hex.EncodeToString(mac.Sum(nil))

	if err := VerifySignature(body, signature, secret); err != nil {
		t.Fatalf("VerifySignature() error = %v", err)
	}
	if err := VerifySignature(body, "bad", secret); err == nil {
		t.Fatal("VerifySignature() error = nil, want mismatch")
	}
}

func TestParseWebhookEventSubscription(t *testing.T) {
	raw := []byte(`{
		"id": "evt_123",
		"eventType": "subscription.paid",
		"created_at": 1782907200000,
		"object": {
			"id": "sub_123",
			"status": "active",
			"product": {"id": "prod_hosted"},
			"customer": "cust_123",
			"current_period_end_date": "2026-08-01T12:00:00Z",
			"metadata": {"beebuzz_user_id": "user_1"}
		}
	}`)

	event, err := ParseWebhookEvent(raw)
	if err != nil {
		t.Fatalf("ParseWebhookEvent() error = %v", err)
	}
	if event.ID != "evt_123" {
		t.Fatalf("ID = %q, want evt_123", event.ID)
	}
	if event.OccurredAt != 1782907200000 {
		t.Fatalf("OccurredAt = %d, want webhook created_at", event.OccurredAt)
	}
	if event.UserID != "user_1" {
		t.Fatalf("UserID = %q, want user_1", event.UserID)
	}
	if event.ProductID != "prod_hosted" {
		t.Fatalf("ProductID = %q, want prod_hosted", event.ProductID)
	}
	if event.CustomerID == nil || *event.CustomerID != "cust_123" {
		t.Fatalf("CustomerID = %v, want cust_123", event.CustomerID)
	}
	if event.SubscriptionID != "sub_123" {
		t.Fatalf("SubscriptionID = %q, want sub_123", event.SubscriptionID)
	}
	wantPeriodEnd := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC).UnixMilli()
	if event.PeriodEndMillis == nil || *event.PeriodEndMillis != wantPeriodEnd {
		t.Fatalf("PeriodEndMillis = %v, want %d", event.PeriodEndMillis, wantPeriodEnd)
	}
}

func TestParseWebhookEventCheckout(t *testing.T) {
	raw := []byte(`{
		"id": "evt_123",
		"eventType": "checkout.completed",
		"created_at": 1782907200000,
		"object": {
			"customer": {"id": "cust_123"},
			"product": {"id": "prod_hosted"},
			"subscription": {
				"id": "sub_123",
				"status": "active",
				"current_period_end_date": "2026-08-01T12:00:00Z"
			},
			"metadata": {"beebuzz_user_id": "user_1"}
		}
	}`)

	event, err := ParseWebhookEvent(raw)
	if err != nil {
		t.Fatalf("ParseWebhookEvent() error = %v", err)
	}
	if event.EventType != "checkout.completed" {
		t.Fatalf("EventType = %q, want checkout.completed", event.EventType)
	}
	if event.UserID != "user_1" {
		t.Fatalf("UserID = %q, want user_1", event.UserID)
	}
	if event.SubscriptionID != "sub_123" {
		t.Fatalf("SubscriptionID = %q, want sub_123", event.SubscriptionID)
	}
	wantPeriodEnd := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC).UnixMilli()
	if event.PeriodEndMillis == nil || *event.PeriodEndMillis != wantPeriodEnd {
		t.Fatalf("PeriodEndMillis = %v, want %d", event.PeriodEndMillis, wantPeriodEnd)
	}
}

func TestParseWebhookEventRequiresMetadataUserID(t *testing.T) {
	raw := []byte(`{
		"id": "evt_123",
		"eventType": "subscription.paid",
		"created_at": 1782907200000,
		"object": {
			"id": "sub_123",
			"status": "active",
			"product": {"id": "prod_hosted"},
			"metadata": {}
		}
	}`)

	if _, err := ParseWebhookEvent(raw); err == nil {
		t.Fatal("ParseWebhookEvent() error = nil, want missing metadata")
	}
}

func TestParseWebhookEventRequiresCheckoutSubscriptionID(t *testing.T) {
	raw := []byte(`{
		"id": "evt_123",
		"eventType": "checkout.completed",
		"created_at": 1782907200000,
		"object": {
			"customer": {"id": "cust_123"},
			"product": {"id": "prod_hosted"},
			"subscription": {
				"status": "active"
			},
			"metadata": {"beebuzz_user_id": "user_1"}
		}
	}`)

	if _, err := ParseWebhookEvent(raw); err == nil {
		t.Fatal("ParseWebhookEvent() error = nil, want missing subscription ID")
	}
}

func TestParseWebhookEventRequiresProductID(t *testing.T) {
	raw := []byte(`{
		"id": "evt_123",
		"eventType": "subscription.paid",
		"created_at": 1782907200000,
		"object": {
			"id": "sub_123",
			"status": "active",
			"metadata": {"beebuzz_user_id": "user_1"}
		}
	}`)

	if _, err := ParseWebhookEvent(raw); err == nil {
		t.Fatal("ParseWebhookEvent() error = nil, want missing product ID")
	}
}
