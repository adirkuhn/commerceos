package repositories

import (
	"context"
	"fmt"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

// UpdateCostPrice writes only cost_price_minor_units -- deliberately
// separate from Update (which the ordinary product-edit save path calls
// and never touches this field), so the cron is the sole writer and can't
// be clobbered by a stale form round-trip.
func (r *SKURepo) UpdateCostPrice(ctx context.Context, skuID string, costPriceMinorUnits int64) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&skuModel{}).Where("id = ?", skuID).
		Update("cost_price_minor_units", costPriceMinorUnits).Error; err != nil {
		return fmt.Errorf("skuRepo.UpdateCostPrice: %w", err)
	}
	return nil
}

// UpdateCostPriceFromReceipt writes cost_price_minor_units and resolves
// cost_price_source from unknown to computed in one update -- the MWAC
// receipt-time hook's write (see application/pms.ApplyReceiptCostUseCase).
// Deliberately not part of pmsDomain.SKURepository: ApplyReceiptCostUseCase
// depends on a narrow local interface (point-of-use ISP) satisfied
// structurally by this type, so adding this method doesn't ripple out to
// every other test double that implements the full SKURepository interface.
// Never called for a SKU whose source is already manual -- the caller
// checks that first (same sticky-override contract as the cron).
func (r *SKURepo) UpdateCostPriceFromReceipt(ctx context.Context, skuID string, costPriceMinorUnits int64) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&skuModel{}).Where("id = ?", skuID).
		Updates(map[string]any{
			"cost_price_minor_units": costPriceMinorUnits,
			"cost_price_source":      string(pmsDomain.SKUCostPriceSourceComputed),
		}).Error; err != nil {
		return fmt.Errorf("skuRepo.UpdateCostPriceFromReceipt: %w", err)
	}
	return nil
}

// UpdateInventoryCost writes only inventory_cost_minor_units -- the moving
// weighted average of the stock on hand (see pmsDomain.SKU). Never touches
// the cost price or its source, so a manual cost price stays in place.
func (r *SKURepo) UpdateInventoryCost(ctx context.Context, skuID string, inventoryCostMinorUnits int64) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&skuModel{}).Where("id = ?", skuID).
		Update("inventory_cost_minor_units", inventoryCostMinorUnits).Error; err != nil {
		return fmt.Errorf("skuRepo.UpdateInventoryCost: %w", err)
	}
	return nil
}

// SetManualCostPrice writes cost_price_minor_units and flips
// cost_price_source to manual in one update -- see
// pmsDomain.SKURepository.SetManualCostPrice.
func (r *SKURepo) SetManualCostPrice(ctx context.Context, skuID string, costPriceMinorUnits int64) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&skuModel{}).Where("id = ?", skuID).
		Updates(map[string]any{
			"cost_price_minor_units": costPriceMinorUnits,
			"cost_price_source":      string(pmsDomain.SKUCostPriceSourceManual),
		}).Error; err != nil {
		return fmt.Errorf("skuRepo.SetManualCostPrice: %w", err)
	}
	return nil
}

// ClearCostPriceOverride resets cost_price_source back to computed --
// cost_price_minor_units is left as-is until the next cron run; see
// pmsDomain.SKURepository.ClearCostPriceOverride.
func (r *SKURepo) ClearCostPriceOverride(ctx context.Context, skuID string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&skuModel{}).Where("id = ?", skuID).
		Update("cost_price_source", string(pmsDomain.SKUCostPriceSourceComputed)).Error; err != nil {
		return fmt.Errorf("skuRepo.ClearCostPriceOverride: %w", err)
	}
	return nil
}
