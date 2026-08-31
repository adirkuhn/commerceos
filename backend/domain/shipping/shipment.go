// Package shipping is the MyParcel shipping-label bounded context: one
// Shipment per SalesOrder shipped through a MyParcel connection, its status
// history, and per-connection carrier configuration. Mirrors domain/channels'
// shape (a SalesChannel/Connection pattern) for a fulfillment concern
// instead of a sales one.
package shipping

import (
	"errors"
	"time"
)

// Carrier is MyParcel's own carrier id -- see
// https://developer.myparcel.nl/api-reference/06.shipments.html.
type Carrier int

const (
	CarrierPostNL           Carrier = 1
	CarrierBpost            Carrier = 2
	CarrierDPD              Carrier = 4
	CarrierDHLForYou        Carrier = 9
	CarrierDHLParcelConnect Carrier = 10
	CarrierDHLEuroplus      Carrier = 11
	CarrierUPSStandard      Carrier = 12
	CarrierUPSExpressSaver  Carrier = 13
)

// PackageType is MyParcel's shipment_options.package_type.
type PackageType int

const (
	PackageTypePackage        PackageType = 1
	PackageTypeMailboxPackage PackageType = 2
	PackageTypeDigitalStamp   PackageType = 4
)

const (
	LabelFormatPDF = "pdf"
	LabelFormatZPL = "zpl"
)

// LabelSizeA4/LabelSizeA6 are the physical page format requested at label
// *retrieval* time (GetShipmentLabelUseCase), not creation time -- unlike
// LabelFormat (pdf/zpl content type, fixed when the shipment is created),
// staff can ask for a different physical size on each print/reprint of the
// same label. The query param this actually sends to MyParcel's real
// shipment_labels endpoint ("format") is NOT confirmed against a live
// sandbox response -- same caveat as the two already flagged in
// infrastructure/myparcel's package doc comment (auth header,
// reference_identifier field name).
const (
	LabelSizeA4 = "A4"
	LabelSizeA6 = "A6"
)

// StatusEventSource distinguishes a status learned from an inbound webhook
// from one learned by ReconcileShipmentsUseCase's own poll -- both write the
// same ShipmentStatusEvent shape, this just says how we found out.
type StatusEventSource string

const (
	StatusEventSourceWebhook        StatusEventSource = "webhook"
	StatusEventSourceReconciliation StatusEventSource = "reconciliation"
	// StatusEventSourceManual is a staff-triggered "refresh from provider"
	// (RefreshShipmentTrackingUseCase) -- same applyStatusUpdate path as the
	// other two sources, just told to check right now instead of waiting for
	// a webhook or the next reconciliation tick.
	StatusEventSourceManual StatusEventSource = "manual"
	// StatusEventSourceBackfill is BackfillShipmentWebhookDispatchUseCase --
	// a one-time reprocessing of webhooks received before dispatch-status
	// tracking existed, so their outcome is visible instead of stuck
	// showing "received" forever. Distinct from Manual so the timeline
	// itself shows which entries came from that one-off run.
	StatusEventSourceBackfill StatusEventSource = "backfill"
)

// StatusMainShipped is the MyParcel status.main value this codebase treats
// as "handed to carrier" -- the only shipment-event effect v1 applies to a
// SalesOrder (Fulfilled=true), per the approved design's conservative
// scope. MyParcel's exact set of status.main string values wasn't fully
// enumerated in what was fetched from the docs this session; "shipped" is
// MyParcel's own terminology for this transition (mirrors "registered" for
// creation, "delivered" for the final state) -- verify against a real
// account's webhook payloads and adjust if MyParcel actually sends a
// different string for this transition.
const StatusMainShipped = "shipped"

var (
	ErrShipmentNotFound    = errors.New("shipping: shipment not found")
	ErrSettingsNotFound    = errors.New("shipping: connection settings not found")
	ErrMissingRecipient    = errors.New("shipping: recipient address is incomplete")
	ErrWebhookSecretNotSet = errors.New("shipping: connection has no webhook secret set yet")
)

// Shipment is one MyParcel shipment created for a SalesOrder. Recipient_*
// fields are a confirmed, MyParcel-shaped snapshot taken at label-creation
// time -- deliberately NOT derived live from a SalesOrder's own
// Shipping/Address (which is Woo-shaped free text with no street/number
// split) on every read; staff review/edit them once in the address-confirm
// step (frontend), and this is what actually gets submitted to MyParcel and
// printed on the label.
type Shipment struct {
	ID                 string
	TenantID           string
	SalesOrderID       string
	ConnectionID       string
	Carrier            Carrier
	MyParcelShipmentID int64
	Barcode            string
	// Status/StatusMain hold only the latest known state -- see
	// ShipmentStatusEvent for the full history.
	Status      int
	StatusMain  string
	LabelFormat string

	PackageType             PackageType
	SignatureRequired       bool
	OnlyRecipient           bool
	InsuranceAmountMinorEUR *int

	RecipientPerson       string
	RecipientCompany      string
	RecipientStreet       string
	RecipientNumber       string
	RecipientNumberSuffix string
	RecipientPostalCode   string
	RecipientCity         string
	RecipientCC           string
	RecipientEmail        string
	RecipientPhone        string

	// CancelledAt is set by CancelShipmentUseCase once staff have tried to
	// delete this shipment (e.g. wrong carrier picked) -- non-nil marks the
	// shipment as no longer "active" for CreateShipmentUseCase's own
	// existing-shipment guard, so a corrected new shipment can be created
	// for the same SalesOrder, while this row (and its status history)
	// stays around for audit instead of being hard-deleted. CancelNote
	// records the outcome of that attempt -- whether MyParcel itself
	// actually deleted the shipment or rejected the request (MyParcel only
	// allows deletion before a label exists, see
	// infrastructure/myparcel.Client.DeleteShipment's doc comment), so staff
	// reviewing history later know whether a wrong-carrier label might
	// still be live on MyParcel's own side.
	CancelledAt *time.Time
	CancelNote  string

	CreatedAt time.Time
	UpdatedAt time.Time
}

