package auth

import (
	"sync"
	"time"
)

// GlobalAuthThrottle caps total login attempts per instance in a rolling window.
// This acts as a mail-channel safety valve when abuse is distributed across emails and IPs.
type GlobalAuthThrottle struct {
	mu       sync.Mutex
	now      func() time.Time
	window   time.Duration
	limit    int
	attempts []time.Time
}

// NewGlobalAuthThrottle creates a rolling-window throttle for all auth login attempts.
func NewGlobalAuthThrottle(limit int, window time.Duration) *GlobalAuthThrottle {
	return &GlobalAuthThrottle{
		now:    time.Now,
		window: window,
		limit:  limit,
	}
}

// Allow reports whether another login attempt can proceed on this instance.
func (g *GlobalAuthThrottle) Allow() bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := g.now().UTC()
	g.attempts = pruneAttempts(g.attempts, now.Add(-g.window))
	if len(g.attempts) >= g.limit {
		return false
	}

	g.attempts = append(g.attempts, now)
	return true
}
