package inventory

import "time"

type MovementType string

const (
	MovementTypeReceipt     MovementType = "receipt"      // stock coming in (purchase, bulk import)
	MovementTypeSale        MovementType = "sale"         // stock going out via an order
	MovementTypeAdjustment  MovementType = "adjustment"   // recount correction
	MovementTypeTransferOut MovementType = "transfer_out" // leaving a warehouse/batch
	MovementTypeTransferIn  MovementType = "transfer_in"  // arriving at a warehouse/batch
	MovementTypeDamaged     MovementType = "damaged"      // written off — damaged goods
	MovementTypeLost        MovementType = "lost"         // written off — lost/missing goods
)

// StockMovement is one immutable ledger entry. Current stock for a SKU (or
// a single batch) is never stored directly — it's always the sum of its
// movements. QuantityDelta is signed: positive for stock coming in
// (receipt, transfer_in), negative for stock going out (sale, transfer_out,
// or a negative adjustment).
type StockMovement struct {
	ID            string
	TenantID      string
	SKUID         string
	WarehouseID   string
	BatchID       *string // nil = un-lotted, general stock
	Type          MovementType
	QuantityDelta int64
	Reason        string
	OccurredAt    time.Time
	CreatedAt     time.Time
}
