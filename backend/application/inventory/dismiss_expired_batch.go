package inventory

import (
	"context"
	"fmt"

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
}

func NewDismissExpiredBatchUseCase(db *gorm.DB, batchRepo invDomain.BatchRepository) *DismissExpiredBatchUseCase {
	return &DismissExpiredBatchUseCase{db: db, batchRepo: batchRepo}
}

func (uc *DismissExpiredBatchUseCase) Execute(ctx context.Context, batchID string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.batchRepo.MarkReviewed(ctx, batchID)
	})
	if err != nil {
		return fmt.Errorf("dismissExpiredBatch: %w", err)
	}
	return nil
}
