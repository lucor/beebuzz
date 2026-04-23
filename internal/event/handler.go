package event

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"lucor.dev/beebuzz/internal/core"
	"lucor.dev/beebuzz/internal/middleware"
)

const defaultDashboardDays = 30
const maxDashboardDays = 90

// Handler handles HTTP requests for the event domain.
type Handler struct {
	svc *Service
	log *slog.Logger
}

// NewHandler creates a new event handler.
func NewHandler(svc *Service, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, log: logger}
}

// Dashboard returns platform-wide dashboard metrics for the admin panel.
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	days := defaultDashboardDays
	if d := r.URL.Query().Get("days"); d != "" {
		parsed, err := strconv.Atoi(d)
		if err != nil || parsed < allTimeRangeDays || parsed > maxDashboardDays {
			core.WriteBadRequest(w, "invalid_days", "days must be 0 or between 1 and 90")
			return
		}
		days = parsed
	}

	dashboard, err := h.svc.GetPlatformDashboard(r.Context(), days)
	if err != nil {
		h.log.Error("failed to get platform dashboard", "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	core.WriteOK(w, dashboard)
}

// AccountUsage returns usage metrics for the authenticated user's account dashboard.
func (h *Handler) AccountUsage(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := middleware.UserFromContext(r.Context())
	if !ok {
		h.log.Error("user not found in context")
		core.WriteInternalError(w, r, errors.New("user not found in context"))
		return
	}

	days := defaultDashboardDays
	if d := r.URL.Query().Get("days"); d != "" {
		parsed, err := strconv.Atoi(d)
		if err != nil || parsed < allTimeRangeDays || parsed > maxDashboardDays {
			core.WriteBadRequest(w, "invalid_days", "days must be 0 or between 1 and 90")
			return
		}
		days = parsed
	}

	usage, err := h.svc.GetAccountUsage(r.Context(), userCtx.ID, days)
	if err != nil {
		h.log.Error("failed to get account usage", "user_id", userCtx.ID, "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	core.WriteOK(w, usage)
}
