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
// The unit takes the case's cost when it has no cost yet, or when none of
// it is on hand (nothing to blend with, so the latest case cost wins).
// While units are on hand their cost is left alone: splitting more cases
// blends in at receipt time instead (see inventory.SplitCaseIntoUnitsUseCase).
// A manual cost is never touched. Every change is a cost ledger row
// (source case_link) and a pms.sku.cost_price_changed event, so the full
// history is kept. Re-running with nothing new changes nothing.
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
	if caseSKU.CaseOfSKUID == nil || caseSKU.UnitsPerCase == nil || *caseSKU.UnitsPerCase <= 0 || caseSKU.CostPriceMinorUnits == nil {
		return nil
	}
	unit, err := uc.skus.FindByID(ctx, *caseSKU.CaseOfSKUID)
	if err != nil {
		return err
	}
	if unit.CostPriceSource == pmsDomain.SKUCostPriceSourceManual {
		return nil
	}
	onHand, err := uc.onHand(ctx, unit.ID)
	if err != nil {
		return err
	}
	if unit.CostPriceMinorUnits != nil && onHand > 0 {
		return nil
	}
	cost := pmsDomain.UnitCostFromCaseMinorUnits(*caseSKU.CostPriceMinorUnits, *caseSKU.UnitsPerCase)
	if unit.CostPriceMinorUnits != nil && *unit.CostPriceMinorUnits == cost {
		return nil
	}

	if err := uc.skus.UpdateCostPriceFromReceipt(ctx, unit.ID, cost); err != nil {
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
	return recordCostPriceChanged(ctx, uc.outbox, *unit, &cost, pmsDomain.SKUCostPriceSourceComputed,
		CostPriceChangeReasonCaseLink, refType, refID)
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
