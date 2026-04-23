package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type testSessionValidator struct {
	user *SessionUser
	err  error
}

func (v testSessionValidator) ValidateSession(_ context.Context, _ string) (*SessionUser, error) {
	if v.err != nil {
		return nil, v.err
	}
	return v.user, nil
}

func decodeErrorBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]string {
	t.Helper()

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	return body
}

func TestBaseSecurityAddsHeaders(t *testing.T) {
	mw := BaseSecurity(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("x-content-type-options: got %q, want %q", got, "nosniff")
	}
	if got := rec.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("x-frame-options: got %q, want %q", got, "DENY")
	}
	if got := rec.Header().Get("Referrer-Policy"); got != "strict-origin-when-cross-origin" {
		t.Fatalf("referrer-policy: got %q, want %q", got, "strict-origin-when-cross-origin")
	}
	if got := rec.Header().Get("Strict-Transport-Security"); got != "max-age=31536000; includeSubDomains" {
		t.Fatalf("strict-transport-security: got %q, want %q", got, "max-age=31536000; includeSubDomains")
	}
	if got := rec.Header().Get("Content-Security-Policy"); got != "" {
		t.Fatalf("content-security-policy: got %q, want empty (CSP enforced at Caddy layer)", got)
	}
}

func TestCORSAllowsConfiguredOrigin(t *testing.T) {
	mw := CORS([]string{"https://allowed.example.com"})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/v1/health", nil)
	req.Header.Set("Origin", "https://allowed.example.com")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://allowed.example.com" {
		t.Fatalf("allow-origin: got %q, want %q", got, "https://allowed.example.com")
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Fatal("allow-methods: got empty value")
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Fatal("allow-headers: got empty value")
	}
}

func TestCORSDeniesUnconfiguredOrigin(t *testing.T) {
	mw := CORS([]string{"https://allowed.example.com"})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/v1/health", nil)
	req.Header.Set("Origin", "https://blocked.example.com")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("allow-origin: got %q, want empty", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "" {
		t.Fatalf("allow-methods: got %q, want empty", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "" {
		t.Fatalf("allow-headers: got %q, want empty", got)
	}
}

func TestRequireSessionRejectsMissingCookie(t *testing.T) {
	handler := RequireSession(testSessionValidator{})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	body := decodeErrorBody(t, rec)
	if got := body["code"]; got != "invalid_session" {
		t.Fatalf("code: got %q, want %q", got, "invalid_session")
	}
	if got := body["message"]; got != messageInvalidSession {
		t.Fatalf("message: got %q, want %q", got, messageInvalidSession)
	}
}

func TestRequireSessionRejectsInvalidSession(t *testing.T) {
	handler := RequireSession(testSessionValidator{err: errors.New("invalid")})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: CookieSessionName, Value: "bad-session"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	body := decodeErrorBody(t, rec)
	if got := body["code"]; got != "invalid_session" {
		t.Fatalf("code: got %q, want %q", got, "invalid_session")
	}
	if got := body["message"]; got != messageInvalidSession {
		t.Fatalf("message: got %q, want %q", got, messageInvalidSession)
	}
}

func TestRequireAdminRejectsNonAdmin(t *testing.T) {
	handler := RequireAdmin()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/users", nil)
	req = req.WithContext(context.WithValue(req.Context(), CtxKeyUser, &CtxUser{
		ID:      "user-1",
		IsAdmin: false,
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusForbidden)
	}

	body := decodeErrorBody(t, rec)
	if got := body["code"]; got != "forbidden" {
		t.Fatalf("code: got %q, want %q", got, "forbidden")
	}
	if got := body["message"]; got != messageForbidden {
		t.Fatalf("message: got %q, want %q", got, messageForbidden)
	}
}
