package admin

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"lucor.dev/beebuzz/internal/core"
)

var (
	ErrInvalidTransition      = fmt.Errorf("invalid status transition")
	ErrConcurrentModification = fmt.Errorf("user status was modified by another request")
	ErrInvalidAccountStatus   = fmt.Errorf("invalid account status value")
)

type SessionRevoker interface {
	RevokeAllSessions(ctx context.Context, userID string) error
}

type Mailer interface {
	SendAccountApproved(ctx context.Context, to string) error
	SendAccountBlocked(ctx context.Context, to string) error
	SendAccountReactivated(ctx context.Context, to string) error
}

// User represents a user in the admin domain.
type User struct {
	ID             string
	Email          string
	IsAdmin        bool
	AccountStatus  core.AccountStatus
	SignupReason   *string
	TrialStartedAt *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Service provides admin business logic.
type Service struct {
	repo           *Repository
	sessionRevoker SessionRevoker
	mailer         Mailer
	log            *slog.Logger
}

// NewService creates a new admin service.
func NewService(repo *Repository, sessionRevoker SessionRevoker, mailer Mailer, logger *slog.Logger) *Service {
	return &Service{
		repo:           repo,
		sessionRevoker: sessionRevoker,
		mailer:         mailer,
		log:            logger,
	}
}

// ListUsers retrieves all users.
func (s *Service) ListUsers(ctx context.Context) ([]User, error) {
	users, err := s.repo.GetAllUsers(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]User, len(users))
	for i, u := range users {
		var trialStartedAt *time.Time
		if u.TrialStartedAt != nil {
			t := time.UnixMilli(*u.TrialStartedAt)
			trialStartedAt = &t
		}
		result[i] = User{
			ID:             u.ID,
			Email:          u.Email,
			IsAdmin:        u.IsAdmin,
			AccountStatus:  u.AccountStatus,
			SignupReason:   u.SignupReason,
			TrialStartedAt: trialStartedAt,
			CreatedAt:      time.UnixMilli(u.CreatedAt),
			UpdatedAt:      time.UnixMilli(u.UpdatedAt),
		}
	}
	return result, nil
}

// UpdateUserStatus updates a user's account status.
func (s *Service) UpdateUserStatus(ctx context.Context, userID string, targetStatus core.AccountStatus, adminID string) (*User, error) {
	validStatuses := map[core.AccountStatus]bool{core.AccountStatusPending: true, core.AccountStatusActive: true, core.AccountStatusBlocked: true}
	if !validStatuses[targetStatus] {
		return nil, ErrInvalidAccountStatus
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	fromStatus := user.AccountStatus

	validTransitions := map[core.AccountStatus]map[core.AccountStatus]bool{
		core.AccountStatusPending: {core.AccountStatusActive: true},
		core.AccountStatusActive:  {core.AccountStatusBlocked: true},
		core.AccountStatusBlocked: {core.AccountStatusActive: true},
	}

	if !validTransitions[fromStatus][targetStatus] {
		return nil, ErrInvalidTransition
	}

	updated, err := s.repo.UpdateAccountStatus(ctx, userID, fromStatus, targetStatus)
	if err != nil {
		return nil, err
	}
	if !updated {
		return nil, ErrConcurrentModification
	}

	if targetStatus == core.AccountStatusBlocked {
		if err := s.sessionRevoker.RevokeAllSessions(ctx, userID); err != nil {
			s.log.Warn("failed to revoke sessions on block", "user_id", userID, "error", err)
		}
	}

	switch targetStatus {
	case core.AccountStatusActive:
		if fromStatus == core.AccountStatusPending {
			if err := s.mailer.SendAccountApproved(ctx, user.Email); err != nil {
				s.log.Error("failed to send account approved email", "user_id", userID, "error", err)
			}
		} else {
			if err := s.mailer.SendAccountReactivated(ctx, user.Email); err != nil {
				s.log.Error("failed to send account reactivated email", "user_id", userID, "error", err)
			}
		}
	case core.AccountStatusBlocked:
		if err := s.mailer.SendAccountBlocked(ctx, user.Email); err != nil {
			s.log.Error("failed to send account blocked email", "user_id", userID, "error", err)
		}
	}

	s.log.Info("user status changed",
		"user_id", userID,
		"from_status", fromStatus,
		"to_status", targetStatus,
		"admin_id", adminID,
	)

	updatedUser, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	var trialStartedAt *time.Time
	if updatedUser.TrialStartedAt != nil {
		t := time.UnixMilli(*updatedUser.TrialStartedAt)
		trialStartedAt = &t
	}

	return &User{
		ID:             updatedUser.ID,
		Email:          updatedUser.Email,
		IsAdmin:        updatedUser.IsAdmin,
		AccountStatus:  updatedUser.AccountStatus,
		SignupReason:   updatedUser.SignupReason,
		TrialStartedAt: trialStartedAt,
		CreatedAt:      time.UnixMilli(updatedUser.CreatedAt),
		UpdatedAt:      time.UnixMilli(updatedUser.UpdatedAt),
	}, nil
}
