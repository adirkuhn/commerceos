package channels

import (
	"context"
	"fmt"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// DismissFailedSyncUseCase is the "Needs attention" view's other resolve
// action -- for a failure staff have decided isn't worth retrying (the Woo
// product was deleted, a known/expected mismatch, etc.), silencing it from
// that view without pretending it succeeded. See SyncLogEntry.DismissedAt's
// doc for why this is the one field ever updated on an otherwise
// append-only log row.
type DismissFailedSyncUseCase struct {
	db          *gorm.DB
	syncLogRepo channelsDomain.SyncLogRepository
}

func NewDismissFailedSyncUseCase(db *gorm.DB, syncLogRepo channelsDomain.SyncLogRepository) *DismissFailedSyncUseCase {
	return &DismissFailedSyncUseCase{db: db, syncLogRepo: syncLogRepo}
}

func (uc *DismissFailedSyncUseCase) Execute(ctx context.Context, syncLogEntryID string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.syncLogRepo.Dismiss(ctx, syncLogEntryID)
	})
	if err != nil {
		return fmt.Errorf("dismissFailedSync: %w", err)
	}
	return nil
}
