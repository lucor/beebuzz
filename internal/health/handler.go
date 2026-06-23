package health

import (
	"context"
	"net/http"

	"go.beebuzz.app/beebuzz/internal/core"
)

// DBPinger checks database connectivity.
type DBPinger interface {
	PingContext(ctx context.Context) error
}

type Handler struct {
	version string
	commit  string
	db      DBPinger
}

func NewHandler(version, commit string, db DBPinger) *Handler {
	return &Handler{version: version, commit: commit, db: db}
}

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

// Health returns 200 if the database is reachable, 503 otherwise.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if err := h.db.PingContext(r.Context()); err != nil {
		core.WriteJSON(w, http.StatusServiceUnavailable, HealthResponse{
			Status:  "unavailable",
			Version: h.version,
			Commit:  h.commit,
		})
		return
	}

	core.WriteJSON(w, http.StatusOK, HealthResponse{
		Status:  "ok",
		Version: h.version,
		Commit:  h.commit,
	})
}
