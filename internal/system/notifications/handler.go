package notifications

import (
	"errors"
	"log/slog"
	"net/http"

	"lucor.dev/beebuzz/internal/core"
	"lucor.dev/beebuzz/internal/middleware"
)

// Handler handles admin system notification settings requests.
type Handler struct {
	svc *Service
	log *slog.Logger
}

// NewHandler creates a system notifications handler.
func NewHandler(svc *Service, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, log: logger}
}

// GetSettings handles GET /admin/system/notifications.
func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.svc.GetSettings(r.Context())
	if err != nil {
		h.log.Error("failed to get system notification settings", "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	core.WriteOK(w, ToSettingsResponse(settings))
}

// UpdateSettings handles PATCH /admin/system/notifications.
func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := middleware.UserFromContext(r.Context())
	if !ok {
		core.WriteUnauthorized(w, "invalid_session", "Invalid or expired session")
		return
	}

	var req UpdateSettingsRequest
	if err := core.DecodeJSON(r.Body, &req); err != nil {
		core.WriteDecodeError(w, err)
		return
	}

	settings, err := h.svc.UpdateSettings(r.Context(), userCtx.ID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrTopicRequired):
			core.WriteBadRequest(w, "topic_required", err.Error())
		case errors.Is(err, ErrInvalidTopicSelection):
			core.WriteError(w, http.StatusUnprocessableEntity, "invalid_topic_selection", err.Error())
		default:
			h.log.Error("failed to update system notification settings", "error", err)
			core.WriteInternalError(w, r, err)
		}
		return
	}

	core.WriteOK(w, ToSettingsResponse(settings))
}
