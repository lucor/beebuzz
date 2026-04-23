package main

import (
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"syscall"
	"testing"

	"lucor.dev/beebuzz/internal/config"
)

// TestRunHTTPServerReturnsStartupError verifies startup bind failures are propagated.
func TestRunHTTPServerReturnsStartupError(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to reserve port: %v", err)
	}
	defer listener.Close()

	httpServer := &http.Server{
		Addr: listener.Addr().String(),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	}

	err = runHTTPServer(context.Background(), httpServer)
	if err == nil {
		t.Fatal("runHTTPServer() error = nil, want startup error")
	}
	if !errors.Is(err, syscall.EADDRINUSE) {
		t.Fatalf("runHTTPServer() error = %v, want address in use", err)
	}
}

func TestRunRequiresExplicitSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run(nil, &stdout, &stderr)
	if err == nil {
		t.Fatal("run() error = nil, want missing subcommand")
	}
	if stderr.Len() == 0 {
		t.Fatal("run() usage output is empty")
	}
}

func TestRunRejectsUnknownSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run([]string{"bogus"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("run() error = nil, want unknown subcommand")
	}
	if stderr.Len() == 0 {
		t.Fatal("run() usage output is empty")
	}
}

func TestRunGenerateVAPIDWritesEnvAssignments(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := run([]string{"vapid", "generate"}, &stdout, &stderr); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	output := stdout.String()
	if !bytes.Contains(stdout.Bytes(), []byte("BEEBUZZ_VAPID_PRIVATE_KEY=")) {
		t.Fatalf("output = %q, want private key assignment", output)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("BEEBUZZ_VAPID_PUBLIC_KEY=")) {
		t.Fatalf("output = %q, want public key assignment", output)
	}
}

func TestRunHealthcheck(t *testing.T) {
	t.Run("succeeds", func(t *testing.T) {
		_, port := newHealthcheckServer(t, http.StatusOK)

		t.Setenv("BEEBUZZ_PORT", port)

		if err := runHealthcheck(); err != nil {
			t.Fatalf("runHealthcheck() error = %v", err)
		}
	})

	t.Run("fails on non-ok response", func(t *testing.T) {
		_, port := newHealthcheckServer(t, http.StatusServiceUnavailable)

		t.Setenv("BEEBUZZ_PORT", port)

		if err := runHealthcheck(); err == nil {
			t.Fatal("runHealthcheck() error = nil, want health check failure")
		}
	})
}

func TestLoadVAPIDKeysFromEnv(t *testing.T) {
	cfg := &config.Config{
		VAPIDPublicKey:  "public-key",
		VAPIDPrivateKey: "private-key",
	}

	keys, err := loadVAPIDKeys(cfg)
	if err != nil {
		t.Fatalf("loadVAPIDKeys() error = %v", err)
	}
	if keys.PublicKey != cfg.VAPIDPublicKey || keys.PrivateKey != cfg.VAPIDPrivateKey {
		t.Fatalf("loadVAPIDKeys() = %#v, want env keys", keys)
	}
}

func TestLoadVAPIDKeysRequiresBothKeys(t *testing.T) {
	t.Run("rejects missing public key", func(t *testing.T) {
		cfg := &config.Config{
			VAPIDPrivateKey: "private-key",
		}

		_, err := loadVAPIDKeys(cfg)
		if err == nil {
			t.Fatal("loadVAPIDKeys() error = nil, want missing key error")
		}
	})

	t.Run("rejects missing private key", func(t *testing.T) {
		cfg := &config.Config{
			VAPIDPublicKey: "public-key",
		}

		_, err := loadVAPIDKeys(cfg)
		if err == nil {
			t.Fatal("loadVAPIDKeys() error = nil, want missing key error")
		}
	})
}

func newHealthcheckServer(t *testing.T, status int) (*httptest.Server, string) {
	t.Helper()

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(status)
	}))

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}

	server.Listener = listener
	server.Start()

	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("failed to parse listener address: %v", err)
	}
	if _, err := strconv.Atoi(port); err != nil {
		t.Fatalf("listener port is invalid: %v", err)
	}

	t.Cleanup(server.Close)

	return server, port
}
