package inventory

import (
	"context"
	"fmt"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type ListBatchesUseCase struct {
	db        *gorm.DB
	batchRepo invDomain.BatchRepository
}

func NewListBatchesUseCase(db *gorm.DB, batchRepo invDomain.BatchRepository) *ListBatchesUseCase {
	return &ListBatchesUseCase{db: db, batchRepo: batchRepo}
}

func (uc *ListBatchesUseCase) Execute(ctx context.Context, skuID string) ([]invDomain.Batch, error) {
	var batches []invDomain.Batch
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		batches, err = uc.batchRepo.ListBySKU(ctx, skuID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listBatches: %w", err)
	}
	return batches, nil
}
