package notification

import (
	"strings"
	"testing"

	"filippo.io/age"
)

// sharedAgeRecipient is a pre-generated age X25519 public key used across fuzz
// iterations to avoid expensive key generation on every call.
var sharedAgeRecipient string

func init() {
	id, err := age.GenerateX25519Identity()
	if err != nil {
		panic("failed to generate age identity for fuzz tests: " + err.Error())
	}
	sharedAgeRecipient = id.Recipient().String()
}

// FuzzEncryptForRecipients verifies that encryptForRecipients never panics
// and returns an error for invalid recipients.
func FuzzEncryptForRecipients(f *testing.F) {
	f.Add([]byte("hello world"), "age1invalidrecipient")
	f.Add([]byte(""), "age1invalidrecipient")
	f.Add([]byte("data"), "")
	f.Add([]byte("data"), "not-an-age-key")
	f.Add([]byte(strings.Repeat("x", 10000)), "age1abc")
	f.Add([]byte("\x00\x01\x02\xff"), "garbage")
	f.Add([]byte("hello world"), sharedAgeRecipient)
	f.Add([]byte(""), sharedAgeRecipient)
	f.Add([]byte("data"), sharedAgeRecipient[:len(sharedAgeRecipient)-1]+"x") // near-valid, broken last char

	f.Fuzz(func(t *testing.T, data []byte, recipient string) {
		if recipient == "" {
			return
		}

		// Must never panic.
		result, err := encryptForRecipients(data, []string{recipient})

		// Invalid recipient must produce an error.
		_, parseErr := age.ParseX25519Recipient(recipient)
		if parseErr != nil && err == nil {
			t.Fatalf("encryptForRecipients succeeded with invalid recipient %q", recipient)
		}

		// Valid recipient with data must produce non-empty ciphertext.
		if err == nil && len(result) == 0 {
			t.Fatal("encryptForRecipients returned empty ciphertext without error")
		}
	})
}

// FuzzEncryptForRecipientsValid verifies the encrypt path with a real age key
// and arbitrary plaintext data.
func FuzzEncryptForRecipientsValid(f *testing.F) {
	f.Add([]byte("hello"))
	f.Add([]byte(""))
	f.Add([]byte(strings.Repeat("a", 100000)))
	f.Add([]byte("\x00\xff"))
	f.Add([]byte("a")) // minimal non-empty plaintext

	f.Fuzz(func(t *testing.T, data []byte) {
		// Must never panic.
		result, err := encryptForRecipients(data, []string{sharedAgeRecipient})
		if err != nil {
			t.Fatalf("encryptForRecipients failed with valid recipient: %v", err)
		}

		if len(result) == 0 {
			t.Fatal("encryptForRecipients returned empty ciphertext")
		}
	})
}

// FuzzSendRequestValidate verifies that SendRequest.Validate never panics.
func FuzzSendRequestValidate(f *testing.F) {
	f.Add("Title", "Body", "", "")
	f.Add("", "", "", "")
	f.Add("Title", "Body", "high", "https://example.com/image.png")
	f.Add("Title", "Body", "invalid", "not-a-url")
	f.Add(strings.Repeat("t", MaxNotificationTitleLen+1), strings.Repeat("b", MaxNotificationBodyLen+1), "normal", "http://example.com")
	f.Add("\x00", "\xff", "\x00", "ftp://evil.com")
	f.Add("", "Body", "", "")                                    // isolated empty title
	f.Add("Title", "", "", "")                                   // isolated empty body
	f.Add("Title", "Body", "normal", "")                         // valid priority, no attachment
	f.Add("Title", "Body", "", "https://example.com/image.png")  // empty priority (valid), with attachment
	f.Add("Title", "Body", "high", "https://127.0.0.1/img.png") // loopback attachment URL
	f.Add("Title", "Body", "high", "https://[::1]/img.png")     // IPv6 loopback attachment
	f.Add("Title", "Body", "high", "https://8.8.8.8/img.png")   // public IP attachment

	f.Fuzz(func(t *testing.T, title, body, priority, attachmentURL string) {
		req := SendRequest{
			Title:         title,
			Body:          body,
			Priority:      priority,
			AttachmentURL: attachmentURL,
		}

		// Must never panic.
		errs := req.Validate()

		// Empty title must produce at least one error.
		if title == "" && len(errs) == 0 {
			t.Fatal("Validate accepted empty title")
		}
		if len(title) > MaxNotificationTitleLen && len(errs) == 0 {
			t.Fatal("Validate accepted title above max length")
		}
		if len(body) > MaxNotificationBodyLen && len(errs) == 0 {
			t.Fatal("Validate accepted body above max length")
		}

	})
}
