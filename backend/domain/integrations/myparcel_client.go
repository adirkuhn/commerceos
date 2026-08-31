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
	// WeightGrams is the shipment's total physical weight, in grams --
	// MyParcel's own docs (07.shipment-related-object-definitions.html)
	// require it for non-EU and digital-stamp shipments, and it's harmless
	// to include otherwise. Nil when it can't be computed (a line item has
	// no SKU on file, or that SKU has no weight recorded) -- sending a
	// wrong/partial total would be worse than sending none.
	WeightGrams *int
}

// ShipmentResult is what MyParcel returns for a created/fetched shipment.
type ShipmentResult struct {
	ShipmentID int64
	Barcode    string
}

// OrderLineInput is one line item on the exported order -- MyParcel's
// Orders API requires this array (see infrastructure/myparcel's package doc
// comment for the confirmed field names and the unconfirmed minor-units
// assumption on the price fields).
type OrderLineInput struct {
	Quantity                int
	SKU                     string
	Name                    string
	PriceMinorUnits         int
	VATMinorUnits           int
	PriceAfterVATMinorUnits int
}

// CreateOrderInput is what ExportMyParcelOrderUseCase needs to export a
// sales order via MyParcel's Orders/Fulfilment API (POST
// /fulfilment/orders) -- a separate, real MyParcel endpoint from
// CreateShipmentInput's /shipments, used for the "export order, let staff
// confirm the address, then create the shipment" flow (see
// ExportMyParcelOrderUseCase's doc). AccountID/ShopID are the tenant's own
// MyParcel merchant identifiers (domain/shipping.ShippingConnectionSettings),
// required by this endpoint specifically -- Shipments API doesn't need them.
// PriceMinorUnits/VATMinorUnits/PriceAfterVATMinorUnits and OrderLines are
// also required by this endpoint (see infrastructure/myparcel's package doc
// comment) -- ExportMyParcelOrderUseCase computes them from the SalesOrder's
// own totals and line items.
type CreateOrderInput struct {
	ExternalIdentifier string
	OrderDate          time.Time
	AccountID          int
	ShopID             int
	Carrier            int
	Recipient          MyParcelAddress
	PackageType        int

	PriceMinorUnits         int
	VATMinorUnits           int
	PriceAfterVATMinorUnits int
	OrderLines              []OrderLineInput
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

// WebhookSubscription is one currently-registered MyParcel webhook
// subscription (GET /webhook_subscriptions/) -- ID is MyParcel's own
// subscription id (for DeleteWebhookSubscription), Hook is the event type
// (e.g. "shipment_status_change"), URL is the callback address MyParcel
// currently has on file for it. See infrastructure/myparcel's package doc
// comment: the exact response envelope shape for this list wasn't
// confirmed against a real sandbox during this build.
type WebhookSubscription struct {
	ID   string
	Hook string
	URL  string
}

// MyParcelClient is one MyParcel connection's credentials + endpoints,
// adapted to exactly what this codebase's use cases need (ISP). Implemented
// by infrastructure/myparcel.
type MyParcelClient interface {
	CreateShipment(ctx context.Context, in CreateShipmentInput) (ShipmentResult, error)
	// DeleteShipment cancels a shipment on MyParcel's side. Per MyParcel's
	// own docs this only succeeds pre-label-creation (status 1, "pending -
	// concept") -- see infrastructure/myparcel.Client.DeleteShipment's doc
	// comment. Callers must treat a rejection as an informative outcome, not
	// necessarily an application error.
	DeleteShipment(ctx context.Context, myParcelShipmentID int64) error
	// CreateOrder exports an order via MyParcel's Orders/Fulfilment API --
	// see CreateOrderInput's doc for how this differs from CreateShipment.
	CreateOrder(ctx context.Context, in CreateOrderInput) (OrderResult, error)
	// GetLabel fetches a shipment's label. format is "pdf" or "zpl" (see
	// domain/shipping.LabelFormatPDF/LabelFormatZPL) -- the response content
	// type, fixed at shipment-creation time. size is the physical page
	// format for THIS fetch (see domain/shipping.LabelSizeA4/LabelSizeA6) --
	// a retrieval-time choice, re-askable on every print.
	GetLabel(ctx context.Context, myParcelShipmentID int64, format, size string) (LabelResult, error)
	GetTrackTrace(ctx context.Context, myParcelShipmentID int64) (TrackTraceResult, error)
	// RegisterWebhook registers (or, per MyParcel's own "adding the same
	// webhook overwrites any existing one" behavior, re-registers) this
	// shop's shipment_status_change callback URL. Safe to call more than
	// once for the same connection. Implemented as a thin wrapper over
	// CreateWebhookSubscription (hook fixed to "shipment_status_change") --
	// kept as its own method rather than folded away so the existing
	// shipping-setup call site (application/shipping.RegisterMyParcelWebhookUseCase)
	// doesn't need to know MyParcel supports other hook types too.
	RegisterWebhook(ctx context.Context, callbackURL string) error
	// ListWebhookSubscriptions lists every webhook subscription currently
	// registered for this shop/API key, across all hook types -- used
	// before creating a new one (MyParcel allows only one subscription per
	// hook type per shop; whatever URL is about to be overwritten can be
	// surfaced to the admin first) and by the provider-webhooks admin UI's
	// read side.
	ListWebhookSubscriptions(ctx context.Context) ([]WebhookSubscription, error)
	// CreateWebhookSubscription registers hook (any of MyParcel's own hook
	// type names, e.g. "shipment_status_change", "shipment_label_created",
	// "order_status_change" -- not validated against an enum here, MyParcel
	// itself rejects an unknown one) pointed at deliveryURL. Silently
	// overwrites any existing subscription of the same hook type (MyParcel's
	// own behavior, see RegisterWebhook's doc comment).
	CreateWebhookSubscription(ctx context.Context, hook, deliveryURL string) (WebhookSubscription, error)
	// DeleteWebhookSubscription removes one subscription by MyParcel's own id.
	DeleteWebhookSubscription(ctx context.Context, id string) error
}

// MyParcelClientFactory builds a MyParcelClient scoped to one connection's
// API key.
type MyParcelClientFactory interface {
	NewClient(apiKey string) (MyParcelClient, error)
}
