package pms

import (
	"context"
	"time"
)

// SKUCostPriceLedgerSourceType is what kind of event wrote a
// SKUCostPriceLedgerEntry.
type SKUCostPriceLedgerSourceType string

const (
	// SKUCostPriceLedgerSourceOpening is the one-time opening-balance row
	// written by the MWAC migration for a SKU's pre-existing on-hand
	// quantity -- cost known (from a prior manual override) or nil
	// (SKUCostPriceSourceUnknown).
	SKUCostPriceLedgerSourceOpening SKUCostPriceLedgerSourceType = "opening"
	// SKUCostPriceLedgerSourceReceipt is a purchase-order goods receipt --
	// the moving average is recomputed and written here at receipt time.
	SKUCostPriceLedgerSourceReceipt SKUCostPriceLedgerSourceType = "receipt"
	// SKUCostPriceLedgerSourceSale is a completed sale -- does not change
	// the average cost itself, but is logged so the ledger has a complete
	// quantity/value timeline (matching the StockMovement ledger).
	SKUCostPriceLedgerSourceSale SKUCostPriceLedgerSourceType = "sale"
	// SKUCostPriceLedgerSourceReturn is stock restored to sellable
	// inventory at its original cost_at_sale.
	SKUCostPriceLedgerSourceReturn SKUCostPriceLedgerSourceType = "return"
	// SKUCostPriceLedgerSourceDamaged/Lost are inventory write-offs -- they
	// remove quantity/value but never change the average cost of what
	// remains.
	SKUCostPriceLedgerSourceDamaged SKUCostPriceLedgerSourceType = "damaged"
	SKUCostPriceLedgerSourceLost    SKUCostPriceLedgerSourceType = "lost"
	// SKUCostPriceLedgerSourceAdjustment is a stock recount correction --
	// changes quantity, not the average unit cost.
	SKUCostPriceLedgerSourceAdjustment SKUCostPriceLedgerSourceType = "adjustment"
	// SKUCostPriceLedgerSourceCorrection is a manual cost correction (e.g. a
	// supplier invoice fixes a prior PO's price) -- changes the average cost
	// going forward only; every ledger row before it, and every sale's
	// already-frozen cost_at_sale, is left untouched.
	SKUCostPriceLedgerSourceCorrection SKUCostPriceLedgerSourceType = "correction"
	// SKUCostPriceLedgerSourceMerge is a duplicate product merged into
	// this SKU (see ProductMerge): its on-hand stock joins at its own cost,
	// blending the average (BlendMovingAverageCost).
	SKUCostPriceLedgerSourceMerge SKUCostPriceLedgerSourceType = "merge"
)

// SKUCostPriceLedgerEntry is one immutable row in a SKU's cost/quantity
// history -- written on every event that changes (or would change) its
// moving-average cost, never updated or deleted. See migration
// 169_sku_cost_price_ledger.sql's doc comment for the full rationale.
type SKUCostPriceLedgerEntry struct {
	ID       string
	TenantID string
	SKUID    string
	// SourceType is one of the SKUCostPriceLedgerSource* constants above.
	SourceType SKUCostPriceLedgerSourceType
	// ReferenceType/ReferenceID point at the record that caused this entry
	// (e.g. "purchase_order_line_item", "sales_order_line_item") -- both
	// nil for an opening-balance row.
	ReferenceType *string
	ReferenceID   *string
	// QuantityDelta is signed, same convention as inventory.StockMovement --
	// positive for a receipt/return, negative for a sale/damaged/lost, zero
	// for a correction that only changes cost.
	QuantityDelta int64
	// QuantityAfter is this SKU's total on-hand quantity immediately after
	// this event -- the ledger's own running total, independent of (but
	// expected to reconcile against) the StockMovement ledger's sum.
	QuantityAfter int64
	// CostPriceMinorUnits is the blended moving-average cost immediately
	// after this event. Nil exactly when the SKU's CostPriceSource is
	// SKUCostPriceSourceUnknown at this point in time -- never a fabricated
	// 0 (see the type's doc comment).
	CostPriceMinorUnits *int64
	// InventoryValueMinorUnits is QuantityAfter * CostPriceMinorUnits, or
	// nil under the same condition as CostPriceMinorUnits.
	InventoryValueMinorUnits *int64
	OccurredAt               time.Time
	CreatedAt                time.Time
}

// SKUCostPriceLedgerRepository is the append-only per-item cost history
// store -- see SKUCostPriceLedgerEntry.
type SKUCostPriceLedgerRepository interface {
	// Create inserts e as a new, immutable row. There is deliberately no
	// Update/Delete on this interface.
	Create(ctx context.Context, e *SKUCostPriceLedgerEntry) error
	// ListBySKU returns a SKU's ledger entries, most recent first, capped
	// at limit (0 means the repository's own default cap) -- the
	// per-item cost-history view's data source.
	ListBySKU(ctx context.Context, skuID string, limit int) ([]SKUCostPriceLedgerEntry, error)
}
