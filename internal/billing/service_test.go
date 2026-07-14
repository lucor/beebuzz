package billing

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.beebuzz.app/beebuzz/internal/core"
	"go.beebuzz.app/beebuzz/internal/testutil"
)

type stubCheckoutCreator struct {
	request CheckoutRequest
}

func (c *stubCheckoutCreator) CreateCheckout(_ context.Context, req CheckoutRequest) (*Checkout, error) {
	c.request = req
	return &Checkout{ID: "ch_123", CheckoutURL: "https://checkout.example.com/ch_123"}, nil
}

type stubCustomerPortalCreator struct {
	providerCustomerID string
}

func (c *stubCustomerPortalCreator) CreateCustomerPortal(_ context.Context, providerCustomerID string) (*CustomerPortal, error) {
	c.providerCustomerID = providerCustomerID
	return &CustomerPortal{PortalURL: "https://portal.example.com/customer"}, nil
}

type stubWebhookVerifier struct {
	payload   *WebhookPayload
	verifyErr error
	parseErr  error
}

func (v *stubWebhookVerifier) Verify([]byte, string) error {
	return v.verifyErr
}

func (v *stubWebhookVerifier) Parse([]byte) (*WebhookPayload, error) {
	if v.parseErr != nil {
		return nil, v.parseErr
	}
	return v.payload, nil
}

type trackingProductNotifier struct {
	activated []string
	ended     []string
	failed    []string
}

func (n *trackingProductNotifier) NotifyHostedActivated(_ context.Context, userID string) error {
	n.activated = append(n.activated, userID)
	return nil
}

func (n *trackingProductNotifier) NotifyHostedEnded(_ context.Context, userID string) error {
	n.ended = append(n.ended, userID)
	return nil
}

func (n *trackingProductNotifier) NotifyBillingWebhookFailed(_ context.Context, eventType string) error {
	n.failed = append(n.failed, eventType)
	return nil
}

func TestServiceCreateCheckoutDoesNotSendEmail(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewDBWithUsers(t, "user_1")
	repo := NewRepository(db)
	checkoutCreator := &stubCheckoutCreator{}
	service := NewService(repo, checkoutCreator, nil, nil, ServiceConfig{
		SuccessURL: "https://dashboard.example.com/account/billing?checkout=success",
	}, nil)

	resp, err := service.CreateCheckout(ctx, "user_1")
	if err != nil {
		t.Fatalf("CreateCheckout() error = %v", err)
	}
	if resp.CheckoutURL == "" {
		t.Fatal("CheckoutURL is empty")
	}
	if checkoutCreator.request.Metadata["beebuzz_user_id"] != "user_1" {
		t.Fatalf("metadata user id = %q, want user_1", checkoutCreator.request.Metadata["beebuzz_user_id"])
	}
	customer, err := repo.GetCustomer(ctx, "user_1", ProviderCreem)
	if err != nil || customer == nil {
		t.Fatalf("GetCustomer() = %v, %v", customer, err)
	}
	if checkoutCreator.request.RequestID != customer.ID {
		t.Fatalf("request ID = %q, want stable billing customer ID %q", checkoutCreator.request.RequestID, customer.ID)
	}
	if checkoutCreator.request.SuccessURL != "https://dashboard.example.com/account/billing?checkout=success" {
		t.Fatalf("success URL = %q, want dashboard billing confirmation URL", checkoutCreator.request.SuccessURL)
	}
	if _, ok := checkoutCreator.request.Metadata["email"]; ok {
		t.Fatal("checkout metadata should not include email")
	}
}

func TestServiceCreateCheckoutRejectsUserWithHostedSubscription(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewDBWithUsers(t, "user_1")
	repo := NewRepository(db)
	periodEnd := time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	if _, err := repo.UpsertSubscription(ctx, UpsertSubscriptionParams{
		UserID:                 "user_1",
		Provider:               ProviderCreem,
		ProviderSubscriptionID: "sub_123",
		Plan:                   core.PlanHosted,
		Status:                 SubscriptionStatusActive,
		CurrentPeriodEnd:       &periodEnd,
	}); err != nil {
		t.Fatalf("UpsertSubscription() error = %v", err)
	}

	checkoutCreator := &stubCheckoutCreator{}
	service := NewService(repo, checkoutCreator, nil, nil, ServiceConfig{}, nil)
	if _, err := service.CreateCheckout(ctx, "user_1"); !errors.Is(err, ErrHostedSubscriptionActive) {
		t.Fatalf("CreateCheckout() error = %v, want %v", err, ErrHostedSubscriptionActive)
	}
	if checkoutCreator.request.RequestID != "" {
		t.Fatalf("checkout request ID = %q, want no provider checkout", checkoutCreator.request.RequestID)
	}
}

