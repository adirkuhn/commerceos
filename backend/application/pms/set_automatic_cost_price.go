package pms

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// SetAutomaticCostPriceUseCase sets a SKU's cost without pinning it: unlike
// SetManualCostPriceUseCase the source becomes computed, so the next goods
// receipt blends into this value (ApplyReceiptCostUseCase) and moves it.
// For a SKU whose cost is unknown or stale and should still follow the
// purchase prices from now on.
type SetAutomaticCostPriceUseCase struct {
	db      *gorm.DB
	skuRepo skuCostRepositoryForReceiptCost
	outbox  eventrecord.Enqueuer
}

func NewSetAutomaticCostPriceUseCase(db *gorm.DB, skuRepo skuCostRepositoryForReceiptCost, outbox eventrecord.Enqueuer) *SetAutomaticCostPriceUseCase {
	return &SetAutomaticCostPriceUseCase{db: db, skuRepo: skuRepo, outbox: outbox}
}

func (uc *SetAutomaticCostPriceUseCase) Execute(ctx context.Context, skuID string, costPriceMinorUnits int64) (*pmsDomain.SKU, error) {
	if costPriceMinorUnits < 0 {
		return nil, fmt.Errorf("setAutomaticCostPrice: %w", pmsDomain.ErrInvalidCostPrice)
	}
	var sku *pmsDomain.SKU
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		sku, err = uc.skuRepo.FindByID(ctx, skuID)
		if err != nil {
			return err
		}
		// Same write as a receipt: cost plus source computed in one update.
		if err := uc.skuRepo.UpdateCostPriceFromReceipt(ctx, skuID, costPriceMinorUnits); err != nil {
			return err
		}
		before := *sku
		sku.CostPriceMinorUnits = &costPriceMinorUnits
		sku.CostPriceSource = pmsDomain.SKUCostPriceSourceComputed
		return recordCostPriceChanged(ctx, uc.outbox, before, sku.CostPriceMinorUnits, sku.CostPriceSource,
			CostPriceChangeReasonSet, "", "")
	})
	if err != nil {
		return nil, fmt.Errorf("setAutomaticCostPrice: %w", err)
	}
	return sku, nil
}
