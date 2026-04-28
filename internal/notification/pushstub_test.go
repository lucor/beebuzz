package notification

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPushStubBroker_PublishNext(t *testing.T) {
	b := NewPushStubBroker(nil)

	ev := PushStubEvent{Endpoint: "https://example/push", DeviceID: "dev-1", Data: `{"k":"v"}`}
	b.Publish(ev)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	got, err := b.Next(ctx)
	if err != nil {
		t.Fatalf("Next returned error: %v", err)
	}
	if got != ev {
		t.Fatalf("got %+v, want %+v", got, ev)
	}
}

func TestPushStubBroker_NextRespectsContextCancellation(t *testing.T) {
	b := NewPushStubBroker(nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := b.Next(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestPushStubBroker_OverflowDropsAndLogs(t *testing.T) {
	var logBuf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelWarn}))
	b := NewPushStubBroker(log)

	for i := 0; i < pushStubBufferSize+5; i++ {
		b.Publish(PushStubEvent{DeviceID: "dev"})
	}

	if !strings.Contains(logBuf.String(), "push stub broker overflow") {
		t.Fatalf("expected overflow warning, got log: %q", logBuf.String())
	}

	// Drain exactly pushStubBufferSize events without blocking.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	for i := 0; i < pushStubBufferSize; i++ {
		if _, err := b.Next(ctx); err != nil {
			t.Fatalf("Next %d returned error: %v", i, err)
		}
	}
}

func TestIsLoopback(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		want       bool
	}{
		{"ipv4 loopback with port", "127.0.0.1:54321", true},
		{"ipv4 loopback without port", "127.0.0.1", true},
		{"ipv6 loopback", "[::1]:8080", true},
		{"ipv4 lan", "192.168.1.10:1234", false},
		{"ipv4 public", "8.8.8.8:443", false},
		{"empty", "", false},
		{"garbage", "not-an-ip", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/_stub/push/next", nil)
			req.RemoteAddr = tt.remoteAddr
			if got := isLoopback(req); got != tt.want {
				t.Fatalf("isLoopback(%q) = %v, want %v", tt.remoteAddr, got, tt.want)
			}
		})
	}
}
