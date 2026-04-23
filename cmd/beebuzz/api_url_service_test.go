package main

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestNormalizeAPIURLUsesDefaultValue(t *testing.T) {
	resolvedAPIURL, err := normalizeAPIURL("")
	if err != nil {
		t.Fatalf("normalizeAPIURL: %v", err)
	}
	if resolvedAPIURL != defaultAPIURL {
		t.Fatalf("APIURL: got %q, want %q", resolvedAPIURL, defaultAPIURL)
	}
}

func TestNormalizeAPIURLAddsHTTPSAndTrimsPath(t *testing.T) {
	resolvedAPIURL, err := normalizeAPIURL("api.example.com/v1")
	if err != nil {
		t.Fatalf("normalizeAPIURL: %v", err)
	}
	if resolvedAPIURL != "https://api.example.com" {
		t.Fatalf("APIURL: got %q, want %q", resolvedAPIURL, "https://api.example.com")
	}
}

func TestNormalizeAPIURLRejectsRemoteHTTP(t *testing.T) {
	_, err := normalizeAPIURL("http://api.example.com")
	if err == nil {
		t.Fatal("expected scheme error")
	}
}

func TestNormalizeAPIURLRejectsLocalHTTP(t *testing.T) {
	for _, input := range []string{
		"http://localhost:8080",
		"http://127.0.0.1:9000",
		"http://[::1]:7000",
	} {
		t.Run(input, func(t *testing.T) {
			_, err := normalizeAPIURL(input)
			if err == nil {
				t.Fatal("expected scheme error")
			}
		})
	}
}

func TestResolveAPIURLRequestsConfiguredAPIURL(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != "https://api.beebuzz.app"+keysEndpointPath {
				t.Fatalf("unexpected URL: %q", req.URL.String())
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"data":[{"device_id":"dev-a","device_name":"phone","paired_at":"2026-04-11T10:00:00Z","age_recipient":"age1abc","age_recipient_fingerprint":"fpabc"}]}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	resolvedURL, deviceKeys, err := resolveAPIURL(context.Background(), client, defaultAPIURL, "token")
	if err != nil {
		t.Fatalf("resolveAPIURL: %v", err)
	}
	if resolvedURL != defaultAPIURL {
		t.Fatalf("resolvedURL: got %q, want %q", resolvedURL, defaultAPIURL)
	}
	if len(deviceKeys) != 1 || deviceKeys[0].AgeRecipient != "age1abc" {
		t.Fatalf("device keys: got %#v", deviceKeys)
	}
}
