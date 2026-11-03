package pms

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// caseCostReferenceType is the ledger/event reference for a cost taken from
// a case: reference_id is the case SKU's id.
const caseCostReferenceType = "sku"

// skuStoreForCaseCost is the SKU persistence SyncUnitCostFromCaseUseCase
// needs (ISP, point of use) -- satisfied by *repositories.SKURepo.
type skuStoreForCaseCost interface {
	FindByID(ctx context.Context, id string) (*pmsDomain.SKU, error)
	ListByVariant(ctx context.Context, variantID string) ([]pmsDomain.SKU, error)
	UpdateCostPriceFromReceipt(ctx context.Context, skuID string, costPriceMinorUnits int64) error
	UpdateInventoryCost(ctx context.Context, skuID string, inventoryCostMinorUnits int64) error
}

type variantListerForCaseCost interface {
	ListByProduct(ctx context.Context, productID string) ([]pmsDomain.Variant, error)
}

// SyncUnitCostFromCaseUseCase gives a unit SKU the cost of the sealed case
// it is sold out of (SKU.CaseOfSKUID/UnitsPerCase): case cost / units per
// case (pmsDomain.UnitCostFromCaseMinorUnits). Without it a unit that is
// only ever stocked by splitting cases never had a cost at all -- only a
// goods receipt (ApplyReceiptCostUseCase) ever wrote one.
//
// Two costs, two rules (COM-11):
//   - cost price (replacement cost): always the case's cost price per
//     unit, so the unit's margins follow the case's last receipt. A manual
//     cost price is never touched.
//   - inventory cost (what the stock cost): the case's per unit when the
//     unit has none yet, or none of it is on hand (nothing to blend with).
//     While units are on hand it is left alone: splitting more cases
//     blends in at receipt time instead (see
//     inventory.SplitCaseIntoUnitsUseCase). Each change is a cost ledger
//     row (source case_link).
//
// Every change raises its event (pms.sku.cost_price_changed or
// pms.sku.inventory_cost_changed, reason case_link). Re-running with
// nothing new changes nothing.
type SyncUnitCostFromCaseUseCase struct {
	db          *gorm.DB
	skus        skuStoreForCaseCost
	variants    variantListerForCaseCost
	ledgerRepo  pmsDomain.SKUCostPriceLedgerRepository
	batchLister batchListerForCostPrice
	outbox      eventrecord.Enqueuer
}

func NewSyncUnitCostFromCaseUseCase(
	db *gorm.DB,
	skus skuStoreForCaseCost,
	variants variantListerForCaseCost,
	ledgerRepo pmsDomain.SKUCostPriceLedgerRepository,
	batchLister batchListerForCostPrice,
	outbox eventrecord.Enqueuer,
) *SyncUnitCostFromCaseUseCase {
	return &SyncUnitCostFromCaseUseCase{db: db, skus: skus, variants: variants, ledgerRepo: ledgerRepo, batchLister: batchLister, outbox: outbox}
}

// SyncFromCase syncs the unit of one SKU; a SKU that is not a case is a
// no-op, so any SKU id (e.g. from a cost change event) can be passed.
func (uc *SyncUnitCostFromCaseUseCase) SyncFromCase(ctx context.Context, skuID string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		sku, err := uc.skus.FindByID(ctx, skuID)
		if err != nil {
			return err
		}
		return uc.sync(ctx, *sku)
	})
	if err != nil {
		return fmt.Errorf("syncUnitCostFromCase: %w", err)
	}
	return nil
}

