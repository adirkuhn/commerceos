package integrations

import (
	"context"
	"time"
)

// BolOfferInput is what PushProductToBolUseCase needs to create/update one
// Bol offer -- domain-owned shape, kept separate from infrastructure/bol's
// own HTTP request struct so domain/application never import infrastructure
// types (see CLAUDE.md's layering rule).
type BolOfferInput struct {
	EAN                    string
	UnknownProductTitle    string
	EconomicOperatorID     string
	Reference              string
	ConditionCategory      string
	FulfilmentMethod       string
	DeliveryCode           string
	UnitPrice              float64
	StockAmount            int
	StockManagedByRetailer bool
}

// BolProductContentInput is what's needed to submit/update Bol catalog
// content for an EAN that isn't yet known to Bol -- only used when
// CreateOffer reports an unknown EAN.
type BolProductContentInput struct {
	EAN         string
	GPCCode     string
	Title       string
	Description string
	ImageURLs   []string
}

type BolCommission struct {
	EAN         string
	FixedAmount float64
	Percentage  float64
}

// BolOrderItem is one line of a BolOrder -- domain-owned shape (see
// BolOfferInput's doc comment on why this stays separate from
// infrastructure/bol's wire structs). Quantity/QuantityShipped/
// QuantityCancelled/CancellationRequest are what ImportBolOrdersUseCase
// derives an aggregate order status from -- Bol's Orders API has no single
// order-level status field, only these per-item facts (confirmed against
// the real Retailer API v10 spec: ReducedOrderItem/OrderOrderItem carry no
// "status" property at all).
type BolOrderItem struct {
	OrderItemID         string
	EAN                 string
	Title               string
	Reference           string
	Quantity            int
	QuantityShipped     int
	QuantityCancelled   int
	CancellationRequest bool
	UnitPriceMinorUnits int64
}

// BolShipmentDetails is a Bol order's delivery address -- only present on
// the full single-order response (GetOrder), not the paginated list
// (ListOrders), same reduced-vs-full split Bol's API itself makes.
type BolShipmentDetails struct {
	FirstName   string
	Surname     string
	StreetName  string
	HouseNumber string
	ZipCode     string
	City        string
	CountryCode string
	Email       string
}

// BolOrder is one Bol order -- ListOrders returns the reduced shape
// (OrderItems only, no ShipmentDetails), GetOrder returns the full shape.
// ImportBolOrdersUseCase pages ListOrders to discover order IDs, then calls
// GetOrder per new one for shipping details.
type BolOrder struct {
	OrderID         string
	OrderPlacedAt   time.Time
	ShipmentDetails BolShipmentDetails
	OrderItems      []BolOrderItem
}

// BolEconomicOperator is one GPSR economic operator registered against a
// Bol account -- confirmed against Bol's separate Economic Operator API v1
// (api.bol.com/registry, same /retailer base path, different media type
// application/vnd.economic-operator.v1+json). Powers a "fetch from Bol"
// button so the merchant doesn't have to manually copy the id out of Bol's
// Partner Platform UI, which is the only place it was previously visible.
type BolEconomicOperator struct {
	ID     string
	Name   string
	Status string // PENDING, VALID, or INVALID
}

// BolClient is one Bol connection's credentials + endpoints, adapted to
// exactly what this codebase's use cases need (ISP) -- mirrors WooClient's
// role. Implemented by infrastructure/bol.
type BolClient interface {
	CreateOffer(ctx context.Context, in BolOfferInput) (offerID string, err error)
	UpdateOffer(ctx context.Context, offerID string, in BolOfferInput) error
	// IsUnknownEAN classifies an error CreateOffer/UpdateOffer returned --
	// true means "this EAN isn't in Bol's catalog yet, submit product
	// content first," false means some other failure.
	IsUnknownEAN(err error) bool
	SubmitProductContent(ctx context.Context, in BolProductContentInput) error
	GetCommission(ctx context.Context, ean string, unitPrice float64) (BolCommission, error)
	// ListOrders returns one page (Bol's fixed 50/page) of orders matching
	// status ("OPEN", "SHIPPED", or "ALL") -- reduced shape, no shipment
	// details. Empty slice (not an error) means no more pages.
	ListOrders(ctx context.Context, page int, status string) ([]BolOrder, error)
	// GetOrder returns one order's full detail, including ShipmentDetails
	// -- called once per newly-discovered order ID from ListOrders.
	GetOrder(ctx context.Context, orderID string) (*BolOrder, error)
	// ListEconomicOperators returns every GPSR economic operator registered
	// against this Bol account -- requires the economic-operators.read (or
	// .manage) OAuth scope, which is granted per-app in Bol's Partner
	// Platform and may not be present on an older/narrower app
	// registration; a scope-denied response surfaces as a normal error
	// here, same as any other Bol API failure.
	ListEconomicOperators(ctx context.Context) ([]BolEconomicOperator, error)
}

// BolClientFactory builds a BolClient scoped to one connection's credentials.
type BolClientFactory interface {
	NewClient(clientID, clientSecret string) (BolClient, error)
}
