package channels

import "time"

type ImportJobStatus string

const (
	ImportJobPending   ImportJobStatus = "pending"
	ImportJobRunning   ImportJobStatus = "running"
	ImportJobCompleted ImportJobStatus = "completed"
	ImportJobFailed    ImportJobStatus = "failed"
)

// ImportJob is a one-off "import all WooCommerce products now" background
// job — created by the manual Import button, picked up and run by
// cmd/scheduler alongside its recurring SyncSchedule polling (see
// sync_schedule.go). At most one pending/running job may exist per
// connection at a time, enforced by a partial unique index on
// (connection_id) — the authoritative guard against double-import-clicks
// spawning duplicate jobs.
type ImportJob struct {
	ID                 string
	TenantID           string
	ConnectionID       string
	DefaultWarehouseID string
	DefaultCurrency    string
	Status             ImportJobStatus
	Imported           int
	AlreadyLinked      int
	SkippedType        int
	Errors             []string
	// ErrorMessage is set only for a hard failure (e.g. bad credentials) —
	// distinct from Errors, which are tolerated per-product failures that
	// don't stop the run (see ImportWooProductsUseCase).
	ErrorMessage string
	CreatedAt    time.Time
	StartedAt    *time.Time
	FinishedAt   *time.Time
	// RetryCount tracks automatic requeue-on-restart attempts (see
	// ImportJobRepository.RecoverOrphanedRunning) — not user-visible,
	// purely so a job whose own execution crashes the scheduler process
	// can't requeue itself forever.
	RetryCount int
	// IsFirstImport is decided once by CreateImportJobUseCase, at job
	// creation, before any attempt runs -- true when the connection had no
	// imported products yet at that moment. Stuck to across retries (never
	// re-derived at run time) so a crash-and-requeue mid-first-import
	// doesn't wrongly skip the orders bootstrap on the retry. See
	// RunManualWooImportUseCase.
	IsFirstImport bool
}