func TestServiceCreateCustomerPortal(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewDBWithUsers(t, "user_1")
	repo := NewRepository(db)
	periodEnd := time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	providerCustomerID := "cust_123"
	if _, err := repo.UpsertSubscription(ctx, UpsertSubscriptionParams{
		UserID:                 "user_1",
		Provider:               ProviderCreem,
		ProviderCustomerID:     &providerCustomerID,
		ProviderSubscriptionID: "sub_123",
		Plan:                   core.PlanHosted,
		Status:                 SubscriptionStatusActive,
		CurrentPeriodEnd:       &periodEnd,
	}); err != nil {
		t.Fatalf("UpsertSubscription() error = %v", err)
	}

	portalCreator := &stubCustomerPortalCreator{}
	service := NewService(repo, nil, portalCreator, nil, ServiceConfig{}, nil)

	resp, err := service.CreateCustomerPortal(ctx, "user_1")
	if err != nil {
		t.Fatalf("CreateCustomerPortal() error = %v", err)
	}
	if resp.PortalURL != "https://portal.example.com/customer" {
		t.Fatalf("PortalURL = %q, want provider portal URL", resp.PortalURL)
	}
	if portalCreator.providerCustomerID != providerCustomerID {
		t.Fatalf("providerCustomerID = %q, want %q", portalCreator.providerCustomerID, providerCustomerID)
	}
}

func TestServiceCreateCustomerPortalRequiresProviderCustomer(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewDBWithUsers(t, "user_1")
	repo := NewRepository(db)
	service := NewService(repo, nil, &stubCustomerPortalCreator{}, nil, ServiceConfig{}, nil)

	if _, err := service.CreateCustomerPortal(ctx, "user_1"); !errors.Is(err, ErrBillingCustomerMissing) {
		t.Fatalf("CreateCustomerPortal() error = %v, want %v", err, ErrBillingCustomerMissing)
	}
}

func TestServiceHandleWebhookAppliesSubscription(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewDBWithUsers(t, "user_1")
	repo := NewRepository(db)
	periodEnd := time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	customerID := "cust_123"
	service := NewService(repo, nil, nil, &stubWebhookVerifier{
		payload: &WebhookPayload{
			EventID:          "evt_123",
			EventType:        "subscription.paid",
			UserID:           "user_1",
			CustomerID:       &customerID,
			SubscriptionID:   "sub_123",
			Status:           SubscriptionStatusActive,
			CurrentPeriodEnd: &periodEnd,
			OccurredAt:       1782907200000,
		},
	}, ServiceConfig{GracePeriodDays: 7}, nil)

	if err := service.HandleWebhook(ctx, []byte(`{"id":"evt_123"}`), "sig"); err != nil {
		t.Fatalf("HandleWebhook() error = %v", err)
	}

	var userPlan core.Plan
	if err := db.QueryRowContext(ctx, `SELECT plan FROM users WHERE id = ?`, "user_1").Scan(&userPlan); err != nil {
		t.Fatalf("read user plan: %v", err)
	}
	if userPlan != core.PlanHosted {
		t.Fatalf("user plan = %q, want hosted", userPlan)
	}
}

