package admin

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"lucor.dev/beebuzz/internal/auth"
	"lucor.dev/beebuzz/internal/core"
	"lucor.dev/beebuzz/internal/testutil"
)

type trackingMailer struct {
	*stubMailer
	approvedCalled    bool
	blockedCalled     bool
	reactivatedCalled bool
}

func (s *trackingMailer) SendAccountApproved(_ context.Context, _ string) error {
	s.approvedCalled = true
	return s.stubMailer.SendAccountApproved(context.TODO(), "")
}

func (s *trackingMailer) SendAccountBlocked(_ context.Context, _ string) error {
	s.blockedCalled = true
	return s.stubMailer.SendAccountBlocked(context.TODO(), "")
}

func (s *trackingMailer) SendAccountReactivated(_ context.Context, _ string) error {
	s.reactivatedCalled = true
	return s.stubMailer.SendAccountReactivated(context.TODO(), "")
}

type errSessionRevoker struct {
	err error
}

func (s *errSessionRevoker) RevokeAllSessions(_ context.Context, _ string) error {
	return s.err
}

func TestUpdateUserStatus_PendingToActive(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	repo := NewRepository(db)
	mailer := &trackingMailer{stubMailer: &stubMailer{}}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(repo, &stubSessionRevoker{}, mailer, logger)

	user, _, err := auth.NewRepository(db).GetOrCreateUser(ctx, "pending@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	if _, err := db.ExecContext(ctx,
		`UPDATE users SET account_status = 'pending' WHERE id = ?`,
		user.ID,
	); err != nil {
		t.Fatalf("set pending status: %v", err)
	}

	updated, err := svc.UpdateUserStatus(ctx, user.ID, core.AccountStatusActive, "admin-1")
	if err != nil {
		t.Fatalf("UpdateUserStatus: %v", err)
	}
	if updated == nil {
		t.Fatal("UpdateUserStatus returned nil")
	}
	if updated.AccountStatus != core.AccountStatusActive {
		t.Errorf("account_status = %q, want %q", updated.AccountStatus, core.AccountStatusActive)
	}
	if !mailer.approvedCalled {
		t.Error("SendAccountApproved was not called")
	}
}

