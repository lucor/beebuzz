package mailer

import (
	"context"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	"lucor.dev/beebuzz/internal/secure"
)

// SMTPSender sends emails via SMTP.
type SMTPSender struct {
	address  string
	host     string
	user     string
	password string
	sender   string
	replyTo  string
}

// NewSMTPSender creates a new SMTP sender with the given configuration.
func NewSMTPSender(address, user, password, sender, replyTo string) (*SMTPSender, error) {
	if address == "" {
		return nil, fmt.Errorf("address is required")
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("invalid address %q: %w", address, err)
	}
	return &SMTPSender{
		address:  address,
		host:     host,
		user:     user,
		password: password,
		sender:   sender,
		replyTo:  replyTo,
	}, nil
}

// smtpMessage builds a multipart/alternative MIME email message.
type smtpMessage struct {
	from     string
	to       string
	replyTo  string
	subject  string
	text     string
	html     string
	boundary string
}

func (m smtpMessage) String() string {
	boundary := m.boundary
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("From: %s\r\n", m.from))
	sb.WriteString(fmt.Sprintf("To: %s\r\n", m.to))
	if m.replyTo != "" {
		sb.WriteString(fmt.Sprintf("Reply-To: %s\r\n", m.replyTo))
	}
	sb.WriteString(fmt.Sprintf("Subject: %s\r\n", m.subject))
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%s\r\n", boundary))
	sb.WriteString("\r\n")
	sb.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
	sb.WriteString(m.text)
	sb.WriteString("\r\n")
	sb.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	sb.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
	sb.WriteString(m.html)
	sb.WriteString("\r\n")
	sb.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	return sb.String()
}

// Send sends an email via SMTP.
func (s *SMTPSender) Send(ctx context.Context, to, subject, text, html string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	auth := smtp.PlainAuth("", s.user, s.password, s.host)
	msg := smtpMessage{
		from:     s.sender,
		to:       to,
		replyTo:  s.replyTo,
		subject:  subject,
		text:     text,
		html:     html,
		boundary: secure.MustRandomHex(16),
	}
	return smtp.SendMail(s.address, auth, s.sender, []string{to}, []byte(msg.String()))
}
