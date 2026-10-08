package channels

import (
	"context"
	"log"
	"strconv"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
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

func wooOrderAddsUp(wo integrationsDomain.WooOrder) bool {
	return !omsDomain.ChannelTotalsDiffer(wooOrderTotals(wo))
}

// completeWooOrder re-reads an order whose parts don't add up to its total
// once, in case it was listed while Woo was still saving it. If the re-read
// fails or doesn't add up either, the listed order is kept -- it is still
// imported 1:1 and the gap is reported (see recordWooOrderDifferences).
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

func wooChannelLines(wo integrationsDomain.WooOrder) []omsDomain.ChannelLine {
	out := make([]omsDomain.ChannelLine, 0, len(wo.LineItems))
	for _, li := range wo.LineItems {
		out = append(out, omsDomain.ChannelLine{SKUCode: li.SKU, Name: li.Name, Quantity: li.Quantity})
	}
	return out
}
