package core

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteDecodeErrorWritesPayloadTooLarge(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteDecodeError(rec, ErrPayloadTooLarge)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
	if body := rec.Body.String(); !strings.Contains(body, "payload_too_large") {
		t.Fatalf("body: got %q, want payload_too_large", body)
	}
}

func TestWriteDecodeErrorWritesBadRequestForOtherDecodeErrors(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteDecodeError(rec, errors.New("unexpected EOF"))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if body := rec.Body.String(); !strings.Contains(body, "invalid_json") {
		t.Fatalf("body: got %q, want invalid_json", body)
	}
}
