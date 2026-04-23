package health

import (
	"context"
	"net/http"

	"lucor.dev/beebuzz/internal/core"
)

// DBPinger checks database connectivity.
type DBPinger interface {
	PingContext(ctx context.Context) error
}

type Handler struct {
	version string
	db      DBPinger
}

func NewHandler(version string, db DBPinger) *Handler {
	return &Handler{version: version, db: db}
}

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// Health returns 200 if the database is reachable, 503 otherwise.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if err := h.db.PingContext(r.Context()); err != nil {
		core.WriteJSON(w, http.StatusServiceUnavailable, HealthResponse{
			Status:  "unavailable",
			Version: h.version,
		})
		return
	}

	core.WriteJSON(w, http.StatusOK, HealthResponse{
		Status:  "ok",
		Version: h.version,
	})
}
