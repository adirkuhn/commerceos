package main

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type countingDue struct{ calls atomic.Int32 }

func (d *countingDue) Execute(context.Context) (int, error) {
	d.calls.Add(1)
	return 0, errors.New("db down")
}

type countingStale struct {
	calls     atomic.Int32
	olderThan time.Duration
}

func (s *countingStale) Execute(_ context.Context, olderThan time.Duration) (int64, error) {
	s.calls.Add(1)
	s.olderThan = olderThan
	return 2, nil
}

func TestRunAssistantWorker_requeuesStaleOnceThenPollsUntilStopped(t *testing.T) {
	due, stale := &countingDue{}, &countingStale{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		runAssistantWorker(ctx, due, stale, 5*time.Millisecond)
		close(done)
	}()
	assert.Eventually(t, func() bool { return due.calls.Load() >= 3 }, time.Second, 5*time.Millisecond,
		"a failing tick is logged and the loop keeps polling")
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker didn't stop when ctx ended")
	}
	assert.EqualValues(t, 1, stale.calls.Load())
	assert.Equal(t, assistantStaleAfter, stale.olderThan)
}
