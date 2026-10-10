package procurement

import (
	"github.com/adirkuhn/commerceos/domain/errs"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

// ErrLineItemNotOnPurchaseOrder guards AllocateLandedUnitCost: the target
// line item ID must actually be one of the supplied lineItems.
var ErrLineItemNotOnPurchaseOrder = errs.New("LINE_ITEM_NOT_ON_PURCHASE_ORDER", "line item does not belong to this purchase order's line items")

// AllocateLandedUnitCost computes a product line item's landed per-unit
// cost: its own UnitCostMinorUnits plus a proportional share of the
// purchase order's shipping/fee/other-charge lines, allocated by each
// product line's own subtotal share of the order's total product value
// (standard landed-cost allocation). Pure function -- lineItems must
// already be every PurchaseOrderLineItem on the same PO as targetLineItemID
// (see PurchaseOrderLineItemRepository.ListByPurchaseOrder), the caller's
// responsibility to have loaded.
//
// Discount/credit_note lines are deliberately excluded from the charges
// pool -- they reduce what's owed, not what it cost to land the goods, so
// folding them in would understate cost price. Only shipping/fee/
// other_charge lines inflate the allocation.
func AllocateLandedUnitCost(lineItems []procurementDomain.PurchaseOrderLineItem, targetLineItemID string) (int64, error) {
	var target *procurementDomain.PurchaseOrderLineItem
	var productSubtotalSum, chargesSum int64
	for i := range lineItems {
		li := &lineItems[i]
		subtotal := li.UnitCostMinorUnits * li.Quantity
		switch {
		case li.LineType.IsProduct():
			productSubtotalSum += subtotal
			if li.ID == targetLineItemID {
				target = li
			}
		case li.LineType == procurementDomain.LineItemTypeShipping || li.LineType == procurementDomain.LineItemTypeFee || li.LineType == procurementDomain.LineItemTypeOtherCharge:
			chargesSum += subtotal
		}
	}
	if target == nil {
		return 0, ErrLineItemNotOnPurchaseOrder
	}
	if productSubtotalSum <= 0 || target.Quantity <= 0 || chargesSum == 0 {
		return target.UnitCostMinorUnits, nil
	}

	targetSubtotal := target.UnitCostMinorUnits * target.Quantity
	// Integer math with a rounding half-up on the final division, same
	// smallest-currency-unit convention as every other money field in this
	// codebase (no float).
	allocatedCharge := (targetSubtotal*chargesSum + productSubtotalSum/2) / productSubtotalSum
	perUnitCharge := (allocatedCharge + target.Quantity/2) / target.Quantity
	return target.UnitCostMinorUnits + perUnitCharge, nil
}