func TestServiceHandleWebhookNotifiesHostedActivationOnce(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewDBWithUsers(t, "user_1")
	repo := NewRepository(db)
	periodEnd := time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	customerID := "cust_123"
	notifier := &trackingProductNotifier{}
	service := NewService(repo, nil, nil, &stubWebhookVerifier{
		payload: &WebhookPayload{
			EventID:          "evt_activated",
			EventType:        "subscription.paid",
			UserID:           "user_1",
			CustomerID:       &customerID,
			SubscriptionID:   "sub_activated",
			Status:           SubscriptionStatusActive,
			CurrentPeriodEnd: &periodEnd,
			OccurredAt:       1782907200000,
		},
	}, ServiceConfig{GracePeriodDays: 7}, nil)
	service.SetProductNotifier(notifier)

	if err := service.HandleWebhook(ctx, []byte(`{"id":"evt_activated"}`), "sig"); err != nil {
		t.Fatalf("HandleWebhook() error = %v", err)
	}
	if err := service.HandleWebhook(ctx, []byte(`{"id":"evt_activated"}`), "sig"); err != nil {
		t.Fatalf("HandleWebhook() duplicate error = %v", err)
	}

	if len(notifier.activated) != 1 || notifier.activated[0] != "user_1" {
		t.Fatalf("activated notices = %#v, want [user_1]", notifier.activated)
	}
	if len(notifier.ended) != 0 {
		t.Fatalf("ended notices = %#v, want none", notifier.ended)
	}
}

func TestServiceHandleWebhookNotifiesHostedEnded(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewDBWithUsers(t, "user_1")
	repo := NewRepository(db)
	periodEnd := time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	customerID := "cust_123"
	if _, err := repo.UpsertSubscription(ctx, UpsertSubscriptionParams{
		UserID:                 "user_1",
		Provider:               ProviderCreem,
		ProviderCustomerID:     &customerID,
		ProviderSubscriptionID: "sub_ended",
		Plan:                   core.PlanHosted,
		Status:                 SubscriptionStatusActive,
		CurrentPeriodEnd:       &periodEnd,
	}); err != nil {
		t.Fatalf("UpsertSubscription() error = %v", err)
	}

	notifier := &trackingProductNotifier{}
	service := NewService(repo, nil, nil, &stubWebhookVerifier{
		payload: &WebhookPayload{
			EventID:          "evt_ended",
			EventType:        "subscription.canceled",
			UserID:           "user_1",
			CustomerID:       &customerID,
			SubscriptionID:   "sub_ended",
			Status:           SubscriptionStatusCanceled,
			CurrentPeriodEnd: &periodEnd,
			OccurredAt:       periodEnd + 1,
		},
	}, ServiceConfig{GracePeriodDays: 7}, nil)
	service.SetProductNotifier(notifier)

	if err := service.HandleWebhook(ctx, []byte(`{"id":"evt_ended"}`), "sig"); err != nil {
		t.Fatalf("HandleWebhook() error = %v", err)
	}

	if len(notifier.ended) != 1 || notifier.ended[0] != "user_1" {
		t.Fatalf("ended notices = %#v, want [user_1]", notifier.ended)
	}
	if len(notifier.activated) != 0 {
		t.Fatalf("activated notices = %#v, want none", notifier.activated)
	}
}

func TestServiceHandleWebhookExpiredKeepsHostedDuringGracePeriod(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewDBWithUsers(t, "user_1")
	repo := NewRepository(db)
	periodEnd := time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	customerID := "cust_123"
	notifier := &trackingProductNotifier{}
	service := NewService(repo, nil, nil, &stubWebhookVerifier{
		payload: &WebhookPayload{
			EventID:          "evt_expired",
			EventType:        "subscription.expired",
			UserID:           "user_1",
			CustomerID:       &customerID,
			SubscriptionID:   "sub_expired",
			Status:           SubscriptionStatusPastDue,
			CurrentPeriodEnd: &periodEnd,
			OccurredAt:       1782907200000,
		},
	}, ServiceConfig{GracePeriodDays: 7}, nil)
	service.SetProductNotifier(notifier)

	if err := service.HandleWebhook(ctx, []byte(`{"id":"evt_expired"}`), "sig"); err != nil {
		t.Fatalf("HandleWebhook() error = %v", err)
	}

	var (
		plan          core.Plan
		planExpiresAt int64
		status        SubscriptionStatus
	)
	if err := db.QueryRowContext(ctx,
		`SELECT users.plan, users.plan_expires_at, billing_subscriptions.status
		 FROM users
		 JOIN billing_subscriptions ON billing_subscriptions.user_id = users.id
		 WHERE users.id = ?`, "user_1",
	).Scan(&plan, &planExpiresAt, &status); err != nil {
		t.Fatalf("read subscription state: %v", err)
	}
	if plan != core.PlanHosted {
		t.Fatalf("plan = %q, want hosted", plan)
	}
	if planExpiresAt != periodEnd+7*24*time.Hour.Milliseconds() {
		t.Fatalf("plan expires at = %d, want grace period after %d", planExpiresAt, periodEnd)
	}
	if status != SubscriptionStatusPastDue {
		t.Fatalf("status = %q, want past_due", status)
	}
	if len(notifier.ended) != 0 {
		t.Fatalf("ended notices = %#v, want none", notifier.ended)
	}
}

