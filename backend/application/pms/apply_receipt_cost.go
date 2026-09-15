package pms

import (
	"context"
	"fmt"
	"time"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// skuCostRepositoryForReceiptCost is the narrow slice of SKU persistence
// ApplyReceiptCostUseCase needs (ISP, point of use) -- deliberately not
// pmsDomain.SKURepository, so adding UpdateCostPriceFromReceipt doesn't
// require touching every other test double that implements the full
// repository interface. Satisfied structurally by *repositories.SKURepo.
type skuCostRepositoryForReceiptCost interface {
	FindByID(ctx context.Context, skuID string) (*pmsDomain.SKU, error)
	UpdateCostPriceFromReceipt(ctx context.Context, skuID string, costPriceMinorUnits int64) error
}

// ApplyReceiptCostInput is what a goods receipt hands to
// ApplyReceiptCostUseCase once it has a landed unit cost to blend in --
// see application/inventory.ReceiveStockUseCase, which calls this in its
// own transaction immediately after a costed receipt's batch/movement
// commit (never nested inside that transaction -- tenant.WithTenant must
// never nest).
type ApplyReceiptCostInput struct {
	SKUID                    string
	ReceivedQuantity         int64
	LandedUnitCostMinorUnits int64
	// ReferenceType/ReferenceID identify what caused this ledger entry --
	// e.g. "purchase_order_line_item"/<id>, or "batch"/<id> when no PO line
	// item is available. Both optional; an opening-balance-style caller may
	// leave them empty.
	ReferenceType string
	ReferenceID   string
	OccurredAt    time.Time
}

// ApplyReceiptCostUseCase is the Moving Weighted Average Cost blend point:
// the ONLY place a SKU's cost_price_minor_units changes going forward (see
// CLAUDE.md-adjacent design: batches keep their own fixed landed cost from
// receipt; this is the separate, single blended number). Skips entirely
// (no error) when the SKU's cost_price_source is manual, same sticky
// contract as the (now reconciliation-only) daily cron -- a human override
// is never silently overwritten by a receipt.
type ApplyReceiptCostUseCase struct {
	db          *gorm.DB
	skuRepo     skuCostRepositoryForReceiptCost
	ledgerRepo  pmsDomain.SKUCostPriceLedgerRepository
	batchLister batchListerForCostPrice
}

func NewApplyReceiptCostUseCase(
	db *gorm.DB,
	skuRepo skuCostRepositoryForReceiptCost,
	ledgerRepo pmsDomain.SKUCostPriceLedgerRepository,
	batchLister batchListerForCostPrice,
) *ApplyReceiptCostUseCase {
	return &ApplyReceiptCostUseCase{db: db, skuRepo: skuRepo, ledgerRepo: ledgerRepo, batchLister: batchLister}
}

func (uc *ApplyReceiptCostUseCase) Execute(ctx context.Context, in ApplyReceiptCostInput) error {
	if in.ReceivedQuantity <= 0 {
		return fmt.Errorf("applyReceiptCost: received quantity %d is not positive", in.ReceivedQuantity)
	}

	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		sku, err := uc.skuRepo.FindByID(ctx, in.SKUID)
		if err != nil {
			return err
		}
		if sku.CostPriceSource == pmsDomain.SKUCostPriceSourceManual {
			return nil
		}

		batches, err := uc.batchLister.Execute(ctx, in.SKUID)
		if err != nil {
			return err
		}
		// batchLister reflects the receipt having already committed (this
		// use case runs after ReceiveStockUseCase's own transaction), so
		// the pre-receipt existing quantity is today's total minus what
		// just arrived.
		var totalQtyAfter int64
		for _, b := range batches {
			totalQtyAfter += b.Quantity
		}
		existingQty := totalQtyAfter - in.ReceivedQuantity
		if existingQty < 0 {
			existingQty = 0
		}

		newAvg, err := ApplyReceiptToMovingAverageCost(existingQty, sku.CostPriceMinorUnits, in.ReceivedQuantity, in.LandedUnitCostMinorUnits)
		if err != nil {
			return err
		}

		if err := uc.skuRepo.UpdateCostPriceFromReceipt(ctx, in.SKUID, newAvg); err != nil {
			return err
		}

		inventoryValue := totalQtyAfter * newAvg
		entry := &pmsDomain.SKUCostPriceLedgerEntry{
			TenantID: tenant.IDFromContext(ctx), SKUID: in.SKUID,
			SourceType:               pmsDomain.SKUCostPriceLedgerSourceReceipt,
			QuantityDelta:            in.ReceivedQuantity,
			QuantityAfter:            totalQtyAfter,
			CostPriceMinorUnits:      &newAvg,
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
		return uc.ledgerRepo.Create(ctx, entry)
	})
	if err != nil {
		return fmt.Errorf("applyReceiptCost: %w", err)
	}
	return nil
}
