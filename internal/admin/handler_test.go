package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"lucor.dev/beebuzz/internal/testutil"
)

// stubSessionRevoker satisfies the SessionRevoker interface for handler tests.
type stubSessionRevoker struct{}

func (s *stubSessionRevoker) RevokeAllSessions(_ context.Context, _ string) error { return nil }

// stubMailer satisfies the admin.Mailer interface for handler tests.
type stubMailer struct{}

func (s *stubMailer) SendAccountApproved(_ context.Context, _ string) error    { return nil }
func (s *stubMailer) SendAccountBlocked(_ context.Context, _ string) error     { return nil }
func (s *stubMailer) SendAccountReactivated(_ context.Context, _ string) error { return nil }

func newTestAdminHandler(t *testing.T) *Handler {
	t.Helper()

	db := testutil.NewDB(t)
	repo := NewRepository(db)
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	svc := NewService(repo, &stubSessionRevoker{}, &stubMailer{}, logger)
	return NewHandler(svc, logger)
}

func seedAdminUser(t *testing.T, ctx context.Context, handler *Handler) {
	t.Helper()

	db := handler.adminService.repo.db
	reason := "Need access"
	if _, err := db.ExecContext(ctx,
		`INSERT INTO users (id, email, is_admin, account_status, signup_reason, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"admin-user-1", "admin@example.com", 1, "active", reason, 1700000000000, 1700000000000,
	); err != nil {
		t.Fatalf("insert admin user: %v", err)
	}
}

func TestListUsersReturnsWrappedData(t *testing.T) {
	handler := newTestAdminHandler(t)
	ctx := context.Background()
	seedAdminUser(t, ctx, handler)

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	w := httptest.NewRecorder()

	handler.ListUsers(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("ListUsers() status = %d, want %d. body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp AdminUsersListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if len(resp.Data) != 1 {
		t.Fatalf("ListUsers() data len = %d, want 1", len(resp.Data))
	}
	if resp.Data[0].Email != "admin@example.com" {
		t.Fatalf("ListUsers() first email = %q, want %q", resp.Data[0].Email, "admin@example.com")
	}
}