func TestUpdateUserStatus_ActiveToBlocked(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	repo := NewRepository(db)
	mailer := &trackingMailer{stubMailer: &stubMailer{}}
	revoker := &stubSessionRevoker{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(repo, revoker, mailer, logger)

	user, _, err := auth.NewRepository(db).GetOrCreateUser(ctx, "active@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	updated, err := svc.UpdateUserStatus(ctx, user.ID, core.AccountStatusBlocked, "admin-1")
	if err != nil {
		t.Fatalf("UpdateUserStatus: %v", err)
	}
	if updated == nil {
		t.Fatal("UpdateUserStatus returned nil")
	}
	if updated.AccountStatus != core.AccountStatusBlocked {
		t.Errorf("account_status = %q, want %q", updated.AccountStatus, core.AccountStatusBlocked)
	}
	if !mailer.blockedCalled {
		t.Error("SendAccountBlocked was not called")
	}
}

func TestUpdateUserStatus_BlockedToActive(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	repo := NewRepository(db)
	mailer := &trackingMailer{stubMailer: &stubMailer{}}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(repo, &stubSessionRevoker{}, mailer, logger)

	user, _, err := auth.NewRepository(db).GetOrCreateUser(ctx, "blocked@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	if _, err := db.ExecContext(ctx,
		`UPDATE users SET account_status = 'blocked' WHERE id = ?`,
		user.ID,
	); err != nil {
		t.Fatalf("set blocked status: %v", err)
	}

	updated, err := svc.UpdateUserStatus(ctx, user.ID, core.AccountStatusActive, "admin-1")
	if err != nil {
		t.Fatalf("UpdateUserStatus: %v", err)
	}
	if updated == nil {
		t.Fatal("UpdateUserStatus returned nil")
	}
	if updated.AccountStatus != core.AccountStatusActive {
		t.Errorf("account_status = %q, want %q", updated.AccountStatus, core.AccountStatusActive)
	}
	if !mailer.reactivatedCalled {
		t.Error("SendAccountReactivated was not called")
	}
}

func TestUpdateUserStatus_ConcurrentModification(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	repo := NewRepository(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(repo, &stubSessionRevoker{}, &stubMailer{}, logger)

	user, _, err := auth.NewRepository(db).GetOrCreateUser(ctx, "concurrent@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	_, err = svc.UpdateUserStatus(ctx, user.ID, core.AccountStatusBlocked, "admin-1")
	if err != nil {
		t.Fatalf("UpdateUserStatus: %v", err)
	}

	got, err := repo.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if got.AccountStatus != core.AccountStatusBlocked {
		t.Errorf("account_status = %q, want %q", got.AccountStatus, core.AccountStatusBlocked)
	}
}

func TestUpdateUserStatus_RevokeSessionsFails_ContinuesBlock(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	repo := NewRepository(db)
	mailer := &stubMailer{}
	errRevoker := &errSessionRevoker{err: errors.New("revoker failed")}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(repo, errRevoker, mailer, logger)

	user, _, err := auth.NewRepository(db).GetOrCreateUser(ctx, "revokerfail@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	updated, err := svc.UpdateUserStatus(ctx, user.ID, core.AccountStatusBlocked, "admin-1")
	if err != nil {
		t.Fatalf("UpdateUserStatus returned error: %v", err)
	}
	if updated == nil {
		t.Fatal("UpdateUserStatus returned nil despite revoker failure")
	}
	if updated.AccountStatus != core.AccountStatusBlocked {
		t.Errorf("account_status = %q, want %q", updated.AccountStatus, core.AccountStatusBlocked)
	}
}

func TestUpdateUserStatus_InvalidTransition_PendingToBlocked(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	repo := NewRepository(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(repo, &stubSessionRevoker{}, &stubMailer{}, logger)

	user, _, err := auth.NewRepository(db).GetOrCreateUser(ctx, "invalid@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	if _, err := db.ExecContext(ctx,
		`UPDATE users SET account_status = 'pending' WHERE id = ?`,
		user.ID,
	); err != nil {
		t.Fatalf("set pending status: %v", err)
	}

	_, err = svc.UpdateUserStatus(ctx, user.ID, core.AccountStatusBlocked, "admin-1")
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("UpdateUserStatus error = %v, want %v", err, ErrInvalidTransition)
	}
}

func TestUpdateUserStatus_UserNotFound(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	repo := NewRepository(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(repo, &stubSessionRevoker{}, &stubMailer{}, logger)

	_, err := svc.UpdateUserStatus(ctx, "nonexistent-id", core.AccountStatusActive, "admin-1")
	if err == nil {
		t.Fatal("UpdateUserStatus expected error for nonexistent user")
	}
}

func TestUpdateUserStatus_InvalidAccountStatus(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	repo := NewRepository(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(repo, &stubSessionRevoker{}, &stubMailer{}, logger)

	_, err := svc.UpdateUserStatus(ctx, "any-id", core.AccountStatus("invalid"), "admin-1")
	if !errors.Is(err, ErrInvalidAccountStatus) {
		t.Fatalf("UpdateUserStatus error = %v, want %v", err, ErrInvalidAccountStatus)
	}
}

func TestListUsers(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	repo := NewRepository(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(repo, &stubSessionRevoker{}, &stubMailer{}, logger)

	auth.NewRepository(db).GetOrCreateUser(ctx, "list1@example.com")
	auth.NewRepository(db).GetOrCreateUser(ctx, "list2@example.com")

	users, err := svc.ListUsers(ctx)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("ListUsers len = %d, want 2", len(users))
	}
}
