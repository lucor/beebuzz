package topic

import (
	"strings"
	"testing"
	"unicode/utf8"

	"lucor.dev/beebuzz/internal/validator"
)

// FuzzCreateTopicRequestValidate verifies that CreateTopicRequest.Validate never panics
// and enforces required name and description length limits.
func FuzzCreateTopicRequestValidate(f *testing.F) {
	f.Add("alerts", "Notifications for alerts")
	f.Add("my_topic2", "")
	f.Add("", "")
	f.Add("", "some description")
	f.Add("UPPERCASE", "desc")
	f.Add("has spaces", "desc")
	f.Add("alerts!", "desc")
	f.Add(strings.Repeat("a", 33), "desc")
	f.Add("a", strings.Repeat("x", validator.MaxDescriptionLen+1))
	f.Add("\x00\xff", "\x00\xff")
	f.Add("topic", "Unicode: \u00e9\u00e0\u00fc\u00f1")
	f.Add("z", strings.Repeat("\u00e9", validator.MaxDescriptionLen))
	f.Add("z", strings.Repeat("\u00e9", validator.MaxDescriptionLen+1))
	f.Add("a", "")                            // minimal valid name
	f.Add(strings.Repeat("a", 32), "")        // exact max length
	f.Add("a"+strings.Repeat("0", 31), "")    // max length with digits
	f.Add("a"+strings.Repeat("_", 31), "")    // max length with underscores
	f.Add("_alerts", "desc")                   // invalid first char: underscore
	f.Add("1alerts", "desc")                   // invalid first char: digit
	f.Add("aA", "desc")                        // invalid uppercase tail

	f.Fuzz(func(t *testing.T, name, description string) {
		req := CreateTopicRequest{
			Name:        name,
			Description: description,
		}

		errs := req.Validate()

		if name == "" && len(errs) == 0 {
			t.Fatal("Validate accepted empty name")
		}

		if utf8.RuneCountInString(description) > validator.MaxDescriptionLen && len(errs) == 0 {
			t.Fatalf("Validate accepted description with %d runes (max %d)",
				utf8.RuneCountInString(description), validator.MaxDescriptionLen)
		}
	})
}

// FuzzUpdateTopicRequestValidate verifies that UpdateTopicRequest.Validate never panics
// and enforces the description length limit.
func FuzzUpdateTopicRequestValidate(f *testing.F) {
	f.Add("")
	f.Add("Updated description")
	f.Add(strings.Repeat("b", validator.MaxDescriptionLen))
	f.Add(strings.Repeat("b", validator.MaxDescriptionLen+1))
	f.Add(strings.Repeat("\u00e9", validator.MaxDescriptionLen))   // multibyte exact boundary
	f.Add(strings.Repeat("\u00e9", validator.MaxDescriptionLen+1))
	f.Add("\x00\xff\xfe")

	f.Fuzz(func(t *testing.T, description string) {
		req := UpdateTopicRequest{
			Description: description,
		}

		errs := req.Validate()

		if utf8.RuneCountInString(description) > validator.MaxDescriptionLen && len(errs) == 0 {
			t.Fatalf("Validate accepted description with %d runes (max %d)",
				utf8.RuneCountInString(description), validator.MaxDescriptionLen)
		}
	})
}
