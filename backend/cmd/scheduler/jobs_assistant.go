package main

import (
	"context"
	"log"
	"time"

	appassistant "github.com/adirkuhn/commerceos/application/assistant"
)

const (
	// assistantPollInterval: a person is waiting on the answer, so the queue
	// is checked often; an empty poll is one indexed query.
	assistantPollInterval = 3 * time.Second
	// assistantBatchSize caps requests claimed per tick -- each attempt can
	// take several model calls, and unclaimed ones stay free for the next
	// tick (or another scheduler process).
	assistantBatchSize = 5
	// assistantStaleAfter: an attempt is bounded by MaxSteps model calls,
	// far below this; a request processing longer was left by a process
	// that died mid-attempt.
	assistantStaleAfter = 10 * time.Minute
)

type dueAssistantRequests interface {
	Execute(ctx context.Context) (int, error)
}

type staleAssistantRequests interface {
	Execute(ctx context.Context, olderThan time.Duration) (int64, error)
}

// runAssistantWorker requeues requests a previous process left mid-attempt,
// then polls the queue in its own goroutine until ctx ends -- model calls
// are slow, so they never hold up the scheduler's other jobs. A tick runs
// to completion before the next starts.
func runAssistantWorker(ctx context.Context, due dueAssistantRequests, stale staleAssistantRequests, interval time.Duration) {
	if n, err := stale.Execute(ctx, assistantStaleAfter); err != nil {
		log.Printf("assistant: requeue stale requests: %v", err)
	} else if n > 0 {
		log.Printf("assistant: requeued %d request(s) left processing by a previous scheduler process", n)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := due.Execute(ctx); err != nil {
				log.Printf("assistant: %v", err)
			}
		}
	}
}

var (
	_ dueAssistantRequests   = (*appassistant.ProcessDueRequestsUseCase)(nil)
	_ staleAssistantRequests = (*appassistant.RequeueStaleRequestsUseCase)(nil)
)
