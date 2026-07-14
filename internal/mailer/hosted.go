package mailer

import "context"

const (
	hostedActivatedTmplName = "hosted_activated"
	hostedActivatedSubject  = "Hosted is now active on BeeBuzz"
	hostedEndedTmplName     = "hosted_ended"
	hostedEndedSubject      = "Your BeeBuzz plan is back to Free"
)

type hostedTemplateData struct {
	DashboardURL string
}

// SendHostedActivated sends a product notice when Hosted entitlements become active.
func (m *mailer) SendHostedActivated(ctx context.Context, to string) error {
	return m.sendTemplate(ctx, to, hostedActivatedTmplName, hostedActivatedSubject, hostedTemplateData{
		DashboardURL: m.siteURL,
	})
}

// SendHostedEnded sends a product notice when Hosted entitlements end.
func (m *mailer) SendHostedEnded(ctx context.Context, to string) error {
	return m.sendTemplate(ctx, to, hostedEndedTmplName, hostedEndedSubject, hostedTemplateData{
		DashboardURL: m.siteURL,
	})
}
