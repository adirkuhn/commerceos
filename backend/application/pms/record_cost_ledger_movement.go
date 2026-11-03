package pms

//events:none cost-ledger audit row mirroring a stock movement, which raises inventory.stock.changed itself; the SKU is not changed.

import (
	"context"
	"fmt"
	"time"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// skuLookupForCostLedgerMovement is the narrow slice of SKU persistence
// RecordCostLedgerMovementUseCase needs (ISP, point of use) -- read-only,
// unlike skuCostRepositoryForReceiptCost (Phase 2's ApplyReceiptCostUseCase),
// since this use case never changes cost_price_minor_units, only logs a
// quantity/value change at whatever cost is already current.
type skuLookupForCostLedgerMovement interface {
	FindByID(ctx context.Context, skuID string) (*pmsDomain.SKU, error)
}

// RecordCostLedgerMovementInput is what a damage/loss write-off or a stock
// adjustment (recount correction, or a customer return credited back via
// inventory.MovementTypeAdjustment -- see that type's doc comment) hands
// to RecordCostLedgerMovementUseCase once its own StockMovement has
// committed.
type RecordCostLedgerMovementInput struct {
	SKUID string
	// QuantityDelta is signed, same convention as inventory.StockMovement
	// -- negative for damage/loss, either sign for an adjustment.
	QuantityDelta int64
	SourceType    pmsDomain.SKUCostPriceLedgerSourceType
	ReferenceType string
	ReferenceID   string
	OccurredAt    time.Time
}

// RecordCostLedgerMovementUseCase logs a quantity/value change that does
// NOT change the SKU's average cost -- damage, loss, or a stock adjustment.
// Writes at whatever cost_price_minor_units the SKU already has (nil stays
// nil, never a fabricated 0 -- same policy as ApplyReceiptCostUseCase);
// never writes to the SKU itself. See application/inventory's
// MarkDamagedOrLostUseCase and RecordStockMovementUseCase (Type Adjustment
// only) for the callers.
type RecordCostLedgerMovementUseCase struct {
	db          *gorm.DB
	skuRepo     skuLookupForCostLedgerMovement
	ledgerRepo  pmsDomain.SKUCostPriceLedgerRepository
	batchLister batchListerForCostPrice
}

func NewRecordCostLedgerMovementUseCase(
	db *gorm.DB,
	skuRepo skuLookupForCostLedgerMovement,
	ledgerRepo pmsDomain.SKUCostPriceLedgerRepository,
	batchLister batchListerForCostPrice,
) *RecordCostLedgerMovementUseCase {
	return &RecordCostLedgerMovementUseCase{db: db, skuRepo: skuRepo, ledgerRepo: ledgerRepo, batchLister: batchLister}
}

func (uc *RecordCostLedgerMovementUseCase) Execute(ctx context.Context, in RecordCostLedgerMovementInput) error {
	if in.QuantityDelta == 0 {
		return fmt.Errorf("recordCostLedgerMovement: quantity delta is zero")
	}

	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		sku, err := uc.skuRepo.FindByID(ctx, in.SKUID)
		if err != nil {
			return err
		}

		batches, err := uc.batchLister.Execute(ctx, in.SKUID)
		if err != nil {
			return err
		}
		var totalQtyAfter int64
		for _, b := range batches {
			totalQtyAfter += b.Quantity
		}

		entry := &pmsDomain.SKUCostPriceLedgerEntry{
			TenantID: tenant.IDFromContext(ctx), SKUID: in.SKUID,
			SourceType:    in.SourceType,
			QuantityDelta: in.QuantityDelta,
			QuantityAfter: totalQtyAfter,
			OccurredAt:    in.OccurredAt,
			CreatedAt:     time.Now(),
		}
		if actual := sku.ActualCostMinorUnits(); actual != nil {
			cost := *actual
			value := totalQtyAfter * cost
			entry.CostPriceMinorUnits = &cost
			entry.InventoryValueMinorUnits = &value
		}
		if in.ReferenceType != "" {
			entry.ReferenceType = &in.ReferenceType
		}
		if in.ReferenceID != "" {
			entry.ReferenceID = &in.ReferenceID
		}
		return uc.ledgerRepo.Create(ctx, entry)
	})
	if err != nil {
		return fmt.Errorf("recordCostLedgerMovement: %w", err)
	}
	return nil
}
