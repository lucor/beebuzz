package auth

import (
	"strings"
	"testing"
)

// FuzzLoginRequestValidate verifies that LoginRequest.Validate never panics
// and correctly rejects empty required fields.
func FuzzLoginRequestValidate(f *testing.F) {
	f.Add("user@example.com", "state123", "")
	f.Add("", "", "")
	f.Add("user@example.com", "", "reason")
	f.Add("not-email", "state", "")
	f.Add(strings.Repeat("a", 1000) + "@example.com", strings.Repeat("b", 1000), "")
	f.Add("\x00@\x00", "\xff", "")
	f.Add("日本語@example.com", "state", "reason")
	f.Add("", "state123", "")                                 // isolated empty email
	f.Add("user@", "state123", "")                            // missing domain
	f.Add("@example.com", "state123", "")                     // missing local part
	f.Add("user+tag@example.com", "state123", "reason")       // valid tagged address
	f.Add("\"Alice Example\" <user@example.com>", "state", "") // display-name form

	f.Fuzz(func(t *testing.T, email, state, reason string) {
		req := LoginRequest{
			Email: email,
			State: state,
		}
		if reason != "" {
			req.Reason = &reason
		}

		// Must never panic.
		errs := req.Validate()

		// Empty email must produce at least one error.
		if email == "" && len(errs) == 0 {
			t.Fatal("Validate accepted empty email")
		}

		// Empty state must produce at least one error.
		if state == "" && len(errs) == 0 {
			t.Fatal("Validate accepted empty state")
		}
	})
}

// FuzzVerifyOTPRequestValidate verifies that VerifyOTPRequest.Validate never panics
// and correctly rejects empty required fields.
func FuzzVerifyOTPRequestValidate(f *testing.F) {
	f.Add("123456", "state123")
	f.Add("", "")
	f.Add("123456", "")
	f.Add("", "state123")
	f.Add(strings.Repeat("9", 1000), strings.Repeat("x", 1000))
	f.Add("\x00\xff", "state")
	f.Add("abc", "state")
	f.Add(" ", "state123")   // whitespace-only OTP
	f.Add("0", "state123")   // smallest non-empty OTP
	f.Add("000000", "state") // leading zeros
	f.Add("123456", " ")     // whitespace-only state

	f.Fuzz(func(t *testing.T, otp, state string) {
		req := VerifyOTPRequest{
			OTP:   otp,
			State: state,
		}

		// Must never panic.
		errs := req.Validate()

		// Empty OTP must produce at least one error.
		if otp == "" && len(errs) == 0 {
			t.Fatal("Validate accepted empty OTP")
		}

		// Empty state must produce at least one error.
		if state == "" && len(errs) == 0 {
			t.Fatal("Validate accepted empty state")
		}
	})
}
