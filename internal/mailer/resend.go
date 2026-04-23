package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const (
	resendAPIURL        = "https://api.resend.com/emails"
	resendClientTimeout = 10 * time.Second
)

// ResendSender sends emails via the Resend API.
type ResendSender struct {
	apiKey  string
	sender  string
	replyTo string
	client  *http.Client
}

// NewResendSender creates a new Resend sender with the given API key.
// Returns an error if the API key is empty.
func NewResendSender(apiKey, sender, replyTo string) (*ResendSender, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}
	return &ResendSender{
		apiKey:  apiKey,
		sender:  sender,
		replyTo: replyTo,
		client:  &http.Client{Timeout: resendClientTimeout},
	}, nil
}

// resendRequest represents a Resend API email request.
type resendRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
	Text    string   `json:"text,omitempty"`
	ReplyTo string   `json:"reply_to,omitempty"`
}

// resendError represents the error payload returned by the Resend API.
type resendError struct {
	Name       string `json:"name"`
	Message    string `json:"message"`
	StatusCode int    `json:"statusCode"`
}

// resendResponse represents a successful Resend API response.
type resendResponse struct {
	ID string `json:"id"`
}

// Send sends an email via the Resend API.
func (s *ResendSender) Send(ctx context.Context, to, subject, text, html string) error {
	req := resendRequest{
		From:    s.sender,
		To:      []string{to},
		Subject: subject,
		HTML:    html,
		Text:    text,
		ReplyTo: s.replyTo,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, resendAPIURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiKey))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr resendError
		if err := json.Unmarshal(respBody, &apiErr); err == nil && apiErr.Message != "" {
			return &ResendError{StatusCode: resp.StatusCode, Name: apiErr.Name, Message: apiErr.Message}
		}
		return &ResendError{StatusCode: resp.StatusCode, Message: string(respBody)}
	}

	var respData resendResponse
	if err := json.Unmarshal(respBody, &respData); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	slog.Info("email sent", "provider", "resend")
	return nil
}

// ResendError represents an error returned by the Resend API.
type ResendError struct {
	StatusCode int
	Name       string
	Message    string
}

func (e *ResendError) Error() string {
	if e.Name != "" {
		return fmt.Sprintf("resend API error %d %s: %s", e.StatusCode, e.Name, e.Message)
	}
	return fmt.Sprintf("resend API error %d: %s", e.StatusCode, e.Message)
}
