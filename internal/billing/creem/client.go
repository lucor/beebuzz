// Package creem adapts Creem's Merchant-of-Record API to BeeBuzz billing.
package creem

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const clientTimeout = 10 * time.Second

// Client creates Creem checkout sessions.
type Client struct {
	apiKey    string
	baseURL   string
	productID string
	client    *http.Client
}

// Config holds Creem adapter configuration.
type Config struct {
	APIKey    string
	BaseURL   string
	ProductID string
}

// NewClient creates a Creem API client.
func NewClient(cfg Config) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("creem API key is required")
	}
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("creem API base URL is required")
	}
	if cfg.ProductID == "" {
		return nil, fmt.Errorf("creem product ID is required")
	}
	return &Client{
		apiKey:    cfg.APIKey,
		baseURL:   strings.TrimRight(cfg.BaseURL, "/"),
		productID: cfg.ProductID,
		client:    &http.Client{Timeout: clientTimeout},
	}, nil
}

// CheckoutRequest describes the provider-neutral checkout inputs BeeBuzz needs.
type CheckoutRequest struct {
	RequestID  string
	SuccessURL string
	Metadata   map[string]string
}

// CheckoutResponse is the Creem checkout URL returned to the dashboard.
type CheckoutResponse struct {
	ID          string
	CheckoutURL string
}

// CustomerPortalResponse is the Creem customer portal link returned to the dashboard.
type CustomerPortalResponse struct {
	PortalURL string
}

type createCheckoutRequest struct {
	ProductID  string            `json:"product_id"`
	RequestID  string            `json:"request_id,omitempty"`
	SuccessURL string            `json:"success_url,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type createCheckoutResponse struct {
	ID          string `json:"id"`
	CheckoutURL string `json:"checkout_url"`
}

type createCustomerPortalRequest struct {
	CustomerID string `json:"customer_id"`
}

type createCustomerPortalResponse struct {
	CustomerPortalLink string `json:"customer_portal_link"`
}

type apiError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// CreateCustomerPortal creates a Creem customer billing management link.
func (c *Client) CreateCustomerPortal(ctx context.Context, providerCustomerID string) (*CustomerPortalResponse, error) {
	if providerCustomerID == "" {
		return nil, fmt.Errorf("creem customer ID is required")
	}

	body, err := json.Marshal(createCustomerPortalRequest{CustomerID: providerCustomerID})
	if err != nil {
		return nil, fmt.Errorf("marshal creem customer portal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/customers/billing", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create creem customer portal request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("creem customer portal request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read creem customer portal response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr apiError
		if err := json.Unmarshal(respBody, &apiErr); err == nil {
			if apiErr.Message != "" {
				return nil, fmt.Errorf("creem customer portal API error %d: %s", resp.StatusCode, apiErr.Message)
			}
			if apiErr.Error != "" {
				return nil, fmt.Errorf("creem customer portal API error %d: %s", resp.StatusCode, apiErr.Error)
			}
		}
		return nil, fmt.Errorf("creem customer portal API error %d", resp.StatusCode)
	}

	var portal createCustomerPortalResponse
	if err := json.Unmarshal(respBody, &portal); err != nil {
		return nil, fmt.Errorf("parse creem customer portal response: %w", err)
	}
	if portal.CustomerPortalLink == "" {
		return nil, fmt.Errorf("creem customer portal response missing customer_portal_link")
	}
	return &CustomerPortalResponse{PortalURL: portal.CustomerPortalLink}, nil
}

// CreateCheckout creates a Creem checkout session without sending BeeBuzz email data.
func (c *Client) CreateCheckout(ctx context.Context, req CheckoutRequest) (*CheckoutResponse, error) {
	body, err := json.Marshal(createCheckoutRequest{
		ProductID:  c.productID,
		RequestID:  req.RequestID,
		SuccessURL: req.SuccessURL,
		Metadata:   req.Metadata,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal creem checkout request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/checkouts", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create creem checkout request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("creem checkout request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read creem checkout response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr apiError
		if err := json.Unmarshal(respBody, &apiErr); err == nil {
			if apiErr.Message != "" {
				return nil, fmt.Errorf("creem checkout API error %d: %s", resp.StatusCode, apiErr.Message)
			}
			if apiErr.Error != "" {
				return nil, fmt.Errorf("creem checkout API error %d: %s", resp.StatusCode, apiErr.Error)
			}
		}
		return nil, fmt.Errorf("creem checkout API error %d", resp.StatusCode)
	}

	var checkout createCheckoutResponse
	if err := json.Unmarshal(respBody, &checkout); err != nil {
		return nil, fmt.Errorf("parse creem checkout response: %w", err)
	}
	if checkout.CheckoutURL == "" {
		return nil, fmt.Errorf("creem checkout response missing checkout_url")
	}
	return &CheckoutResponse{
		ID:          checkout.ID,
		CheckoutURL: checkout.CheckoutURL,
	}, nil
}
