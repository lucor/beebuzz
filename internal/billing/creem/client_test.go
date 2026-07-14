package creem

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientCreateCheckout(t *testing.T) {
	var gotRequest createCheckoutRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/checkouts" {
			t.Fatalf("path = %s, want /v1/checkouts", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "creem_test_key" {
			t.Fatal("missing x-api-key header")
		}
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"ch_123","checkout_url":"https://checkout.creem.io/ch_123"}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{
		APIKey:    "creem_test_key",
		BaseURL:   server.URL,
		ProductID: "prod_123",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	resp, err := client.CreateCheckout(context.Background(), CheckoutRequest{
		RequestID:  "req_123",
		SuccessURL: "https://dashboard.example.com/account/billing?checkout=success",
		Metadata: map[string]string{
			"beebuzz_user_id": "user_1",
		},
	})
	if err != nil {
		t.Fatalf("CreateCheckout() error = %v", err)
	}
	if resp.CheckoutURL != "https://checkout.creem.io/ch_123" {
		t.Fatalf("CheckoutURL = %q, want provider URL", resp.CheckoutURL)
	}
	if gotRequest.ProductID != "prod_123" {
		t.Fatalf("ProductID = %q, want prod_123", gotRequest.ProductID)
	}
	if gotRequest.RequestID != "req_123" {
		t.Fatalf("RequestID = %q, want req_123", gotRequest.RequestID)
	}
	if gotRequest.Metadata["beebuzz_user_id"] != "user_1" {
		t.Fatalf("metadata user ID = %q, want user_1", gotRequest.Metadata["beebuzz_user_id"])
	}
}

func TestClientCreateCheckoutRejectsMissingCheckoutURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"ch_123"}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{
		APIKey:    "creem_test_key",
		BaseURL:   server.URL,
		ProductID: "prod_123",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if _, err := client.CreateCheckout(context.Background(), CheckoutRequest{}); err == nil {
		t.Fatal("CreateCheckout() error = nil, want missing checkout_url")
	}
}

func TestClientCreateCustomerPortal(t *testing.T) {
	var gotRequest createCustomerPortalRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/customers/billing" {
			t.Fatalf("path = %s, want /v1/customers/billing", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "creem_test_key" {
			t.Fatal("missing x-api-key header")
		}
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"customer_portal_link":"https://creem.io/customer/portal"}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{
		APIKey:    "creem_test_key",
		BaseURL:   server.URL,
		ProductID: "prod_123",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	resp, err := client.CreateCustomerPortal(context.Background(), "cust_123")
	if err != nil {
		t.Fatalf("CreateCustomerPortal() error = %v", err)
	}
	if resp.PortalURL != "https://creem.io/customer/portal" {
		t.Fatalf("PortalURL = %q, want provider URL", resp.PortalURL)
	}
	if gotRequest.CustomerID != "cust_123" {
		t.Fatalf("CustomerID = %q, want cust_123", gotRequest.CustomerID)
	}
}
