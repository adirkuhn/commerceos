package pms

import "fmt"

// ApplyReceiptToMovingAverageCost computes a SKU's new blended Moving
// Weighted Average Cost after a goods receipt:
//
//	new_avg = (existing_qty*existing_avg + received_qty*received_unit_cost)
//	          / (existing_qty + received_qty)
//
// existingAvgCostMinorUnits nil means the SKU currently has
// pms.SKUCostPriceSourceUnknown -- there is no reliable cost to blend for
// the existing quantity. Rather than treating unknown as 0 (which would
// silently understate the result), the existing quantity merges in at the
// received unit cost, so the blended result becomes exactly that cost. This
// is the "resolved automatically at first real receipt" policy documented
// on SKUCostPriceSourceUnknown.
//
// Result is rounded half-up to the nearest minor currency unit, same
// convention as every other money calculation in this codebase (no float).
func ApplyReceiptToMovingAverageCost(existingQty int64, existingAvgCostMinorUnits *int64, receivedQty int64, receivedUnitCostMinorUnits int64) (int64, error) {
	if existingQty < 0 {
		return 0, fmt.Errorf("applyReceiptToMovingAverageCost: existing quantity %d is negative", existingQty)
	}
	if receivedQty <= 0 {
		return 0, fmt.Errorf("applyReceiptToMovingAverageCost: received quantity %d is not positive", receivedQty)
	}

	existingAvg := receivedUnitCostMinorUnits
	if existingAvgCostMinorUnits != nil {
		existingAvg = *existingAvgCostMinorUnits
	}

	totalQty := existingQty + receivedQty
	totalValue := existingQty*existingAvg + receivedQty*receivedUnitCostMinorUnits
	return (totalValue + totalQty/2) / totalQty, nil
}
