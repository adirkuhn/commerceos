package bol

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// --- Orders API (v10) ---
// Confirmed against Bol's real Retailer API v10 OpenAPI spec (downloaded
// and inspected directly, unlike GetCommission's path above which was a
// verified-correct guess): GET /orders (paginated, 50/page, reduced shape
// -- no shipment/billing details, no per-item price) and GET /orders/{id}
// (single order, full shape). Neither supports a "modified since" cursor
// -- only a status filter -- so ImportBolOrdersUseCase polls status=OPEN
// every run rather than tracking an incremental cursor.

type reducedOrdersResponse struct {
	Orders []reducedOrder `json:"orders"`
}

type reducedOrder struct {
	OrderID             string             `json:"orderId"`
	OrderPlacedDateTime time.Time          `json:"orderPlacedDateTime"`
	OrderItems          []reducedOrderItem `json:"orderItems"`
}

type reducedOrderItem struct {
	OrderItemID         string `json:"orderItemId"`
	EAN                 string `json:"ean"`
	Quantity            int    `json:"quantity"`
	QuantityShipped     int    `json:"quantityShipped"`
	QuantityCancelled   int    `json:"quantityCancelled"`
	CancellationRequest bool   `json:"cancellationRequest"`
}

type orderResponse struct {
	OrderID             string          `json:"orderId"`
	OrderPlacedDateTime time.Time       `json:"orderPlacedDateTime"`
	ShipmentDetails     shipmentDetails `json:"shipmentDetails"`
	OrderItems          []orderItem     `json:"orderItems"`
}

type shipmentDetails struct {
	FirstName   string `json:"firstName"`
	Surname     string `json:"surname"`
	StreetName  string `json:"streetName"`
	HouseNumber string `json:"houseNumber"`
	ZipCode     string `json:"zipCode"`
	City        string `json:"city"`
	CountryCode string `json:"countryCode"`
	Email       string `json:"email"`
}

type orderItem struct {
	OrderItemID         string           `json:"orderItemId"`
	CancellationRequest bool             `json:"cancellationRequest"`
	Fulfilment          *orderFulfilment `json:"fulfilment"`
	Offer               orderOffer       `json:"offer"`
	Product             orderProduct     `json:"product"`
	Quantity            int              `json:"quantity"`
	QuantityShipped     int              `json:"quantityShipped"`
	QuantityCancelled   int              `json:"quantityCancelled"`
	UnitPrice           float64          `json:"unitPrice"`
}

// orderFulfilment is v10's OrderFulfilment -- only the delivery-promise
// dates are read. Both are "format: date" strings ("2017-02-10") and
// nullable; Bol fills at most one of them (exactDeliveryDate only when the
// customer picked a delivery day).
type orderFulfilment struct {
	LatestDeliveryDate string `json:"latestDeliveryDate"`
	ExactDeliveryDate  string `json:"exactDeliveryDate"`
}

type orderOffer struct {
	Reference string `json:"reference"`
}

type orderProduct struct {
	EAN   string `json:"ean"`
	Title string `json:"title"`
}

// ListOrders returns one page of orders (Bol's fixed 50/page) filtered by
// status ("OPEN", "SHIPPED", or "ALL"). Empty slice (not an error) means no
// more pages -- caller stops paging.
func (c *Client) ListOrders(ctx context.Context, page int, status string) ([]reducedOrder, error) {
	var resp reducedOrdersResponse
	path := "/orders?page=" + strconv.Itoa(page) + "&status=" + url.QueryEscape(status) + "&fulfilment-method=FBR"
	if err := c.doJSON(ctx, http.MethodGet, path, contentAccept, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Orders, nil
}

// GetOrder returns one order's full detail, including ShipmentDetails and
// per-item pricing -- neither present in ListOrders' reduced shape.
func (c *Client) GetOrder(ctx context.Context, orderID string) (*orderResponse, error) {
	var resp orderResponse
	path := "/orders/" + url.PathEscape(orderID)
	if err := c.doJSON(ctx, http.MethodGet, path, contentAccept, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// --- Economic Operator API (v1) ---
// Confirmed against Bol's real Economic Operator API v1 spec
// (api.bol.com/registry/api-definitions/economic-operators/economic-operators-v1.yaml,
// same /retailer base path this client already uses, different media type
// -- economicOperatorAccept below, not contentAccept). Only the list
// endpoint is used here (a "fetch from Bol" convenience for the
// Economic Operator ID field); create/update/delete/get-by-id are not
// needed by this codebase.

const economicOperatorAccept = "application/vnd.economic-operator.v1+json"

type economicOperatorResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type economicOperatorsPageResponse struct {
	Operators []economicOperatorResponse `json:"operators"`
}

// ListEconomicOperators returns every GPSR economic operator registered
// against this Bol account. Requires the economic-operators.read (or
// .manage) OAuth scope -- a 403 here means the app registration behind
// this connection's client_id/secret hasn't been granted that scope in
// Bol's Partner Platform, not a bug in this client.
func (c *Client) ListEconomicOperators(ctx context.Context) ([]economicOperatorResponse, error) {
	var resp economicOperatorsPageResponse
	if err := c.doJSON(ctx, http.MethodGet, "/economic-operators", economicOperatorAccept, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Operators, nil
}
