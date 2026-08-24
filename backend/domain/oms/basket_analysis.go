package oms

import (
	"context"
	"time"
)

// OrderSKUs is one qualifying order's distinct SKU IDs -- one "transaction"
// for the Phase 1 "Frequently Bought Together" report's FP-Growth mining to
// consume. Adhoc/unmatched line items (no linked SKU) are excluded --
// there's no stable item identity to mine co-occurrence over.
type OrderSKUs struct {
	OrderID string
	SKUIDs  []string
}

// BasketAnalysisRepository backs the Phase 1 "Frequently Bought Together"
// report -- same status/currency filtering as SalesReportRepository.
type BasketAnalysisRepository interface {
	// ListOrderSKUsBetween returns one row per qualifying order within
	// [from, to), each carrying that order's distinct SKU IDs. An order
	// with fewer than 2 distinct SKUs is still included (a size-1
	// transaction) -- FP-Growth itself naturally never produces an
	// itemset larger than what a transaction actually contains.
	ListOrderSKUsBetween(ctx context.Context, from, to time.Time, currencyCode string) ([]OrderSKUs, error)
}
