package device

import (
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"
	"net/url"

	"rsc.io/qr"

	"lucor.dev/beebuzz/internal/core"
	"lucor.dev/beebuzz/internal/middleware"
)

// Handler handles device HTTP requests.
type Handler struct {
	svc     *Service
	hiveURL string
	log     *slog.Logger
}

// NewHandler creates a new device handler.
func NewHandler(svc *Service, hiveURL string, logger *slog.Logger) *Handler {
	return &Handler{
		svc:     svc,
		hiveURL: hiveURL,
		log:     logger,
	}
}

// GetDevices handles GET /v1/devices.
func (h *Handler) GetDevices(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := middleware.UserFromContext(r.Context())
	if !ok {
		core.WriteUnauthorized(w, "invalid_session", "Invalid or expired session")
		return
	}

	devices, err := h.svc.ListDevices(r.Context(), userCtx.ID)
	if err != nil {
		h.log.Error("failed to fetch devices", "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	core.WriteOK(w, DevicesListResponse{Data: devices})
}

// CreateDevice handles POST /v1/devices.
func (h *Handler) CreateDevice(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := middleware.UserFromContext(r.Context())
	if !ok {
		core.WriteUnauthorized(w, "invalid_session", "Invalid or expired session")
		return
	}

	var req CreateDeviceRequest
	if err := core.DecodeJSON(r.Body, &req); err != nil {
		core.WriteDecodeError(w, err)
		return
	}

	if errs := req.Validate(); len(errs) > 0 {
		core.WriteValidationErrors(w, errs)
		return
	}

	device, otp, expiresAt, err := h.svc.CreateDevice(r.Context(), userCtx.ID, req.Name, req.Description, req.Topics)
	if err != nil {
		if errors.Is(err, ErrAtLeastOneTopic) {
			core.WriteError(w, http.StatusUnprocessableEntity, "at_least_one_topic", err.Error())
			return
		}
		if errors.Is(err, ErrInvalidTopicSelection) {
			core.WriteError(w, http.StatusUnprocessableEntity, "invalid_topic_selection", err.Error())
			return
		}
		h.log.Error("failed to create device", "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	hiveURL := h.hiveURL + "/pair"

	core.WriteJSON(w, http.StatusCreated, CreatedDeviceResponse{
		Device:      ToDeviceResponse(device, req.Topics),
		PairingCode: otp,
		PairingURL:  hiveURL,
		QRCode:      generateQRCode(hiveURL, h.log),
		ExpiresAt:   expiresAt,
	})
}

// UpdateDevice handles PATCH /v1/devices/{deviceID}.
func (h *Handler) UpdateDevice(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := middleware.UserFromContext(r.Context())
	if !ok {
		core.WriteUnauthorized(w, "invalid_session", "Invalid or expired session")
		return
	}

	deviceID := core.GetURLParam(r, "deviceID")
	if deviceID == "" {
		core.WriteBadRequest(w, "missing_param", "deviceID is required")
		return
	}

	var req UpdateDeviceRequest
	if err := core.DecodeJSON(r.Body, &req); err != nil {
		core.WriteDecodeError(w, err)
		return
	}

	if errs := req.Validate(); len(errs) > 0 {
		core.WriteValidationErrors(w, errs)
		return
	}

	if err := h.svc.UpdateDevice(r.Context(), userCtx.ID, deviceID, req.Name, req.Description, req.Topics); err != nil {
		if errors.Is(err, ErrDeviceNotFound) {
			core.WriteNotFound(w, "device_not_found", "Device not found")
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
		h.log.Error("failed to update device", "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	core.WriteNoContent(w)
}

// DeleteDevice handles DELETE /v1/devices/{deviceID}.
func (h *Handler) DeleteDevice(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := middleware.UserFromContext(r.Context())
	if !ok {
		core.WriteUnauthorized(w, "invalid_session", "Invalid or expired session")
		return
	}

	deviceID := core.GetURLParam(r, "deviceID")
	if deviceID == "" {
		core.WriteBadRequest(w, "missing_param", "deviceID is required")
		return
	}

	if err := h.svc.DeleteDevice(r.Context(), userCtx.ID, deviceID); err != nil {
		if errors.Is(err, ErrDeviceNotFound) {
			core.WriteNotFound(w, "device_not_found", "Device not found")
			return
		}
		h.log.Error("failed to delete device", "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	core.WriteNoContent(w)
}

// RegeneratePairingOTP handles POST /v1/devices/{deviceID}/pairing-code.
func (h *Handler) RegeneratePairingOTP(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := middleware.UserFromContext(r.Context())
	if !ok {
		core.WriteUnauthorized(w, "invalid_session", "Invalid or expired session")
		return
	}

	deviceID := core.GetURLParam(r, "deviceID")
	if deviceID == "" {
		core.WriteBadRequest(w, "missing_param", "deviceID is required")
		return
	}

	otp, expiresAt, err := h.svc.RegeneratePairingOTP(r.Context(), userCtx.ID, deviceID)
	if err != nil {
		if errors.Is(err, ErrDeviceNotFound) {
			core.WriteNotFound(w, "device_not_found", "Device not found")
			return
		}
		h.log.Error("failed to regenerate pairing OTP", "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	hiveURL := h.hiveURL + "/pair"

	core.WriteOK(w, PairingCodeResponse{
		PairingCode: otp,
		PairingURL:  hiveURL,
		QRCode:      generateQRCode(hiveURL, h.log),
		ExpiresAt:   expiresAt,
	})
}

// UnpairDevice handles POST /v1/devices/{deviceID}/unpair.
func (h *Handler) UnpairDevice(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := middleware.UserFromContext(r.Context())
	if !ok {
		core.WriteUnauthorized(w, "invalid_session", "Invalid or expired session")
		return
	}

	deviceID := core.GetURLParam(r, "deviceID")
	if deviceID == "" {
		core.WriteBadRequest(w, "missing_param", "deviceID is required")
		return
	}

	if err := h.svc.UnpairDevice(r.Context(), userCtx.ID, deviceID); err != nil {
		if errors.Is(err, ErrDeviceNotFound) {
			core.WriteNotFound(w, "device_not_found", "Device not found")
			return
		}
		h.log.Error("failed to unpair device", "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	core.WriteNoContent(w)
}

// Pair handles POST /v1/pairing (public endpoint).
func (h *Handler) Pair(w http.ResponseWriter, r *http.Request) {
	var req PairRequest
	if err := core.DecodeJSON(r.Body, &req); err != nil {
		core.WriteDecodeError(w, err)
		return
	}

	if errs := req.Validate(); len(errs) > 0 {
		core.WriteValidationErrors(w, errs)
		return
	}

	deviceID, deviceToken, err := h.svc.Pair(r.Context(), req.PairingCode, req.Endpoint, req.P256dh, req.Auth, req.AgeRecipient)
	if err != nil {
		if errors.Is(err, ErrPairingCodeInvalid) {
			core.WriteUnauthorized(w, "invalid_pairing_code", "Pairing code is invalid or expired")
			return
		}
		if errors.Is(err, ErrInvalidPushEndpoint) {
			core.WriteError(w, http.StatusUnprocessableEntity, "invalid_push_endpoint", "Push endpoint is not supported")
			return
		}
		if errors.Is(err, ErrInvalidAgeRecipient) {
			core.WriteError(w, http.StatusUnprocessableEntity, "invalid_age_recipient", "age_recipient must be a valid age X25519 recipient")
			return
		}
		h.log.Error("failed to pair device", "error", err)
		core.WriteInternalError(w, r, err)
		return
	}

	pushHost := ""
	if parsed, parseErr := url.Parse(req.Endpoint); parseErr == nil {
		pushHost = parsed.Host
	}
	h.log.Info(
		"pairing registered push subscription",
		"device_id", deviceID,
		"push_host", pushHost,
		"p256dh_len", len(req.P256dh),
		"auth_len", len(req.Auth),
		"has_age_recipient", req.AgeRecipient != "",
	)

	core.WriteOK(w, PairResponse{DeviceID: deviceID, DeviceToken: deviceToken})
}

// PairingStatus handles GET /v1/pairing/{deviceID} (public, device-token-authenticated).
func (h *Handler) PairingStatus(w http.ResponseWriter, r *http.Request) {
	deviceID := core.GetURLParam(r, "deviceID")
	if deviceID == "" {
		core.WriteBadRequest(w, "missing_param", "deviceID is required")
		return
	}

	deviceToken, ok := middleware.BearerTokenFromContext(r.Context())
	if !ok || deviceToken == "" {
		core.WriteUnauthorized(w, "missing_device_token", "Device token is required")
		return
	}

	status, err := h.svc.GetPairingStatus(r.Context(), deviceID, deviceToken)
	if err != nil {
		if errors.Is(err, ErrInvalidDeviceToken) {
			core.WriteUnauthorized(w, "invalid_device_token", "Invalid device token")
			return
		}
		h.log.Error("failed to get pairing status", "error", err, "device_id", deviceID)
		core.WriteInternalError(w, r, err)
		return
	}

	core.WriteOK(w, status)
}

// generateQRCode encodes value into a QR PNG and returns a base64 data URI.
// Returns empty string on failure and logs the error.
func generateQRCode(value string, log *slog.Logger) string {
	code, err := qr.Encode(value, qr.M)
	if err != nil {
		log.Error("failed to generate QR code", "error", err)
		return ""
	}
	png := code.PNG()
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
}
