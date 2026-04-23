package token

import (
	"errors"
	"log/slog"
	"net/http"

	"lucor.dev/beebuzz/internal/core"
	"lucor.dev/beebuzz/internal/middleware"
)

// Handler handles API token HTTP requests.
type Handler struct {
	svc *Service
	log *slog.Logger
}

// NewHandler creates a new token handler.
func NewHandler(svc *Service, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, log: logger}
}

// ListAPITokens handles GET /tokens.
func (h *Handler) ListAPITokens(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := middleware.UserFromContext(r.Context())
	if !ok {
		core.WriteUnauthorized(w, "invalid_session", "Invalid or expired session")
		return
	}

	tokens, err := h.svc.ListAPITokens(r.Context(), userCtx.ID)
	if err != nil {
		h.log.Error("failed to list API tokens", "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	core.WriteOK(w, APITokensListResponse{Data: tokens})
}

// CreateAPIToken handles POST /tokens.
func (h *Handler) CreateAPIToken(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := middleware.UserFromContext(r.Context())
	if !ok {
		core.WriteUnauthorized(w, "invalid_session", "Invalid or expired session")
		return
	}

	var req CreateAPITokenRequest

	if err := core.DecodeJSON(r.Body, &req); err != nil {
		core.WriteDecodeError(w, err)
		return
	}

	if errs := req.Validate(); len(errs) > 0 {
		core.WriteValidationErrors(w, errs)
		return
	}

	tokenStr, _, err := h.svc.CreateAPIToken(r.Context(), userCtx.ID, req.Name, req.Description, req.Topics)
	if err != nil {
		if errors.Is(err, ErrAtLeastOneTopic) {
			core.WriteError(w, http.StatusUnprocessableEntity, "at_least_one_topic", err.Error())
			return
		}
		if errors.Is(err, ErrInvalidTopicSelection) {
			core.WriteError(w, http.StatusUnprocessableEntity, "invalid_topic_selection", err.Error())
			return
		}
		h.log.Error("failed to create API token", "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	core.WriteJSON(w, http.StatusCreated, CreatedAPITokenResponse{
		Token: tokenStr,
		Name:  req.Name,
	})
}

// UpdateAPIToken handles PATCH /tokens/{tokenID}.
func (h *Handler) UpdateAPIToken(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := middleware.UserFromContext(r.Context())
	if !ok {
		core.WriteUnauthorized(w, "invalid_session", "Invalid or expired session")
		return
	}

	tokenID := core.GetURLParam(r, "tokenID")
	if tokenID == "" {
		core.WriteBadRequest(w, "missing_param", "tokenID is required")
		return
	}

	var req UpdateAPITokenRequest

	if err := core.DecodeJSON(r.Body, &req); err != nil {
		core.WriteDecodeError(w, err)
		return
	}

	if errs := req.Validate(); len(errs) > 0 {
		core.WriteValidationErrors(w, errs)
		return
	}

	if err := h.svc.UpdateAPIToken(r.Context(), userCtx.ID, tokenID, req.Name, req.Description, req.Topics); err != nil {
		if errors.Is(err, ErrTokenNotFound) {
			core.WriteNotFound(w, "token_not_found", "API token not found")
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
		h.log.Error("failed to update API token", "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	core.WriteNoContent(w)
}

// RevokeAPIToken handles DELETE /tokens/{tokenID}.
func (h *Handler) RevokeAPIToken(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := middleware.UserFromContext(r.Context())
	if !ok {
		core.WriteUnauthorized(w, "invalid_session", "Invalid or expired session")
		return
	}

	tokenID := core.GetURLParam(r, "tokenID")
	if tokenID == "" {
		core.WriteBadRequest(w, "missing_param", "tokenID is required")
		return
	}

	if err := h.svc.RevokeAPIToken(r.Context(), userCtx.ID, tokenID); err != nil {
		if errors.Is(err, ErrTokenNotFound) {
			core.WriteNotFound(w, "token_not_found", "API token not found")
			return
		}
		h.log.Error("failed to revoke API token", "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	core.WriteNoContent(w)
}
