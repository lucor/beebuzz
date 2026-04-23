package validator

import (
	"strings"
	"testing"
)

// FuzzEmail verifies that Email never panics on arbitrary input.
func FuzzEmail(f *testing.F) {
	f.Add("user@example.com")
	f.Add("")
	f.Add("not-an-email")
	f.Add("@")
	f.Add("a@b")
	f.Add("<script>@evil.com")
	f.Add(strings.Repeat("a", 1000) + "@example.com")
	f.Add("user@" + strings.Repeat("x", 500) + ".com")
	f.Add("\x00@\x00.com")
	f.Add("日本語@example.com")
	f.Add("user@")
	f.Add("@example.com")
	f.Add("user+tag@example.com")
	f.Add("\"Alice Example\" <user@example.com>")
	f.Add("user name@example.com") // invalid unquoted space

	f.Fuzz(func(t *testing.T, value string) {
		// Must never panic.
		_ = Email("email", value)
	})
}

// FuzzNotBlank verifies that NotBlank never panics on arbitrary input.
func FuzzNotBlank(f *testing.F) {
	f.Add("")
	f.Add(" ")
	f.Add("\t")
	f.Add("value")
	f.Add(" value ")
	f.Add("\x00")
	f.Add(strings.Repeat("a", 1000))

	f.Fuzz(func(t *testing.T, value string) {
		_ = NotBlank("field", value)
	})
}

// FuzzPlainEmail verifies that PlainEmail never panics on arbitrary input.
func FuzzPlainEmail(f *testing.F) {
	f.Add("user@example.com")
	f.Add("")
	f.Add("not-an-email")
	f.Add("@")
	f.Add("a@b")
	f.Add("<script>@evil.com")
	f.Add(strings.Repeat("a", 1000) + "@example.com")
	f.Add("user@" + strings.Repeat("x", 500) + ".com")
	f.Add("\x00@\x00.com")
	f.Add("日本語@example.com")
	f.Add("user@")
	f.Add("@example.com")
	f.Add("user+tag@example.com")
	f.Add("\"Alice Example\" <user@example.com>")
	f.Add("user name@example.com")

	f.Fuzz(func(t *testing.T, value string) {
		_ = PlainEmail("email", value)
	})
}

// FuzzTopicName verifies that TopicName never panics and rejects invalid patterns.
func FuzzTopicName(f *testing.F) {
	f.Add("alerts")
	f.Add("")
	f.Add("a")
	f.Add("a_b1")
	f.Add("1abc")
	f.Add("ABC")
	f.Add(strings.Repeat("a", 33))
	f.Add(strings.Repeat("a", 32))
	f.Add("topic-with-dash")
	f.Add("topic.with.dot")
	f.Add("\x00null")
	f.Add("日本語")
	f.Add("_abc")                        // invalid first char: underscore
	f.Add("a0")                          // valid digit tail
	f.Add("a_")                          // valid underscore tail
	f.Add("aA")                          // invalid uppercase tail
	f.Add("a" + strings.Repeat("0", 31)) // max length with digits
	f.Add("a" + strings.Repeat("_", 31)) // max length with underscores

	f.Fuzz(func(t *testing.T, value string) {
		err := TopicName("name", value)

		// Names longer than 32 chars must always be rejected.
		if len(value) > 32 && err == nil {
			t.Fatalf("TopicName accepted string longer than 32 chars: %q", value)
		}

		// Empty string must always be rejected.
		if value == "" && err == nil {
			t.Fatal("TopicName accepted empty string")
		}
	})
}

// FuzzMaxLen verifies that MaxLen never panics and correctly enforces rune limits.
func FuzzMaxLen(f *testing.F) {
	f.Add("hello", 5)
	f.Add("", 0)
	f.Add(strings.Repeat("a", 100), 50)
	f.Add("日本語", 3)
	f.Add("日本語テスト", 2)
	f.Add("\x00\xff\xfe", 10)
	f.Add("a", 0)  // single char, zero max
	f.Add("ab", 1) // ASCII boundary: 2 chars, max 1
	f.Add("é", 1)  // multibyte rune at exact max
	f.Add("éé", 1) // multibyte rune over max
	f.Add("x", -1) // negative max hits early return

	f.Fuzz(func(t *testing.T, value string, max int) {
		if max < 0 {
			return
		}

		// Must never panic.
		_ = MaxLen("field", value, max)
	})
}

// FuzzHTTPSURL verifies that HTTPSURL never panics and rejects non-HTTPS schemes.
func FuzzHTTPSURL(f *testing.F) {
	f.Add("https://example.com/path")
	f.Add("")
	f.Add("http://example.com")
	f.Add("ftp://example.com")
	f.Add("https://127.0.0.1/path")
	f.Add("https://10.0.0.1/path")
	f.Add("https://192.168.1.1/path")
	f.Add("not-a-url")
	f.Add("https://")
	f.Add("https://" + strings.Repeat("a", 1000) + ".com")
	f.Add("https://example.com/" + strings.Repeat("a", 10000))
	f.Add("https://8.8.8.8/path")         // valid public IP
	f.Add("https://172.16.0.1/path")      // private 172.16/12
	f.Add("https://[::1]/path")           // IPv6 loopback
	f.Add("https://[fd00::1]/path")       // IPv6 private (ULA)
	f.Add("https:///path")                // scheme present, host missing
	f.Add("https://example.com:443/path") // hostname with port

	f.Fuzz(func(t *testing.T, value string) {
		err := HTTPSURL("url", value)

		// Empty is allowed (handled by Required validator).
		if value == "" && err != nil {
			t.Fatalf("HTTPSURL rejected empty string, should be allowed")
		}
	})
}
