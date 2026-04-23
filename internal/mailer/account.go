package mailer

import (
	"context"
)

const (
	accountApprovedTmplName    = "account_approved"
	accountApprovedSubject     = "Your BeeBuzz account has been approved"
	accountBlockedTmplName     = "account_blocked"
	accountBlockedSubject      = "Your BeeBuzz account has been restricted"
	accountReactivatedTmplName = "account_reactivated"
	accountReactivatedSubject  = "Your BeeBuzz account has been reactivated"
)

// SendAccountApproved sends an account approval notification email.
func (m *mailer) SendAccountApproved(ctx context.Context, to string) error {
	data := struct {
		LoginURL string
	}{
		LoginURL: m.siteURL + "/login",
	}
	return m.sendTemplate(ctx, to, accountApprovedTmplName, accountApprovedSubject, data)
}

// SendAccountBlocked sends an account blocked notification email.
func (m *mailer) SendAccountBlocked(ctx context.Context, to string) error {
	return m.sendTemplate(ctx, to, accountBlockedTmplName, accountBlockedSubject, nil)
}

// SendAccountReactivated sends an account reactivation notification email.
func (m *mailer) SendAccountReactivated(ctx context.Context, to string) error {
	return m.sendTemplate(ctx, to, accountReactivatedTmplName, accountReactivatedSubject, nil)
}
