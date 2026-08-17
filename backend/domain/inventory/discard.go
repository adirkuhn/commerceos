package inventory

import "time"

// DiscardRecord is the audit trail for a "this stock was thrown away"
// action -- expired/near-expiry write-off (desktop review queue or the
// scanner's Discard app), with an optional photo of the discarded goods.
// It always references the StockMovement that actually removed the stock
// (Type: MovementTypeLost) -- DiscardRecord is the richer, queryable report
// row; the movement is what the ledger/stock-level math actually reads.
type DiscardRecord struct {
	ID          string
	TenantID    string
	SKUID       string
	WarehouseID string
	// BatchID is nil for a general-ledger (un-lotted) discard.
	BatchID *string
	// MovementID is the StockMovement this discard produced.
	MovementID string
	Quantity   int64
	Reason     string
	// PhotoURL is nil when no photo was attached.
	PhotoURL    *string
	DiscardedBy string
	DiscardedAt time.Time
	CreatedAt   time.Time
}

// DiscardRecordReport is DiscardRecord enriched with SKU/product/warehouse
// display fields, the same join-at-the-repo shape as ExpiringBatch/
// ExpiredBatchReview -- the discarded-items report's data source.
type DiscardRecordReport struct {
	DiscardRecord
	SKUCode       string
	ProductName   string
	WarehouseName string
	LotNumber     string
}

// DiscardRecordFilter narrows ListDiscardsUseCase's report -- all fields
// optional, nil/zero means unfiltered.
type DiscardRecordFilter struct {
	WarehouseID *string
	SKUID       *string
	From        *time.Time
	To          *time.Time
	Limit       int
	Offset      int
}
