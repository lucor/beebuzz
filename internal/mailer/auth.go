package mailer

import (
	"context"
)

const (
	sendRequestAuthTmplName = "send_request_auth"
	sendRequestAuthSubject  = "Your BeeBuzz verification code"
)

// SendRequestAuth sends an authentication request email with an OTP code.
func (m *mailer) SendRequestAuth(ctx context.Context, to, otp string) error {
	data := struct {
		OTP string
	}{
		OTP: otp,
	}
	return m.sendTemplate(ctx, to, sendRequestAuthTmplName, sendRequestAuthSubject, data)
}
