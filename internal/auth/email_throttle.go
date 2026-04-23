package auth

import (
	"sync"
	"time"
)

// EmailThrottle silently drops abusive login requests for a single email identity.
// This keeps the auth response uniform while still protecting the mail channel.
type EmailThrottle struct {
	mu       sync.Mutex
	now      func() time.Time
	window   time.Duration
	cooldown time.Duration
	limit    int
	entries  map[string]*emailThrottleEntry
}

type emailThrottleEntry struct {
	lastAttemptAt time.Time
	attempts      []time.Time
}

// NewEmailThrottle creates an email throttle with a rolling window and cooldown.
// Cooldown is clamped to the window so config changes cannot keep a key blocked
// longer than the rolling history used to justify the throttle.
func NewEmailThrottle(limit int, window time.Duration, cooldown time.Duration) *EmailThrottle {
	if cooldown > window {
		cooldown = window
	}

	return &EmailThrottle{
		now:      time.Now,
		window:   window,
		cooldown: cooldown,
		limit:    limit,
		entries:  make(map[string]*emailThrottleEntry),
	}
}

// Allow reports whether a request for the given email should proceed.
// Throttled attempts refresh the cooldown so repeated hammering stays quiet.
func (t *EmailThrottle) Allow(email string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := t.now().UTC()
	entry, ok := t.entries[email]
	if !ok {
		entry = &emailThrottleEntry{}
		t.entries[email] = entry
	}

	entry.attempts = pruneAttempts(entry.attempts, now.Add(-t.window))
	if len(entry.attempts) == 0 && !entry.lastAttemptAt.IsZero() && now.Sub(entry.lastAttemptAt) >= t.cooldown {
		delete(t.entries, email)
		entry = &emailThrottleEntry{}
		t.entries[email] = entry
	}

	if len(entry.attempts) == 0 && entry.lastAttemptAt.IsZero() {
		entry.attempts = append(entry.attempts, now)
		entry.lastAttemptAt = now
		return true
	}

	if !entry.lastAttemptAt.IsZero() && now.Sub(entry.lastAttemptAt) < t.cooldown {
		entry.lastAttemptAt = now
		return false
	}

	if len(entry.attempts) >= t.limit {
		entry.lastAttemptAt = now
		return false
	}

	entry.attempts = append(entry.attempts, now)
	entry.lastAttemptAt = now
	return true
}

func pruneAttempts(attempts []time.Time, cutoff time.Time) []time.Time {
	firstValid := 0
	for firstValid < len(attempts) && attempts[firstValid].Before(cutoff) {
		firstValid++
	}
	return attempts[firstValid:]
}
