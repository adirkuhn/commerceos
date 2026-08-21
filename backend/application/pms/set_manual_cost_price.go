package pms

import (
	"context"
	"fmt"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// SetManualCostPriceUseCase lets a human override a SKU's cost price
// directly -- for cases the daily recalculation cron has no basis to
// compute from at all (e.g. a purchase order received before landed-cost
// capture existed, or stock that never came through procurement). The cron
// (RecalculateCostPricesUseCase) skips any SKU set this way entirely, so the
// value sticks until ClearCostPriceOverrideUseCase runs.
type SetManualCostPriceUseCase struct {
	db      *gorm.DB
	skuRepo pmsDomain.SKURepository
}

func NewSetManualCostPriceUseCase(db *gorm.DB, skuRepo pmsDomain.SKURepository) *SetManualCostPriceUseCase {
	return &SetManualCostPriceUseCase{db: db, skuRepo: skuRepo}
}

func (uc *SetManualCostPriceUseCase) Execute(ctx context.Context, skuID string, costPriceMinorUnits int64) (*pmsDomain.SKU, error) {
	if costPriceMinorUnits < 0 {
		return nil, fmt.Errorf("setManualCostPrice: %w", pmsDomain.ErrInvalidCostPrice)
	}
	var sku *pmsDomain.SKU
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		sku, err = uc.skuRepo.FindByID(ctx, skuID)
		if err != nil {
			return err
		}
		if err := uc.skuRepo.SetManualCostPrice(ctx, skuID, costPriceMinorUnits); err != nil {
			return err
		}
		sku.CostPriceMinorUnits = &costPriceMinorUnits
		sku.CostPriceSource = pmsDomain.SKUCostPriceSourceManual
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("setManualCostPrice: %w", err)
	}
	return sku, nil
}
