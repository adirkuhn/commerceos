package oms

import "context"

// AppliedChangeLister lists the order changes one fulfilment step (picking
// or packing) has applied to a sales order -- ReplayedAt set -- oldest
// first. Powers the order page's "what was replaced, added or refunded".
type AppliedChangeLister interface {
	ListAppliedBySalesOrder(ctx context.Context, salesOrderID string) ([]StagedChange, error)
}

// IsExtraItem reports whether the change added a product the order never
// had (free or charged) -- it became a line of its own.
func (c StagedChange) IsExtraItem() bool {
	return c.Action == ChangeFreebie || c.Action == ChangeNewItem
}

// MatchAddedLines finds the line each applied extra item became, keyed by
// change ID. The applier creates that line with the change's SKU, quantity
// and Reason as its name, and nothing links the two, so they are matched on
// those -- each line used once, oldest change first. An extra staff chose
// to ship without (no stock) never became a line and stays unmatched.
func MatchAddedLines(changes []StagedChange, lines []SalesOrderLineItem) map[string]string {
	matched := make(map[string]string)
	used := make(map[string]bool)
	for _, c := range changes {
		if !c.IsExtraItem() {
			continue
		}
		for _, li := range lines {
			if used[li.ID] || li.SKUID == nil || *li.SKUID != c.SKUID || li.Quantity != c.Quantity || li.Name != c.Reason {
				continue
			}
			used[li.ID] = true
			matched[c.ID] = li.ID
			break
		}
	}
	return matched
}
