package auth

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"lucor.dev/beebuzz/internal/config"
	"lucor.dev/beebuzz/internal/mailer"
	"lucor.dev/beebuzz/internal/middleware"
	"lucor.dev/beebuzz/internal/secure"
	"lucor.dev/beebuzz/internal/testutil"
	"lucor.dev/beebuzz/internal/topic"
)

func newTestAuthHandler() *Handler {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	return NewHandler(nil, "", logger)
}

// TestLoginRejectsUnknownFields verifies strict JSON decoding rejects unknown fields.
func TestLoginRejectsUnknownFields(t *testing.T) {
	handler := newTestAuthHandler()

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(`{"email":"test@example.com","state":"abc","unexpected":true}`))
	w := httptest.NewRecorder()

	handler.Login(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Login() status = %d, want %d. body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

// TestLoginRejectsTrailingJSON verifies strict JSON decoding rejects trailing JSON payloads.
func TestLoginRejectsTrailingJSON(t *testing.T) {
	handler := newTestAuthHandler()

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(`{"email":"test@example.com","state":"abc"}{"extra":true}`))
	w := httptest.NewRecorder()

	handler.Login(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Login() status = %d, want %d. body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

// TestLoginRejectsDisplayNameEmail verifies plain login email validation rejects display-name forms.
func TestLoginRejectsDisplayNameEmail(t *testing.T) {
	handler := newTestAuthHandler()

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(`{"email":"\"Alice Example\" <user@example.com>","state":"abc"}`))
	w := httptest.NewRecorder()

	handler.Login(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("Login() status = %d, want %d. body=%s", w.Code, http.StatusUnprocessableEntity, w.Body.String())
	}
}

// TestLoginRejectsWhitespaceOnlyState verifies blank state values are rejected.
func TestLoginRejectsWhitespaceOnlyState(t *testing.T) {
	handler := newTestAuthHandler()

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(`{"email":"user@example.com","state":"   "}`))
	w := httptest.NewRecorder()

	handler.Login(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("Login() status = %d, want %d. body=%s", w.Code, http.StatusUnprocessableEntity, w.Body.String())
	}
}

func TestLoginReturnsNoContentForApprovedUser(t *testing.T) {
	handler := newAuthFlowTestHandler(t, false)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(`{"email":"approved@example.com","state":"abc"}`))
	w := httptest.NewRecorder()

	handler.Login(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("Login() status = %d, want %d. body=%s", w.Code, http.StatusNoContent, w.Body.String())
	}
	if w.Body.Len() != 0 {
		t.Fatalf("Login() body = %q, want empty", w.Body.String())
	}
}

func TestLoginReturnsNoContentForWaitlistUser(t *testing.T) {
	handler := newAuthFlowTestHandler(t, true)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(`{"email":"waitlist@example.com","state":"abc"}`))
	w := httptest.NewRecorder()

	handler.Login(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("Login() status = %d, want %d. body=%s", w.Code, http.StatusNoContent, w.Body.String())
	}
	if w.Body.Len() != 0 {
		t.Fatalf("Login() body = %q, want empty", w.Body.String())
	}
}

func TestLoginReturnsTooManyRequestsForGlobalThrottle(t *testing.T) {
	db := testutil.NewDB(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repo := NewRepository(db)
	svc := NewService(repo, nil, "", nil, logger)
	svc.SetGlobalThrottle(NewGlobalAuthThrottle(0, time.Minute))
	handler := NewHandler(svc, "", logger)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(`{"email":"throttle@example.com","state":"abc"}`))
	w := httptest.NewRecorder()

	handler.Login(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("Login() status = %d, want %d. body=%s", w.Code, http.StatusTooManyRequests, w.Body.String())
	}
	if got := w.Header().Get("Retry-After"); got != "60" {
		t.Fatalf("Retry-After = %q, want %q", got, "60")
	}
}

func TestLogoutReturnsNoContentAndClearsCookies(t *testing.T) {
	db := testutil.NewDB(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repo := NewRepository(db)
	svc := NewService(repo, nil, "", nil, logger)
	handler := NewHandler(svc, "", logger)
	ctx := context.Background()

	user, _, err := repo.GetOrCreateUser(ctx, "logout@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	rawToken := "logout-session-token"
	if err := repo.CreateSession(ctx, secure.Hash(rawToken), user.ID, time.Now().Add(time.Hour).UnixMilli()); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: middleware.CookieSessionName, Value: rawToken})
	req = req.WithContext(testutil.WithUserContext(req.Context(), user.ID))
	w := httptest.NewRecorder()

	handler.Logout(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("Logout() status = %d, want %d. body=%s", w.Code, http.StatusNoContent, w.Body.String())
	}
	if w.Body.Len() != 0 {
		t.Fatalf("Logout() body = %q, want empty", w.Body.String())
	}

	resp := w.Result()
	cookies := resp.Cookies()
	if len(cookies) != 2 {
		t.Fatalf("Logout() cookies len = %d, want 2", len(cookies))
	}

	got := map[string]*http.Cookie{}
	for _, cookie := range cookies {
		got[cookie.Name] = cookie
	}

	sessionCookie, ok := got[middleware.CookieSessionName]
	if !ok {
		t.Fatalf("missing cookie %q", middleware.CookieSessionName)
	}
	if sessionCookie.Value != "" || sessionCookie.MaxAge != -1 {
		t.Fatalf("session cookie = %+v, want cleared cookie", *sessionCookie)
	}

	loggedInCookie, ok := got[middleware.CookieLoggedInName]
	if !ok {
		t.Fatalf("missing cookie %q", middleware.CookieLoggedInName)
	}
	if loggedInCookie.Value != "" || loggedInCookie.MaxAge != -1 {
		t.Fatalf("logged-in cookie = %+v, want cleared cookie", *loggedInCookie)
	}
}

func newAuthFlowTestHandler(t *testing.T, privateBeta bool) *Handler {
	t.Helper()

	db := testutil.NewDB(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	authRepo := NewRepository(db)
	topicRepo := topic.NewRepository(db)
	topicSvc := topic.NewService(topicRepo, logger)
	testMailer, err := mailer.New(&config.Mailer{
		Sender:  "noreply@example.com",
		ReplyTo: "support@example.com",
	})
	if err != nil {
		t.Fatalf("mailer.New: %v", err)
	}

	svc := NewService(authRepo, testMailer, "", topicSvc, logger)
	svc.UsePrivateBeta(privateBeta)

	return NewHandler(svc, "", logger)
}
