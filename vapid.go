package main

import (
	"fmt"
	"io"

	"github.com/SherClockHolmes/webpush-go"
)

// runGenerateVAPID prints a copy-pasteable VAPID env pair so operators can
// provision keys without hand-editing secrets.
func runGenerateVAPID(stdout io.Writer) error {
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		return fmt.Errorf("generate VAPID keys: %w", err)
	}

	if _, err := fmt.Fprintf(stdout, "BEEBUZZ_VAPID_PRIVATE_KEY=%s\nBEEBUZZ_VAPID_PUBLIC_KEY=%s\n", privateKey, publicKey); err != nil {
		return fmt.Errorf("write VAPID keys: %w", err)
	}

	return nil
}
