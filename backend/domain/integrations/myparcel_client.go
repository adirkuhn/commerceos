package integrations

import (
	"context"
	"time"
)

// MyParcelAddress is a shipment's structured recipient address -- the shape
// MyParcel's API requires (street/number/number_suffix split), distinct
// from oms.Address's free-text Address1/Address2 (Woo-shaped). Staff
// confirm/edit this once per shipment at label-creation time; it is never
// derived live from a SalesOrder on every read (see domain/shipping.Shipment's
// doc comment).
type MyParcelAddress struct {
	Person       string
	Company      string
	Street       string
	Number       string
	NumberSuffix string
	PostalCode   string
	City         string
	CC           string
	Email        string
	Phone        string
}

// MyParcelShipmentOptions mirrors the subset of MyParcel's ShipmentOptions
// object this codebase exposes in v1 (package_type, signature, insurance,
// only_recipient) -- see infrastructure/myparcel's package doc for the full
// object and why the rest are left at MyParcel's own defaults.
type MyParcelShipmentOptions struct {
	PackageType       int
	SignatureRequired bool
	OnlyRecipient     bool
	// InsuranceAmountMinorEUR's name is historical (predates Currency
	// existing) -- the amount is in minor units of Currency, not
	// necessarily EUR. Left unrenamed to avoid touching the
	// domain/shipping.Shipment column of the same name across every
	// existing caller/migration for a field-name-only change.
	InsuranceAmountMinorEUR *int
	// Currency is the tenant's Settings -> Localization -> Default Currency
	// (domain/settings.TenantSettings.DefaultCurrency) at shipment-create
	// time -- MyParcel's insurance option requires an explicit currency
	// per shipment (see infrastructure/myparcel's package doc, section
	// 6.A.3), it does not infer one. Empty means "use the client's own
	// fallback" (see infrastructure/myparcel.Client.CreateShipment).
	Currency string
}

// CreateShipmentInput is what CreateShipmentUseCase needs to create one
// MyParcel shipment. OrderReference is set to our own SalesOrder.ID so it
// round-trips on every webhook event for that shipment (see
// domain/shipping.HandleShipmentWebhook's doc comment).
type CreateShipmentInput struct {
	Recipient      MyParcelAddress
	Carrier        int
	Options        MyParcelShipmentOptions
	OrderReference string
}

// ShipmentResult is what MyParcel returns for a created/fetched shipment.
type ShipmentResult struct {
	ShipmentID int64
	Barcode    string
}

// CreateOrderInput is what ExportMyParcelOrderUseCase needs to export a
// sales order via MyParcel's Orders/Fulfilment API (POST
// /fulfilment/orders) -- a separate, real MyParcel endpoint from
// CreateShipmentInput's /shipments, used for the "export order, let staff
// confirm the address, then create the shipment" flow (see
// ExportMyParcelOrderUseCase's doc). AccountID/ShopID are the tenant's own
// MyParcel merchant identifiers (domain/shipping.ShippingConnectionSettings),
// required by this endpoint specifically -- Shipments API doesn't need them.
type CreateOrderInput struct {
	ExternalIdentifier string
	OrderDate          time.Time
	AccountID          int
	ShopID             int
	Carrier            int
	Recipient          MyParcelAddress
	PackageType        int
}

// OrderResult is what MyParcel returns for a created order -- UUID is
// persisted (MyParcelFulfilmentOrderRepository) so a later shipment-create
// call and idempotency checks (don't re-export an already-exported order)
// can look it back up.
type OrderResult struct {
	UUID   string
	Status string
}

// LabelResult is a fetched shipment label -- Data is the raw response body
// (a PDF by default; ZPL if requested, see GetLabel's Format parameter),
// ContentType is the response's Content-Type header, passed straight
// through so the frontend print view doesn't need to guess it.
type LabelResult struct {
	Data        []byte
	ContentType string
}

// TrackTraceResult is one shipment's current tracking status, as reported
// by GET /tracktraces/{id}. StatusMain is MyParcel's own string category
// (e.g. "registered", "delivered") -- matched on in preference to the
// numeric StatusCode, since the full numeric status-code table wasn't
// confirmed against MyParcel's docs during this build (see
// infrastructure/myparcel.Client.GetTrackTrace's doc comment).
type TrackTraceResult struct {
	StatusCode int
	StatusMain string
}

// MyParcelClient is one MyParcel connection's credentials + endpoints,
// adapted to exactly what this codebase's use cases need (ISP). Implemented
// by infrastructure/myparcel.
type MyParcelClient interface {
	CreateShipment(ctx context.Context, in CreateShipmentInput) (ShipmentResult, error)
	// CreateOrder exports an order via MyParcel's Orders/Fulfilment API --
	// see CreateOrderInput's doc for how this differs from CreateShipment.
	CreateOrder(ctx context.Context, in CreateOrderInput) (OrderResult, error)
	// GetLabel fetches a shipment's label. format is "pdf" or "zpl" (see
	// domain/shipping.LabelFormatPDF/LabelFormatZPL).
	GetLabel(ctx context.Context, myParcelShipmentID int64, format string) (LabelResult, error)
	GetTrackTrace(ctx context.Context, myParcelShipmentID int64) (TrackTraceResult, error)
	// RegisterWebhook registers (or, per MyParcel's own "adding the same
	// webhook overwrites any existing one" behavior, re-registers) this
	// shop's shipment_status_change callback URL. Safe to call more than
	// once for the same connection.
	RegisterWebhook(ctx context.Context, callbackURL string) error
}

// MyParcelClientFactory builds a MyParcelClient scoped to one connection's
// API key.
type MyParcelClientFactory interface {
	NewClient(apiKey string) (MyParcelClient, error)
}
