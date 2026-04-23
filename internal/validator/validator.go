// Package validator provides simple, composable validation functions.
// Validators return nil on success and an error on failure.
// Use [Validate] to collect multiple validation results into a single slice.
package validator

import (
	"fmt"
	"net/mail"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

// MaxDisplayNameLen is the maximum number of Unicode characters for a display name or short label.
const MaxDisplayNameLen = 64

// MaxDescriptionLen is the maximum number of Unicode characters for a description field.
const MaxDescriptionLen = 128

var topicNameRegexp = regexp.MustCompile(`^[a-z][a-z0-9_]{0,31}$`)
var jsonPathRegexp = regexp.MustCompile(`^[A-Za-z0-9_]+(?:\[[0-9]+\])*(?:\.[A-Za-z0-9_]+(?:\[[0-9]+\])*)*$`)

// isBlank reports whether value is empty after trimming surrounding whitespace.
func isBlank(value string) bool {
	return strings.TrimSpace(value) == ""
}

// Validate collects the non-nil errors from the provided checks.
// Returns nil if all checks pass.
func Validate(checks ...error) []error {
	var errs []error
	for _, err := range checks {
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// Email returns an error if value is not a valid email address.
// It accepts general RFC-style addresses, including display-name forms such as
// `"Alice Example" <user@example.com>`. Use PlainEmail for login-style input.
func Email(field, value string) error {
	if _, err := mail.ParseAddress(value); err != nil {
		return fmt.Errorf("%s is not a valid email", field)
	}
	return nil
}

// PlainEmail returns an error unless value is a plain email address with no display name.
// It is intended for login-style input such as `user@example.com`, not generic RFC address forms.
func PlainEmail(field, value string) error {
	trimmedValue := strings.TrimSpace(value)
	parsedAddress, err := mail.ParseAddress(trimmedValue)
	if err != nil {
		return fmt.Errorf("%s is not a valid email", field)
	}
	if parsedAddress.Name != "" || parsedAddress.Address != trimmedValue {
		return fmt.Errorf("%s is not a valid email", field)
	}
	return nil
}

// Required returns an error if value is the zero value for its type.
func Required[T comparable](field string, value T) error {
	var zero T
	if value == zero {
		return fmt.Errorf("%s: is required", field)
	}
	return nil
}

// NotBlank returns an error if the string is empty or contains only whitespace.
func NotBlank(field, value string) error {
	if isBlank(value) {
		return fmt.Errorf("%s: is required", field)
	}
	return nil
}

// Blank returns an error if the string contains any non-whitespace content.
func Blank(field, value string) error {
	if !isBlank(value) {
		return fmt.Errorf("%s: must be blank", field)
	}
	return nil
}

// RequiredSlice returns an error if the slice is empty.
func RequiredSlice[T any](field string, values []T) error {
	if len(values) == 0 {
		return fmt.Errorf("%s: is required", field)
	}
	return nil
}

// UniqueStrings returns an error if the slice contains duplicate string values.
func UniqueStrings(field string, values []string) error {
	sortedValues := slices.Clone(values)
	slices.Sort(sortedValues)

	for i := 1; i < len(sortedValues); i++ {
		if sortedValues[i] == sortedValues[i-1] {
			return fmt.Errorf("%s: duplicate values are not allowed", field)
		}
	}
	return nil
}

// TopicName returns an error if value does not match ^[a-z][a-z0-9_]{0,31}$.
func TopicName(field, value string) error {
	if !topicNameRegexp.MatchString(value) {
		return fmt.Errorf("%s: must be lowercase letters, digits, or underscores, start with a letter, max 32 chars", field)
	}
	return nil
}

// MaxLen returns an error if value exceeds max Unicode characters (rune count, not byte length).
func MaxLen(field, value string, max int) error {
	if utf8.RuneCountInString(value) > max {
		return fmt.Errorf("%s: must be %d characters or less", field, max)
	}
	return nil
}

// OneOf returns an error if value is not one of the allowed values.
func OneOf(field, value string, allowed []string) error {
	if slices.Contains(allowed, value) {
		return nil
	}
	return fmt.Errorf("%s: must be one of: %s", field, strings.Join(allowed, ", "))
}

// NoLeadingDot returns an error if value begins with a dot after trimming whitespace.
func NoLeadingDot(field, value string) error {
	trimmedValue := strings.TrimSpace(value)
	if strings.HasPrefix(trimmedValue, ".") {
		return fmt.Errorf("%s: must not start with '.'", field)
	}
	return nil
}

// JSONPath returns an error unless value is a simple dot-separated path with optional numeric indexes.
func JSONPath(field, value string) error {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return fmt.Errorf("%s: is required", field)
	}
	if !jsonPathRegexp.MatchString(trimmedValue) {
		return fmt.Errorf("%s: must use dot-separated field names with optional numeric indexes", field)
	}
	return nil
}