// IsActive reports whether this shipment still counts as the current
// shipment for its SalesOrder -- false once CancelShipmentUseCase has
// marked it cancelled, at which point a new shipment may be created for the
// same order/connection.
func (s Shipment) IsActive() bool {
	return s.CancelledAt == nil
}

// IsTerminal reports whether this shipment's status.main means no further
// status changes are expected -- used by ReconcileShipmentsUseCase to skip
// already-finished shipments. "delivered" is the only terminal value
// confirmed from what was fetched; MyParcel's docs mention returns/failed
// deliveries exist too but their exact status_main strings weren't
// captured -- verify and extend this against a real account if a shipment
// ever gets stuck being re-polled forever.
func (s Shipment) IsTerminal() bool {
	return s.StatusMain == "delivered"
}

// ShipmentStatusEvent is an append-only history row for every status
// MyParcel has reported for a Shipment, whether learned via webhook or
// ReconcileShipmentsUseCase's poll -- Shipment.Status/StatusMain only ever
// holds the latest.
type ShipmentStatusEvent struct {
	ID         string
	TenantID   string
	ShipmentID string
	Status     int
	StatusMain string
	Source     StatusEventSource
	OccurredAt time.Time
}

// ShippingConnectionSettings holds one MyParcel connection's global
// shipping defaults -- "configure general configuration and settings
// global" from the approved design. One row per connection.
type ShippingConnectionSettings struct {
	ConnectionID       string
	TenantID           string
	DefaultCarrier     Carrier
	DefaultPackageType PackageType
	DefaultLabelFormat string
	// DefaultLabelSize (LabelSizeA4/LabelSizeA6) is GetShipmentLabelUseCase's
	// fallback physical page format when a print request doesn't ask for a
	// specific one explicitly -- unlike DefaultLabelFormat, this isn't
	// stored on the Shipment itself (see LabelSizeA4's doc comment).
	DefaultLabelSize         string
	DefaultSignatureRequired bool
	DefaultOnlyRecipient     bool
	// InsureByDefault turns on insurance for every shipment on this
	// connection unless a carrier override says otherwise -- the actual
	// insured amount is never configured here, it's computed at
	// shipment-creation time from that specific SalesOrder's own value
	// (CreateShipmentUseCase), so a fixed cents figure can't go stale
	// against what's actually being shipped. See ResolveOptions for the
	// mandatory-signature/only_recipient rule MyParcel imposes whenever a
	// shipment ends up insured.
	InsureByDefault bool
	// MyParcelAccountID/MyParcelShopID are the tenant's own MyParcel
	// merchant account/shop identifiers -- required by MyParcel's
	// Orders/Fulfilment API (ExportMyParcelOrderUseCase), not secrets, so
	// unlike the API key they're plain columns here rather than encrypted.
	MyParcelAccountID *int
	MyParcelShopID    *int
	// AutoPrintEnabled: once a shipment/label is created, auto-open the
	// print view (window.print()) instead of requiring a manual "Print"
	// click. Not silent printing -- the OS print dialog still appears --
	// see MyParcelSyncSettingsPage's doc for why (cloud-hosted backend,
	// LAN-local printer, no print-agent infra).
	AutoPrintEnabled bool
	// ExportOrderToMyParcel, when true, makes CreateShipmentUseCase also
	// register the sales order via MyParcel's separate Orders/Fulfilment API
	// (ExportMyParcelOrderUseCase) -- the only thing that makes it show up
	// on MyParcel's own Orders dashboard (backoffice.myparcel.nl/orders),
	// which the plain Shipments API this codebase uses by default never
	// populates. Off by default: exporting the full order (line items,
	// prices) hands MyParcel data/insights the merchant may not want to
	// share when they only need a label.
	ExportOrderToMyParcel bool
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// ShippingCarrierOverride overrides ShippingConnectionSettings' defaults for
// one specific carrier on one connection -- "and per carrier" from the
// approved design. A nil field on this struct means "fall back to the
// connection's global default for that field," not "false"/"zero" -- see
// ResolveShippingOptionsUseCase for how the three layers (this override,
// the connection settings, MyParcel's own defaults) combine.
type ShippingCarrierOverride struct {
	ID                string
	TenantID          string
	ConnectionID      string
	Carrier           Carrier
	PackageType       *PackageType
	SignatureRequired *bool
	OnlyRecipient     *bool
	// InsureByDefault nil means "fall back to the connection's InsureByDefault" --
	// same convention as PackageType/SignatureRequired/OnlyRecipient. A
	// non-nil value (true or false) always wins over the connection default.
	InsureByDefault *bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
