package channels

import (
	"context"
	"fmt"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// RetryFailedSyncUseCase is the "Needs attention" view's resolve action --
// re-runs a full product push to the same connection the failed entry was
// for (not a narrow replay of just the one field that failed), so the
// retry always leaves the listing fully caught up regardless of what
// originally went wrong. Writes its own new SyncLogEntry via
// PushProductToWooUseCase's own logging (see sync_log_shared.go) -- the
// original failed entry is never edited, only ever superseded by a newer
// one, same append-only convention as the rest of the log. productPusher
// is *PushProductToWooUseCase (see run_scheduled_product_export.go).
type RetryFailedSyncUseCase struct {
	db          *gorm.DB
	syncLogRepo channelsDomain.SyncLogRepository
	pusher      productPusher
}

func NewRetryFailedSyncUseCase(db *gorm.DB, syncLogRepo channelsDomain.SyncLogRepository, pusher productPusher) *RetryFailedSyncUseCase {
	return &RetryFailedSyncUseCase{db: db, syncLogRepo: syncLogRepo, pusher: pusher}
}

func (uc *RetryFailedSyncUseCase) Execute(ctx context.Context, syncLogEntryID string) error {
	var entry *channelsDomain.SyncLogEntry
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		entry, err = uc.syncLogRepo.FindByID(ctx, syncLogEntryID)
		return err
	})
	if err != nil {
		return fmt.Errorf("retryFailedSync: %w", err)
	}
	if entry.ListingID == nil {
		return fmt.Errorf("retryFailedSync: %w", channelsDomain.ErrSyncLogHasNoListing)
	}

	// PushProductToWooUseCase opens its own tenant.WithTenant internally --
	// called outside the block above, never nested.
	if _, err := uc.pusher.Execute(ctx, PushProductToWooInput{ProductID: entry.ProductID, ConnectionID: entry.ConnectionID}); err != nil {
		return fmt.Errorf("retryFailedSync: %w", err)
	}
	return nil
}
