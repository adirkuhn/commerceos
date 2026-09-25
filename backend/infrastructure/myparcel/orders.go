package myparcel

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// OrderResult is this package's own response shape for CreateOrder.
type OrderResult struct {
	UUID   string
	Status string
}

// --- Orders (Fulfilment API) ---

// OrderInput is this package's own request shape for CreateOrder -- kept
// separate from domain/integrations.CreateOrderInput the same way
// ShipmentInput is kept separate from CreateShipmentInput (see the comment
// above ShipmentInput); adapter.go maps between the two.
// OrderLine is one line item on the order -- see the package doc comment
// for the confirmed field names/nesting and the unconfirmed minor-units
// assumption on Price/VAT/PriceAfterVAT.
type OrderLine struct {
	Quantity                int
	SKU                     string
	Name                    string
	PriceMinorUnits         int
	VATMinorUnits           int
	PriceAfterVATMinorUnits int
}

type OrderInput struct {
	ExternalIdentifier string
	OrderDate          time.Time
	AccountID          int
	ShopID             int
	Carrier            int
	Recipient          Recipient
	PackageType        int
	// PriceMinorUnits/VATMinorUnits/PriceAfterVATMinorUnits are the order's
	// own totals -- required by MyParcel's Orders API (see the package doc
	// comment); PriceAfterVATMinorUnits should equal
	// PriceMinorUnits+VATMinorUnits.
	PriceMinorUnits         int
	VATMinorUnits           int
	PriceAfterVATMinorUnits int
	OrderLines              []OrderLine
}

type orderShipment struct {
	Carrier   int             `json:"carrier"`
	Recipient recipient       `json:"recipient"`
	Options   shipmentOptions `json:"options"`
}

type orderLineProduct struct {
	SKU  string `json:"sku"`
	Name string `json:"name"`
}

type orderLine struct {
	Quantity      int              `json:"quantity"`
	Price         int              `json:"price"`
	VAT           int              `json:"vat"`
	PriceAfterVAT int              `json:"price_after_vat"`
	Product       orderLineProduct `json:"product"`
}

type createOrderRequest struct {
	ExternalIdentifier string        `json:"external_identifier"`
	OrderDate          string        `json:"order_date"`
	AccountID          int           `json:"account_id"`
	ShopID             int           `json:"shop_id"`
	Price              int           `json:"price"`
	VAT                int           `json:"vat"`
	PriceAfterVAT      int           `json:"price_after_vat"`
	Shipment           orderShipment `json:"shipment"`
	OrderLines         []orderLine   `json:"order_lines"`
}

type ordersEnvelope struct {
	Data struct {
		Orders []struct {
			UUID   string `json:"uuid"`
			Status string `json:"status"`
		} `json:"orders"`
	} `json:"data"`
}

// CreateOrder exports one order via MyParcel's Orders/Fulfilment API
// (POST /fulfilment/orders) -- see the package doc comment's caveats;
// this endpoint specifically has not been exercised against a real
// sandbox response during this build (needs verification with real
// credentials before relying on it in production, same open item as the
// rest of this integration -- see project memory on Bol/MyParcel
// verification).
func (c *Client) CreateOrder(ctx context.Context, in OrderInput) (OrderResult, error) {
	req := struct {
		Data struct {
			Orders []createOrderRequest `json:"orders"`
		} `json:"data"`
	}{}
	lines := make([]orderLine, len(in.OrderLines))
	for i, l := range in.OrderLines {
		lines[i] = orderLine{
			Quantity: l.Quantity, Price: l.PriceMinorUnits, VAT: l.VATMinorUnits, PriceAfterVAT: l.PriceAfterVATMinorUnits,
			Product: orderLineProduct{SKU: l.SKU, Name: l.Name},
		}
	}
	req.Data.Orders = []createOrderRequest{{
		ExternalIdentifier: in.ExternalIdentifier,
		OrderDate:          in.OrderDate.Format("2006-01-02 15:04:05"),
		AccountID:          in.AccountID,
		ShopID:             in.ShopID,
		Price:              in.PriceMinorUnits, VAT: in.VATMinorUnits, PriceAfterVAT: in.PriceAfterVATMinorUnits,
		Shipment: orderShipment{
			Carrier: in.Carrier,
			Recipient: recipient{
				Person: in.Recipient.Person, Company: in.Recipient.Company,
				Street: in.Recipient.Street, Number: in.Recipient.Number, NumberSuffix: in.Recipient.NumberSuffix,
				PostalCode: in.Recipient.PostalCode, City: in.Recipient.City, CC: in.Recipient.CC,
				Email: in.Recipient.Email, Phone: in.Recipient.Phone,
			},
			Options: shipmentOptions{PackageType: in.PackageType},
		},
		OrderLines: lines,
	}}

	var resp ordersEnvelope
	if err := c.doJSON(ctx, http.MethodPost, "/fulfilment/orders", orderAccept, req, &resp); err != nil {
		return OrderResult{}, err
	}
	if len(resp.Data.Orders) == 0 {
		return OrderResult{}, fmt.Errorf("myparcel: create order: empty response")
	}
	o := resp.Data.Orders[0]
	return OrderResult{UUID: o.UUID, Status: o.Status}, nil
}
