package inventory

import (
	"time"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
)

// StockLevel is a SKU's current computed stock, derived from its
// StockMovement ledger — never stored directly.
type StockLevel struct {
	SKUID string
	// Total is the sum of every movement, regardless of batch expiry.
	Total int64
	// Syncable is what should be reported to external channels: the
	// default rule sums un-lotted movements plus movements against
	// batches that aren't expired as of now. This is the one stock rule
	// in place today — a pluggable rules engine (excluding reserved
	// stock, specific warehouses, etc.) is future work, not built yet.
	Syncable int64
}

// computeStockLevel is the one place the Total/Syncable rule above is
// implemented — shared by GetSKUStockLevelUseCase (one SKU) and
// ListStockLevelsUseCase (many SKUs at once) so the business rule can't
// drift between the two.
func computeStockLevel(skuID string, movements []invDomain.StockMovement, batches []invDomain.Batch) StockLevel {
	level := StockLevel{SKUID: skuID}
	expiryByBatch := make(map[string]*time.Time, len(batches))
	for _, b := range batches {
		expiryByBatch[b.ID] = b.ExpiryDate
	}

	now := time.Now()
	for _, m := range movements {
		level.Total += m.QuantityDelta
		if m.BatchID == nil {
			level.Syncable += m.QuantityDelta
			continue
		}
		expiry := expiryByBatch[*m.BatchID]
		if expiry == nil || expiry.After(now) {
			level.Syncable += m.QuantityDelta
		}
	}
	return level
}
