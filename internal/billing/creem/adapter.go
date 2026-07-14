package creem

import (
	"context"
	"fmt"

	"go.beebuzz.app/beebuzz/internal/billing"
)

// BillingAdapter adapts Creem checkout and webhook APIs to the billing service.
type BillingAdapter struct {
	client        *Client
	webhookSecret string
	productID     string
}

// NewBillingAdapter creates a Creem billing adapter.
func NewBillingAdapter(client *Client, webhookSecret, productID string) *BillingAdapter {
	return &BillingAdapter{client: client, webhookSecret: webhookSecret, productID: productID}
}

// CreateCheckout creates a checkout session through Creem.
func (a *BillingAdapter) CreateCheckout(ctx context.Context, req billing.CheckoutRequest) (*billing.Checkout, error) {
	resp, err := a.client.CreateCheckout(ctx, CheckoutRequest{
		RequestID:  req.RequestID,
		SuccessURL: req.SuccessURL,
		Metadata:   req.Metadata,
	})
	if err != nil {
		return nil, err
	}
	return &billing.Checkout{ID: resp.ID, CheckoutURL: resp.CheckoutURL}, nil
}

// CreateCustomerPortal creates a customer billing management link through Creem.
func (a *BillingAdapter) CreateCustomerPortal(ctx context.Context, providerCustomerID string) (*billing.CustomerPortal, error) {
	resp, err := a.client.CreateCustomerPortal(ctx, providerCustomerID)
	if err != nil {
		return nil, err
	}
	return &billing.CustomerPortal{PortalURL: resp.PortalURL}, nil
}

// Verify verifies a Creem webhook signature.
func (a *BillingAdapter) Verify(rawBody []byte, signature string) error {
	return VerifySignature(rawBody, signature, a.webhookSecret)
}

// Parse parses a Creem webhook into normalized billing data.
func (a *BillingAdapter) Parse(rawBody []byte) (*billing.WebhookPayload, error) {
	eventType, err := ParseWebhookEventType(rawBody)
	if err != nil {
		return nil, err
	}
	if !handlesEventType(eventType) {
		return nil, billing.ErrWebhookEventIgnored
	}

	event, err := ParseWebhookEvent(rawBody)
	if err != nil {
		return nil, err
	}
	if event.ProductID != a.productID {
		return nil, fmt.Errorf("unexpected creem product")
	}

	status, cancelAtPeriodEnd, err := normalizeStatus(event.EventType, event.Status)
	if err != nil {
		return nil, err
	}

	return &billing.WebhookPayload{
		EventID:           event.ID,
		EventType:         event.EventType,
		UserID:            event.UserID,
		CustomerID:        event.CustomerID,
		SubscriptionID:    event.SubscriptionID,
		Status:            status,
		CurrentPeriodEnd:  event.PeriodEndMillis,
		CancelAtPeriodEnd: cancelAtPeriodEnd,
		OccurredAt:        event.OccurredAt,
	}, nil
}

func normalizeStatus(eventType string, providerStatus string) (billing.SubscriptionStatus, bool, error) {
	switch eventType {
	case eventTypeCheckoutCompleted:
		if providerStatus == providerStatusActive {
			return billing.SubscriptionStatusActive, false, nil
		}
		return billing.SubscriptionStatusIncomplete, false, nil
	case eventTypeSubscriptionPaid:
		return billing.SubscriptionStatusActive, false, nil
	case eventTypeSubscriptionActive:
		return billing.SubscriptionStatusActive, false, nil
	case eventTypeSubscriptionScheduled:
		return billing.SubscriptionStatusScheduled, true, nil
	case eventTypeSubscriptionPastDue:
		return billing.SubscriptionStatusPastDue, false, nil
	case eventTypeSubscriptionCanceled:
		return billing.SubscriptionStatusCanceled, false, nil
	case eventTypeSubscriptionPaused:
		// BeeBuzz does not offer paused Hosted plans. A provider pause must not
		// leave a paid entitlement active.
		return billing.SubscriptionStatusCanceled, false, nil
	case eventTypeSubscriptionExpired:
		// Creem can keep retrying a payment after subscription.expired. Treat it
		// as past due so Hosted access follows the configured grace period; the
		// terminal subscription.canceled event revokes it.
		return billing.SubscriptionStatusPastDue, false, nil
	case eventTypeSubscriptionUpdate:
		return normalizeProviderStatus(providerStatus)
	}
	return "", false, fmt.Errorf("unsupported creem event type %q", eventType)
}

func handlesEventType(eventType string) bool {
	switch eventType {
	case eventTypeCheckoutCompleted,
		eventTypeSubscriptionActive,
		eventTypeSubscriptionPaid,
		eventTypeSubscriptionScheduled,
		eventTypeSubscriptionPastDue,
		eventTypeSubscriptionExpired,
		eventTypeSubscriptionCanceled,
		eventTypeSubscriptionPaused,
		eventTypeSubscriptionUpdate:
		return true
	}
	return false
}

func normalizeProviderStatus(status string) (billing.SubscriptionStatus, bool, error) {
	switch status {
	case providerStatusActive:
		return billing.SubscriptionStatusActive, false, nil
	case providerStatusScheduled:
		return billing.SubscriptionStatusScheduled, true, nil
	case providerStatusPastDue:
		return billing.SubscriptionStatusPastDue, false, nil
	case providerStatusCanceled:
		return billing.SubscriptionStatusCanceled, false, nil
	case providerStatusExpired:
		return billing.SubscriptionStatusPastDue, false, nil
	case providerStatusIncomplete:
		return billing.SubscriptionStatusIncomplete, false, nil
	}
	return "", false, fmt.Errorf("unsupported creem subscription status %q", status)
}
