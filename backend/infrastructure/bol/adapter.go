package bol

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

// Adapter implements domain/integrations.BolClient on top of Client,
// mapping domain-owned request/response shapes to this package's own HTTP
// types at the boundary -- domain/application never import this package
// directly, only through the BolClient interface.
type Adapter struct{ client *Client }

func NewAdapter(client *Client) *Adapter { return &Adapter{client: client} }

// toFulfilment sets Schedule to BOL_DELIVERY_PROMISE for FBR -- Bol
// requires this field for FBR (a real 400 confirmed it: "Field 'schedule'
// is required when 'method' is FBR") even though the static v11.5.0 spec
// doesn't mark it schema-required. BOL_DELIVERY_PROMISE (let Bol calculate
// the promise) is the simplest correct choice: MY_DELIVERY_PROMISE needs a
// separately-configured Delivery Promise API profile this codebase
// doesn't integrate, and SHIPPING_VIA_BOL is a different fulfilment
// concept entirely. Not set for FBB, which Bol fully controls itself.
//
// Bol also requires DeliveryPromise whenever Schedule is
// BOL_DELIVERY_PROMISE (another real 400: "Field 'deliveryPromise' is
// required when 'method' = FBR and 'schedule' is BOL_DELIVERY_PROMISE") --
// parsed from deliveryCode, the connection's stored day-range setting
// ("3-5d" etc, still the human-friendly input the settings UI collects;
// see domain/integrations.DefaultBolDeliveryCode). The 24uurs-XX/
// MijnLeverbelofte/VVB codes from that same setting's original v10-based
// enum don't parse as an "N-Md" range and fall back to the 3-5 day
// default -- they'd actually need a different Schedule value entirely,
// not handled here.
func toFulfilment(method, deliveryCode string) Fulfilment {
	f := Fulfilment{Method: method}
	if method == "FBR" {
		f.Schedule = "BOL_DELIVERY_PROMISE"
		promise := parseDeliveryPromise(deliveryCode)
		f.DeliveryPromise = &promise
	}
	return f
}

func parseDeliveryPromise(code string) DeliveryPromise {
	fallback := DeliveryPromise{MinimumDaysToCustomer: 3, MaximumDaysToCustomer: 5}
	trimmed := strings.TrimSuffix(code, "d")
	if trimmed == code {
		return fallback
	}
	parts := strings.SplitN(trimmed, "-", 2)
	if len(parts) != 2 {
		return fallback
	}
	min, errMin := strconv.Atoi(parts[0])
	max, errMax := strconv.Atoi(parts[1])
	if errMin != nil || errMax != nil {
		return fallback
	}
	return DeliveryPromise{MinimumDaysToCustomer: min, MaximumDaysToCustomer: max}
}

// toCreateOfferRequest builds a full offer payload for POST /offers. Note
// in.DeliveryCode feeds toFulfilment's DeliveryPromise parsing below --
// it's no longer sent as its own field (v11 has no such field, that was a
// v10-deprecated-only requirement) but is reused as the day-range input
// for the real v11 requirement, DeliveryPromise.
func toCreateOfferRequest(in integrationsDomain.BolOfferInput) CreateOfferRequest {
	var stock *Stock
	if in.FulfilmentMethod == "FBR" {
		stock = &Stock{Amount: in.StockAmount, ManagedByRetailer: in.StockManagedByRetailer}
	}
	return CreateOfferRequest{
		EAN: in.EAN, UnknownProductTitle: in.UnknownProductTitle,
		EconomicOperatorID: in.EconomicOperatorID, Reference: in.Reference,
		Condition:  Condition{Category: in.ConditionCategory},
		Pricing:    Pricing{BundlePrices: toBundlePrices(in.Prices)},
		Fulfilment: toFulfilment(in.FulfilmentMethod, in.DeliveryCode),
		Stock:      stock,
	}
}

func toBundlePrices(prices []integrationsDomain.BolPrice) []BundlePrice {
	out := make([]BundlePrice, len(prices))
	for i, p := range prices {
		out[i] = BundlePrice{Quantity: p.Quantity, UnitPrice: p.UnitPrice}
	}
	return out
}

