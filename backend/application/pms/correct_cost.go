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

// ApplyValueCorrectionToCost re-blends a SKU's average cost after a value
// correction (e.g. a supplier invoice correcting a prior PO's price) --
// adjustmentValueMinorUnits (signed) is added to the SKU's current total
// inventory value, then the result is divided back across existingQty:
//
//	new_avg = (existingQty*existingAvgCostMinorUnits + adjustmentValueMinorUnits) / existingQty
//
// Rounded half-up to the nearest minor currency unit, same convention as
// ApplyReceiptToMovingAverageCost.
func ApplyValueCorrectionToCost(existingQty int64, existingAvgCostMinorUnits int64, adjustmentValueMinorUnits int64) (int64, error) {
	if existingQty <= 0 {
		return 0, fmt.Errorf("applyValueCorrectionToCost: existing quantity %d is not positive", existingQty)
	}
	newValue := existingQty*existingAvgCostMinorUnits + adjustmentValueMinorUnits
	return (newValue + existingQty/2) / existingQty, nil
}

// CorrectCostInput is a manual cost correction -- e.g. a supplier invoice
// arrives correcting what a prior PO's landed cost should have been.
// AdjustmentValueMinorUnits is signed: how much total inventory value
// changes, not a new unit cost directly (see ApplyValueCorrectionToCost).
type CorrectCostInput struct {
	SKUID                     string
	AdjustmentValueMinorUnits int64
	ReferenceType             string
	ReferenceID               string
	OccurredAt                time.Time
}

// CorrectCostUseCase is the locked design decision from the MWAC costing
// discussion made concrete: a cost correction changes the SKU's current
// inventory cost -- the moving average of the stock on hand -- (and writes
// a "correction" ledger entry) going forward only. The cost price (last
// receipt) is the supplier cost history's concern. It NEVER touches historical data -- no ledger row before
// it is edited, and (deliberately, see this type's test coverage) it has
// no dependency on any sales-order repository at all, so an already-frozen
// cost_at_sale/cogs on a past sale (see application/oms's
// freezeCostAtSale) can never be altered by a correction made today.
type CorrectCostUseCase struct {
	db          *gorm.DB
	skuRepo     skuCostRepositoryForReceiptCost
	ledgerRepo  pmsDomain.SKUCostPriceLedgerRepository
	batchLister batchListerForCostPrice
	outbox      eventrecord.Enqueuer
}

func NewCorrectCostUseCase(
	db *gorm.DB,
	skuRepo skuCostRepositoryForReceiptCost,
	ledgerRepo pmsDomain.SKUCostPriceLedgerRepository,
	batchLister batchListerForCostPrice,
	outbox eventrecord.Enqueuer,
) *CorrectCostUseCase {
	return &CorrectCostUseCase{db: db, skuRepo: skuRepo, ledgerRepo: ledgerRepo, batchLister: batchLister, outbox: outbox}
}

func (uc *CorrectCostUseCase) Execute(ctx context.Context, in CorrectCostInput) (*pmsDomain.SKU, error) {
	var sku *pmsDomain.SKU
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		sku, err = uc.skuRepo.FindByID(ctx, in.SKUID)
		if err != nil {
			return err
		}
		actual := sku.ActualCostMinorUnits()
		if actual == nil {
			return pmsDomain.ErrCostPriceUnknownForCorrection
		}

		batches, err := uc.batchLister.Execute(ctx, in.SKUID)
		if err != nil {
			return err
		}
		var totalQty int64
		for _, b := range batches {
			totalQty += b.Quantity
		}
		if totalQty <= 0 {
			return pmsDomain.ErrNoStockOnHandForCorrection
		}

		newCost, err := ApplyValueCorrectionToCost(totalQty, *actual, in.AdjustmentValueMinorUnits)
		if err != nil {
			return err
		}

		if err := uc.skuRepo.UpdateInventoryCost(ctx, in.SKUID, newCost); err != nil {
			return err
		}
		before := *sku
		sku.InventoryCostMinorUnits = &newCost

		inventoryValue := totalQty * newCost
		entry := &pmsDomain.SKUCostPriceLedgerEntry{
			TenantID: tenant.IDFromContext(ctx), SKUID: in.SKUID,
			SourceType:               pmsDomain.SKUCostPriceLedgerSourceCorrection,
			QuantityDelta:            0,
			QuantityAfter:            totalQty,
			CostPriceMinorUnits:      &newCost,
			InventoryValueMinorUnits: &inventoryValue,
			OccurredAt:               in.OccurredAt,
			CreatedAt:                time.Now(),
		}
		if in.ReferenceType != "" {
			entry.ReferenceType = &in.ReferenceType
		}
		if in.ReferenceID != "" {
			entry.ReferenceID = &in.ReferenceID
		}
		if err := uc.ledgerRepo.Create(ctx, entry); err != nil {
			return err
		}
		return recordInventoryCostChanged(ctx, uc.outbox, before, newCost,
			CostPriceChangeReasonCorrection, in.ReferenceType, in.ReferenceID)
	})
	if err != nil {
		return nil, fmt.Errorf("correctCost: %w", err)
	}
	return sku, nil
}
