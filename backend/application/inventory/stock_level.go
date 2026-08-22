package inventory

import (
	"sort"
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
	// SourceBatch is set only when Syncable was attributed to one specific
	// batch (the tier-3 nearest-expiry rule below) -- which lot/expiry the
	// synced quantity refers to, for channel pushes that surface it as
	// product metadata (see application/channels' toWooMetaData). Nil for
	// every other case: rules disabled, no batches, or batches with no
	// expiry (Syncable is a sum across batches there, not attributable to
	// one lot).
	SourceBatch *BatchSyncInfo
}

// BatchSyncInfo is the lot identity behind a StockLevel.SourceBatch.
type BatchSyncInfo struct {
	LotNumber  string
	ExpiryDate time.Time
}

// computeStockLevel is the one place the Total/Syncable rule is
// implemented — shared by GetSKUStockLevelUseCase (one SKU) and
// ListStockLevelsUseCase (many SKUs at once) so the business rule can't
// drift between the two.
//
// rulesEnabled off (default, tenant_settings.inventory_sync_rules_enabled
// unset/false) keeps the original flat rule: sum of un-lotted movements
// plus movements against batches that aren't expired as of now -- an
// already-expired batch never counts toward Syncable, rules on or off.
//
// rulesEnabled on switches to a 3-tier, batch-expiry-aware rule (Settings ->
// Inventory Rules):
//   - No batches at all for the SKU: Syncable = Total (the whole ledger).
//   - Batches exist but none carry an ExpiryDate: Syncable = Total (same
//     sum as the flat rule produces, since nothing is ever excluded).
//   - At least one batch carries an ExpiryDate: Syncable = only the
//     quantity of the not-yet-expired batch whose expiry is soonest AMONG
//     THOSE STILL HOLDING STOCK — a depleted lot is skipped in favor of the
//     next-nearest-expiring one that still has quantity, same FEFO
//     reasoning (an empty lot has nothing left to sell through first).
//     Every other batch (expiry or not) and any un-lotted movement is
//     still deliberately excluded once a usable lot is found. This
//     under-reports true stock on purpose, to push sell-through of the
//     oldest lot first. If every expiry-bearing batch has already
//     expired, Syncable is 0 — there is no batch left that's safe to
//     promise on a channel. If every not-yet-expired batch is merely
//     depleted (not expired, just empty), there's no lot left to enforce
//     FEFO on at all, so Syncable falls back to Total instead of a false
//     zero -- unlike the expired case, an empty-but-valid lot isn't an
//     "unsafe to sell" signal, it's just an accounting dead end.
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

	var notExpired []*invDomain.Batch
	hasExpiryBatch := false
	for i, b := range batches {
		if b.ExpiryDate == nil {
			continue
		}
		hasExpiryBatch = true
		if !b.ExpiryDate.After(now) {
			continue // already expired -- not eligible at all
		}
		notExpired = append(notExpired, &batches[i])
	}

	if !hasExpiryBatch {
		level.Syncable = level.Total // covers both "no batches" and "batches, no expiry"
		return level
	}
	if len(notExpired) == 0 {
		return level // all expiry batches already expired -- Syncable stays 0
	}

	sort.Slice(notExpired, func(i, j int) bool { return notExpired[i].ExpiryDate.Before(*notExpired[j].ExpiryDate) })
	for _, b := range notExpired {
		qty := qtyByBatch[b.ID]
		if qty <= 0 {
			continue // depleted lot -- move to the next-nearest-expiring one
		}
		level.Syncable = qty
		level.SourceBatch = &BatchSyncInfo{LotNumber: b.LotNumber, ExpiryDate: *b.ExpiryDate}
		return level
	}
	// Every not-yet-expired batch is depleted -- no lot left to enforce FEFO
	// on, so report the real ledger total instead of a false zero.
	level.Syncable = level.Total
	return level
}
