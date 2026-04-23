package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// buildAuthorizedRequest creates an authenticated HTTP request for the BeeBuzz API.
func buildAuthorizedRequest(ctx context.Context, method, requestURL, apiToken string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, requestURL, body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiToken)
	req.Header.Set("User-Agent", "BeeBuzz-CLI/"+version)
	return req, nil
}

// decodeJSONResponse decodes a successful JSON response into the provided output value.
func decodeJSONResponse(resp *http.Response, out any) error {
	if resp == nil {
		return fmt.Errorf("response is required")
	}
	if out == nil {
		return fmt.Errorf("decode output is required")
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}
