package middleware

import (
	"sync"
	"time"
)

// IPRateLimiter is an in-memory, single-instance sliding-window limiter
// keyed by an arbitrary string (typically client IP). Not shared across
// replicas -- fine for a single-instance deploy; horizontal scaling would
// need a shared store (Redis) instead.
type IPRateLimiter struct {
	mu       sync.Mutex
	attempts map[string]*rateWindow
	max      int
	interval time.Duration
	now      func() time.Time
}

type rateWindow struct {
	count   int
	resetAt time.Time
}

func NewIPRateLimiter(max int, interval time.Duration) *IPRateLimiter {
	return &IPRateLimiter{attempts: make(map[string]*rateWindow), max: max, interval: interval, now: time.Now}
}

// Allow reports whether key may proceed, incrementing its attempt count.
// Returns false once key has exceeded max attempts within the current window.
func (l *IPRateLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	w, ok := l.attempts[key]
	if !ok || now.After(w.resetAt) {
		w = &rateWindow{resetAt: now.Add(l.interval)}
		l.attempts[key] = w
	}
	w.count++
	return w.count <= l.max
}
