package myparcel

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

// deliveryOptionsAccept is GET /delivery_options' required Accept header --
// a versioned media type distinct from shipmentAccept/orderAccept, per
// developer.myparcel.nl/api-reference/08.delivery-options.html.
const deliveryOptionsAccept = "application/json;charset=utf-8;version=2.0"

// DeliveryOptionsInput drives GetDeliveryOptions -- see that method's doc
// comment for the CC "NL"/"BE"-only limitation.
type DeliveryOptionsInput struct {
	CC         string
	PostalCode string
	Number     string
	Carrier    int
}

// DeliveryPriceOption is one priced delivery slot from GET /delivery_options
// -- see GetDeliveryOptions' doc comment.
type DeliveryPriceOption struct {
	PriceComment     string
	AmountMinorUnits int
	Currency         string
}

// deliveryOptionsEnvelope is GET /delivery_options' response envelope --
// data.delivery[].time[]{price:{currency,amount},price_comment}, confirmed
// against 08.delivery-options.html's own example response. data.pickup[]
// (parcel locker options) exists in the same envelope but is never modeled
// here -- GetDeliveryOptions only surfaces home-delivery prices, matching
// what CreateShipment actually books.
type deliveryOptionsEnvelope struct {
	Data struct {
		Delivery []struct {
			Time []struct {
				Price struct {
					Currency string `json:"currency"`
					Amount   int    `json:"amount"`
				} `json:"price"`
				PriceComment string `json:"price_comment"`
			} `json:"time"`
		} `json:"delivery"`
	} `json:"data"`
}

// GetDeliveryOptions fetches MyParcel's own priced delivery slots for an
// address -- the only pre-shipment-creation price MyParcel's API exposes at
// all (every other price field, e.g. on GET /shipments/{id}, only exists
// after a shipment is actually created). Per MyParcel's own docs this
// endpoint only accepts in.CC "NL" or "BE" -- this client does not enforce
// that itself (see application/shipping.GetDeliveryPriceEstimateUseCase,
// which rejects any other country before ever reaching here). NOT
// confirmed against a real sandbox response during this build (see the
// package doc comment's caveat list): specifically, the carrier query
// param's exact format -- the docs example showed a string slug
// ("postnl") but this client sends MyParcel's numeric carrier id instead,
// matching every other carrier field in this codebase (e.g.
// ShipmentInput.Carrier) -- if a real key 400s on this param, that's the
// first thing to check.
func (c *Client) GetDeliveryOptions(ctx context.Context, in DeliveryOptionsInput) ([]DeliveryPriceOption, error) {
	q := url.Values{}
	q.Set("cc", in.CC)
	q.Set("postal_code", in.PostalCode)
	q.Set("number", in.Number)
	// platform is required by this endpoint -- MyParcel rejects the request
	// with 400 "platform veld is verplicht" ("platform field is required")
	// without it. Fixed value, not per-call, since this client only ever
	// talks to MyParcel's own platform.
	q.Set("platform", "myparcel")
	if in.Carrier != 0 {
		q.Set("carrier", strconv.Itoa(in.Carrier))
	}

	var resp deliveryOptionsEnvelope
	path := "/delivery_options?" + q.Encode()
	rawBody, err := c.doJSONRaw(ctx, http.MethodGet, path, deliveryOptionsAccept, nil, &resp)
	if err != nil {
		return nil, err
	}

	var options []DeliveryPriceOption
	for _, d := range resp.Data.Delivery {
		for _, t := range d.Time {
			options = append(options, DeliveryPriceOption{
				PriceComment: t.PriceComment, AmountMinorUnits: t.Price.Amount, Currency: t.Price.Currency,
			})
		}
	}
	if len(options) == 0 {
		// MyParcel returned 200 with no priced slots -- happens legitimately
		// when a shop has no delivery-options/dropoff-point config for this
		// carrier in the MyParcel dashboard, but could also mean this
		// client's response parsing (deliveryOptionsEnvelope) no longer
		// matches a real response shape. Logging the raw body here (rather
		// than silently returning empty) is the only way to tell the two
		// apart without a live sandbox to inspect.
		log.Printf("myparcel: GetDeliveryOptions returned no priced slots for %s: %s", path, string(rawBody))
	}
	return options, nil
}