// SyncProduct syncs the unit of every case SKU of a product -- run when a
// product changes, which is when a case link is made or changed.
func (uc *SyncUnitCostFromCaseUseCase) SyncProduct(ctx context.Context, productID string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		variants, err := uc.variants.ListByProduct(ctx, productID)
		if err != nil {
			return err
		}
		for _, v := range variants {
			skus, err := uc.skus.ListByVariant(ctx, v.ID)
			if err != nil {
				return err
			}
			for _, s := range skus {
				if err := uc.sync(ctx, s); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("syncUnitCostFromCase: product %s: %w", productID, err)
	}
	return nil
}

func (uc *SyncUnitCostFromCaseUseCase) sync(ctx context.Context, caseSKU pmsDomain.SKU) error {
	if caseSKU.CaseOfSKUID == nil || caseSKU.UnitsPerCase == nil || *caseSKU.UnitsPerCase <= 0 {
		return nil
	}
	unit, err := uc.skus.FindByID(ctx, *caseSKU.CaseOfSKUID)
	if err != nil {
		return err
	}
	if err := uc.syncCostPrice(ctx, caseSKU, *unit); err != nil {
		return err
	}
	return uc.syncInventoryCost(ctx, caseSKU, *unit)
}

// syncCostPrice: the unit's cost price is the case's, per unit.
func (uc *SyncUnitCostFromCaseUseCase) syncCostPrice(ctx context.Context, caseSKU, unit pmsDomain.SKU) error {
	if caseSKU.CostPriceMinorUnits == nil || unit.CostPriceSource == pmsDomain.SKUCostPriceSourceManual {
		return nil
	}
	cost := pmsDomain.UnitCostFromCaseMinorUnits(*caseSKU.CostPriceMinorUnits, *caseSKU.UnitsPerCase)
	if unit.CostPriceMinorUnits != nil && *unit.CostPriceMinorUnits == cost {
		return nil
	}
	if err := uc.skus.UpdateCostPriceFromReceipt(ctx, unit.ID, cost); err != nil {
		return err
	}
	return recordCostPriceChanged(ctx, uc.outbox, unit, &cost, pmsDomain.SKUCostPriceSourceComputed,
		CostPriceChangeReasonCaseLink, caseCostReferenceType, caseSKU.ID)
}

// syncInventoryCost: the unit's inventory cost takes the case's, per unit,
// only when there is nothing on hand to blend with.
func (uc *SyncUnitCostFromCaseUseCase) syncInventoryCost(ctx context.Context, caseSKU, unit pmsDomain.SKU) error {
	caseCost := caseSKU.ActualCostMinorUnits()
	if caseCost == nil {
		return nil
	}
	onHand, err := uc.onHand(ctx, unit.ID)
	if err != nil {
		return err
	}
	if unit.InventoryCostMinorUnits != nil && onHand > 0 {
		return nil
	}
	cost := pmsDomain.UnitCostFromCaseMinorUnits(*caseCost, *caseSKU.UnitsPerCase)
	if unit.InventoryCostMinorUnits != nil && *unit.InventoryCostMinorUnits == cost {
		return nil
	}
	if err := uc.skus.UpdateInventoryCost(ctx, unit.ID, cost); err != nil {
		return err
	}
	now := time.Now()
	value := onHand * cost
	refType, refID := caseCostReferenceType, caseSKU.ID
	if err := uc.ledgerRepo.Create(ctx, &pmsDomain.SKUCostPriceLedgerEntry{
		TenantID: tenant.IDFromContext(ctx), SKUID: unit.ID,
		SourceType:    pmsDomain.SKUCostPriceLedgerSourceCaseLink,
		ReferenceType: &refType, ReferenceID: &refID,
		QuantityAfter:       onHand,
		CostPriceMinorUnits: &cost, InventoryValueMinorUnits: &value,
		OccurredAt: now, CreatedAt: now,
	}); err != nil {
		return err
	}
	return recordInventoryCostChanged(ctx, uc.outbox, unit, cost, CostPriceChangeReasonCaseLink, refType, refID)
}

func (uc *SyncUnitCostFromCaseUseCase) onHand(ctx context.Context, skuID string) (int64, error) {
	batches, err := uc.batchLister.Execute(ctx, skuID)
	if err != nil {
		return 0, err
	}
	var total int64
	for _, b := range batches {
		total += b.Quantity
	}
	return total, nil
}
