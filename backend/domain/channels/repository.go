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
	Update(ctx context.Context, l *ProductChannelListing) error
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
}
