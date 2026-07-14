// Package user manages user accounts and profile data.
package user

import (
	"time"

	"go.beebuzz.app/beebuzz/internal/billing"
	"go.beebuzz.app/beebuzz/internal/core"
)

type User struct {
	ID                 string                      `db:"id"`
	Email              string                      `db:"email"`
	IsAdmin            bool                        `db:"is_admin"`
	AccountStatus      core.AccountStatus          `db:"account_status"`
	Plan               core.Plan                   `db:"plan"`
	PlanExpiresAt      *int64                      `db:"plan_expires_at"`
	SubscriptionStatus *billing.SubscriptionStatus `db:"subscription_status"`
	CreatedAt          int64                       `db:"created_at"`
	UpdatedAt          int64                       `db:"updated_at"`
}

type UserResponse struct {
	ID                 string                      `json:"id"`
	Email              string                      `json:"email"`
	IsAdmin            bool                        `json:"is_admin"`
	AccountStatus      core.AccountStatus          `json:"account_status"`
	Plan               core.Plan                   `json:"plan"`
	PlanExpiresAt      *time.Time                  `json:"plan_expires_at,omitempty"`
	SubscriptionStatus *billing.SubscriptionStatus `json:"subscription_status"`
	CreatedAt          time.Time                   `json:"created_at"`
	UpdatedAt          time.Time                   `json:"updated_at"`
}

func ToUserResponse(u *User) UserResponse {
	var planExpiresAt *time.Time
	if u.PlanExpiresAt != nil {
		t := time.UnixMilli(*u.PlanExpiresAt).UTC()
		planExpiresAt = &t
	}
	return UserResponse{
		ID:                 u.ID,
		Email:              u.Email,
		IsAdmin:            u.IsAdmin,
		AccountStatus:      u.AccountStatus,
		Plan:               u.Plan,
		PlanExpiresAt:      planExpiresAt,
		SubscriptionStatus: u.SubscriptionStatus,
		CreatedAt:          time.UnixMilli(u.CreatedAt).UTC(),
		UpdatedAt:          time.UnixMilli(u.UpdatedAt).UTC(),
	}
}