// toPatchOfferRequest mirrors toCreateOfferRequest for PATCH -- no
// Condition field at all (Bol's real PatchOfferRequest schema has none;
// an offer's condition can't be changed after creation).
func toPatchOfferRequest(in integrationsDomain.BolOfferInput) PatchOfferRequest {
	var stock *Stock
	if in.FulfilmentMethod == "FBR" {
		stock = &Stock{Amount: in.StockAmount, ManagedByRetailer: in.StockManagedByRetailer}
	}
	fulfilment := toFulfilment(in.FulfilmentMethod, in.DeliveryCode)
	return PatchOfferRequest{
		EconomicOperatorID: in.EconomicOperatorID, Reference: in.Reference,
		Pricing:    &Pricing{BundlePrices: toBundlePrices(in.Prices)},
		Fulfilment: &fulfilment,
		Stock:      stock,
	}
}

func (a *Adapter) CreateOffer(ctx context.Context, in integrationsDomain.BolOfferInput) (string, error) {
	return a.client.CreateOffer(ctx, toCreateOfferRequest(in))
}

func (a *Adapter) UpdateOffer(ctx context.Context, offerID string, in integrationsDomain.BolOfferInput) error {
	return a.client.UpdateOffer(ctx, offerID, toPatchOfferRequest(in))
}

func (a *Adapter) SetOfferOnHold(ctx context.Context, offerID string, onHold bool) error {
	return a.client.SetOfferOnHold(ctx, offerID, onHold)
}

func (a *Adapter) FindOfferByEAN(ctx context.Context, ean string) (string, bool, error) {
	return a.client.FindOfferByEAN(ctx, ean)
}

func (a *Adapter) IsUnknownEAN(err error) bool { return IsUnknownEAN(err) }

func (a *Adapter) SubmitProductContent(ctx context.Context, in integrationsDomain.BolProductContentInput) error {
	attrs := []ContentAttribute{
		{ID: "EAN", Values: []AttributeValue{{Value: in.EAN}}},
	}
	if in.Title != "" {
		// Real id is "Name", not "Title" -- see BolProductContentInput's
		// doc comment.
		attrs = append(attrs, ContentAttribute{ID: "Name", Values: []AttributeValue{{Value: in.Title}}})
	}
	if in.Description != "" {
		attrs = append(attrs, ContentAttribute{ID: "Description", Values: []AttributeValue{{Value: in.Description}}})
	}
	for id, value := range in.Attributes {
		attrs = append(attrs, ContentAttribute{ID: id, Values: []AttributeValue{{Value: value, UnitID: in.AttributeUnits[id]}}})
	}
	assets := make([]Asset, len(in.ImageURLs))
	for i, u := range in.ImageURLs {
		assets[i] = Asset{URL: u, Labels: []string{"FRONT"}}
	}
	return a.client.SubmitProductContent(ctx, ProductContentRequest{
		Language: "nl", Attributes: attrs, Assets: assets,
	}, PollOptions{})
}

func (a *Adapter) GetCommission(ctx context.Context, ean string, unitPrice float64) (integrationsDomain.BolCommission, error) {
	c, err := a.client.GetCommission(ctx, ean, unitPrice)
	if err != nil {
		return integrationsDomain.BolCommission{}, err
	}
	return integrationsDomain.BolCommission{EAN: c.EAN, FixedAmount: c.FixedAmount, Percentage: c.Percentage}, nil
}

func (a *Adapter) GetNotForSaleReasons(ctx context.Context, offerID string) ([]integrationsDomain.BolNotForSaleReason, error) {
	resp, err := a.client.GetNotForSaleReasons(ctx, offerID)
	if err != nil {
		return nil, err
	}
	var reasons []integrationsDomain.BolNotForSaleReason
	for _, country := range resp.Countries {
		for _, r := range country.Reasons {
			missing := make([]string, len(r.MissingAttributes))
			for i, m := range r.MissingAttributes {
				missing[i] = m.Name
			}
			reasons = append(reasons, integrationsDomain.BolNotForSaleReason{
				CountryCode: country.CountryCode, Category: r.Category, Code: r.Code,
				Description: r.Description, MissingAttributes: missing,
			})
		}
	}
	return reasons, nil
}

