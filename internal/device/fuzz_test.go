package device

import (
	"strings"
	"testing"
	"unicode/utf8"

	"lucor.dev/beebuzz/internal/validator"
)

// FuzzPairRequestValidate verifies that PairRequest.Validate never panics
// and rejects requests with missing required fields.
func FuzzPairRequestValidate(f *testing.F) {
	f.Add("123456", "https://push.example.com/sub", "base64key", "authkey", "age1recipient")
	f.Add("", "", "", "", "")
	f.Add("code", "https://push.example.com", "", "auth", "age1key")
	f.Add(strings.Repeat("9", 100), strings.Repeat("a", 1000), strings.Repeat("b", 500), strings.Repeat("c", 500), strings.Repeat("d", 500))
	f.Add("\x00\xff", "not-a-url", "\x00", "\xff", "\x00\x01")
	f.Add("123456", "https://fcm.googleapis.com/fcm/send/abc", "BNcR...", "auth123", "age1qyqsz...")
	f.Add("", "https://push.example.com/sub", "base64key", "authkey", "age1recipient")   // isolated empty pairing code
	f.Add("123456", "", "base64key", "authkey", "age1recipient")                          // isolated empty endpoint
	f.Add("123456", "https://push.example.com/sub", "base64key", "", "age1recipient")     // isolated empty auth
	f.Add("123456", "https://push.example.com/sub", "base64key", "authkey", "")           // isolated empty age recipient
	f.Add("1", "h", "k", "a", "r")                                                       // minimal non-empty values
	f.Add(" ", " ", " ", " ", " ")                                                        // whitespace-only values

	f.Fuzz(func(t *testing.T, pairingCode, endpoint, p256dh, auth, ageRecipient string) {
		req := PairRequest{
			PairingCode:  pairingCode,
			Endpoint:     endpoint,
			P256dh:       p256dh,
			Auth:         auth,
			AgeRecipient: ageRecipient,
		}

		// Must never panic.
		errs := req.Validate()

		// All fields are required — if any is empty, must produce errors.
		hasEmpty := pairingCode == "" || endpoint == "" || p256dh == "" || auth == "" || ageRecipient == ""
		if hasEmpty && len(errs) == 0 {
			t.Fatal("Validate accepted request with empty required field(s)")
		}
	})
}

// FuzzCreateDeviceRequestValidate verifies that CreateDeviceRequest.Validate never panics.
func FuzzCreateDeviceRequestValidate(f *testing.F) {
	f.Add("My Device", "A description")
	f.Add("", "")
	f.Add(strings.Repeat("a", 65), strings.Repeat("b", 129))
	f.Add("Device", strings.Repeat("x", 128))
	f.Add("\x00", "\xff")
	f.Add("", "desc")                                                                       // isolated empty name
	f.Add(strings.Repeat("a", validator.MaxDisplayNameLen), "")                              // exact name boundary
	f.Add(strings.Repeat("a", validator.MaxDisplayNameLen+1), "")                            // name overflow isolated
	f.Add(strings.Repeat("é", validator.MaxDisplayNameLen), "")                              // multibyte name exact
	f.Add(strings.Repeat("é", validator.MaxDisplayNameLen+1), "")                            // multibyte name overflow
	f.Add("Device", strings.Repeat("é", validator.MaxDescriptionLen))                        // multibyte desc exact
	f.Add("Device", strings.Repeat("é", validator.MaxDescriptionLen+1))                      // multibyte desc overflow

	f.Fuzz(func(t *testing.T, name, description string) {
		req := CreateDeviceRequest{
			Name:        name,
			Description: description,
			Topics:      []string{"topic1"},
		}

		// Must never panic.
		errs := req.Validate()

		// Empty name must produce at least one error.
		if name == "" && len(errs) == 0 {
			t.Fatal("Validate accepted empty name")
		}
	})
}

// FuzzUpdateDeviceRequestValidate verifies that UpdateDeviceRequest.Validate never panics
// and enforces required name and length limits.
func FuzzUpdateDeviceRequestValidate(f *testing.F) {
	f.Add("My Device", "A description")
	f.Add("", "")
	f.Add("", "description")
	f.Add("d", "")
	f.Add(strings.Repeat("a", validator.MaxDisplayNameLen+1), strings.Repeat("b", validator.MaxDescriptionLen+1))
	f.Add(strings.Repeat("a", validator.MaxDisplayNameLen), strings.Repeat("b", validator.MaxDescriptionLen))
	f.Add("\x00\xff", "\x00\xff")
	f.Add("device", strings.Repeat("\u00e9", validator.MaxDescriptionLen+1))
	f.Add(strings.Repeat("a", validator.MaxDisplayNameLen+1), "ok")                      // isolated name overflow
	f.Add(strings.Repeat("é", validator.MaxDisplayNameLen), "desc")                      // multibyte name exact
	f.Add(strings.Repeat("é", validator.MaxDisplayNameLen+1), "desc")                    // multibyte name overflow
	f.Add("device", strings.Repeat("é", validator.MaxDescriptionLen))                    // multibyte desc exact

	f.Fuzz(func(t *testing.T, name, description string) {
		req := UpdateDeviceRequest{
			Name:        name,
			Description: description,
			Topics:      []string{"topic1"},
		}

		errs := req.Validate()

		if name == "" && len(errs) == 0 {
			t.Fatal("Validate accepted empty name")
		}

		if utf8.RuneCountInString(name) > validator.MaxDisplayNameLen && len(errs) == 0 {
			t.Fatalf("Validate accepted name with %d runes (max %d)",
				utf8.RuneCountInString(name), validator.MaxDisplayNameLen)
		}

		if utf8.RuneCountInString(description) > validator.MaxDescriptionLen && len(errs) == 0 {
			t.Fatalf("Validate accepted description with %d runes (max %d)",
				utf8.RuneCountInString(description), validator.MaxDescriptionLen)
		}
	})
}
