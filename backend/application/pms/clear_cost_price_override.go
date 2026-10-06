package pms

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// ClearCostPriceOverrideUseCase reverts a SKU set by
// SetManualCostPriceUseCase back to automatic -- the value on the SKU is
// left as-is (still the last manual number) until the next run of
// RecalculateCostPricesUseCase recomputes it from actual receiving history.
type ClearCostPriceOverrideUseCase struct {
	db      *gorm.DB
	skuRepo pmsDomain.SKURepository
	outbox  eventrecord.Enqueuer
}

func NewClearCostPriceOverrideUseCase(db *gorm.DB, skuRepo pmsDomain.SKURepository, outbox eventrecord.Enqueuer) *ClearCostPriceOverrideUseCase {
	return &ClearCostPriceOverrideUseCase{db: db, skuRepo: skuRepo, outbox: outbox}
}

func (uc *ClearCostPriceOverrideUseCase) Execute(ctx context.Context, skuID string) (*pmsDomain.SKU, error) {
	var sku *pmsDomain.SKU
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		sku, err = uc.skuRepo.FindByID(ctx, skuID)
		if err != nil {
			return err
		}
		if err := uc.skuRepo.ClearCostPriceOverride(ctx, skuID); err != nil {
			return err
		}
		before := *sku
		sku.CostPriceSource = pmsDomain.SKUCostPriceSourceComputed
		return recordCostPriceChanged(ctx, uc.outbox, before, sku.CostPriceMinorUnits, sku.CostPriceSource,
			CostPriceChangeReasonOverrideCleared, "", "")
	})
	if err != nil {
		return nil, fmt.Errorf("clearCostPriceOverride: %w", err)
	}
	return sku, nil
}
