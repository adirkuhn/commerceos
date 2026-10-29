package moneybird

import (
	"context"
	"sync"
	"time"
)

// limiter is a token bucket: up to burst calls at once, then one call per
// interval. Moneybird allows 150 requests per 5 minutes per IP; shared by
// every client this process builds (see NewFactory).
type limiter struct {
	mu       sync.Mutex
	tokens   float64
	burst    float64
	interval time.Duration
	last     time.Time

	now   func() time.Time
	sleep func(ctx context.Context, d time.Duration) error
}

func newLimiter(burst int, interval time.Duration) *limiter {
	return &limiter{
		tokens: float64(burst), burst: float64(burst), interval: interval,
		now: time.Now, sleep: sleepCtx,
	}
}

// Wait blocks until a call may be made, or ctx is done.
func (l *limiter) Wait(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	l.mu.Lock()
	now := l.now()
	if !l.last.IsZero() {
		l.tokens += float64(now.Sub(l.last)) / float64(l.interval)
		if l.tokens > l.burst {
			l.tokens = l.burst
		}
	}
	l.last = now
	l.tokens--
	var wait time.Duration
	if l.tokens < 0 {
		// The token is taken now; the caller waits until it would have refilled.
		wait = time.Duration(-l.tokens * float64(l.interval))
	}
	l.mu.Unlock()
	if wait == 0 {
		return nil
	}
	return l.sleep(ctx, wait)
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
