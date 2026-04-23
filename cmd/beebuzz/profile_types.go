package main

import (
	"fmt"
	"strings"

	"lucor.dev/beebuzz/internal/secure"
)

const (
	configDirName           = ".beebuzz"
	profilesDirName         = "profiles"
	configFileName          = "config.json"
	defaultProfileFileName  = "default-profile"
	fallbackProfileName     = "default"
	configDirPerm           = 0o700
	configFilePerm          = 0o600
	maxProfileNameLen       = 64
	envBeeBuzzAPIURL        = "BEEBUZZ_API_URL"
	envBeeBuzzAPIToken      = "BEEBUZZ_API_TOKEN"
	errPrefixConfigNotFound = "config not found: "
)

// Config stores the BeeBuzz CLI configuration on disk.
type Config struct {
	APIURL     string      `json:"api_url"`
	APIToken   string      `json:"api_token"`
	DeviceKeys []DeviceKey `json:"device_keys"`
	Profile    string      `json:"-"` // loaded from effective config, not stored
}

// DeviceKey stores the paired device key descriptor cached by the CLI.
type DeviceKey struct {
	DeviceID                string  `json:"device_id"`
	DeviceName              string  `json:"device_name"`
	PairedAt                *string `json:"paired_at,omitempty"`
	AgeRecipient            string  `json:"age_recipient"`
	AgeRecipientFingerprint string  `json:"age_recipient_fingerprint"`
}

// Normalize trims whitespace and removes a trailing slash from the API URL.
func (c *Config) Normalize() {
	c.APIURL = strings.TrimSpace(c.APIURL)
	c.APIURL = strings.TrimRight(c.APIURL, "/")
	c.APIToken = strings.TrimSpace(c.APIToken)
	if c.DeviceKeys == nil {
		c.DeviceKeys = []DeviceKey{}
	}
	for i := range c.DeviceKeys {
		c.DeviceKeys[i].DeviceID = strings.TrimSpace(c.DeviceKeys[i].DeviceID)
		c.DeviceKeys[i].DeviceName = strings.TrimSpace(c.DeviceKeys[i].DeviceName)
		c.DeviceKeys[i].AgeRecipient = strings.TrimSpace(c.DeviceKeys[i].AgeRecipient)
		if c.DeviceKeys[i].AgeRecipient != "" && c.DeviceKeys[i].AgeRecipientFingerprint == "" {
			c.DeviceKeys[i].AgeRecipientFingerprint = secure.Fingerprint(c.DeviceKeys[i].AgeRecipient)
		}
	}
}

// AgeRecipients returns the cached age recipients in config order.
func (c *Config) AgeRecipients() []string {
	if c == nil || len(c.DeviceKeys) == 0 {
		return []string{}
	}

	recipients := make([]string, 0, len(c.DeviceKeys))
	for _, deviceKey := range c.DeviceKeys {
		if deviceKey.AgeRecipient == "" {
			continue
		}
		recipients = append(recipients, deviceKey.AgeRecipient)
	}

	return recipients
}

// validateProfileName checks that profile name is valid.
func validateProfileName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: profile name is required", ErrUsage)
	}

	if len(name) > maxProfileNameLen {
		return fmt.Errorf("%w: profile name too long (max %d characters)", ErrUsage, maxProfileNameLen)
	}

	validChars := "abcdefghijklmnopqrstuvwxyz0123456789_-"
	for _, c := range name {
		if !strings.Contains(validChars, string(c)) {
			return fmt.Errorf("%w: invalid profile name %q: only [a-z0-9_-] allowed", ErrUsage, name)
		}
	}

	return nil
}

// maskToken returns a masked token representation.
func maskToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	if len(token) <= 4 {
		return strings.Repeat("*", len(token))
	}

	return strings.Repeat("*", len(token)-4) + token[len(token)-4:]
}

// isConfigNotFoundError reports whether the config load failed because the file does not exist.
func isConfigNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	return strings.HasPrefix(err.Error(), errPrefixConfigNotFound)
}
