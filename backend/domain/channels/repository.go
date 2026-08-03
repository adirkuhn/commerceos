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
