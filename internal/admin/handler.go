package admin

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"lucor.dev/beebuzz/internal/core"
	"lucor.dev/beebuzz/internal/middleware"
)

type Handler struct {
	adminService *Service
	log          *slog.Logger
}

type AdminUserResponse struct {
	ID             string             `json:"id"`
	Email          string             `json:"email"`
	IsAdmin        bool               `json:"is_admin"`
	AccountStatus  core.AccountStatus `json:"account_status"`
	SignupReason   *string            `json:"signup_reason,omitempty"`
	TrialStartedAt *time.Time         `json:"trial_started_at,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

type AdminUsersListResponse struct {
	Data []AdminUserResponse `json:"data"`
}

func NewHandler(adminSvc *Service, logger *slog.Logger) *Handler {
	return &Handler{
		adminService: adminSvc,
		log:          logger,
	}
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.adminService.ListUsers(r.Context())
	if err != nil {
		h.log.Error("failed to list users", "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	core.WriteJSON(w, http.StatusOK, toListUsersResponse(users))
}

type UpdateUserStatusRequest struct {
	AccountStatus core.AccountStatus `json:"account_status"`
}

func (h *Handler) UpdateUserStatus(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")

	var req UpdateUserStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		core.WriteBadRequest(w, "invalid_request", "Invalid request body")
		return
	}

	validStatuses := map[core.AccountStatus]bool{core.AccountStatusPending: true, core.AccountStatusActive: true, core.AccountStatusBlocked: true}
	if !validStatuses[req.AccountStatus] {
		core.WriteBadRequest(w, "invalid_status", "account_status must be one of: pending, active, blocked")
		return
	}

	adminID := ""
	if userCtx, ok := middleware.UserFromContext(r.Context()); ok {
		adminID = userCtx.ID
	}

	user, err := h.adminService.UpdateUserStatus(r.Context(), userID, req.AccountStatus, adminID)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidTransition):
			core.WriteError(w, http.StatusBadRequest, "invalid_transition", err.Error())
		case errors.Is(err, ErrConcurrentModification):
			core.WriteError(w, http.StatusConflict, "concurrent_modification", err.Error())
		default:
			h.log.Error("failed to update user status", "error", err)
			core.WriteInternalError(w, r, err)
		}
		return
	}

	core.WriteJSON(w, http.StatusOK, ToAdminUserResponse(user))
}

func ToAdminUserResponse(u *User) AdminUserResponse {
	return AdminUserResponse(*u)
}

func toListUsersResponse(users []User) AdminUsersListResponse {
	result := make([]AdminUserResponse, len(users))
	for i, u := range users {
		result[i] = AdminUserResponse(u)
	}
	return AdminUsersListResponse{Data: result}
}
