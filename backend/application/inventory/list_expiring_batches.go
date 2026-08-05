package inventory

import (
	"context"
	"fmt"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type ListExpiringBatchesUseCase struct {
	db        *gorm.DB
	batchRepo invDomain.BatchRepository
}

func NewListExpiringBatchesUseCase(db *gorm.DB, batchRepo invDomain.BatchRepository) *ListExpiringBatchesUseCase {
	return &ListExpiringBatchesUseCase{db: db, batchRepo: batchRepo}
}

func (uc *ListExpiringBatchesUseCase) Execute(ctx context.Context, withinDays int) ([]invDomain.ExpiringBatch, error) {
	var batches []invDomain.ExpiringBatch
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		batches, err = uc.batchRepo.ListExpiringSoon(ctx, withinDays)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listExpiringBatches: %w", err)
	}
	return batches, nil
}
