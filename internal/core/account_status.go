package core

// AccountStatus represents the admin moderation state of a user account.
type AccountStatus string

const (
	AccountStatusPending AccountStatus = "pending"
	AccountStatusActive  AccountStatus = "active"
	AccountStatusBlocked AccountStatus = "blocked"
)

// IsValid reports whether s is a recognised account status value.
func (s AccountStatus) IsValid() bool {
	switch s {
	case AccountStatusPending, AccountStatusActive, AccountStatusBlocked:
		return true
	}
	return false
}
