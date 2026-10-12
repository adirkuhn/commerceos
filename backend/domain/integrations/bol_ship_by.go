package integrations

import "time"

// ShipByDate is the calendar date this order must be handed to the carrier
// to still reach the customer on time, or nil when Bol gave no delivery
// date for any still-open item.
//
// Bol's Retailer API v10 (OrderOrderItem.fulfilment) carries two delivery
// dates and never both: latestDeliveryDate ("The ultimate delivery date at
// which this order must be delivered at the customer's shipping address")
// and exactDeliveryDate (filled only when the customer chose a delivery
// day). Both are DELIVERY dates, not ship dates -- Bol states no handover
// date. The rule here:
//   - delivery date = exactDeliveryDate, else latestDeliveryDate;
//   - ship by = the day before it, since FBR parcels go through standard
//     NL/BE next-day carrier networks (PostNL, DHL, ...);
//   - a ship-by that lands on a Sunday moves back to Saturday, because
//     those networks don't collect on Sundays;
//   - across items, only ones with quantity still open (not fully
//     shipped/cancelled) count, and the earliest date wins -- the order is
//     late as soon as any one of its items is.
//
// Bol's expiryDate (the day an item is auto-cancelled) is deliberately not
// used: it's a cancellation date, not a delivery promise.
func (o BolOrder) ShipByDate() *time.Time {
	var earliest *time.Time
	for _, it := range o.OrderItems {
		if it.QuantityShipped+it.QuantityCancelled >= it.Quantity {
			continue
		}
		delivery := it.ExactDeliveryDate
		if delivery == nil {
			delivery = it.LatestDeliveryDate
		}
		if delivery == nil {
			continue
		}
		shipBy := time.Date(delivery.Year(), delivery.Month(), delivery.Day()-1, 0, 0, 0, 0, time.UTC)
		if shipBy.Weekday() == time.Sunday {
			shipBy = shipBy.AddDate(0, 0, -1)
		}
		if earliest == nil || shipBy.Before(*earliest) {
			earliest = &shipBy
		}
	}
	return earliest
}
