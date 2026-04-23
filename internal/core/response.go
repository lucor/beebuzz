package core

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/getsentry/sentry-go"
)

// WriteOK writes a 200 OK response with JSON body.
func WriteOK(w http.ResponseWriter, data any) {
	WriteJSON(w, http.StatusOK, data)
}

// WriteNoContent writes a 204 No Content response.
func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func WriteJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err, "status", statusCode)
	}
}

func WriteBadRequest(w http.ResponseWriter, code, message string) {
	WriteError(w, http.StatusBadRequest, code, message)
}

func WritePayloadTooLarge(w http.ResponseWriter) {
	WriteError(w, http.StatusRequestEntityTooLarge, "payload_too_large", "payload too large")
}

// WriteDecodeError maps shared JSON decode failures to the correct HTTP response
// while keeping a stable public error shape for normal decode failures.
func WriteDecodeError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrPayloadTooLarge) {
		WritePayloadTooLarge(w)
		return
	}

	WriteBadRequest(w, "invalid_json", "invalid JSON")
}

func WriteUnauthorized(w http.ResponseWriter, code, message string) {
	WriteError(w, http.StatusUnauthorized, code, message)
}

func WriteForbidden(w http.ResponseWriter, code, message string) {
	WriteError(w, http.StatusForbidden, code, message)
}

func WriteNotFound(w http.ResponseWriter, code, message string) {
	WriteError(w, http.StatusNotFound, code, message)
}

func WriteTooManyRequests(w http.ResponseWriter, code, message string) {
	WriteError(w, http.StatusTooManyRequests, code, message)
}

func WriteConflict(w http.ResponseWriter, code, message string) {
	WriteError(w, http.StatusConflict, code, message)
}

// WriteInternalError writes a 500 Internal Server Error response.
// Captures the error in Sentry if available. The caller should log the error.
func WriteInternalError(w http.ResponseWriter, r *http.Request, err error) {
	if hub := sentry.GetHubFromContext(r.Context()); hub != nil {
		hub.CaptureException(err)
	}
	WriteJSON(w, http.StatusInternalServerError, map[string]string{"code": "internal_error"})
}

func WriteError(w http.ResponseWriter, statusCode int, code, message string) {
	WriteJSON(w, statusCode, map[string]string{"code": code, "message": message})
}

const codeValidationErr = "validation_error"

type validationErrorResponse struct {
	Code   string   `json:"code"`
	Errors []string `json:"errors,omitempty"`
}

// WriteValidationErrors writes a 422 Unprocessable Entity with all validation error messages.
func WriteValidationErrors(w http.ResponseWriter, errs []error) {
	resp := validationErrorResponse{Code: codeValidationErr}
	for _, err := range errs {
		resp.Errors = append(resp.Errors, err.Error())
	}
	WriteJSON(w, http.StatusUnprocessableEntity, resp)
}
