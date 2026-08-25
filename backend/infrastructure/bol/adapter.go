package bol

import (
	"context"
	"math"

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
func toFulfilment(method string) Fulfilment {
	f := Fulfilment{Method: method}
	if method == "FBR" {
		f.Schedule = "BOL_DELIVERY_PROMISE"
	}
	return f
}

// toCreateOfferRequest builds a full offer payload for POST /offers. Note
// in.DeliveryCode is deliberately not mapped -- v11's Fulfilment has no
// such field (that was a v10-deprecated-only requirement); BolOfferInput
// keeps the field for now (harmless, just unused here) rather than
// ripping out the connection-level setting/UI mid-incident.
func toCreateOfferRequest(in integrationsDomain.BolOfferInput) CreateOfferRequest {
	var stock *Stock
	if in.FulfilmentMethod == "FBR" {
		stock = &Stock{Amount: in.StockAmount, ManagedByRetailer: in.StockManagedByRetailer}
	}
	return CreateOfferRequest{
		EAN: in.EAN, UnknownProductTitle: in.UnknownProductTitle,
		EconomicOperatorID: in.EconomicOperatorID, Reference: in.Reference,
		Condition:  Condition{Category: in.ConditionCategory},
		Pricing:    Pricing{BundlePrices: []BundlePrice{{Quantity: 1, UnitPrice: in.UnitPrice}}},
		Fulfilment: toFulfilment(in.FulfilmentMethod),
		Stock:      stock,
	}
}

// toPatchOfferRequest mirrors toCreateOfferRequest for PATCH -- no
// Condition field at all (Bol's real PatchOfferRequest schema has none;
// an offer's condition can't be changed after creation).
func toPatchOfferRequest(in integrationsDomain.BolOfferInput) PatchOfferRequest {
	var stock *Stock
	if in.FulfilmentMethod == "FBR" {
		stock = &Stock{Amount: in.StockAmount, ManagedByRetailer: in.StockManagedByRetailer}
	}
	fulfilment := toFulfilment(in.FulfilmentMethod)
	return PatchOfferRequest{
		EconomicOperatorID: in.EconomicOperatorID, Reference: in.Reference,
		Pricing:    &Pricing{BundlePrices: []BundlePrice{{Quantity: 1, UnitPrice: in.UnitPrice}}},
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

func (a *Adapter) FindOfferByEAN(ctx context.Context, ean string) (string, bool, error) {
	return a.client.FindOfferByEAN(ctx, ean)
}

func (a *Adapter) IsUnknownEAN(err error) bool { return IsUnknownEAN(err) }

func (a *Adapter) SubmitProductContent(ctx context.Context, in integrationsDomain.BolProductContentInput) error {
	attrs := []ContentAttribute{
		{ID: "EAN", Values: []AttributeValue{{Value: in.EAN}}},
	}
	if in.GPCCode != "" {
		attrs = append(attrs, ContentAttribute{ID: "GPC Code", Values: []AttributeValue{{Value: in.GPCCode}}})
	}
	if in.Title != "" {
		attrs = append(attrs, ContentAttribute{ID: "Title", Values: []AttributeValue{{Value: in.Title}}})
	}
	if in.Description != "" {
		attrs = append(attrs, ContentAttribute{ID: "Description", Values: []AttributeValue{{Value: in.Description}}})
	}
	assets := make([]Asset, len(in.ImageURLs))
	for i, u := range in.ImageURLs {
		assets[i] = Asset{URL: u, Labels: "FRONT"}
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
