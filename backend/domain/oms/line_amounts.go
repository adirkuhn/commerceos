package oms

// LineAmounts is one order line's money, in minor units. Subtotal equals
// Total: a line carries no discount of its own.
type LineAmounts struct {
	Subtotal    int64
	SubtotalTax int64
	Total       int64
	TotalTax    int64
}

// PriceLine prices quantity x unitPrice at a VAT rate (basis points). With
// pricesIncludeTax the VAT is the part of the amount above amount/(1+rate),
// otherwise it comes on top -- the same split the admin's order form makes.
func PriceLine(quantity, unitPriceMinorUnits, taxRateBasisPoints int64, pricesIncludeTax bool) LineAmounts {
	amount := quantity * unitPriceMinorUnits
	var tax int64
	switch {
	case taxRateBasisPoints <= 0:
	case pricesIncludeTax:
		tax = amount - roundDiv(amount*10000, 10000+taxRateBasisPoints)
	default:
		tax = roundDiv(amount*taxRateBasisPoints, 10000)
	}
	return LineAmounts{Subtotal: amount, SubtotalTax: tax, Total: amount, TotalTax: tax}
}

// OrderTotals sums lines into the order's total and VAT: inclusive prices
// already contain their VAT, exclusive ones get it added.
func OrderTotals(lines []LineAmounts, pricesIncludeTax bool) (total, tax int64) {
	for _, l := range lines {
		tax += l.TotalTax
		total += l.Total
		if !pricesIncludeTax {
			total += l.TotalTax
		}
	}
	return total, tax
}

// roundDiv is a/b rounded half up, for non-negative a and positive b.
func roundDiv(a, b int64) int64 {
	return (a*2 + b) / (b * 2)
}
