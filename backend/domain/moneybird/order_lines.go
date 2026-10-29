package moneybird

import (
	"fmt"
	"strconv"

	"github.com/adirkuhn/commerceos/domain/errs"
	"github.com/adirkuhn/commerceos/domain/oms"
)

// ErrTotalsDontAddUp: the order's lines, shipping and fee don't sum to what
// the customer was charged -- booking them would put a different amount in
// the books than on the invoice.
var ErrTotalsDontAddUp = errs.NewWithCause("MONEYBIRD_TOTALS_DONT_ADD_UP", "the order's lines don't add up to its total")

// BookedLine is one detail row before Moneybird ids are attached: its VAT
// rate still as a rate, mapped to a Moneybird tax rate by Settings.TaxRateID.
type BookedLine struct {
	Description     string
	Amount          string
	PriceMinorUnits int64
	RateBasisPoints int32
}

// QuoteLine is one quote line at its quoted unit price.
func QuoteLine(description string, quantity, unitPriceMinorUnits int64, rateBasisPoints int32) BookedLine {
	return BookedLine{Description: description, Amount: strconv.FormatInt(quantity, 10), PriceMinorUnits: unitPriceMinorUnits, RateBasisPoints: rateBasisPoints}
}

// OrderLines books an order with prices incl. VAT: one row per line item,
// then shipping and fee. The rows always add up to the order total, or it
// fails with ErrTotalsDontAddUp.
//
// Shipping and fee are stored excl. VAT for Woo orders and as typed (incl.
// VAT) for manual ones; which one applies is read from the order's own
// totals -- whatever the line items don't account for is shipping + fee,
// with or without the rest of the VAT on top.
func OrderLines(order oms.SalesOrder, items []oms.SalesOrderLineItem) ([]BookedLine, error) {
	var out []BookedLine
	var itemsIncl, itemsTax int64
	for _, li := range items {
		incl := li.TotalInclTax()
		itemsIncl += incl
		itemsTax += li.TotalTaxMinorUnits
		out = append(out, itemLine(li, incl))
	}

	extras := extraCharges(order)
	if len(out) == 0 && len(extras) == 0 {
		return nil, ErrNothingToBook
	}
	restIncl := order.TotalMinorUnits - itemsIncl
	restTax := order.TaxTotalMinorUnits - itemsTax
	var extrasSum int64
	for _, e := range extras {
		extrasSum += e.amount
	}

	var inclVAT bool
	switch restIncl {
	case extrasSum:
		inclVAT = true
	case extrasSum + restTax:
		inclVAT = false
	default:
		return nil, fmt.Errorf("%w: lines %s + shipping/fee %s, order total %s",
			ErrTotalsDontAddUp, FormatAmount(itemsIncl), FormatAmount(extrasSum), FormatAmount(order.TotalMinorUnits))
	}

	taxLeft := restTax
	for i, e := range extras {
		tax := taxLeft
		if i < len(extras)-1 && extrasSum > 0 {
			tax = (restTax*e.amount*2 + extrasSum) / (extrasSum * 2)
		}
		taxLeft -= tax
		incl, excl := e.amount, e.amount-tax
		if !inclVAT {
			incl, excl = e.amount+tax, e.amount
		}
		out = append(out, BookedLine{Description: e.description, Amount: "1", PriceMinorUnits: incl, RateBasisPoints: RateBasisPoints(tax, excl)})
	}
	return out, nil
}

// itemLine is quantity x unit price when the total divides into whole
// cents, else one row for the whole total ("3 x name").
func itemLine(li oms.SalesOrderLineItem, incl int64) BookedLine {
	rate := RateBasisPoints(li.TotalTaxMinorUnits, li.TotalExclTax())
	if li.Quantity > 0 && incl%li.Quantity == 0 {
		return BookedLine{Description: li.Name, Amount: strconv.FormatInt(li.Quantity, 10), PriceMinorUnits: incl / li.Quantity, RateBasisPoints: rate}
	}
	return BookedLine{Description: fmt.Sprintf("%d x %s", li.Quantity, li.Name), Amount: "1", PriceMinorUnits: incl, RateBasisPoints: rate}
}

type extraCharge struct {
	description string
	amount      int64
}

func extraCharges(order oms.SalesOrder) []extraCharge {
	var out []extraCharge
	if order.ShippingTotalMinorUnits != 0 {
		d := "Shipping"
		if order.ShippingMethodTitle != "" {
			d += ": " + order.ShippingMethodTitle
		}
		out = append(out, extraCharge{d, order.ShippingTotalMinorUnits})
	}
	if order.FeeTotalMinorUnits != 0 {
		d := order.FeeLabel
		if d == "" {
			d = "Fee"
		}
		out = append(out, extraCharge{d, order.FeeTotalMinorUnits})
	}
	return out
}
