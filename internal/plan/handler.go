package plan

import (
	"errors"
	"log/slog"
	"net/http"

	"go.beebuzz.app/beebuzz/internal/core"
	"go.beebuzz.app/beebuzz/internal/middleware"
)

// Handler handles plan HTTP requests.
type Handler struct {
	svc *Service
	log *slog.Logger
}

// NewHandler creates a plan handler.
func NewHandler(svc *Service, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, log: logger}
}

// AccountUsage returns current plan quota state for the authenticated user.
func (h *Handler) AccountUsage(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := middleware.UserFromContext(r.Context())
	if !ok {
		h.log.Error("user not found in context")
		core.WriteInternalError(w, r, errors.New("user not found in context"))
		return
	}

	usage, err := h.svc.GetUsage(r.Context(), userCtx.ID)
	if err != nil {
		h.log.Error("failed to get plan usage", "user_id", userCtx.ID, "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	core.WriteOK(w, usage)
}
