package inventory

import (
	"context"
	"fmt"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type GetSKUStockLevelUseCase struct {
	db           *gorm.DB
	movementRepo invDomain.StockMovementRepository
	batchRepo    invDomain.BatchRepository
}

func NewGetSKUStockLevelUseCase(db *gorm.DB, movementRepo invDomain.StockMovementRepository, batchRepo invDomain.BatchRepository) *GetSKUStockLevelUseCase {
	return &GetSKUStockLevelUseCase{db: db, movementRepo: movementRepo, batchRepo: batchRepo}
}

func (uc *GetSKUStockLevelUseCase) Execute(ctx context.Context, skuID string) (*StockLevel, error) {
	var level StockLevel
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		movements, err := uc.movementRepo.ListBySKU(ctx, skuID)
		if err != nil {
			return err
		}
		batches, err := uc.batchRepo.ListBySKU(ctx, skuID)
		if err != nil {
			return err
		}
		level = computeStockLevel(skuID, movements, batches)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getSKUStockLevel: %w", err)
	}
	return &level, nil
}
