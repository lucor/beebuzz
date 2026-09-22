package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
)

func runGenerateSalt(stdout io.Writer) error {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return fmt.Errorf("generate salt: %w", err)
	}
	if _, err := fmt.Fprintf(stdout, "BEEBUZZ_IP_HASH_SALT=%s\n", hex.EncodeToString(b)); err != nil {
		return fmt.Errorf("write salt: %w", err)
	}
	return nil
}