package billing

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type acceptedWebhookVerifier struct{}

func (acceptedWebhookVerifier) Verify([]byte, string) error {
	return nil
}

func (acceptedWebhookVerifier) Parse([]byte) (*WebhookPayload, error) {
	return nil, ErrWebhookEventIgnored
}

func TestReceiveWebhookReturnsOKForAcceptedDelivery(t *testing.T) {
	service := NewService(nil, nil, nil, acceptedWebhookVerifier{}, ServiceConfig{}, nil)
	handler := NewHandler(service, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/billing/webhooks/creem", strings.NewReader(`{"eventType":"refund.created"}`))
	rec := httptest.NewRecorder()

	handler.ReceiveWebhook(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ReceiveWebhook() status = %d, want %d", rec.Code, http.StatusOK)
	}
}
