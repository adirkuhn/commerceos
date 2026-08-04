package channels

import (
	"context"
	"time"
)

type ProductChannelListingRepository interface {
	Create(ctx context.Context, l *ProductChannelListing) error
	FindByExternalID(ctx context.Context, connectionID, externalID string) (*ProductChannelListing, error)
	FindByProductAndConnection(ctx context.Context, productID, connectionID string) (*ProductChannelListing, error)
	ListByConnection(ctx context.Context, connectionID string) ([]ProductChannelListing, error)
	// ListByProduct returns every channel this product is linked to, across
	// all connections — used to fan a single product change out to every
	// channel it's synced to.
	ListByProduct(ctx context.Context, productID string) ([]ProductChannelListing, error)
	Update(ctx context.Context, l *ProductChannelListing) error
}

// SyncLogRepository is insert-only, no-op-free by design — a sync log
// entry is never updated once written.
type SyncLogRepository interface {
	Create(ctx context.Context, e *SyncLogEntry) error
	// ListByListing returns entries newest-first, capped at limit.
	ListByListing(ctx context.Context, listingID string, limit int) ([]SyncLogEntry, error)
	// ListAll returns every sync log entry for the current tenant
	// (RLS-scoped), newest first, paginated — powers the tenant-wide sync
	// history page (as opposed to ListByListing's single-product view).
	ListAll(ctx context.Context, page, perPage int) ([]SyncLogEntry, int64, error)
}

type SyncScheduleRepository interface {
	Upsert(ctx context.Context, s *SyncSchedule) error
	ListByConnection(ctx context.Context, connectionID string) ([]SyncSchedule, error)
	ListDue(ctx context.Context, now time.Time) ([]SyncSchedule, error)
	RecordRunResult(ctx context.Context, id string, ranAt time.Time, status, errMsg string, nextRunAt time.Time) error
}

type ImportJobRepository interface {
	Create(ctx context.Context, j *ImportJob) error
	// HasActiveForConnection checks for an existing pending/running job —
	// the proactive half of the double-job guard (Create's unique-index
	// violation is the backstop for the race window between check and
	// insert).
	HasActiveForConnection(ctx context.Context, connectionID string) (bool, error)
	// ListPending is a system-level, cross-tenant query — cmd/scheduler
	// polls it under tenant.WithSuperAdmin, same as SyncSchedule.ListDue.
	ListPending(ctx context.Context) ([]ImportJob, error)
	MarkRunning(ctx context.Context, id string, startedAt time.Time) error
	MarkCompleted(ctx context.Context, id string, imported, alreadyLinked, skippedType int, errs []string, finishedAt time.Time) error
	MarkFailed(ctx context.Context, id string, errMsg string, finishedAt time.Time) error
	// MarkOrphanedRunningAsFailed fails every job still marked "running" —
	// called once at scheduler startup. A job can only be "running" while
	// some scheduler process is actively inside runImportJob; if the
	// process that set it died (crash, redeploy) before reaching a
	// terminal state, the row is stuck forever and — since Create's
	// double-job guard checks pending/running — permanently blocks any
	// further import for that connection. Returns the count recovered.
	MarkOrphanedRunningAsFailed(ctx context.Context, errMsg string, finishedAt time.Time) (int64, error)
}
