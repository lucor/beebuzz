package mailer

import (
	"context"
	"strings"
	"testing"
	"text/template"
	"time"
)

type captureTransport struct {
	subject string
	text    string
	html    string
}

func (t *captureTransport) Send(_ context.Context, _, subject, text, html string) error {
	t.subject = subject
	t.text = text
	t.html = html
	return nil
}

func TestHostedLifecycleEmails(t *testing.T) {
	parsed, err := template.ParseFS(templates, "templates/*.html.tmpl", "templates/*.txt.tmpl")
	if err != nil {
		t.Fatalf("ParseFS(): %v", err)
	}
	transport := &captureTransport{}
	m := &mailer{
		transport: transport,
		templates: parsed,
		siteURL:   "https://dashboard.example.com/",
		replyTo:   "hello@example.com",
	}
	periodEnd := time.Date(2027, 9, 3, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		send        func() error
		wantSubject string
		wantText    []string
	}{
		{
			name:        "activation",
			send:        func() error { return m.SendHostedActivated(context.Background(), "user@example.com") },
			wantSubject: hostedActivatedSubject,
			wantText:    []string{"Hosted plan active", "Your Hosted plan is now active", "Manage billing", "https://dashboard.example.com/account/billing"},
		},
		{
			name: "scheduled cancellation",
			send: func() error {
				return m.SendHostedCancellationScheduled(context.Background(), "user@example.com", periodEnd)
			},
			wantSubject: hostedCancellationScheduledSubject,
			wantText:    []string{"Hosted plan scheduled to end", "Your Hosted plan will end on 3 Sep 2027", "Manage subscription", "https://dashboard.example.com/account/billing"},
		},
		{
			name:        "resume",
			send:        func() error { return m.SendHostedResumed(context.Background(), "user@example.com", periodEnd) },
			wantSubject: hostedResumedSubject,
			wantText:    []string{"Hosted plan is active again", "Your Hosted plan will remain active", "renew on 3 Sep 2027", "Manage subscription", "https://dashboard.example.com/account/billing"},
		},
		{
			name:        "payment issue",
			send:        func() error { return m.SendHostedPaymentIssue(context.Background(), "user@example.com", periodEnd) },
			wantSubject: hostedPaymentIssueSubject,
			wantText:    []string{"Payment issue", "Hosted plan payment", "Your Hosted plan will remain active until 3 Sep 2027", "Update payment details", "https://dashboard.example.com/account/billing"},
		},
		{
			name:        "ended",
			send:        func() error { return m.SendHostedEnded(context.Background(), "user@example.com") },
			wantSubject: hostedEndedSubject,
			wantText:    []string{"Hosted plan ended", "Your Hosted plan has ended", "Free plan", "View your plan", "https://dashboard.example.com/account/billing"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.send(); err != nil {
				t.Fatalf("send email: %v", err)
			}
			if transport.subject != tt.wantSubject {
				t.Fatalf("subject = %q, want %q", transport.subject, tt.wantSubject)
			}
			for _, want := range tt.wantText {
				if !strings.Contains(transport.text, want) || !strings.Contains(transport.html, want) {
					t.Fatalf("rendered email missing %q", want)
				}
			}
		})
	}
}
