package user

import (
	"errors"
	"log/slog"
	"net/http"

	"lucor.dev/beebuzz/internal/core"
	"lucor.dev/beebuzz/internal/middleware"
)

// Handler handles user HTTP requests.
type Handler struct {
	userService *Service
	log         *slog.Logger
}

// NewHandler creates a new user handler.
func NewHandler(userSvc *Service, logger *slog.Logger) *Handler {
	return &Handler{
		userService: userSvc,
		log:         logger,
	}
}

// Me handles requests for the current user's info.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := middleware.UserFromContext(r.Context())
	if !ok {
		core.WriteInternalError(w, r, errors.New("failed to fetch user from context"))
		return
	}

	user, err := h.userService.GetMe(r.Context(), userCtx.ID)
	if err != nil {
		h.log.Error("failed to get user", "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	core.WriteJSON(w, http.StatusOK, ToUserResponse(user))
}
