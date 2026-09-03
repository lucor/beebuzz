package mailer

import (
	"context"
	"strings"
	"time"
)

const (
	hostedActivatedTmplName             = "hosted_activated"
	hostedActivatedSubject              = "Your BeeBuzz Hosted plan is active"
	hostedCancellationScheduledTmplName = "hosted_cancellation_scheduled"
	hostedCancellationScheduledSubject  = "Your BeeBuzz Hosted plan is scheduled to end"
	hostedResumedTmplName               = "hosted_resumed"
	hostedResumedSubject                = "Your BeeBuzz Hosted plan is active again"
	hostedPaymentIssueTmplName          = "hosted_payment_issue"
	hostedPaymentIssueSubject           = "There was a problem with your BeeBuzz Hosted payment"
	hostedEndedTmplName                 = "hosted_ended"
	hostedEndedSubject                  = "Your BeeBuzz Hosted plan has ended"
)

type hostedTemplateData struct {
	BillingURL   string
	Date         string
	SupportEmail string
}

// SendHostedActivated sends a product notice when Hosted entitlements become active.
func (m *mailer) SendHostedActivated(ctx context.Context, to string) error {
	return m.sendTemplate(ctx, to, hostedActivatedTmplName, hostedActivatedSubject, hostedTemplateData{
		BillingURL:   m.billingURL(),
		SupportEmail: m.replyTo,
	})
}

// SendHostedCancellationScheduled confirms that Hosted will end after the paid period.
func (m *mailer) SendHostedCancellationScheduled(ctx context.Context, to string, accessUntil time.Time) error {
	return m.sendTemplate(ctx, to, hostedCancellationScheduledTmplName, hostedCancellationScheduledSubject, hostedTemplateData{
		BillingURL:   m.billingURL(),
		Date:         formatHostedDate(accessUntil),
		SupportEmail: m.replyTo,
	})
}

// SendHostedResumed confirms that a scheduled cancellation was reversed.
func (m *mailer) SendHostedResumed(ctx context.Context, to string, renewsAt time.Time) error {
	return m.sendTemplate(ctx, to, hostedResumedTmplName, hostedResumedSubject, hostedTemplateData{
		BillingURL:   m.billingURL(),
		Date:         formatHostedDate(renewsAt),
		SupportEmail: m.replyTo,
	})
}

// SendHostedPaymentIssue asks the customer to update their payment method.
func (m *mailer) SendHostedPaymentIssue(ctx context.Context, to string, accessUntil time.Time) error {
	return m.sendTemplate(ctx, to, hostedPaymentIssueTmplName, hostedPaymentIssueSubject, hostedTemplateData{
		BillingURL:   m.billingURL(),
		Date:         formatHostedDate(accessUntil),
		SupportEmail: m.replyTo,
	})
}

// SendHostedEnded sends a product notice when Hosted entitlements end.
func (m *mailer) SendHostedEnded(ctx context.Context, to string) error {
	return m.sendTemplate(ctx, to, hostedEndedTmplName, hostedEndedSubject, hostedTemplateData{
		BillingURL:   m.billingURL(),
		SupportEmail: m.replyTo,
	})
}

func (m *mailer) billingURL() string {
	return strings.TrimRight(m.siteURL, "/") + "/account/billing"
}

func formatHostedDate(value time.Time) string {
	return value.UTC().Format("2 Jan 2006")
}
