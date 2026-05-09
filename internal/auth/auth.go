// Package auth handles user authentication via OTP and session management.
package auth

import (
	"fmt"
	"time"

	"lucor.dev/beebuzz/internal/core"
	"lucor.dev/beebuzz/internal/validator"
)

type User struct {
	ID             string             `db:"id"`
	Email          string             `db:"email"`
	IsAdmin        bool               `db:"is_admin"`
	AccountStatus  core.AccountStatus `db:"account_status"`
	TrialStartedAt *int64             `db:"trial_started_at"`
	CreatedAt      int64              `db:"created_at"`
	UpdatedAt      int64              `db:"updated_at"`
}

type CreateUserOptions struct {
	AccountStatus      core.AccountStatus
	SignupReason       *string
	StartTrialOnCreate bool
}

const (
	TrialDuration = 14 * 24 * time.Hour
)

type Session struct {
	Token     string
	ExpiresAt time.Time
}

type LoginRequest struct {
	Email  string  `json:"email"`
	State  string  `json:"state"`
	Reason *string `json:"reason,omitempty"`
	// Keep in sync with web/packages/shared/src/constants/auth.ts.
	ReferralCode string `json:"referral_code,omitempty"`
}

// Validate enforces login-style email input and required request fields.
func (r *LoginRequest) Validate() []error {
	return validator.Validate(
		validator.NotBlank("email", r.Email),
		validator.PlainEmail("email", r.Email),
		validator.NotBlank("state", r.State),
	)
}

type MessageResponse struct {
	Message string `json:"message"`
}

type VerifyOTPRequest struct {
	OTP   string `json:"otp"`
	State string `json:"state"`
}

func (r *VerifyOTPRequest) Validate() []error {
	return validator.Validate(
		validator.NotBlank("otp", r.OTP),
		validator.NotBlank("state", r.State),
	)
}

var (
	ErrOTPNotFound     = fmt.Errorf("otp challenge not found: %w", core.ErrUnauthorized)
	ErrOTPExpired      = fmt.Errorf("otp challenge expired: %w", core.ErrUnauthorized)
	ErrOTPUsed         = fmt.Errorf("otp challenge already used: %w", core.ErrUnauthorized)
	ErrOTPInvalid      = fmt.Errorf("invalid otp: %w", core.ErrUnauthorized)
	ErrOTPMaxAttempts  = fmt.Errorf("max otp attempts exceeded: %w", core.ErrUnauthorized)
	ErrGlobalRateLimit = fmt.Errorf("global auth rate limit exceeded")
)
