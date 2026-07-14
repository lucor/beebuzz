package billing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"

	"go.beebuzz.app/beebuzz/internal/core"
)

var (
	ErrBillingDisabled          = errors.New("billing is disabled")
	ErrBillingCustomerMissing   = errors.New("billing customer is missing")
	ErrHostedSubscriptionActive = errors.New("hosted subscription is already active")
	ErrInvalidWebhook           = errors.New("invalid billing webhook")
	ErrWebhookEventIgnored      = errors.New("billing webhook event ignored")
)

// CheckoutCreator creates hosted checkout sessions with the configured provider.
type CheckoutCreator interface {
	CreateCheckout(ctx context.Context, req CheckoutRequest) (*Checkout, error)
}

// CustomerPortalCreator creates customer billing management links.
type CustomerPortalCreator interface {
	CreateCustomerPortal(ctx context.Context, providerCustomerID string) (*CustomerPortal, error)
}

// CheckoutRequest carries provider-neutral checkout inputs.
type CheckoutRequest struct {
	RequestID  string
	SuccessURL string
	Metadata   map[string]string
}

// Checkout is a provider checkout session.
type Checkout struct {
	ID          string
	CheckoutURL string
}

// CustomerPortal is a provider billing management session.
type CustomerPortal struct {
	PortalURL string
}

// WebhookVerifier verifies and parses provider webhook payloads.
type WebhookVerifier interface {
	Verify(rawBody []byte, signature string) error
	Parse(rawBody []byte) (*WebhookPayload, error)
}

// ProductNotifier sends BeeBuzz product notices for billing-derived entitlement changes.
type ProductNotifier interface {
	NotifyHostedActivated(ctx context.Context, userID string) error
	NotifyHostedEnded(ctx context.Context, userID string) error
	NotifyBillingWebhookFailed(ctx context.Context, eventType string) error
}

// WebhookPayload is normalized provider webhook data.
type WebhookPayload struct {
	EventID           string
	EventType         string
	UserID            string
	CustomerID        *string
	SubscriptionID    string
	Status            SubscriptionStatus
	CurrentPeriodEnd  *int64
	CancelAtPeriodEnd bool
	OccurredAt        int64
}

// Service owns provider-neutral billing workflows.
type Service struct {
	repo                  *Repository
	checkoutCreator       CheckoutCreator
	customerPortalCreator CustomerPortalCreator
	webhookVerifier       WebhookVerifier
	productNotifier       ProductNotifier
	successURL            string
	gracePeriodDays       int
	log                   *slog.Logger
}

// ServiceConfig configures billing workflows.
type ServiceConfig struct {
	SuccessURL      string
	GracePeriodDays int
}

// NewService creates a billing service.
func NewService(repo *Repository, checkoutCreator CheckoutCreator, customerPortalCreator CustomerPortalCreator, webhookVerifier WebhookVerifier, cfg ServiceConfig, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{
		repo:                  repo,
		checkoutCreator:       checkoutCreator,
		customerPortalCreator: customerPortalCreator,
		webhookVerifier:       webhookVerifier,
		successURL:            cfg.SuccessURL,
		gracePeriodDays:       cfg.GracePeriodDays,
		log:                   log,
	}
}

// SetProductNotifier configures best-effort product notices for billing transitions.
func (s *Service) SetProductNotifier(notifier ProductNotifier) {
	s.productNotifier = notifier
}

// CreateCheckout starts a Hosted checkout without sending BeeBuzz email data.
func (s *Service) CreateCheckout(ctx context.Context, userID string) (*CheckoutResponse, error) {
	if s == nil || s.checkoutCreator == nil {
		return nil, ErrBillingDisabled
	}
	active, err := s.repo.HasHostedSubscription(ctx, userID)
	if err != nil {
		return nil, err
	}
	if active {
		return nil, ErrHostedSubscriptionActive
	}

	customer, err := s.repo.EnsureCustomer(ctx, userID, ProviderCreem)
	if err != nil {
		return nil, err
	}

	checkout, err := s.checkoutCreator.CreateCheckout(ctx, CheckoutRequest{
		RequestID:  customer.ID,
		SuccessURL: s.successURL,
		Metadata: map[string]string{
			"beebuzz_user_id": userID,
		},
	})
	if err != nil {
		return nil, err
	}
	return &CheckoutResponse{CheckoutURL: checkout.CheckoutURL}, nil
}

