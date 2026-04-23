package health_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"lucor.dev/beebuzz/internal/health"
)

type stubPinger struct {
	err error
}

func (s *stubPinger) PingContext(_ context.Context) error { return s.err }

func TestHealth_OK(t *testing.T) {
	h := health.NewHandler("v1.0.0", &stubPinger{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	h.Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp health.HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("expected status ok, got %s", resp.Status)
	}
	if resp.Version != "v1.0.0" {
		t.Errorf("expected version v1.0.0, got %s", resp.Version)
	}
}

func TestHealth_DBDown(t *testing.T) {
	h := health.NewHandler("v1.0.0", &stubPinger{err: errors.New("connection refused")})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	h.Health(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}

	var resp health.HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Status != "unavailable" {
		t.Errorf("expected status unavailable, got %s", resp.Status)
	}
}
