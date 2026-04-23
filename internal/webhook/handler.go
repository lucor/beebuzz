package webhook

import (
	"errors"
	"io"
	"log/slog"
	"net/http"

	"lucor.dev/beebuzz/internal/core"
	"lucor.dev/beebuzz/internal/middleware"
)

const maxWebhookBodyBytes = 64 * 1024 // 64 KB

// Handler handles webhook HTTP requests.
type Handler struct {
	service *Service
	hookURL string
	log     *slog.Logger
}

// NewHandler creates a new webhook handler.
func NewHandler(svc *Service, hookURL string, logger *slog.Logger) *Handler {
	return &Handler{service: svc, hookURL: hookURL, log: logger}
}

// Receive handles incoming webhook payloads.
func (h *Handler) Receive(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.RequestIDFromContext(r.Context())
	log := h.log.With("request_id", reqID)

	token := core.GetURLParam(r, "token")
	if token == "" {
		core.WriteBadRequest(w, "missing_param", "token is required")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxWebhookBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		core.WritePayloadTooLarge(w)
		return
	}

	response, err := h.service.Receive(r.Context(), token, body, log)
	if err != nil {
		if errors.Is(err, ErrWebhookNotFound) {
			core.WriteNotFound(w, "webhook_not_found", "Webhook not found")
			return
		}
		if errors.Is(err, ErrWebhookInactive) {
			core.WriteForbidden(w, "webhook_inactive", "Webhook is inactive")
			return
		}
		if errors.Is(err, ErrPayloadExtraction) {
			core.WriteError(w, http.StatusUnprocessableEntity, "payload_extraction_failed", err.Error())
			return
		}
		if errors.Is(err, ErrWebhookDeliveryFailed) {
			core.WriteError(w, http.StatusBadGateway, "webhook_delivery_failed", "Webhook dispatch failed for all topics")
			return
		}
		log.Error("webhook receive failed", "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	if response.Status == ReceiveStatusDelivered && response.TotalCount == 0 {
		core.WriteJSON(w, http.StatusAccepted, response)
		return
	}
	core.WriteOK(w, response)
}

// ListWebhooks handles GET /webhooks.
func (h *Handler) ListWebhooks(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := middleware.UserFromContext(r.Context())
	if !ok {
		core.WriteUnauthorized(w, "invalid_session", "Invalid or expired session")
		return
	}

	webhooks, err := h.service.ListWebhooks(r.Context(), userCtx.ID)
	if err != nil {
		h.log.Error("failed to list webhooks", "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	core.WriteOK(w, WebhooksListResponse{Data: webhooks})
}

// CreateWebhook handles POST /webhooks.
func (h *Handler) CreateWebhook(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := middleware.UserFromContext(r.Context())
	if !ok {
		core.WriteUnauthorized(w, "invalid_session", "Invalid or expired session")
		return
	}

	var req CreateWebhookRequest

	if err := core.DecodeJSON(r.Body, &req); err != nil {
		core.WriteDecodeError(w, err)
		return
	}

	if errs := req.Validate(); len(errs) > 0 {
		core.WriteValidationErrors(w, errs)
		return
	}

	tokenStr, webhookID, err := h.service.CreateWebhook(r.Context(), userCtx.ID, req.Name, req.Description, req.PayloadType, req.TitlePath, req.BodyPath, req.Priority, req.Topics)
	if err != nil {
		if errors.Is(err, ErrAtLeastOneTopic) {
			core.WriteError(w, http.StatusUnprocessableEntity, "at_least_one_topic", err.Error())
			return
		}
		if errors.Is(err, ErrInvalidTopicSelection) {
			core.WriteError(w, http.StatusUnprocessableEntity, "invalid_topic_selection", err.Error())
			return
		}
		h.log.Error("failed to create webhook", "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	core.WriteJSON(w, http.StatusCreated, CreatedWebhookResponse{
		ID:    webhookID,
		Token: tokenStr,
		Name:  req.Name,
	})
}

// UpdateWebhook handles PATCH /webhooks/{webhookID}.
func (h *Handler) UpdateWebhook(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := middleware.UserFromContext(r.Context())
	if !ok {
		core.WriteUnauthorized(w, "invalid_session", "Invalid or expired session")
		return
	}

	webhookID := core.GetURLParam(r, "webhookID")
	if webhookID == "" {
		core.WriteBadRequest(w, "missing_param", "webhookID is required")
		return
	}

	var req UpdateWebhookRequest

	if err := core.DecodeJSON(r.Body, &req); err != nil {
		core.WriteDecodeError(w, err)
		return
	}

	if errs := req.Validate(); len(errs) > 0 {
		core.WriteValidationErrors(w, errs)
		return
	}

	if err := h.service.UpdateWebhook(r.Context(), userCtx.ID, webhookID, req.Name, req.Description, req.PayloadType, req.TitlePath, req.BodyPath, req.Priority, req.Topics); err != nil {
		if errors.Is(err, ErrWebhookNotFound) {
			core.WriteNotFound(w, "webhook_not_found", "Webhook not found")
			return
		}
		if errors.Is(err, ErrAtLeastOneTopic) {
			core.WriteError(w, http.StatusUnprocessableEntity, "at_least_one_topic", err.Error())
			return
		}
		if errors.Is(err, ErrInvalidTopicSelection) {
			core.WriteError(w, http.StatusUnprocessableEntity, "invalid_topic_selection", err.Error())
			return
		}
		h.log.Error("failed to update webhook", "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	core.WriteNoContent(w)
}

// DeleteWebhook handles DELETE /webhooks/{webhookID}.
func (h *Handler) DeleteWebhook(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := middleware.UserFromContext(r.Context())
	if !ok {
		core.WriteUnauthorized(w, "invalid_session", "Invalid or expired session")
		return
	}

	webhookID := core.GetURLParam(r, "webhookID")
	if webhookID == "" {
		core.WriteBadRequest(w, "missing_param", "webhookID is required")
		return
	}

	if err := h.service.RevokeWebhook(r.Context(), userCtx.ID, webhookID); err != nil {
		if errors.Is(err, ErrWebhookNotFound) {
			core.WriteNotFound(w, "webhook_not_found", "Webhook not found")
			return
		}
		h.log.Error("failed to delete webhook", "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	core.WriteNoContent(w)
}

// RegenerateToken handles POST /webhooks/{webhookID}/token.
func (h *Handler) RegenerateToken(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := middleware.UserFromContext(r.Context())
	if !ok {
		core.WriteUnauthorized(w, "invalid_session", "Invalid or expired session")
		return
	}

	webhookID := core.GetURLParam(r, "webhookID")
	if webhookID == "" {
		core.WriteBadRequest(w, "missing_param", "webhookID is required")
		return
	}

	rawToken, err := h.service.RegenerateToken(r.Context(), userCtx.ID, webhookID)
	if err != nil {
		if errors.Is(err, ErrWebhookNotFound) {
			core.WriteNotFound(w, "webhook_not_found", "Webhook not found")
			return
		}
		h.log.Error("failed to regenerate webhook token", "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	core.WriteOK(w, RegenerateTokenResponse{Token: rawToken})
}

// CreateInspectSession handles POST /webhooks/inspect.
func (h *Handler) CreateInspectSession(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := middleware.UserFromContext(r.Context())
	if !ok {
		core.WriteUnauthorized(w, "invalid_session", "Invalid or expired session")
		return
	}

	var req CreateInspectSessionRequest
	if err := core.DecodeJSON(r.Body, &req); err != nil {
		core.WriteDecodeError(w, err)
		return
	}

	if errs := req.Validate(); len(errs) > 0 {
		core.WriteValidationErrors(w, errs)
		return
	}

	response, err := h.service.CreateInspectSession(r.Context(), userCtx.ID, req.Name, req.Description, req.Priority, req.Topics, h.hookURL)
	if err != nil {
		if errors.Is(err, ErrAtLeastOneTopic) {
			core.WriteError(w, http.StatusUnprocessableEntity, "at_least_one_topic", err.Error())
			return
		}
		if errors.Is(err, ErrInvalidTopicSelection) {
			core.WriteError(w, http.StatusUnprocessableEntity, "invalid_topic_selection", err.Error())
			return
		}
		h.log.Error("failed to create inspect session", "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	core.WriteJSON(w, http.StatusCreated, response)
}

// GetInspectSession handles GET /webhooks/inspect.
func (h *Handler) GetInspectSession(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := middleware.UserFromContext(r.Context())
	if !ok {
		core.WriteUnauthorized(w, "invalid_session", "Invalid or expired session")
		return
	}

	response := h.service.GetInspectSession(r.Context(), userCtx.ID)
	if response == nil {
		core.WriteNotFound(w, "inspect_session_not_found", "No active inspect session found")
		return
	}

	core.WriteOK(w, response)
}

// FinalizeInspect handles POST /webhooks/inspect/finalize.
func (h *Handler) FinalizeInspect(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := middleware.UserFromContext(r.Context())
	if !ok {
		core.WriteUnauthorized(w, "invalid_session", "Invalid or expired session")
		return
	}

	var req FinalizeInspectRequest
	if err := core.DecodeJSON(r.Body, &req); err != nil {
		core.WriteDecodeError(w, err)
		return
	}

	if errs := req.Validate(); len(errs) > 0 {
		core.WriteValidationErrors(w, errs)
		return
	}

	rawToken, webhookID, err := h.service.FinalizeInspect(r.Context(), userCtx.ID, req.TitlePath, req.BodyPath)
	if err != nil {
		if errors.Is(err, ErrInspectSessionNotFound) {
			core.WriteNotFound(w, "inspect_session_not_found", "Inspect session not found")
			return
		}
		if errors.Is(err, ErrInspectNotCaptured) {
			core.WriteError(w, http.StatusUnprocessableEntity, "inspect_not_captured", "No payload has been captured yet")
			return
		}
		if errors.Is(err, ErrPayloadExtraction) {
			core.WriteError(w, http.StatusUnprocessableEntity, "payload_extraction_failed", err.Error())
			return
		}
		h.log.Error("failed to finalize inspect session", "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	core.WriteJSON(w, http.StatusCreated, CreatedWebhookResponse{
		ID:    webhookID,
		Token: rawToken,
		Name:  req.TitlePath,
	})
}
