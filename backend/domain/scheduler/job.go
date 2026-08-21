// Package scheduler holds the small cross-tenant "global job" concept --
// recurring work that touches every tenant in one pass (cost-price
// recalculation, expired sale-price/batch sweeps), as opposed to
// domain/channels.SyncSchedule which is one row per tenant connection. See
// migration 085_scheduled_jobs.sql's doc comment for why this has no
// tenant_id/RLS.
package scheduler

import (
	"context"
	"errors"
	"time"
)

var ErrJobNotFound = errors.New("scheduled job not found")

// Known job keys -- application/scheduler's use cases are keyed by these,
// and cmd/scheduler dispatches to them the same way dispatch.go's runners
// map dispatches SyncSchedule rows.
const (
	JobExpiredSalePrices      = "expired_sale_prices"
	JobExpiredBatches         = "expired_batches"
	JobCostPriceRecalculation = "cost_price_recalculation"
)

type Job struct {
	ID              int64
	JobKey          string
	Enabled         bool
	IntervalMinutes int
	LastRunAt       *time.Time
	LastRunStatus   string // "" | "success" | "error"
	LastRunError    string
	NextRunAt       time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type JobRepository interface {
	ListAll(ctx context.Context) ([]Job, error)
	ListDue(ctx context.Context, now time.Time) ([]Job, error)
	FindByKey(ctx context.Context, jobKey string) (*Job, error)
	// RecordRunResult stamps the outcome of a run that actually happened
	// (ranAt becomes LastRunAt) and schedules the next one.
	RecordRunResult(ctx context.Context, jobKey string, ranAt time.Time, status, errMsg string, nextRunAt time.Time) error
	// TriggerNow pulls NextRunAt to now, without touching LastRun* -- the
	// job hasn't run yet, this just makes cmd/scheduler's next poll tick
	// pick it up (or, for a job whose "run now" instead runs synchronously
	// in the API process, still keeps the persisted schedule consistent so
	// the periodic poll doesn't immediately re-run it again seconds later
	// -- see application/scheduler.RunJobNowUseCase).
	TriggerNow(ctx context.Context, jobKey string) error
}
