package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"filippo.io/age"
)

// sharedAgeRecipient is a pre-generated age X25519 public key used across fuzz
// iterations to avoid expensive key generation on every call.
var sharedCLIAgeRecipient string

func init() {
	id, err := age.GenerateX25519Identity()
	if err != nil {
		panic("failed to generate age identity for fuzz tests: " + err.Error())
	}
	sharedCLIAgeRecipient = id.Recipient().String()
}

// FuzzValidateProfileName verifies that validateProfileName never panics
// and enforces length and character constraints.
func FuzzValidateProfileName(f *testing.F) {
	f.Add("default")
	f.Add("")
	f.Add("a")
	f.Add("my-profile")
	f.Add("my_profile")
	f.Add("profile123")
	f.Add("UPPERCASE")
	f.Add("has space")
	f.Add("has.dot")
	f.Add("has/slash")
	f.Add("../traversal")
	f.Add(strings.Repeat("a", maxProfileNameLen))
	f.Add(strings.Repeat("a", maxProfileNameLen+1))
	f.Add("\x00null")
	f.Add("日本語")
	f.Add("-leading-dash")
	f.Add("_leading-underscore")

	f.Fuzz(func(t *testing.T, name string) {
		err := validateProfileName(name)

		// Empty name must always be rejected.
		if name == "" && err == nil {
			t.Fatal("validateProfileName accepted empty name")
		}

		// Names longer than max must always be rejected.
		if len(name) > maxProfileNameLen && err == nil {
			t.Fatalf("validateProfileName accepted name longer than %d chars: %q", maxProfileNameLen, name)
		}

		// Names with invalid characters must always be rejected.
		validChars := "abcdefghijklmnopqrstuvwxyz0123456789_-"
		for _, c := range name {
			if !strings.ContainsRune(validChars, c) {
				if err == nil {
					t.Fatalf("validateProfileName accepted name with invalid char %q: %q", string(c), name)
				}
				return
			}
		}
	})
}

// FuzzNormalizeAPIURL verifies that normalizeAPIURL never panics
// and always returns a URL with a host when successful.
func FuzzNormalizeAPIURL(f *testing.F) {
	f.Add("https://api.beebuzz.app")
	f.Add("")
	f.Add("api.example.com")
	f.Add("api.example.com/v1")
	f.Add("http://api.example.com")
	f.Add("https://")
	f.Add("not-a-url")
	f.Add("ftp://example.com")
	f.Add("https://127.0.0.1")
	f.Add("https://[::1]")
	f.Add(strings.Repeat("a", 1000) + ".com")
	f.Add("https://" + strings.Repeat("a", 10000) + ".com")
	f.Add("\x00")
	f.Add("://missing-scheme")
	f.Add("https://example.com:8080/path?query=1#fragment")

	f.Fuzz(func(t *testing.T, rawURL string) {
		result, err := normalizeAPIURL(rawURL)

		// On success the result must not be empty and must not have a trailing slash.
		if err == nil {
			if result == "" {
				t.Fatal("normalizeAPIURL returned empty string without error")
			}
			if strings.HasSuffix(result, "/") {
				t.Fatalf("normalizeAPIURL returned URL with trailing slash: %q", result)
			}
		}
	})
}

// FuzzMaskToken verifies that maskToken never panics
// and never returns the full original token for tokens longer than 4 chars.
func FuzzMaskToken(f *testing.F) {
	f.Add("beebuzz_token_123456")
	f.Add("")
	f.Add("a")
	f.Add("ab")
	f.Add("abc")
	f.Add("abcd")
	f.Add("abcde")
	f.Add(strings.Repeat("x", 1000))
	f.Add(" ")
	f.Add("  spaced  ")
	f.Add("\x00\x01\x02")

	f.Fuzz(func(t *testing.T, token string) {
		masked := maskToken(token)
		trimmed := strings.TrimSpace(token)

		// Empty or whitespace-only tokens must return empty.
		if trimmed == "" && masked != "" {
			t.Fatalf("maskToken returned non-empty for blank token: %q", masked)
		}

		// Tokens longer than 4 chars must not appear in full in the masked output.
		if len(trimmed) > 4 && masked == trimmed {
			t.Fatalf("maskToken returned unmasked token: %q", masked)
		}

		// Masked output must never be longer than the trimmed token.
		if len(masked) > len(trimmed) {
			t.Fatalf("maskToken output longer than input: %q -> %q", trimmed, masked)
		}
	})
}

