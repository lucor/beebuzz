package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"syscall"
	"testing"

	"beebuzz.app/beebuzz/internal/config"
)

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
