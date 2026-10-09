package channels

import (
	"context"
	"log"
	"strconv"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

// wooOrderTotals returns a Woo order's own total and what its parts add up
// to: every line (after discount, plus its tax), shipping and its tax, and
// every fee and its tax. Woo always saves these consistently, so a mismatch
// means the order was read while Woo was still saving it.
func wooOrderTotals(wo integrationsDomain.WooOrder) (total, partsTotal int64) {
	total, _ = ParseWooPriceToMinorUnits(wo.Total)
	add := func(amounts ...string) {
		for _, a := range amounts {
			v, _ := ParseWooPriceToMinorUnits(a)
			partsTotal += v
		}
	}
	for _, li := range wo.LineItems {
		if li.Quantity > 0 {
			add(li.Total, li.TotalTax)
		}
	}
	add(wo.ShippingTotal, wo.ShippingTax)
	for _, f := range wo.FeeLines {
		add(f.Total, f.TotalTax)
	}
	return total, partsTotal
}

// wooTotalToleranceMinorUnits absorbs per-line tax rounding: Woo's parts
// can add up to a cent or two off its own total.
const wooTotalToleranceMinorUnits = 2

func wooOrderAddsUp(wo integrationsDomain.WooOrder) bool {
	total, partsTotal := wooOrderTotals(wo)
	diff := total - partsTotal
	return diff >= -wooTotalToleranceMinorUnits && diff <= wooTotalToleranceMinorUnits
}

// completeWooOrder re-reads an order whose parts don't add up to its total
// once, in case it was listed while Woo was still saving it (its total
// already saved, not all of its lines yet). If the re-read fails or doesn't
// add up either, the listed order is kept: an order is always imported
// exactly as Woo returns it.
func completeWooOrder(ctx context.Context, client integrationsDomain.WooClient, wo integrationsDomain.WooOrder) integrationsDomain.WooOrder {
	if wooOrderAddsUp(wo) {
		return wo
	}
	fresh, err := client.GetOrder(ctx, strconv.Itoa(wo.ID))
	if err != nil {
		log.Printf("importWooOrders: woo order %d doesn't add up; re-read failed: %v", wo.ID, err)
		return wo
	}
	return *fresh
}