func (a *Adapter) ListOrders(ctx context.Context, page int, status string) ([]integrationsDomain.BolOrder, error) {
	reduced, err := a.client.ListOrders(ctx, page, status)
	if err != nil {
		return nil, err
	}
	orders := make([]integrationsDomain.BolOrder, len(reduced))
	for i, o := range reduced {
		items := make([]integrationsDomain.BolOrderItem, len(o.OrderItems))
		for j, it := range o.OrderItems {
			items[j] = integrationsDomain.BolOrderItem{
				OrderItemID: it.OrderItemID, EAN: it.EAN,
				Quantity: it.Quantity, QuantityShipped: it.QuantityShipped,
				QuantityCancelled: it.QuantityCancelled, CancellationRequest: it.CancellationRequest,
			}
		}
		orders[i] = integrationsDomain.BolOrder{OrderID: o.OrderID, OrderPlacedAt: o.OrderPlacedDateTime, OrderItems: items}
	}
	return orders, nil
}

func (a *Adapter) GetOrder(ctx context.Context, orderID string) (*integrationsDomain.BolOrder, error) {
	o, err := a.client.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	items := make([]integrationsDomain.BolOrderItem, len(o.OrderItems))
	for i, it := range o.OrderItems {
		items[i] = integrationsDomain.BolOrderItem{
			OrderItemID: it.OrderItemID, EAN: it.Product.EAN, Title: it.Product.Title,
			Reference: it.Offer.Reference, Quantity: it.Quantity, QuantityShipped: it.QuantityShipped,
			QuantityCancelled: it.QuantityCancelled, CancellationRequest: it.CancellationRequest,
			UnitPriceMinorUnits: int64(math.Round(it.UnitPrice * 100)),
		}
		if it.Fulfilment != nil {
			latest, err := parseBolDate(it.Fulfilment.LatestDeliveryDate)
			if err != nil {
				return nil, fmt.Errorf("bol order %s item %s: latestDeliveryDate: %w", o.OrderID, it.OrderItemID, err)
			}
			exact, err := parseBolDate(it.Fulfilment.ExactDeliveryDate)
			if err != nil {
				return nil, fmt.Errorf("bol order %s item %s: exactDeliveryDate: %w", o.OrderID, it.OrderItemID, err)
			}
			items[i].LatestDeliveryDate, items[i].ExactDeliveryDate = latest, exact
		}
	}
	return &integrationsDomain.BolOrder{
		OrderID: o.OrderID, OrderPlacedAt: o.OrderPlacedDateTime,
		ShipmentDetails: integrationsDomain.BolShipmentDetails{
			FirstName: o.ShipmentDetails.FirstName, Surname: o.ShipmentDetails.Surname,
			StreetName: o.ShipmentDetails.StreetName, HouseNumber: o.ShipmentDetails.HouseNumber,
			ZipCode: o.ShipmentDetails.ZipCode, City: o.ShipmentDetails.City,
			CountryCode: o.ShipmentDetails.CountryCode, Email: o.ShipmentDetails.Email,
		},
		OrderItems: items,
	}, nil
}

func (a *Adapter) ListEconomicOperators(ctx context.Context) ([]integrationsDomain.BolEconomicOperator, error) {
	operators, err := a.client.ListEconomicOperators(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]integrationsDomain.BolEconomicOperator, len(operators))
	for i, o := range operators {
		out[i] = integrationsDomain.BolEconomicOperator{ID: o.ID, Name: o.Name, Status: o.Status}
	}
	return out, nil
}

// ClientFactory implements domain/integrations.BolClientFactory.
type ClientFactory struct{}

func NewClientFactory() ClientFactory { return ClientFactory{} }

func (ClientFactory) NewClient(clientID, clientSecret string) (integrationsDomain.BolClient, error) {
	return NewAdapter(NewClient(Config{ClientID: clientID, ClientSecret: clientSecret})), nil
}

// parseBolDate reads one of Bol's "format: date" fields ("2017-02-10") as
// UTC midnight; empty (Bol sends null or omits it) is nil.
func parseBolDate(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	d, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return nil, err
	}
	return &d, nil
}
