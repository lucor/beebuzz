package mailer

import (
	"context"
	"log/slog"
)

// NoOpTransport is a no-op transport that logs emails instead of sending them.
type NoOpTransport struct {
	sender  string
	replyTo string
}

// NewNoOpTransport creates a new no-op transport.
func NewNoOpTransport(sender, replyTo string) (*NoOpTransport, error) {
	return &NoOpTransport{
		sender:  sender,
		replyTo: replyTo,
	}, nil
}

// Send logs the email instead of sending it.
func (s *NoOpTransport) Send(_ context.Context, to, subject, text, html string) error {
	slog.Info("email logged (no transport configured)",
		"from", s.sender,
		"to", to,
		"reply_to", s.replyTo,
		"subject", subject,
		"text", text)

	return nil
}
