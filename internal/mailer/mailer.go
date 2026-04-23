// Package mailer provides an abstraction for sending HTML emails
// It supports multiple sending backends (SMTP, API-based services),
// asynchronous sending, retry logic, and fallback to logging when no email provider is configured.
package mailer

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"text/template"

	"lucor.dev/beebuzz/internal/config"
	"lucor.dev/beebuzz/internal/validator"
)

//go:embed templates/*.html.tmpl templates/*.txt.tmpl
var templates embed.FS

// Transport defines the interface for sending emails via a backend service.
type Transport interface {
	Send(ctx context.Context, to, subject, text, html string) error
}

// Mailer defines the interface for sending templated emails.
type Mailer interface {
	SendRequestAuth(ctx context.Context, to, otp string) error
	SendAccountApproved(ctx context.Context, to string) error
	SendAccountBlocked(ctx context.Context, to string) error
	SendAccountReactivated(ctx context.Context, to string) error
}

// mailer sends emails using a configured transport backend.
type mailer struct {
	transport Transport
	templates *template.Template
	siteURL   string
}

// New creates a new Mailer instance based on the provided config.
// Priority: SMTP > Resend > no-op fallback.
func New(cfg *config.Mailer) (Mailer, error) {
	errs := errors.Join(
		validator.NotBlank("sender", cfg.Sender),
		validator.Email("sender", cfg.Sender),

		validator.NotBlank("reply_to", cfg.ReplyTo),
		validator.Email("reply_to", cfg.ReplyTo),
	)

	if errs != nil {
		return nil, fmt.Errorf("invalid conf for mailer: %w", errs)
	}

	// Parse templates
	tmpl, err := template.ParseFS(templates, "templates/*.html.tmpl", "templates/*.txt.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	transport, err := newTransport(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create transport: %w", err)
	}

	return &mailer{
		transport: transport,
		templates: tmpl,
		siteURL:   cfg.SiteURL,
	}, nil
}

// sendTemplate renders and sends a templated email.
func (m *mailer) sendTemplate(ctx context.Context, to, tmplName, subject string, data any) error {
	htmlTmpl, txtTmpl, err := m.lookupTemplates(tmplName)
	if err != nil {
		return err
	}

	var htmlBuf bytes.Buffer
	if err := htmlTmpl.Execute(&htmlBuf, data); err != nil {
		return fmt.Errorf("failed to render HTML template: %w", err)
	}

	var textBuf bytes.Buffer
	if err := txtTmpl.Execute(&textBuf, data); err != nil {
		return fmt.Errorf("failed to render text template: %w", err)
	}

	return m.transport.Send(ctx, to, subject, textBuf.String(), htmlBuf.String())
}

// lookupTemplates returns the HTML and text templates for a given base name.
// name should be without the .html.tmpl or .txt.tmpl extension (e.g., "send_request_auth").
func (m *mailer) lookupTemplates(name string) (*template.Template, *template.Template, error) {
	htmlTmplName := name + ".html.tmpl"
	txtTmplName := name + ".txt.tmpl"

	htmlTmpl := m.templates.Lookup(htmlTmplName)
	if htmlTmpl == nil {
		return nil, nil, fmt.Errorf("template %s not found", htmlTmplName)
	}

	txtTmpl := m.templates.Lookup(txtTmplName)
	if txtTmpl == nil {
		return nil, nil, fmt.Errorf("template %s not found", txtTmplName)
	}

	return htmlTmpl, txtTmpl, nil
}

func newTransport(cfg *config.Mailer) (Transport, error) {
	if cfg.SMTPAddress != "" {
		return NewSMTPSender(cfg.SMTPAddress, cfg.SMTPUser, cfg.SMTPPassword, cfg.Sender, cfg.ReplyTo)
	}
	if cfg.ResendAPIKey != "" {
		return NewResendSender(cfg.ResendAPIKey, cfg.Sender, cfg.ReplyTo)
	}
	// Fallback to no-op transport
	return NewNoOpTransport(cfg.Sender, cfg.ReplyTo)
}
