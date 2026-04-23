package token

import (
	"strings"
	"testing"
	"unicode/utf8"

	"lucor.dev/beebuzz/internal/validator"
)

// FuzzCreateAPITokenRequestValidate verifies that CreateAPITokenRequest.Validate never panics
// and enforces required name and length limits.
func FuzzCreateAPITokenRequestValidate(f *testing.F) {
	f.Add("my-token", "A test token")
	f.Add("", "")
	f.Add("", "description")
	f.Add("t", "")
	f.Add(strings.Repeat("a", validator.MaxDisplayNameLen), strings.Repeat("b", validator.MaxDisplayNameLen))
	f.Add(strings.Repeat("a", validator.MaxDisplayNameLen+1), strings.Repeat("b", validator.MaxDisplayNameLen+1))
	f.Add("\x00\xff", "\x00\xff")
	f.Add("token", strings.Repeat("\u00e9", validator.MaxDisplayNameLen+1))
	f.Add(strings.Repeat("a", validator.MaxDisplayNameLen+1), "ok")                          // isolated name overflow
	f.Add(strings.Repeat("é", validator.MaxDisplayNameLen), "")                              // multibyte name exact
	f.Add(strings.Repeat("é", validator.MaxDisplayNameLen+1), "")                            // multibyte name overflow
	f.Add("token", strings.Repeat("é", validator.MaxDisplayNameLen))                         // multibyte desc exact (uses MaxDisplayNameLen)
	f.Add(" ", "")                                                                           // whitespace-only name

	f.Fuzz(func(t *testing.T, name, description string) {
		req := CreateAPITokenRequest{
			Name:        name,
			Description: description,
			Topics:      []string{"general"},
		}

		errs := req.Validate()

		if name == "" && len(errs) == 0 {
			t.Fatal("Validate accepted empty name")
		}

		if utf8.RuneCountInString(name) > validator.MaxDisplayNameLen && len(errs) == 0 {
			t.Fatalf("Validate accepted name with %d runes (max %d)",
				utf8.RuneCountInString(name), validator.MaxDisplayNameLen)
		}

		if utf8.RuneCountInString(description) > validator.MaxDisplayNameLen && len(errs) == 0 {
			t.Fatalf("Validate accepted description with %d runes (max %d)",
				utf8.RuneCountInString(description), validator.MaxDisplayNameLen)
		}
	})
}

// FuzzUpdateAPITokenRequestValidate verifies that UpdateAPITokenRequest.Validate never panics
// and enforces required name and length limits.
func FuzzUpdateAPITokenRequestValidate(f *testing.F) {
	f.Add("updated-token", "Updated description")
	f.Add("", "")
	f.Add("", "description")
	f.Add("t", "")
	f.Add(strings.Repeat("a", validator.MaxDisplayNameLen), strings.Repeat("b", validator.MaxDisplayNameLen))
	f.Add(strings.Repeat("a", validator.MaxDisplayNameLen+1), strings.Repeat("b", validator.MaxDisplayNameLen+1))
	f.Add("\x00\xff", "\x00\xff")
	f.Add("token", strings.Repeat("\u00e9", validator.MaxDisplayNameLen+1))
	f.Add(strings.Repeat("a", validator.MaxDisplayNameLen+1), "ok")                          // isolated name overflow
	f.Add(strings.Repeat("é", validator.MaxDisplayNameLen), "")                              // multibyte name exact
	f.Add(strings.Repeat("é", validator.MaxDisplayNameLen+1), "")                            // multibyte name overflow
	f.Add("token", strings.Repeat("é", validator.MaxDisplayNameLen))                         // multibyte desc exact (uses MaxDisplayNameLen)
	f.Add(" ", "")                                                                           // whitespace-only name

	f.Fuzz(func(t *testing.T, name, description string) {
		req := UpdateAPITokenRequest{
			Name:        name,
			Description: description,
			Topics:      []string{"general"},
		}

		errs := req.Validate()

		if name == "" && len(errs) == 0 {
			t.Fatal("Validate accepted empty name")
		}

		if utf8.RuneCountInString(name) > validator.MaxDisplayNameLen && len(errs) == 0 {
			t.Fatalf("Validate accepted name with %d runes (max %d)",
				utf8.RuneCountInString(name), validator.MaxDisplayNameLen)
		}

		if utf8.RuneCountInString(description) > validator.MaxDisplayNameLen && len(errs) == 0 {
			t.Fatalf("Validate accepted description with %d runes (max %d)",
				utf8.RuneCountInString(description), validator.MaxDisplayNameLen)
		}
	})
}
