package pms

// UnitCostFromCaseMinorUnits is what one unit of a sealed case costs: the
// case's cost spread evenly over its units (see SKU.CaseOfSKUID/
// UnitsPerCase), rounded half up to a whole minor unit -- e.g. a 13.99 box
// of 20 makes each unit 0.70. The sub-cent remainder across the case is
// deliberately not carried anywhere. unitsPerCase must be positive (the
// SKU invariant ErrUnitsPerCaseMustBePositive guarantees it).
func UnitCostFromCaseMinorUnits(caseCostMinorUnits, unitsPerCase int64) int64 {
	return (caseCostMinorUnits*2 + unitsPerCase) / (unitsPerCase * 2)
}
