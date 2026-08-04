package channels

import (
	"context"
	"fmt"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

const maxSyncLogEntries = 50

// ListSyncLogsUseCase powers the audit trail shown on a product's
// per-channel tab: everything that's been synced for one listing, newest
// first.
type ListSyncLogsUseCase struct {
	db          *gorm.DB
	listingRepo channelsDomain.ProductChannelListingRepository
	syncLogRepo channelsDomain.SyncLogRepository
}

func NewListSyncLogsUseCase(db *gorm.DB, listingRepo channelsDomain.ProductChannelListingRepository, syncLogRepo channelsDomain.SyncLogRepository) *ListSyncLogsUseCase {
	return &ListSyncLogsUseCase{db: db, listingRepo: listingRepo, syncLogRepo: syncLogRepo}
}

func (uc *ListSyncLogsUseCase) Execute(ctx context.Context, productID, connectionID string) ([]channelsDomain.SyncLogEntry, error) {
	var entries []channelsDomain.SyncLogEntry
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		listing, err := uc.listingRepo.FindByProductAndConnection(ctx, productID, connectionID)
		if err != nil {
			return err
		}
		entries, err = uc.syncLogRepo.ListByListing(ctx, listing.ID, maxSyncLogEntries)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listSyncLogs: %w", err)
	}
	return entries, nil
}
