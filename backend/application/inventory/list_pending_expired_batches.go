package inventory

import (
	"context"
	"fmt"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// ListPendingExpiredBatchesUseCase is the expired-batch review queue's data
// source -- every Expired batch the tenant hasn't acted on yet.
type ListPendingExpiredBatchesUseCase struct {
	db        *gorm.DB
	batchRepo invDomain.BatchRepository
}

func NewListPendingExpiredBatchesUseCase(db *gorm.DB, batchRepo invDomain.BatchRepository) *ListPendingExpiredBatchesUseCase {
	return &ListPendingExpiredBatchesUseCase{db: db, batchRepo: batchRepo}
}

func (uc *ListPendingExpiredBatchesUseCase) Execute(ctx context.Context) ([]invDomain.ExpiredBatchReview, error) {
	var result []invDomain.ExpiredBatchReview
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		result, err = uc.batchRepo.ListPendingReview(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listPendingExpiredBatches: %w", err)
	}
	return result, nil
}
