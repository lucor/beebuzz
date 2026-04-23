package validator

import (
	"fmt"

	"filippo.io/age"
)

// AgeRecipient returns an error if value is not a valid age X25519 recipient.
func AgeRecipient(field, value string) error {
	if _, err := age.ParseX25519Recipient(value); err != nil {
		return fmt.Errorf("%s: must be a valid age X25519 recipient", field)
	}
	return nil
}
