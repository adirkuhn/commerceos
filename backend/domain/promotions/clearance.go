package promotions

import "time"

// ClearanceBatch is this domain's own view of the inventory batch a
// clearance line (PromotionSKU.BatchID set) is selling off -- domain
// packages don't import each other (see CLAUDE.md), so the application
// layer maps inventory's Batch into this shape.
type ClearanceBatch struct {
	// Found is false once the batch no longer exists (deleted).
	Found bool
	// Quantity is the batch's current ledger quantity -- sales, discards
	// and write-offs all reduce it.
	Quantity   int64
	ExpiryDate *time.Time
	// MarkedExpired is true once inventory's auto-expire job has flipped
	// the batch's status to expired.
	MarkedExpired bool
}

// IsFinished reports whether the clearance is over at `at`: the batch was
// removed, has no stock left (sold out, or fully discarded/written off), or
// has expired. Expiry uses the same "expiry date is not after now" rule as
// inventory's own stock-level math, so the promotion ends at the same
// moment the batch stops counting as sellable stock.
func (b ClearanceBatch) IsFinished(at time.Time) bool {
	if !b.Found || b.Quantity <= 0 || b.MarkedExpired {
		return true
	}
	return b.ExpiryDate != nil && !b.ExpiryDate.After(at)
}
