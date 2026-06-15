package debugreport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"go.beebuzz.app/beebuzz/internal/core"
	"go.beebuzz.app/beebuzz/internal/middleware"
)

// DeviceAuthenticator validates Hive device credentials by token alone.
type DeviceAuthenticator interface {
	AuthenticateDeviceByToken(ctx context.Context, deviceToken string) (deviceID string, err error)
}

// Handler handles debug report HTTP requests.
type Handler struct {
	svc        *Service
	deviceAuth DeviceAuthenticator
	log        *slog.Logger
}

// NewHandler creates a new debug report handler.
func NewHandler(svc *Service, logger *slog.Logger) *Handler {
	return &Handler{
		svc: svc,
		log: logger,
	}
}

// SetDeviceAuthenticator enables device token auth for this handler.
func (h *Handler) SetDeviceAuthenticator(auth DeviceAuthenticator) {
	h.deviceAuth = auth
}

// Submit handles POST /v1/hive/debug-reports.
// Requires a valid Hive device Bearer token.
func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.RequestIDFromContext(r.Context())
	log := h.log.With("request_id", reqID)

	rawToken, ok := middleware.BearerTokenFromContext(r.Context())
	if !ok || rawToken == "" {
		core.WriteUnauthorized(w, "missing_token", "Bearer device token required")
		return
	}

	if !strings.HasPrefix(rawToken, "beebuzz_device_") {
		core.WriteUnauthorized(w, "invalid_token", "Invalid device token")
		return
	}

	deviceID, err := h.deviceAuth.AuthenticateDeviceByToken(r.Context(), rawToken)
	if err != nil {
		log.Error("device authentication failed", "error", err)
		core.WriteUnauthorized(w, "invalid_device_token", "Invalid or expired device token")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, MaxPayloadSize+1))
	if err != nil {
		core.WriteBadRequest(w, "read_error", "Failed to read request body")
		return
	}
	if len(body) > MaxPayloadSize {
		core.WritePayloadTooLarge(w)
		return
	}

	// Decode with strict unknown field rejection.
	var req SubmitRequest
	dec := json.NewDecoder(strings.NewReader(string(body)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		var syntaxErr *json.SyntaxError
		if errors.As(err, &syntaxErr) {
			core.WriteBadRequest(w, "invalid_json", "Invalid JSON body")
		} else {
			core.WriteBadRequest(w, "unknown_field", fmt.Sprintf("Invalid field: %s", err.Error()))
		}
		return
	}

	// Check for trailing data
	var trailing struct{}
	if err := dec.Decode(&trailing); err != io.EOF {
		core.WriteBadRequest(w, "unexpected_data", "Unexpected data after JSON body")
		return
	}

	if errs := req.Validate(time.Now()); len(errs) > 0 {
		core.WriteValidationErrors(w, errs)
		return
	}

	// Marshal canonical JSON for storage
	canonical, err := json.Marshal(req)
	if err != nil {
		log.Error("failed to marshal canonical report JSON", "error", err)
		core.WriteInternalError(w, r, fmt.Errorf("failed to process report"))
		return
	}

	report := &DebugReport{
		ReportID:    uuid.New().String(),
		DeviceID:    deviceID,
		CreatedAt:   time.Now().UnixMilli(),
		PayloadJSON: string(canonical),
	}

	if err := h.svc.Save(r.Context(), report); err != nil {
		log.Error("failed to save debug report", "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	log.Info("debug report saved", "report_id", report.ReportID, "device_id", deviceID)

	core.WriteJSON(w, http.StatusCreated, SubmitResponse{
		ReportID:  report.ReportID,
		CreatedAt: time.UnixMilli(report.CreatedAt).Format(time.RFC3339Nano),
	})
}
