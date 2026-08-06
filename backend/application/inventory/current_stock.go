package inventory

import invDomain "github.com/adirkuhn/commerceos/domain/inventory"

// currentStock sums the ledger scoped to one warehouse and one batch (nil
// batchID = general/un-lotted stock) -- the same "sum the movements" rule
// GetSKUStockLevelUseCase uses SKU-wide, narrowed to a single scope so a
// removal can be checked against what that specific batch/general ledger
// actually has, not the SKU's total across every batch and warehouse.
func currentStock(movements []invDomain.StockMovement, warehouseID string, batchID *string) int64 {
	var total int64
	for _, m := range movements {
		if m.WarehouseID != warehouseID || !sameBatchScope(m.BatchID, batchID) {
			continue
		}
		total += m.QuantityDelta
	}
	return total
}

func sameBatchScope(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// requireSufficientStock is a no-op for a non-negative delta (nothing is
// being removed). For a negative delta it rejects the movement unless the
// scoped ledger already has at least that much.
func requireSufficientStock(movements []invDomain.StockMovement, warehouseID string, batchID *string, delta int64) error {
	if delta >= 0 {
		return nil
	}
	if currentStock(movements, warehouseID, batchID)+delta < 0 {
		return invDomain.ErrInsufficientStock
	}
	return nil
}
