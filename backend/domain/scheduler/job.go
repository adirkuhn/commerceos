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
var ErrInvalidIntervalMinutes = errors.New("interval_minutes must be at least 1")

// Known job keys -- application/scheduler's use cases are keyed by these,
// and cmd/scheduler dispatches to them the same way dispatch.go's runners
// map dispatches SyncSchedule rows.
const (
	JobExpiredSalePrices           = "expired_sale_prices"
	JobExpiredBatches              = "expired_batches"
	JobCostPriceRecalculation      = "cost_price_recalculation"
	JobResyncAllProductsToChannels = "resync_all_products_to_channels"
	JobPromotionActivation         = "promotion_activation"
	JobReconcileFulfilledOrders    = "reconcile_fulfilled_orders"
	// JobCheckBolShipmentConfirmations asks Bol for the result of every
	// pending shipment confirmation (see application/channels.CheckBolShipmentConfirmationsUseCase).
	JobCheckBolShipmentConfirmations = "check_bol_shipment_confirmations"
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
	// UpdateConfig sets Enabled/IntervalMinutes -- the super_admin
	// "Scheduled Jobs" page's edit action. Does not touch NextRunAt:
	// enabling a job that was previously disabled picks it up on the next
	// poll tick using whatever NextRunAt already holds (normally already
	// due, since disabling never advances it), and changing the interval
	// only affects the schedule computed after the next run finishes --
	// consistent with how every other config change here behaves, and
	// avoids a wide gap of inactivity if this were called right after a
	// run had just pushed NextRunAt far into the future under the old
	// interval.
	UpdateConfig(ctx context.Context, jobKey string, enabled bool, intervalMinutes int) error
}
