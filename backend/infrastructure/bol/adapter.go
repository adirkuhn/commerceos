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

func toOfferRequest(in integrationsDomain.BolOfferInput) CreateOfferRequest {
	var stock *Stock
	if in.FulfilmentMethod == "FBR" {
		stock = &Stock{Amount: in.StockAmount, ManagedByRetailer: in.StockManagedByRetailer}
	}
	return CreateOfferRequest{
		EAN: in.EAN, UnknownProductTitle: in.UnknownProductTitle,
		EconomicOperatorID: in.EconomicOperatorID, Reference: in.Reference,
		Condition:  Condition{Category: in.ConditionCategory},
		Pricing:    Pricing{BundlePrices: []BundlePrice{{Quantity: 1, UnitPrice: in.UnitPrice}}},
		Fulfilment: Fulfilment{Method: in.FulfilmentMethod},
		Stock:      stock,
	}
}

func (a *Adapter) CreateOffer(ctx context.Context, in integrationsDomain.BolOfferInput) (string, error) {
	return a.client.CreateOffer(ctx, toOfferRequest(in))
}

func (a *Adapter) UpdateOffer(ctx context.Context, offerID string, in integrationsDomain.BolOfferInput) error {
	return a.client.UpdateOffer(ctx, offerID, toOfferRequest(in))
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

// ClientFactory implements domain/integrations.BolClientFactory.
type ClientFactory struct{}

func NewClientFactory() ClientFactory { return ClientFactory{} }

func (ClientFactory) NewClient(clientID, clientSecret string) (integrationsDomain.BolClient, error) {
	return NewAdapter(NewClient(Config{ClientID: clientID, ClientSecret: clientSecret})), nil
}
