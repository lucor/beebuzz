package main

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestFormatHTTPErrorUsesMessageField(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusUnauthorized,
		Body:       io.NopCloser(strings.NewReader(`{"code":"unauthorized","message":"invalid token"}`)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}

	err := FormatHTTPError("push request", resp)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "push request failed with status 401 (unauthorized): invalid token" {
		t.Fatalf("error: got %q", err)
	}
}

func TestFormatHTTPErrorUsesValidationErrors(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusUnprocessableEntity,
		Body:       io.NopCloser(strings.NewReader(`{"code":"validation_error","errors":["body: is required","priority: must be one of: , high, normal"]}`)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}

	err := FormatHTTPError("push request", resp)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "push request failed with status 422 (validation_error): body: is required, priority: must be one of: , high, normal" {
		t.Fatalf("error: got %q", err)
	}
}

func TestFormatHTTPErrorSuppressesPlainTextBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusBadGateway,
		Body:       io.NopCloser(strings.NewReader("upstream error")),
		Header:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}},
	}

	err := FormatHTTPError("keys request", resp)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "keys request failed with status 502 (text/plain; charset=utf-8): unexpected non-JSON response" {
		t.Fatalf("error: got %q", err)
	}
}

func TestFormatHTTPErrorSuppressesHTMLBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(strings.NewReader("<!doctype html><html><body>not api</body></html>")),
		Header:     http.Header{"Content-Type": []string{"text/html; charset=utf-8"}},
	}

	err := FormatHTTPError("keys request", resp)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "keys request failed with status 404 (text/html; charset=utf-8): unexpected non-JSON response" {
		t.Fatalf("error: got %q", err)
	}
}
