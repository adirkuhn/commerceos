package pms

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// skuRepoForClearOverride is the SKU persistence clearing an override
// needs (ISP).
type skuRepoForClearOverride interface {
	FindByID(ctx context.Context, skuID string) (*pmsDomain.SKU, error)
	ClearCostPriceOverride(ctx context.Context, skuID string) error
	UpdateCostPriceFromReceipt(ctx context.Context, skuID string, costPriceMinorUnits int64) error
}

// ClearCostPriceOverrideUseCase reverts a SKU set by
// SetManualCostPriceUseCase back to automatic: the cost price goes back to
// the last receipt still standing in the supplier cost history (COM-11).
// With no paid receipt on record the last manual number stays until the
// next receipt replaces it.
type ClearCostPriceOverrideUseCase struct {
	db      *gorm.DB
	skuRepo skuRepoForClearOverride
	last    lastReceiptCostLookup
	outbox  eventrecord.Enqueuer
}

func NewClearCostPriceOverrideUseCase(db *gorm.DB, skuRepo skuRepoForClearOverride, last lastReceiptCostLookup, outbox eventrecord.Enqueuer) *ClearCostPriceOverrideUseCase {
	return &ClearCostPriceOverrideUseCase{db: db, skuRepo: skuRepo, last: last, outbox: outbox}
}

func (uc *ClearCostPriceOverrideUseCase) Execute(ctx context.Context, skuID string) (*pmsDomain.SKU, error) {
	// read before our transaction: the lookup opens its own
	last, err := uc.last.LastReceiptCost(ctx, skuID)
	if err != nil {
		return nil, fmt.Errorf("clearCostPriceOverride: %w", err)
	}
	var sku *pmsDomain.SKU
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
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
		if last != nil && (sku.CostPriceMinorUnits == nil || *sku.CostPriceMinorUnits != last.LandedUnitCostMinorUnits) {
			cost := last.LandedUnitCostMinorUnits
			if err := uc.skuRepo.UpdateCostPriceFromReceipt(ctx, skuID, cost); err != nil {
				return err
			}
			sku.CostPriceMinorUnits = &cost
		}
		return recordCostPriceChanged(ctx, uc.outbox, before, sku.CostPriceMinorUnits, sku.CostPriceSource,
			CostPriceChangeReasonOverrideCleared, "", "")
	})
	if err != nil {
		return nil, fmt.Errorf("clearCostPriceOverride: %w", err)
	}
	return sku, nil
}
