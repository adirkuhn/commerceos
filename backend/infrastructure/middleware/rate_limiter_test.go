package middleware_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func TestIPRateLimiter_allowsUpToMaxAttemptsThenBlocks(t *testing.T) {
	l := middleware.NewIPRateLimiter(3, time.Minute)

	assert.True(t, l.Allow("1.2.3.4"))
	assert.True(t, l.Allow("1.2.3.4"))
	assert.True(t, l.Allow("1.2.3.4"))
	assert.False(t, l.Allow("1.2.3.4"), "4th attempt within the window must be blocked")
	assert.False(t, l.Allow("1.2.3.4"), "stays blocked, does not reset on its own mid-window")
}

func TestIPRateLimiter_tracksKeysIndependently(t *testing.T) {
	l := middleware.NewIPRateLimiter(1, time.Minute)

	assert.True(t, l.Allow("1.2.3.4"))
	assert.False(t, l.Allow("1.2.3.4"))
	assert.True(t, l.Allow("5.6.7.8"), "a different key must have its own independent budget")
}

func TestIPRateLimiter_resetsAfterWindowElapses(t *testing.T) {
	l := middleware.NewIPRateLimiter(1, 20*time.Millisecond)

	assert.True(t, l.Allow("1.2.3.4"))
	assert.False(t, l.Allow("1.2.3.4"))

	time.Sleep(25 * time.Millisecond)

	assert.True(t, l.Allow("1.2.3.4"), "window elapsed, budget must reset")
}
