package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// resolveAPIURL validates and normalizes the configured BeeBuzz API base URL.
func resolveAPIURL(ctx context.Context, client *http.Client, rawAPIURL, apiToken string) (string, []DeviceKey, error) {
	if client == nil {
		return "", nil, fmt.Errorf("http client is required")
	}

	resolvedAPIURL, err := normalizeAPIURL(rawAPIURL)
	if err != nil {
		return "", nil, err
	}

	config := &Config{
		APIURL:   resolvedAPIURL,
		APIToken: apiToken,
	}

	resp, err := doKeysRequest(ctx, client, config)
	if err != nil {
		return "", nil, fmt.Errorf("request keys from %s: %w", resolvedAPIURL, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return "", nil, FormatHTTPError(fmt.Sprintf("keys request to %s", resolvedAPIURL), resp)
	}

	var keysResponse KeysResponse
	if err := decodeJSONResponse(resp, &keysResponse); err != nil {
		return "", nil, fmt.Errorf("decode keys response from %s: %w", resolvedAPIURL, err)
	}
	if keysResponse.Data == nil {
		keysResponse.Data = []DeviceKey{}
	}

	return resolvedAPIURL, keysResponse.Data, nil
}

// normalizeAPIURL resolves the configured BeeBuzz API URL to a canonical base URL.
func normalizeAPIURL(rawAPIURL string) (string, error) {
	normalized := strings.TrimSpace(rawAPIURL)
	if normalized == "" {
		normalized = defaultAPIURL
	}
	if !strings.Contains(normalized, "://") {
		normalized = "https://" + normalized
	}

	parsedURL, err := url.Parse(normalized)
	if err != nil {
		return "", fmt.Errorf("parse API URL: %w", err)
	}
	if parsedURL.Host == "" {
		return "", fmt.Errorf("API URL must include a host")
	}

	scheme := parsedURL.Scheme
	if scheme == "" {
		scheme = "https"
	}
	if scheme != "https" {
		return "", fmt.Errorf("API URL must use https")
	}

	return strings.TrimRight((&url.URL{Scheme: scheme, Host: parsedURL.Host}).String(), "/"), nil
}
