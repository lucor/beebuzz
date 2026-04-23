package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestID_GeneratesWhenMissing(t *testing.T) {
	var ctxID string
	rid := NewRequestID("")
	handler := rid.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxID = RequestIDFromContext(r.Context())
	}))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))

	if len(ctxID) != 32 {
		t.Fatalf("expected 32-char generated ID, got %q (len %d)", ctxID, len(ctxID))
	}
	if rr.Header().Get("X-Request-ID") != ctxID {
		t.Fatalf("response header %q != context %q", rr.Header().Get("X-Request-ID"), ctxID)
	}
}

func TestRequestID_PreservesValidHeader(t *testing.T) {
	var ctxID string
	rid := NewRequestID("")
	handler := rid.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxID = RequestIDFromContext(r.Context())
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-ID", "abc-123_XYZ")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if ctxID != "abc-123_XYZ" {
		t.Fatalf("expected preserved ID %q, got %q", "abc-123_XYZ", ctxID)
	}
	if rr.Header().Get("X-Request-ID") != "abc-123_XYZ" {
		t.Fatalf("response header mismatch")
	}
}

func TestRequestID_RejectsInvalidChars(t *testing.T) {
	var ctxID string
	rid := NewRequestID("")
	handler := rid.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxID = RequestIDFromContext(r.Context())
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-ID", "bad id with spaces!")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if ctxID == "bad id with spaces!" {
		t.Fatal("should have rejected invalid ID")
	}
	if len(ctxID) != 32 {
		t.Fatalf("expected 32-char generated ID, got len %d", len(ctxID))
	}
	if rr.Header().Get("X-Request-ID") != ctxID {
		t.Fatal("response header should match generated ID")
	}
}

func TestRequestID_TruncatesLongValidHeader(t *testing.T) {
	var ctxID string
	rid := NewRequestID("")
	handler := rid.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxID = RequestIDFromContext(r.Context())
	}))

	longID := strings.Repeat("a", 100)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-ID", longID)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if len(ctxID) != 64 {
		t.Fatalf("expected truncated to 64 chars, got %d", len(ctxID))
	}
	if rr.Header().Get("X-Request-ID") != ctxID {
		t.Fatal("response header should match truncated ID")
	}
}

func TestRequestID_CustomHeader(t *testing.T) {
	var ctxID string
	rid := NewRequestID("X-Trace-ID")
	handler := rid.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxID = RequestIDFromContext(r.Context())
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Trace-ID", "custom-trace-123")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if ctxID != "custom-trace-123" {
		t.Fatalf("expected %q, got %q", "custom-trace-123", ctxID)
	}
	if rr.Header().Get("X-Trace-ID") != "custom-trace-123" {
		t.Fatal("response header should use custom header name")
	}
	if rr.Header().Get("X-Request-ID") != "" {
		t.Fatal("default header should not be set when using custom header")
	}
}
