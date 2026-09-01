package shipping

import "errors"

// ErrDeliveryEstimateUnsupportedCountry is returned when CC isn't NL or BE
// -- MyParcel's own GET /delivery_options endpoint (the only pre-shipment
// price MyParcel's API exposes at all) only accepts those two country
// codes; every other destination has no live pre-creation price quote
// available from MyParcel, only the actual price returned after a shipment
// is created (see infrastructure/myparcel's package doc comment).
var ErrDeliveryEstimateUnsupportedCountry = errors.New("shipping: delivery price estimate is only available for NL/BE addresses")

// DeliveryPriceOption is one priced delivery slot MyParcel offers for a
// given address/carrier (e.g. standard, morning, evening) -- callers
// typically show the cheapest of these as "the" estimate, but every slot
// found is returned since price_comment differentiates them meaningfully.
type DeliveryPriceOption struct {
	PriceComment     string
	AmountMinorUnits int
	Currency         string
}
