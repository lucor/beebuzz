// Package user manages user accounts and profile data.
package user

import (
	"time"

	"lucor.dev/beebuzz/internal/core"
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

type UserResponse struct {
	ID             string             `json:"id"`
	Email          string             `json:"email"`
	IsAdmin        bool               `json:"is_admin"`
	AccountStatus  core.AccountStatus `json:"account_status"`
	TrialStartedAt *time.Time         `json:"trial_started_at,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

func ToUserResponse(u *User) UserResponse {
	var trialStartedAt *time.Time
	if u.TrialStartedAt != nil {
		t := time.UnixMilli(*u.TrialStartedAt).UTC()
		trialStartedAt = &t
	}
	return UserResponse{
		ID:             u.ID,
		Email:          u.Email,
		IsAdmin:        u.IsAdmin,
		AccountStatus:  u.AccountStatus,
		TrialStartedAt: trialStartedAt,
		CreatedAt:      time.UnixMilli(u.CreatedAt).UTC(),
		UpdatedAt:      time.UnixMilli(u.UpdatedAt).UTC(),
	}
}
