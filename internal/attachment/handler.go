package attachment

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"lucor.dev/beebuzz/internal/core"
)

// Handler handles attachment HTTP requests.
type Handler struct {
	service *Service
	log     *slog.Logger
}

// NewHandler creates a new attachment handler.
func NewHandler(svc *Service, logger *slog.Logger) *Handler {
	return &Handler{service: svc, log: logger}
}

// Get handles GET /v1/attachments/{token}.
// No session auth — access is controlled by the opaque token in the URL.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	token := core.GetURLParam(r, "token")

	data, att, err := h.service.GetByToken(r.Context(), token)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) || errors.Is(err, ErrAttachmentExpired) {
			core.WriteNotFound(w, "not_found", "attachment not found or expired")
			return
		}
		h.log.Error("failed to get attachment", "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	_ = att // metadata available for future use (e.g., logging)
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		h.log.Error("failed to write attachment response", "error", err)
	}
}