// CreateCustomerPortal creates a provider billing management link for the current user.
func (s *Service) CreateCustomerPortal(ctx context.Context, userID string) (*CustomerPortalResponse, error) {
	if s == nil || s.customerPortalCreator == nil {
		return nil, ErrBillingDisabled
	}

	customer, err := s.repo.GetCustomer(ctx, userID, ProviderCreem)
	if err != nil {
		return nil, err
	}
	if customer == nil || customer.ProviderCustomerID == nil || *customer.ProviderCustomerID == "" {
		return nil, ErrBillingCustomerMissing
	}

	portal, err := s.customerPortalCreator.CreateCustomerPortal(ctx, *customer.ProviderCustomerID)
	if err != nil {
		return nil, err
	}
	return &CustomerPortalResponse{PortalURL: portal.PortalURL}, nil
}

// HandleWebhook verifies and applies a provider webhook.
func (s *Service) HandleWebhook(ctx context.Context, rawBody []byte, signature string) error {
	if s == nil || s.webhookVerifier == nil {
		return ErrBillingDisabled
	}
	if err := s.webhookVerifier.Verify(rawBody, signature); err != nil {
		// Do not notify admins on signature failures: the webhook endpoint is
		// public, so unsigned or probe requests would otherwise let anyone
		// spam admin notifications. Only failures past verification (below)
		// come from a genuine provider payload and warrant a notice.
		return fmt.Errorf("%w: %v", ErrInvalidWebhook, err)
	}

	payload, err := s.webhookVerifier.Parse(rawBody)
	if errors.Is(err, ErrWebhookEventIgnored) {
		s.log.Info("ignored billing webhook event")
		return nil
	}
	if err != nil {
		s.notifyBillingWebhookFailed(ctx, "")
		return fmt.Errorf("%w: %v", ErrInvalidWebhook, err)
	}

	previousPlan, err := s.repo.GetUserPlan(ctx, payload.UserID)
	if err != nil {
		s.notifyBillingWebhookFailed(ctx, payload.EventType)
		return err
	}

	_, err = s.repo.ApplySubscriptionEvent(ctx, Event{
		Provider:        ProviderCreem,
		ProviderEventID: payload.EventID,
		EventType:       payload.EventType,
		PayloadSHA256:   sha256Hex(rawBody),
	}, UpsertSubscriptionParams{
		UserID:                 payload.UserID,
		Provider:               ProviderCreem,
		ProviderCustomerID:     payload.CustomerID,
		ProviderSubscriptionID: payload.SubscriptionID,
		Plan:                   core.PlanHosted,
		Status:                 payload.Status,
		CurrentPeriodEnd:       payload.CurrentPeriodEnd,
		CancelAtPeriodEnd:      payload.CancelAtPeriodEnd,
		GracePeriodDays:        s.gracePeriodDays,
		ProviderEventAt:        payload.OccurredAt,
	})
	if errors.Is(err, ErrEventAlreadyProcessed) {
		s.log.Info("billing webhook already processed", "event_type", payload.EventType)
		return nil
	}
	if errors.Is(err, ErrStaleSubscriptionEvent) {
		s.log.Info("ignored stale billing webhook", "event_type", payload.EventType)
		return nil
	}
	if err != nil {
		s.notifyBillingWebhookFailed(ctx, payload.EventType)
		return err
	}

	currentPlan, err := s.repo.GetUserPlan(ctx, payload.UserID)
	if err != nil {
		s.notifyBillingWebhookFailed(ctx, payload.EventType)
		return err
	}
	s.notifyHostedPlanTransition(ctx, payload.UserID, previousPlan, currentPlan)
	return nil
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (s *Service) notifyBillingWebhookFailed(ctx context.Context, eventType string) {
	if s.productNotifier == nil {
		return
	}
	if err := s.productNotifier.NotifyBillingWebhookFailed(ctx, eventType); err != nil {
		s.log.Warn("failed to send billing webhook failure notice", "event_type", eventType, "error", err)
	}
}

func (s *Service) notifyHostedPlanTransition(ctx context.Context, userID string, previousPlan, currentPlan core.Plan) {
	if s.productNotifier == nil {
		return
	}

	previousHosted := previousPlan == core.PlanHosted
	currentHosted := currentPlan == core.PlanHosted

	if !previousHosted && currentHosted {
		if err := s.productNotifier.NotifyHostedActivated(ctx, userID); err != nil {
			s.log.Warn("failed to send hosted activation notice", "user_id", userID, "error", err)
		}
		return
	}
	if previousHosted && !currentHosted {
		if err := s.productNotifier.NotifyHostedEnded(ctx, userID); err != nil {
			s.log.Warn("failed to send hosted ended notice", "user_id", userID, "error", err)
		}
	}
}
