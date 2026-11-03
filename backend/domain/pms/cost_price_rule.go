package pms

// CostPriceAfterReceipt is the cost price rule (COM-11): a SKU's cost price
// is the landed unit cost of its last goods receipt -- the replacement
// cost pricing and margins are judged against. It returns the new cost
// price and whether it changes:
//
//   - a manual cost always wins, a receipt never overwrites it;
//   - a zero-cost receipt (bonus goods, samples) never becomes the last
//     cost -- the previous paid receipt stays;
//   - a receipt against a purchase order sets the cost price;
//   - a costed receipt without a purchase order (no supplier behind it)
//     only fills a cost price that is still unknown.
//
// The actual cost of stock (COGS, stock value) is separate: see
// SKU.InventoryCostMinorUnits.
func CostPriceAfterReceipt(sku SKU, landedUnitCostMinorUnits int64, fromPurchaseOrder bool) (int64, bool) {
	if sku.CostPriceSource == SKUCostPriceSourceManual || landedUnitCostMinorUnits <= 0 {
		return 0, false
	}
	if sku.CostPriceMinorUnits != nil && (!fromPurchaseOrder || *sku.CostPriceMinorUnits == landedUnitCostMinorUnits) {
		return 0, false
	}
	return landedUnitCostMinorUnits, true
}

// ActualCostMinorUnits is what the stock on hand actually cost per unit --
// the moving weighted average (InventoryCostMinorUnits), used for COGS and
// stock value. Falls back to the cost price when no average exists yet
// (e.g. a cost only ever set by hand); nil when neither is known.
func (s SKU) ActualCostMinorUnits() *int64 {
	if s.InventoryCostMinorUnits != nil {
		return s.InventoryCostMinorUnits
	}
	return s.CostPriceMinorUnits
}
