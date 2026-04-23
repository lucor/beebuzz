// Package secure provides utilities for generating and handling
// cryptographically secure tokens intended for authentication,
// API keys, and other security-sensitive use cases.
package secure

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"

	"golang.org/x/crypto/blake2b"
)

const (
	// tokenSizeLong is used for long-lived tokens (API keys, webhooks).
	tokenSizeLong = 32
	// inspectTokenLen is used for short-lived inspect tokens.
	inspectTokenLen = 16

	// tokenPrefix is the global prefix for all BeeBuzz tokens.
	tokenPrefix = "beebuzz"

	// tokenPrefixAPI is the prefix for API authentication tokens.
	tokenPrefixAPI = "api"
	// tokenPrefixWebhook is the prefix for webhook endpoint tokens.
	tokenPrefixWebhook = "wh"
	// tokenPrefixWebhookInspect is the prefix for webhook inspect tokens.
	tokenPrefixWebhookInspect = "whi"
	// tokenPrefixSession is the prefix for session tokens.
	tokenPrefixSession = "session"
	// tokenPrefixDevice is the prefix for device tokens.
	tokenPrefixDevice = "device"

	// fingerprintHexLen is the number of hex characters exposed in UI/CLI fingerprints.
	fingerprintHexLen = 16
)

// NewAPIToken generates a cryptographically secure token for API authentication.
// Returns a prefixed token in the format beebuzz_api_<token>.
func NewAPIToken() (string, error) {
	return newPrefixedToken(tokenPrefixAPI, tokenSizeLong)
}

// NewWebhookToken generates a cryptographically secure token for webhook endpoints.
// Returns a prefixed token in the format beebuzz_wh_<token>.
func NewWebhookToken() (string, error) {
	return newPrefixedToken(tokenPrefixWebhook, tokenSizeLong)
}

// NewInspectToken generates a cryptographically secure token for webhook inspect sessions.
// Returns a prefixed token in the format beebuzz_whi_<token>.
func NewInspectToken() (string, error) {
	return newPrefixedToken(tokenPrefixWebhookInspect, inspectTokenLen)
}

// NewOTP generates a cryptographically secure 6-digit one-time password.
func NewOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", fmt.Errorf("new otp: %w", err)
	}
	return fmt.Sprintf("%06d", n), nil
}

// NewSessionToken generates a cryptographically secure token for user sessions.
// Returns a prefixed token in the format beebuzz_session_<token>.
func NewSessionToken() (string, error) {
	return newPrefixedToken(tokenPrefixSession, tokenSizeLong)
}

// NewDeviceToken generates a cryptographically secure token for device authentication.
// Returns a prefixed token in the format beebuzz_device_<token>.
func NewDeviceToken() (string, error) {
	return newPrefixedToken(tokenPrefixDevice, tokenSizeLong)
}

// Hash returns the BLAKE2b-256 hash of the given value as a hex-encoded string.
func Hash(value string) string {
	sum := blake2b.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// Verify reports whether the hash of token matches the given hash.
// Uses constant time comparison to prevent timing attacks.
func Verify(token, hash string) bool {
	return subtle.ConstantTimeCompare(
		[]byte(Hash(token)),
		[]byte(hash),
	) == 1
}

// Fingerprint returns a short BLAKE2b-derived fingerprint for display purposes.
func Fingerprint(value string) string {
	sum := blake2b.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])[:fingerprintHexLen]
}

// MustRandomHex returns n cryptographically secure random bytes as a hex-encoded string.
// Returns 2*n hex characters. Suitable for request IDs, boundaries, and nonces.
// Panics on random source failure (indicates system-level issue).
func MustRandomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}
	return hex.EncodeToString(b)
}

// newToken generates a cryptographically secure random token of n bytes.
// The returned string is base64 URL-encoded without padding.
// It is safe for use in HTTP headers, URLs, and API keys.
func newToken(n int) (string, error) {
	b := make([]byte, n)

	// Read fills b with cryptographically secure random bytes.
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate secure token: %w", err)
	}

	// RawURLEncoding produces a URL-safe base64 string without "=" padding.
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// newPrefixedToken generates a token with a beebuzz_{name}_ prefix.
func newPrefixedToken(name string, size int) (string, error) {
	tok, err := newToken(size)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s_%s_%s", tokenPrefix, name, tok), nil
}