func TestServiceHandleWebhookSignatureFailureDoesNotNotify(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewDBWithUsers(t, "user_1")
	repo := NewRepository(db)
	notifier := &trackingProductNotifier{}
	service := NewService(repo, nil, nil, &stubWebhookVerifier{verifyErr: errors.New("bad signature")}, ServiceConfig{}, nil)
	service.SetProductNotifier(notifier)

	if err := service.HandleWebhook(ctx, []byte(`{"id":"evt_failed"}`), "sig"); !errors.Is(err, ErrInvalidWebhook) {
		t.Fatalf("HandleWebhook() error = %v, want %v", err, ErrInvalidWebhook)
	}

	// The webhook endpoint is public: unsigned/probe requests must not be able
	// to spam admin notifications.
	if len(notifier.failed) != 0 {
		t.Fatalf("failed notices = %#v, want none on signature failure", notifier.failed)
	}
}

func TestServiceHandleWebhookIgnoresVerifiedNonEntitlementEvent(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewDBWithUsers(t, "user_1")
	repo := NewRepository(db)
	notifier := &trackingProductNotifier{}
	service := NewService(repo, nil, nil, &stubWebhookVerifier{parseErr: ErrWebhookEventIgnored}, ServiceConfig{}, nil)
	service.SetProductNotifier(notifier)

	if err := service.HandleWebhook(ctx, []byte(`{"id":"evt_refund"}`), "sig"); err != nil {
		t.Fatalf("HandleWebhook() error = %v", err)
	}
	if len(notifier.failed) != 0 {
		t.Fatalf("failed notices = %#v, want none", notifier.failed)
	}
}

func TestServiceHandleWebhookNotifiesFailureAfterVerification(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewDBWithUsers(t, "user_1")
	repo := NewRepository(db)
	notifier := &trackingProductNotifier{}
	service := NewService(repo, nil, nil, &stubWebhookVerifier{parseErr: errors.New("bad payload")}, ServiceConfig{}, nil)
	service.SetProductNotifier(notifier)

	if err := service.HandleWebhook(ctx, []byte(`{"id":"evt_failed"}`), "sig"); !errors.Is(err, ErrInvalidWebhook) {
		t.Fatalf("HandleWebhook() error = %v, want %v", err, ErrInvalidWebhook)
	}

	// A payload that passed signature verification is a genuine provider
	// message, so a processing failure should notify admins.
	if len(notifier.failed) != 1 {
		t.Fatalf("failed notices = %#v, want one", notifier.failed)
	}
}

func TestServiceDisabled(t *testing.T) {
	service := NewService(nil, nil, nil, nil, ServiceConfig{}, nil)

	if _, err := service.CreateCheckout(context.Background(), "user_1"); !errors.Is(err, ErrBillingDisabled) {
		t.Fatalf("CreateCheckout() error = %v, want %v", err, ErrBillingDisabled)
	}
	if _, err := service.CreateCustomerPortal(context.Background(), "user_1"); !errors.Is(err, ErrBillingDisabled) {
		t.Fatalf("CreateCustomerPortal() error = %v, want %v", err, ErrBillingDisabled)
	}
	if err := service.HandleWebhook(context.Background(), nil, ""); !errors.Is(err, ErrBillingDisabled) {
		t.Fatalf("HandleWebhook() error = %v, want %v", err, ErrBillingDisabled)
	}
}
