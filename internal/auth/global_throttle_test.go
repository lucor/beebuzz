package auth

import (
	"testing"
	"time"
)

func TestGlobalAuthThrottleAllow(t *testing.T) {
	base := time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)
	current := base

	throttle := NewGlobalAuthThrottle(2, time.Minute)
	throttle.now = func() time.Time {
		return current
	}

	t.Run("allows requests under the limit", func(t *testing.T) {
		if !throttle.Allow() {
			t.Fatal("Allow() first request = false, want true")
		}

		current = base.Add(30 * time.Second)
		if !throttle.Allow() {
			t.Fatal("Allow() second request = false, want true")
		}
	})

	t.Run("rejects requests once the rolling limit is reached", func(t *testing.T) {
		current = base.Add(45 * time.Second)
		if throttle.Allow() {
			t.Fatal("Allow() third request = true, want false")
		}
	})

	t.Run("allows requests again after the rolling window expires", func(t *testing.T) {
		current = base.Add(90 * time.Second)
		if !throttle.Allow() {
			t.Fatal("Allow() after window expiry = false, want true")
		}
	})
}