// FuzzFormatHTTPError verifies that FormatHTTPError never panics
// on arbitrary response bodies and status codes.
func FuzzFormatHTTPError(f *testing.F) {
	f.Add(401, `{"code":"unauthorized","message":"invalid token"}`)
	f.Add(422, `{"code":"validation_error","errors":["field: required"]}`)
	f.Add(500, "internal server error")
	f.Add(502, "")
	f.Add(200, `{"code":"ok","message":"success"}`)
	f.Add(400, `not json at all`)
	f.Add(503, `{"code":"","message":""}`)
	f.Add(401, `{"code":"err","errors":[]}`)
	f.Add(400, strings.Repeat("a", 5000))
	f.Add(500, "\x00\xff\xfe")
	f.Add(422, `{"code":123}`) // wrong type for code field

	f.Fuzz(func(t *testing.T, statusCode int, body string) {
		if statusCode < 100 || statusCode > 999 {
			return
		}

		resp := &http.Response{
			StatusCode: statusCode,
			Body:       io.NopCloser(strings.NewReader(body)),
		}

		// Must never panic.
		err := FormatHTTPError("test operation", resp)
		if err == nil {
			t.Fatal("FormatHTTPError returned nil error")
		}
	})
}

// FuzzEncryptPayload verifies that encryptPayload never panics
// and returns an error for invalid recipients.
func FuzzEncryptPayload(f *testing.F) {
	f.Add([]byte("hello world"), "age1invalidkey")
	f.Add([]byte(""), "age1invalidkey")
	f.Add([]byte("data"), "")
	f.Add([]byte("data"), "not-an-age-key")
	f.Add([]byte(strings.Repeat("x", 10000)), "garbage")
	f.Add([]byte("\x00\x01\x02\xff"), "garbage")
	f.Add([]byte("hello world"), sharedCLIAgeRecipient)
	f.Add([]byte(""), sharedCLIAgeRecipient)

	f.Fuzz(func(t *testing.T, plaintext []byte, key string) {
		if key == "" {
			return
		}

		// Must never panic.
		result, err := encryptPayload(plaintext, []string{key})

		// Invalid key must produce an error.
		_, parseErr := age.ParseX25519Recipient(key)
		if parseErr != nil && err == nil {
			t.Fatalf("encryptPayload succeeded with invalid key %q", key)
		}

		// Valid key must produce non-empty ciphertext.
		if err == nil && len(result) == 0 {
			t.Fatal("encryptPayload returned empty ciphertext without error")
		}
	})
}

// FuzzIsConfigNotFoundError verifies that isConfigNotFoundError never panics.
func FuzzIsConfigNotFoundError(f *testing.F) {
	f.Add("config not found: /path/to/config.json")
	f.Add("")
	f.Add("other error")
	f.Add("config not found: ")
	f.Add("CONFIG NOT FOUND: /path")
	f.Add(strings.Repeat("a", 10000))

	f.Fuzz(func(t *testing.T, msg string) {
		err := fmt.Errorf("%s", msg)

		// Must never panic.
		result := isConfigNotFoundError(err)

		// Must return true only when the message starts with the expected prefix.
		expected := strings.HasPrefix(msg, errPrefixConfigNotFound)
		if result != expected {
			t.Fatalf("isConfigNotFoundError(%q) = %v, want %v", msg, result, expected)
		}
	})
}

// FuzzSummarizeRecipient verifies that summarizeRecipient never panics
// and returns a result no longer than the input.
func FuzzSummarizeRecipient(f *testing.F) {
	f.Add("age1abcdefghijklmnop")
	f.Add("")
	f.Add("short")
	f.Add(strings.Repeat("a", 16))
	f.Add(strings.Repeat("a", 17))
	f.Add(strings.Repeat("x", 1000))
	f.Add("\x00\xff")
	f.Add("日本語テスト文字列を長くする")

	f.Fuzz(func(t *testing.T, key string) {
		result := summarizeRecipient(key)

		// Short keys must be returned verbatim.
		if len(key) <= 16 && result != key {
			t.Fatalf("summarizeRecipient(%q) = %q, want verbatim", key, result)
		}

		// Long keys must contain the ellipsis marker.
		if len(key) > 16 && !strings.Contains(result, "...") {
			t.Fatalf("summarizeRecipient(%q) missing ellipsis: %q", key, result)
		}
	})
}
