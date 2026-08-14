package channels

import (
	"context"
	"time"
)

type ProductChannelListingRepository interface {
	Create(ctx context.Context, l *ProductChannelListing) error
	FindByExternalID(ctx context.Context, connectionID, externalID string) (*ProductChannelListing, error)
	// FindByProductAndConnection returns one representative listing for
	// this (product, connection) pair if any row exists — the
	// product-level row (VariantID nil) when one exists, else an
	// arbitrary variant-level row. Answers "is this product linked to this
	// connection at all" for idempotency/status checks that don't care
	// which specific row they get back. Callers that need every variant's
	// own listing use ListByProductAndConnection.
	FindByProductAndConnection(ctx context.Context, productID, connectionID string) (*ProductChannelListing, error)
	// FindByProductVariantAndConnection looks up one specific row —
	// variantID nil means the product-level row (whole simple product, or
	// the parent of a variable product).
	FindByProductVariantAndConnection(ctx context.Context, productID string, variantID *string, connectionID string) (*ProductChannelListing, error)
	ListByConnection(ctx context.Context, connectionID string) ([]ProductChannelListing, error)
	// ListByProduct returns every channel this product is linked to, across
	// all connections — used to fan a single product change out to every
	// channel it's synced to.
	ListByProduct(ctx context.Context, productID string) ([]ProductChannelListing, error)
	// ListByProductAndConnection returns every listing row (product-level
	// and/or per-variant) for one product on one connection — used by the
	// sync-mode migration to enumerate everything that needs tearing down
	// when the actual shape on the channel no longer matches the product's
	// current SyncVariantsAsWooVariations flag.
	ListByProductAndConnection(ctx context.Context, productID, connectionID string) ([]ProductChannelListing, error)
	Update(ctx context.Context, l *ProductChannelListing) error
	// Delete removes one listing row — used to tear down a stale
	// representation during a sync-mode migration.
	Delete(ctx context.Context, id string) error
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

type SalesChannelRepository interface {
	Create(ctx context.Context, c *SalesChannel) error
	FindByID(ctx context.Context, id string) (*SalesChannel, error)
	ListByTenant(ctx context.Context) ([]SalesChannel, error)
	Update(ctx context.Context, c *SalesChannel) error
	Delete(ctx context.Context, id string) error
}

// SalesChannelProductOverrideRepository is upsert-shaped per (ChannelID,
// ProductID) -- at most one override per product per channel, so a repeat
// "add"/"remove" action overwrites the prior override rather than
// accumulating rows.
type SalesChannelProductOverrideRepository interface {
	Upsert(ctx context.Context, o *SalesChannelProductOverride) error
	FindByChannelAndProduct(ctx context.Context, channelID, productID string) (*SalesChannelProductOverride, error)
	// ListByChannel returns every override for a channel -- used to compute
	// enrollment in bulk for the channel's product picker.
	ListByChannel(ctx context.Context, channelID string) ([]SalesChannelProductOverride, error)
	Delete(ctx context.Context, channelID, productID string) error
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
	// RecoverOrphanedRunning handles every job still marked "running" at
	// scheduler startup — called once, before polling begins. A job can
	// only be "running" while some scheduler process is actively inside
	// runImportJob; if the process that set it died (crash, redeploy)
	// before reaching a terminal state, the row is stuck forever and —
	// since Create's double-job guard checks pending/running — permanently
	// blocks any further import for that connection.
	//
	// Rather than just failing it (requiring a manual retrigger), a row
	// with RetryCount below maxRetries is reset to pending — the next poll
	// picks it right back up automatically, with RetryCount incremented
	// and StartedAt cleared. A row already at maxRetries is failed instead,
	// so a job whose own execution reliably crashes the scheduler can't
	// requeue itself forever. Returns (requeued, failed).
	RecoverOrphanedRunning(ctx context.Context, maxRetries int, errMsg string, finishedAt time.Time) (requeued, failed int64, err error)
}
