package validator

import (
	"errors"
	"testing"
)

var ErrTest = errors.New("test error")

func TestValidate(t *testing.T) {
	tests := []struct {
		name   string
		checks []error
		want   int
	}{
		{
			name:   "no errors",
			checks: []error{nil, nil},
			want:   0,
		},
		{
			name:   "one error",
			checks: []error{nil, ErrTest, nil},
			want:   1,
		},
		{
			name:   "multiple errors",
			checks: []error{ErrTest, ErrTest, ErrTest},
			want:   3,
		},
		{
			name:   "nil input",
			checks: nil,
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Validate(tt.checks...)
			if len(got) != tt.want {
				t.Errorf("Validate() got %d errors, want %d", len(got), tt.want)
			}
		})
	}
}

func TestEmail(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		value     string
		wantError bool
	}{
		{
			name:      "valid email",
			field:     "email",
			value:     "test@example.com",
			wantError: false,
		},
		{
			name:      "valid email with name",
			field:     "email",
			value:     "John Doe <john@example.com>",
			wantError: false,
		},
		{
			name:      "invalid email no domain",
			field:     "email",
			value:     "test@",
			wantError: true,
		},
		{
			name:      "invalid email no local",
			field:     "email",
			value:     "@example.com",
			wantError: true,
		},
		{
			name:      "invalid email format",
			field:     "email",
			value:     "not-an-email",
			wantError: true,
		},
		{
			name:      "empty email",
			field:     "email",
			value:     "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Email(tt.field, tt.value)
			if (err != nil) != tt.wantError {
				t.Errorf("Email() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestPlainEmail(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		value     string
		wantError bool
	}{
		{
			name:      "valid plain email",
			field:     "email",
			value:     "test@example.com",
			wantError: false,
		},
		{
			name:      "valid tagged email",
			field:     "email",
			value:     "user+tag@example.com",
			wantError: false,
		},
		{
			name:      "valid email with surrounding whitespace",
			field:     "email",
			value:     "  test@example.com  ",
			wantError: false,
		},
		{
			name:      "display name form rejected",
			field:     "email",
			value:     "John Doe <john@example.com>",
			wantError: true,
		},
		{
			name:      "invalid email format",
			field:     "email",
			value:     "not-an-email",
			wantError: true,
		},
		{
			name:      "empty email",
			field:     "email",
			value:     "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := PlainEmail(tt.field, tt.value)
			if (err != nil) != tt.wantError {
				t.Errorf("PlainEmail() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestRequired(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		value     string
		wantError bool
	}{
		{
			name:      "non-empty string",
			field:     "username",
			value:     "john",
			wantError: false,
		},
		{
			name:      "empty string",
			field:     "username",
			value:     "",
			wantError: true,
		},
		{
			name:      "whitespace only",
			field:     "username",
			value:     "   ",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Required(tt.field, tt.value)
			if (err != nil) != tt.wantError {
				t.Errorf("Required() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestNotBlank(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		value     string
		wantError bool
	}{
		{
			name:      "non-empty string",
			field:     "username",
			value:     "john",
			wantError: false,
		},
		{
			name:      "empty string",
			field:     "username",
			value:     "",
			wantError: true,
		},
		{
			name:      "whitespace only",
			field:     "username",
			value:     "   \t",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NotBlank(tt.field, tt.value)
			if (err != nil) != tt.wantError {
				t.Errorf("NotBlank() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestBlank(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		value     string
		wantError bool
	}{
		{
			name:      "empty string",
			field:     "title_path",
			value:     "",
			wantError: false,
		},
		{
			name:      "whitespace only",
			field:     "title_path",
			value:     "  \t",
			wantError: false,
		},
		{
			name:      "non-empty string",
			field:     "title_path",
			value:     "data.title",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Blank(tt.field, tt.value)
			if (err != nil) != tt.wantError {
				t.Errorf("Blank() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestNoLeadingDot(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		value     string
		wantError bool
	}{
		{
			name:      "plain path",
			field:     "title_path",
			value:     "data.title",
			wantError: false,
		},
		{
			name:      "trimmed plain path",
			field:     "title_path",
			value:     "  data.title  ",
			wantError: false,
		},
		{
			name:      "leading dot",
			field:     "title_path",
			value:     ".data.title",
			wantError: true,
		},
		{
			name:      "leading dot after whitespace",
			field:     "title_path",
			value:     "  .data.title",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NoLeadingDot(tt.field, tt.value)
			if (err != nil) != tt.wantError {
				t.Errorf("NoLeadingDot() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestJSONPath(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		value     string
		wantError bool
	}{
		{
			name:      "plain path",
			field:     "title_path",
			value:     "data.title",
			wantError: false,
		},
		{
			name:      "path with array index",
			field:     "title_path",
			value:     "data.items[0].title",
			wantError: false,
		},
		{
			name:      "leading dot",
			field:     "title_path",
			value:     ".data.title",
			wantError: true,
		},
		{
			name:      "recursive operator",
			field:     "title_path",
			value:     "data..title",
			wantError: true,
		},
		{
			name:      "query operator",
			field:     "title_path",
			value:     "data.#.title",
			wantError: true,
		},
		{
			name:      "modifier operator",
			field:     "title_path",
			value:     "@this",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := JSONPath(tt.field, tt.value)
			if (err != nil) != tt.wantError {
				t.Errorf("JSONPath() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestUniqueStrings(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		values    []string
		wantError bool
	}{
		{
			name:      "unique values",
			field:     "topics",
			values:    []string{"a", "b", "c"},
			wantError: false,
		},
		{
			name:      "duplicate values",
			field:     "topics",
			values:    []string{"a", "b", "a"},
			wantError: true,
		},
		{
			name:      "empty slice",
			field:     "topics",
			values:    []string{},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UniqueStrings(tt.field, tt.values)
			if (err != nil) != tt.wantError {
				t.Errorf("UniqueStrings() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestRequiredInt(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		value     int
		wantError bool
	}{
		{
			name:      "non-zero int",
			field:     "count",
			value:     42,
			wantError: false,
		},
		{
			name:      "zero int",
			field:     "count",
			value:     0,
			wantError: true,
		},
		{
			name:      "negative int",
			field:     "count",
			value:     -1,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Required(tt.field, tt.value)
			if (err != nil) != tt.wantError {
				t.Errorf("Required() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestOneOf(t *testing.T) {
	allowed := []string{"", "high", "normal", "low"}

	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{"empty string in allowed list", "", false},
		{"first value", "high", false},
		{"last value", "low", false},
		{"value not in list", "ultra", true},
		{"case sensitive mismatch", "High", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := OneOf("priority", tt.value, allowed)
			if (err != nil) != tt.wantError {
				t.Errorf("OneOf() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
