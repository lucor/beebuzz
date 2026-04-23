package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const maxErrorBodyBytes = 4096

// ErrorResponse is the CLI view of the API error response.
type ErrorResponse struct {
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Errors  []string `json:"errors"`
}

// FormatHTTPError builds a user-facing error from an unsuccessful HTTP response.
func FormatHTTPError(operation string, resp *http.Response) error {
	if resp == nil {
		return fmt.Errorf("%s failed", operation)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
	if err != nil {
		return fmt.Errorf("%s failed with status %d", operation, resp.StatusCode)
	}

	var errorResponse ErrorResponse
	if err := json.Unmarshal(body, &errorResponse); err == nil {
		if len(errorResponse.Errors) > 0 {
			return fmt.Errorf(
				"%s failed with status %d (%s): %s",
				operation,
				resp.StatusCode,
				fallbackErrorCode(errorResponse.Code),
				strings.Join(errorResponse.Errors, ", "),
			)
		}
		if errorResponse.Message != "" {
			return fmt.Errorf(
				"%s failed with status %d (%s): %s",
				operation,
				resp.StatusCode,
				fallbackErrorCode(errorResponse.Code),
				errorResponse.Message,
			)
		}
	}

	trimmedBody := strings.TrimSpace(string(body))
	contentType := strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Type")))
	if contentType != "" {
		return fmt.Errorf("%s failed with status %d (%s): unexpected non-JSON response", operation, resp.StatusCode, contentType)
	}
	if trimmedBody != "" && looksLikeHTML(trimmedBody) {
		return fmt.Errorf("%s failed with status %d: unexpected HTML response", operation, resp.StatusCode)
	}

	return fmt.Errorf("%s failed with status %d", operation, resp.StatusCode)
}

// fallbackErrorCode returns a printable code value for formatted error messages.
func fallbackErrorCode(code string) string {
	if strings.TrimSpace(code) == "" {
		return "unknown_error"
	}

	return code
}

func looksLikeHTML(body string) bool {
	body = strings.ToLower(strings.TrimSpace(body))
	return strings.HasPrefix(body, "<!doctype html") || strings.HasPrefix(body, "<html")
}
