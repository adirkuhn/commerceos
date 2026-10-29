package moneybird

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLimiter_burstPassesThenWaitsForRefill(t *testing.T) {
	now := time.Unix(0, 0)
	var slept time.Duration
	l := newLimiter(2, time.Second)
	l.now = func() time.Time { return now }
	l.sleep = func(ctx context.Context, d time.Duration) error { slept += d; now = now.Add(d); return nil }

	require.NoError(t, l.Wait(context.Background()))
	require.NoError(t, l.Wait(context.Background()))
	assert.Zero(t, slept, "the burst needs no waiting")

	require.NoError(t, l.Wait(context.Background()))
	assert.Equal(t, time.Second, slept, "the third call waits for one token to refill")
}

func TestLimiter_refillsOverTimeUpToBurst(t *testing.T) {
	now := time.Unix(0, 0)
	l := newLimiter(2, time.Second)
	l.now = func() time.Time { return now }
	l.sleep = func(ctx context.Context, d time.Duration) error { t.Fatalf("should not wait, asked %v", d); return nil }

	require.NoError(t, l.Wait(context.Background()))
	require.NoError(t, l.Wait(context.Background()))
	now = now.Add(10 * time.Second) // refills, but never above the burst of 2
	require.NoError(t, l.Wait(context.Background()))
	require.NoError(t, l.Wait(context.Background()))
}

func TestLimiter_cancelledContextStopsWaiting(t *testing.T) {
	l := newLimiter(1, time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	require.NoError(t, l.Wait(ctx))
	cancel()
	assert.ErrorIs(t, l.Wait(ctx), context.Canceled)
}
