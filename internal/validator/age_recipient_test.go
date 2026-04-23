package validator

import "testing"

const validAgeRecipient = "age1lnhya3u0lkqlw2txk56ltge6n9gm3p4lj2snmt4yaftwx005wetsmyx0zx"

func TestAgeRecipient(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{
			name:      "valid age x25519 recipient",
			value:     validAgeRecipient,
			wantError: false,
		},
		{
			name:      "invalid recipient",
			value:     "age1recipient",
			wantError: true,
		},
		{
			name:      "blank recipient",
			value:     "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AgeRecipient("age_recipient", tt.value)
			if (err != nil) != tt.wantError {
				t.Fatalf("AgeRecipient() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
