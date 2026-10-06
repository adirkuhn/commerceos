package inventory

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// DismissExpiredBatchUseCase acks an Expired batch from the review queue
// without changing anything about the batch itself -- Status stays
// Expired and the quantity is untouched, on record for audit, per the
// "Dismiss" action's spec.
type DismissExpiredBatchUseCase struct {
	db        *gorm.DB
	batchRepo invDomain.BatchRepository
	outbox    eventrecord.Enqueuer
}

func NewDismissExpiredBatchUseCase(db *gorm.DB, batchRepo invDomain.BatchRepository, outbox eventrecord.Enqueuer) *DismissExpiredBatchUseCase {
	return &DismissExpiredBatchUseCase{db: db, batchRepo: batchRepo, outbox: outbox}
}

func (uc *DismissExpiredBatchUseCase) Execute(ctx context.Context, batchID string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return markBatchReviewed(ctx, uc.batchRepo, uc.outbox, batchID, BatchReviewOutcomeDismissed)
	})
	if err != nil {
		return fmt.Errorf("dismissExpiredBatch: %w", err)
	}
	return nil
}
