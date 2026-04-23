package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestRefreshKeysUpdatesConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != keysEndpointPath {
			t.Fatalf("path: got %q, want %q", r.URL.Path, keysEndpointPath)
		}
		if r.Header.Get("Authorization") != "Bearer beebuzz_token" {
			t.Fatalf("authorization: got %q", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"data":[{"device_id":"dev-a","device_name":"phone","paired_at":"2026-04-11T10:00:00Z","age_recipient":"age1abc","age_recipient_fingerprint":"fpabc"},{"device_id":"dev-b","device_name":"laptop","paired_at":"2026-04-11T11:00:00Z","age_recipient":"age1def","age_recipient_fingerprint":"fpdef"}]}`))
		if err != nil {
			t.Fatalf("Write: %v", err)
		}
	}))
	defer server.Close()

	config := &Config{
		APIURL:     server.URL,
		APIToken:   "beebuzz_token",
		DeviceKeys: []DeviceKey{},
	}

	if err := refreshKeys(context.Background(), server.Client(), config); err != nil {
		t.Fatalf("refreshKeys: %v", err)
	}

	if len(config.DeviceKeys) != 2 {
		t.Fatalf("keys len: got %d, want 2", len(config.DeviceKeys))
	}
	if config.DeviceKeys[0].DeviceName != "phone" || config.DeviceKeys[1].AgeRecipient != "age1def" {
		t.Fatalf("device keys: got %#v", config.DeviceKeys)
	}
}

func TestRefreshKeysReturnsFormattedHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, err := w.Write([]byte(`{"code":"unauthorized","message":"invalid token"}`))
		if err != nil {
			t.Fatalf("Write: %v", err)
		}
	}))
	defer server.Close()

	config := &Config{
		APIURL:   server.URL,
		APIToken: "beebuzz_token",
	}

	err := refreshKeys(context.Background(), server.Client(), config)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "keys request failed with status 401 (unauthorized): invalid token" {
		t.Fatalf("error: got %q", err)
	}
}

func TestAppRunKeysUsesAPIURLFlagOverrideOverEnvAndConfig(t *testing.T) {
	t.Setenv(envBeeBuzzAPIURL, "https://env.example.com")

	baseDir := t.TempDir()
	configPath := filepath.Join(baseDir, profilesDirName, fallbackProfileName, configFileName)
	store := newProfileStore(func() (string, error) { return configBasePath() })
	if err := store.saveConfigToPath(configPath, &Config{
		APIURL:     "https://file.example.com",
		APIToken:   "beebuzz_token",
		DeviceKeys: []DeviceKey{},
	}); err != nil {
		t.Fatalf("saveConfigToPath: %v", err)
	}

	originalBasePath := configBasePath
	defer func() {
		configBasePath = originalBasePath
	}()
	configBasePath = func() (string, error) { return baseDir, nil }

	requestHit := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestHit = true
		if r.URL.Path != keysEndpointPath {
			t.Fatalf("path: got %q, want %q", r.URL.Path, keysEndpointPath)
		}
		if r.Header.Get("Authorization") != "Bearer beebuzz_token" {
			t.Fatalf("authorization: got %q", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "application/json")
		_, writeErr := w.Write([]byte(`{"data":[{"device_id":"dev-a","device_name":"phone","paired_at":"2026-04-11T10:00:00Z","age_recipient":"age1abc","age_recipient_fingerprint":"fpabc"}]}`))
		if writeErr != nil {
			t.Fatalf("Write: %v", writeErr)
		}
	}))
	defer server.Close()

	app := NewApp(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	app.httpClient = server.Client()

	if err := app.Run([]string{"keys", "--api-url", server.URL}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !requestHit {
		t.Fatal("expected request to api-url flag server")
	}
}
