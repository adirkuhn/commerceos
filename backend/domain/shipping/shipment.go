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

// StatusEventSource distinguishes a status learned from an inbound webhook
// from one learned by ReconcileShipmentsUseCase's own poll -- both write the
// same ShipmentStatusEvent shape, this just says how we found out.
type StatusEventSource string

const (
	StatusEventSourceWebhook        StatusEventSource = "webhook"
	StatusEventSourceReconciliation StatusEventSource = "reconciliation"
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

	CreatedAt time.Time
	UpdatedAt time.Time
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
	ConnectionID             string
	TenantID                 string
	DefaultCarrier           Carrier
	DefaultPackageType       PackageType
	DefaultLabelFormat       string
	DefaultSignatureRequired bool
	DefaultOnlyRecipient     bool
	CreatedAt                time.Time
	UpdatedAt                time.Time
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
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
