package auth

import (
	"testing"
	"time"
)

func TestEmailThrottleAllow(t *testing.T) {
	base := time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)
	current := base

	throttle := NewEmailThrottle(3, 15*time.Minute, time.Minute)
	throttle.now = func() time.Time {
		return current
	}

	t.Run("allows first request", func(t *testing.T) {
		if !throttle.Allow("user@example.com") {
			t.Fatal("Allow() = false, want true")
		}
	})

	t.Run("rejects requests within cooldown", func(t *testing.T) {
		current = base.Add(30 * time.Second)
		if throttle.Allow("user@example.com") {
			t.Fatal("Allow() = true, want false")
		}
	})

	t.Run("allows requests after cooldown", func(t *testing.T) {
		current = base.Add(90 * time.Second)
		if !throttle.Allow("user@example.com") {
			t.Fatal("Allow() = false, want true")
		}
	})

	t.Run("allows third request within rolling window", func(t *testing.T) {
		current = base.Add(3 * time.Minute)
		if !throttle.Allow("user@example.com") {
			t.Fatal("Allow() third request = false, want true")
		}
	})

	t.Run("rejects fourth request within rolling window", func(t *testing.T) {
		current = base.Add(5 * time.Minute)
		if throttle.Allow("user@example.com") {
			t.Fatal("Allow() fourth request = true, want false")
		}
	})

	t.Run("purges attempts outside the rolling window", func(t *testing.T) {
		current = base.Add(16 * time.Minute)
		if !throttle.Allow("user@example.com") {
			t.Fatal("Allow() after window expiry = false, want true")
		}
	})

	t.Run("evicts idle entries after the window and cooldown pass", func(t *testing.T) {
		current = base.Add(32 * time.Minute)
		if !throttle.Allow("new-user@example.com") {
			t.Fatal("Allow() = false, want true")
		}

		current = base.Add(48 * time.Minute)
		if !throttle.Allow("new-user@example.com") {
			t.Fatal("Allow() after eviction = false, want true")
		}

		if entry, ok := throttle.entries["new-user@example.com"]; !ok || len(entry.attempts) != 1 {
			t.Fatal("expected idle entry to be rebuilt with a single fresh attempt")
		}
	})
}

func TestNewEmailThrottleClampsCooldownToWindow(t *testing.T) {
	throttle := NewEmailThrottle(3, time.Minute, 5*time.Minute)
	if throttle.cooldown != time.Minute {
		t.Fatalf("cooldown = %v, want %v", throttle.cooldown, time.Minute)
	}
}
