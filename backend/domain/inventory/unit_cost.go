package inventory

// UnitCostSource says which cost a close-to-expiry row's UnitCostMinorUnits
// is.
type UnitCostSource string

const (
	// UnitCostSourceNone: neither the lot nor the SKU has a cost.
	UnitCostSourceNone UnitCostSource = ""
	// UnitCostSourceBatch: the lot's own landed cost (Batch.LandedCostMinorUnits).
	UnitCostSourceBatch UnitCostSource = "batch"
	// UnitCostSourceSKU: the SKU's actual cost (pms.SKU.ActualCostMinorUnits).
	UnitCostSourceSKU UnitCostSource = "sku"
)

// ResolveUnitCost picks the cost to judge a sale of one lot against: the
// lot's own landed cost -- what this stock actually cost -- else the SKU's
// average cost. Nil when neither is known.
func ResolveUnitCost(lotCost, skuAverageCost *int64) (*int64, UnitCostSource) {
	if lotCost != nil {
		return lotCost, UnitCostSourceBatch
	}
	if skuAverageCost != nil {
		return skuAverageCost, UnitCostSourceSKU
	}
	return nil, UnitCostSourceNone
}
