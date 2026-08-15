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
	// Syncable is what should be reported to external channels. See
	// computeStockLevel for the two rules that can produce it.
	Syncable int64
}

// computeStockLevel is the one place the Total/Syncable rule is
// implemented — shared by GetSKUStockLevelUseCase (one SKU) and
// ListStockLevelsUseCase (many SKUs at once) so the business rule can't
// drift between the two.
//
// rulesEnabled off (default, tenant_settings.inventory_sync_rules_enabled
// unset/false) keeps the original flat rule: sum of un-lotted movements
// plus movements against batches that aren't expired as of now.
//
// rulesEnabled on switches to a 3-tier, batch-expiry-aware rule (Settings ->
// Inventory Rules):
//   - No batches at all for the SKU: Syncable = Total (the whole ledger).
//   - Batches exist but none carry an ExpiryDate: Syncable = Total (same
//     sum as the flat rule produces, since nothing is ever excluded).
//   - At least one batch carries an ExpiryDate: Syncable = only the
//     quantity of the single not-yet-expired batch whose expiry is
//     soonest — every other batch (expiry or not) and any un-lotted
//     movement is deliberately excluded. This under-reports true stock on
//     purpose, to push sell-through of the oldest lot first (FEFO). If
//     every expiry-bearing batch has already expired, Syncable is 0 —
//     there is no batch left that's safe to promise on a channel.
func computeStockLevel(skuID string, movements []invDomain.StockMovement, batches []invDomain.Batch, rulesEnabled bool) StockLevel {
	level := StockLevel{SKUID: skuID}
	expiryByBatch := make(map[string]*time.Time, len(batches))
	for _, b := range batches {
		expiryByBatch[b.ID] = b.ExpiryDate
	}

	now := time.Now()
	qtyByBatch := make(map[string]int64, len(batches))
	for _, m := range movements {
		level.Total += m.QuantityDelta
		if m.BatchID != nil {
			qtyByBatch[*m.BatchID] += m.QuantityDelta
		}
		if rulesEnabled {
			continue
		}
		if m.BatchID == nil {
			level.Syncable += m.QuantityDelta
			continue
		}
		expiry := expiryByBatch[*m.BatchID]
		if expiry == nil || expiry.After(now) {
			level.Syncable += m.QuantityDelta
		}
	}
	if !rulesEnabled {
		return level
	}

	var nearestBatchID string
	var nearestExpiry *time.Time
	hasExpiryBatch := false
	for _, b := range batches {
		if b.ExpiryDate == nil {
			continue
		}
		hasExpiryBatch = true
		if !b.ExpiryDate.After(now) {
			continue // already expired -- not eligible to be "nearest"
		}
		if nearestExpiry == nil || b.ExpiryDate.Before(*nearestExpiry) {
			nearestExpiry = b.ExpiryDate
			nearestBatchID = b.ID
		}
	}

	if !hasExpiryBatch {
		level.Syncable = level.Total // covers both "no batches" and "batches, no expiry"
		return level
	}
	level.Syncable = qtyByBatch[nearestBatchID] // 0 if nearestBatchID is "" (all expiry batches already expired)
	return level
}
