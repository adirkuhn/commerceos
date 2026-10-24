package oms

// TotalExclTax is the line's total without VAT, whichever way it was stored
// (see TotalIncludesTax).
func (li SalesOrderLineItem) TotalExclTax() int64 {
	if li.TotalIncludesTax {
		return li.TotalMinorUnits - li.TotalTaxMinorUnits
	}
	return li.TotalMinorUnits
}

// TotalInclTax is the line's total with VAT -- what the customer paid for it.
func (li SalesOrderLineItem) TotalInclTax() int64 {
	return li.TotalExclTax() + li.TotalTaxMinorUnits
}

// RefundAmount is what refunding quantity units of the line gives back: their
// share of the total the customer paid, VAT included.
func (li SalesOrderLineItem) RefundAmount(quantity int64) int64 {
	if li.Quantity <= 0 {
		return 0
	}
	return roundDiv(li.TotalInclTax()*quantity, li.Quantity)
}
