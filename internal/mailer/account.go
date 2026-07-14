package mailer

import (
	"context"
)

const (
	accountBlockedTmplName     = "account_blocked"
	accountBlockedSubject      = "Your BeeBuzz account has been suspended"
	accountReactivatedTmplName = "account_reactivated"
	accountReactivatedSubject  = "Your BeeBuzz account is active again"
)

type accountBlockedTemplateData struct {
	SupportEmail string
}

// SendAccountBlocked sends an account suspension notification email.
func (m *mailer) SendAccountBlocked(ctx context.Context, to string) error {
	return m.sendTemplate(ctx, to, accountBlockedTmplName, accountBlockedSubject, accountBlockedTemplateData{
		SupportEmail: m.replyTo,
	})
}

// SendAccountReactivated sends an account reactivation notification email.
func (m *mailer) SendAccountReactivated(ctx context.Context, to string) error {
	return m.sendTemplate(ctx, to, accountReactivatedTmplName, accountReactivatedSubject, nil)
}
