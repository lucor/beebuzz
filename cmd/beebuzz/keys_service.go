package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
)

const keysEndpointPath = "/v1/push/keys"

// KeysResponse is the CLI view of the server keys response.
type KeysResponse struct {
	Data []DeviceKey `json:"data"`
}

// doKeysRequest performs the authenticated GET /v1/push/keys request.
func doKeysRequest(ctx context.Context, client *http.Client, config *Config) (*http.Response, error) {
	req, err := buildAuthorizedRequest(ctx, http.MethodGet, config.APIURL+keysEndpointPath, config.APIToken, nil)
	if err != nil {
		return nil, fmt.Errorf("build keys request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request keys: %w", err)
	}

	return resp, nil
}

// refreshKeys fetches the current device age keys and updates the config cache.
func refreshKeys(ctx context.Context, client *http.Client, config *Config) error {
	resp, err := doKeysRequest(ctx, client, config)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return FormatHTTPError("keys request", resp)
	}

	var keysResponse KeysResponse
	if err := decodeJSONResponse(resp, &keysResponse); err != nil {
		return fmt.Errorf("decode keys response: %w", err)
	}

	if keysResponse.Data == nil {
		keysResponse.Data = []DeviceKey{}
	}

	config.DeviceKeys = keysResponse.Data
	config.Normalize()
	return nil
}

func writeKeyRefreshSummary(output io.Writer, previousKeys, currentKeys []DeviceKey) error {
	addedKeys, removedKeys := diffKeys(previousKeys, currentKeys)
	addedCount := len(addedKeys)
	removedCount := len(removedKeys)
	if addedCount == 0 && removedCount == 0 {
		return nil
	}

	if _, err := fmt.Fprintf(output, "warning: device keys changed (%d added, %d removed)\n", addedCount, removedCount); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	for _, deviceKey := range addedKeys {
		if _, err := fmt.Fprintf(output, "added: %s\n", summarizeDeviceKey(deviceKey)); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
	}
	for _, deviceKey := range removedKeys {
		if _, err := fmt.Fprintf(output, "removed: %s\n", summarizeDeviceKey(deviceKey)); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
	}

	return nil
}

func diffKeys(previousKeys, currentKeys []DeviceKey) (addedKeys, removedKeys []DeviceKey) {
	previousSet := make(map[string]DeviceKey, len(previousKeys))
	currentSet := make(map[string]DeviceKey, len(currentKeys))

	for _, deviceKey := range previousKeys {
		previousSet[deviceKey.AgeRecipient] = deviceKey
	}
	for _, deviceKey := range currentKeys {
		currentSet[deviceKey.AgeRecipient] = deviceKey
	}

	for recipient, deviceKey := range currentSet {
		if _, ok := previousSet[recipient]; !ok {
			addedKeys = append(addedKeys, deviceKey)
		}
	}
	for recipient, deviceKey := range previousSet {
		if _, ok := currentSet[recipient]; !ok {
			removedKeys = append(removedKeys, deviceKey)
		}
	}

	sort.Slice(addedKeys, func(i, j int) bool {
		return addedKeys[i].AgeRecipient < addedKeys[j].AgeRecipient
	})
	sort.Slice(removedKeys, func(i, j int) bool {
		return removedKeys[i].AgeRecipient < removedKeys[j].AgeRecipient
	})

	return addedKeys, removedKeys
}

func summarizeDeviceKey(deviceKey DeviceKey) string {
	name := deviceKey.DeviceName
	if name == "" {
		name = "unnamed device"
	}

	return fmt.Sprintf("%s [%s] %s", name, deviceKey.AgeRecipientFingerprint, summarizeRecipient(deviceKey.AgeRecipient))
}

func summarizeRecipient(recipient string) string {
	if len(recipient) <= 16 {
		return recipient
	}

	return recipient[:8] + "..." + recipient[len(recipient)-8:]
}
