package billing

import (
	"errors"
	"io"
	"log/slog"
	"net/http"

	"go.beebuzz.app/beebuzz/internal/core"
	"go.beebuzz.app/beebuzz/internal/middleware"
)

const maxBillingWebhookBodyBytes = 256 * 1024

// Handler handles billing HTTP requests.
type Handler struct {
	service *Service
	log     *slog.Logger
}

// NewHandler creates a billing handler.
func NewHandler(service *Service, log *slog.Logger) *Handler {
	return &Handler{service: service, log: log}
}

// CreateCheckout starts a Hosted plan checkout for the current user.
func (h *Handler) CreateCheckout(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := middleware.UserFromContext(r.Context())
	if !ok {
		core.WriteUnauthorized(w, "invalid_session", "Invalid or expired session")
		return
	}

	response, err := h.service.CreateCheckout(r.Context(), userCtx.ID)
	if err != nil {
		if errors.Is(err, ErrBillingDisabled) {
			core.WriteNotFound(w, "billing_disabled", "Billing is not enabled")
			return
		}
		if errors.Is(err, ErrHostedSubscriptionActive) {
			core.WriteConflict(w, "hosted_subscription_active", "Hosted is already active. Manage billing through the billing portal.")
			return
		}
		h.log.Error("failed to create billing checkout", "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	core.WriteJSON(w, http.StatusCreated, response)
}

// CreateCustomerPortal creates a customer billing management link.
func (h *Handler) CreateCustomerPortal(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := middleware.UserFromContext(r.Context())
	if !ok {
		core.WriteUnauthorized(w, "invalid_session", "Invalid or expired session")
		return
	}

	response, err := h.service.CreateCustomerPortal(r.Context(), userCtx.ID)
	if err != nil {
		if errors.Is(err, ErrBillingDisabled) {
			core.WriteNotFound(w, "billing_disabled", "Billing is not enabled")
			return
		}
		if errors.Is(err, ErrBillingCustomerMissing) {
			core.WriteNotFound(w, "billing_customer_missing", "Billing customer not found")
			return
		}
		h.log.Error("failed to create billing portal", "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	core.WriteJSON(w, http.StatusCreated, response)
}

// ReceiveWebhook handles provider billing webhooks.
func (h *Handler) ReceiveWebhook(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBillingWebhookBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		core.WritePayloadTooLarge(w)
		return
	}

	if err := h.service.HandleWebhook(r.Context(), body, r.Header.Get("creem-signature")); err != nil {
		if errors.Is(err, ErrBillingDisabled) {
			core.WriteNotFound(w, "billing_disabled", "Billing is not enabled")
			return
		}
		if errors.Is(err, ErrInvalidWebhook) {
			core.WriteUnauthorized(w, "invalid_signature", "Invalid billing webhook")
			return
		}
		h.log.Error("failed to process billing webhook", "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	core.WriteNoContent(w)
}
