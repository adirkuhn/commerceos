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
//
// GPCCode was removed 2026-08-25: it was submitted as an attribute id
// "GPC Code", which does not exist in Bol's real data model (confirmed by
// downloading and searching it directly -- no attribute with that id, or
// any "gpc"/"classification" id, exists). Bol's real
// CreateProductContentSingleRequest schema has no classification field at
// all (only language/attributes/assets) -- a GPC/chunk id is only ever
// returned, as a prediction, from the separate POST
// /content/chunk-recommendations endpoint, which this client doesn't call.
// So this field never did anything; it was silently dropped on every
// submission, same failure mode as the other wrong ids fixed this session.
// If genuinely-new-product classification is ever needed, it requires
// wiring up chunk-recommendations first -- not done here, out of scope for
// this bug fix (the reported bug and this whole content-mapping cleanup
// are about updating content for products that already exist on Bol).
type BolProductContentInput struct {
	EAN         string
	Title       string // wire attribute id is "Name", not "Title" -- see adapter.go
	Description string
	ImageURLs   []string
	// Attributes is category-specific product-content fields (manufacturer
	// info, ingredients, shelf life, etc), keyed by Bol's real attribute id
	// -- an English data-model key (e.g. "Manufacturer Name"), not the
	// Dutch UI label. Confirmed 2026-08-25 against Bol's published data
	// model (productdatamodel.s-bol.com/v10/datamodel_v10_en.json) for
	// every key application/channels.bolProductContentAttributes sends --
	// see that function's doc comment for the full field-by-field
	// breakdown (units, LOV constraints, what's still free text and why).
	Attributes map[string]string
	// AttributeUnits is the Bol unece unit code for any Attributes entry
	// that's number+unit-typed (Capacity, Width/Height/Length, Weight,
	// Number Pieces In Package) -- keyed by the same attribute id as
	// Attributes. Bol's AttributeValue carries value and unitId as sibling
	// fields, so a flat map[string]string can't express both; this is the
	// companion map. Absent/empty for every other (text/boolean/LOV)
	// attribute, which carries no unit.
	AttributeUnits map[string]string
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

// BolNotForSaleReason is one reason Bol currently won't list an offer for
// sale in one country -- mirrors the real v11 not-for-sale-reasons
// response (see infrastructure/bol.NotForSaleReasons). MissingAttributes
// is only ever populated for a CONTENT-category reason.
type BolNotForSaleReason struct {
	CountryCode       string
	Category          string
	Code              string
	Description       string
	MissingAttributes []string
}

// BolShipmentItem is one order item to mark as shipped, by Bol's own
// orderItemId.
type BolShipmentItem struct {
	OrderItemID string
	Quantity    int
}

// BolShipmentInput tells Bol that order items left the warehouse with a
// transporter (Bol's transporter code, see channels.BolTransporterCode)
// and track & trace code.
type BolShipmentInput struct {
	OrderItems        []BolShipmentItem
	ShipmentReference string
	TransporterCode   string
	TrackAndTrace     string
}

// BolTransportInput is the new transporter + track & trace for a
// shipment Bol already has.
type BolTransportInput struct {
	TransporterCode string
	TrackAndTrace   string
}

// BolProcessStatus is Bol's answer to an asynchronous request: Status is
// PENDING until Bol has processed it, then SUCCESS, FAILURE or TIMEOUT
// (ErrorMessage explains a failure).
type BolProcessStatus struct {
	ID           string
	Status       string
	ErrorMessage string
}

// BolClient is one Bol connection's credentials + endpoints, adapted to
// exactly what this codebase's use cases need (ISP) -- mirrors WooClient's
// role. Implemented by infrastructure/bol.
type BolClient interface {
	CreateOffer(ctx context.Context, in BolOfferInput) (offerID string, err error)
	UpdateOffer(ctx context.Context, offerID string, in BolOfferInput) error
	// SetOfferOnHold toggles onHoldByRetailer alone via a minimal PATCH,
	// never touching price/stock/fulfilment -- used to gate/ungate a
	// listing when the local product's status crosses the Active boundary,
	// see application/channels.SyncProductToChannelsUseCase.
	SetOfferOnHold(ctx context.Context, offerID string, onHold bool) error
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
	// FindOfferByEAN looks up whether an offer already exists on Bol for
	// this EAN -- lets a caller link+update a pre-existing offer instead of
	// blindly creating a duplicate. Empty string + false (not an error)
	// means no offer exists yet.
	FindOfferByEAN(ctx context.Context, ean string) (offerID string, found bool, err error)
	// ListEconomicOperators returns every GPSR economic operator registered
	// against this Bol account -- requires the economic-operators.read (or
	// .manage) OAuth scope, which is granted per-app in Bol's Partner
	// Platform and may not be present on an older/narrower app
	// registration; a scope-denied response surfaces as a normal error
	// here, same as any other Bol API failure.
	ListEconomicOperators(ctx context.Context) ([]BolEconomicOperator, error)
	// GetNotForSaleReasons reports why offerID currently isn't for sale, per
	// country -- empty slice means it's for sale everywhere. Flattened
	// across countries (one entry per country x reason) since callers here
	// only ever display/store these, never need to group by country.
	GetNotForSaleReasons(ctx context.Context, offerID string) ([]BolNotForSaleReason, error)
	// CreateShipment tells Bol that order items shipped. Bol processes it
	// asynchronously -- poll GetProcessStatus with the returned ID. A
	// request Bol refuses outright wraps ErrBolRequestRejected.
	CreateShipment(ctx context.Context, in BolShipmentInput) (BolProcessStatus, error)
	GetProcessStatus(ctx context.Context, processStatusID string) (BolProcessStatus, error)
	// ListShipmentTransportIDs returns the transport id of every shipment
	// Bol has for this order.
	ListShipmentTransportIDs(ctx context.Context, orderID string) ([]string, error)
	// UpdateTransport replaces a shipment's transporter + track & trace
	// (asynchronous, like CreateShipment).
	UpdateTransport(ctx context.Context, transportID string, in BolTransportInput) (BolProcessStatus, error)
}

// BolClientFactory builds a BolClient scoped to one connection's credentials.
type BolClientFactory interface {
	NewClient(clientID, clientSecret string) (BolClient, error)
}
