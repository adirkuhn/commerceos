package shipping

import (
	"context"
	"time"
)

// ShipmentRepository persists Shipment rows. Implemented by
// infrastructure/repositories.
type ShipmentRepository interface {
	Create(ctx context.Context, s *Shipment) error
	FindByID(ctx context.Context, id string) (*Shipment, error)
	// FindByExternalShipmentID correlates an inbound webhook's shipment id
	// (converted to string at the caller's boundary -- see
	// domain/shipping.Shipment.ExternalShipmentID's doc) back to our own
	// Shipment row. Returns ErrShipmentNotFound (not a generic error) when
	// none matches.
	FindByExternalShipmentID(ctx context.Context, externalShipmentID string) (*Shipment, error)
	// FindBySalesOrderID returns every shipment created for a SalesOrder,
	// newest first -- normally one, but nothing stops staff from creating a
	// second after a failed/lost first attempt.
	FindBySalesOrderID(ctx context.Context, salesOrderID string) ([]Shipment, error)
	// FindStaleNonTerminal returns every non-terminal shipment whose
	// updated_at is older than olderThan -- ReconcileShipmentsUseCase's
	// "missed webhook" scan.
	FindStaleNonTerminal(ctx context.Context, olderThan time.Time, limit int) ([]Shipment, error)
	// trackingURL is MyParcel's `link_consumer_portal` for this status
	// update -- persisted alongside status so it's available later without
	// a live MyParcel call (see Shipment.TrackingURL's doc). Empty is a
	// valid value (MyParcel doesn't always have one yet), and overwrites
	// whatever was there before -- same "latest known value" convention as
	// status/statusMain.
	//
	// barcode is different: an empty barcode here means "the caller has no
	// barcode to offer" (only the shipment_status_change webhook payload
	// carries one -- MyParcel's tracktrace GET doesn't), not "MyParcel says
	// there isn't one" -- so unlike trackingURL, an empty barcode must NOT
	// overwrite an already-known one. Non-empty always overwrites.
	UpdateStatus(ctx context.Context, id string, status int, statusMain, trackingURL, barcode string) error
	// MarkCancelled soft-cancels a shipment -- see Shipment.CancelledAt's
	// doc comment. cancelledAt/note are set unconditionally by the caller
	// (CancelShipmentUseCase), not derived here.
	MarkCancelled(ctx context.Context, id string, cancelledAt time.Time, outcome CancelOutcome) error
}

// ShipmentStatusEventRepository appends to and reads back a Shipment's
// status history.
type ShipmentStatusEventRepository interface {
	Append(ctx context.Context, e *ShipmentStatusEvent) error
	// ListByShipmentID returns every event for one shipment, newest first --
	// the Sales Order detail page's tracking timeline.
	ListByShipmentID(ctx context.Context, shipmentID string) ([]ShipmentStatusEvent, error)
}

// ShippingConnectionSettingsRepository persists one MyParcel connection's
// global defaults.
type ShippingConnectionSettingsRepository interface {
	// FindByConnectionID returns nil, nil (not an error) when no settings
	// row exists yet -- the common state right after connecting, before the
	// merchant has configured anything; callers fall back to hardcoded
	// package-level defaults (see ResolveShippingOptionsUseCase).
	FindByConnectionID(ctx context.Context, connectionID string) (*ShippingConnectionSettings, error)
	Upsert(ctx context.Context, s *ShippingConnectionSettings) error
}

// ShippingCarrierOverrideRepository persists per-carrier override rows for
// one connection.
type ShippingCarrierOverrideRepository interface {
	ListByConnection(ctx context.Context, connectionID string) ([]ShippingCarrierOverride, error)
	FindByConnectionAndCarrier(ctx context.Context, connectionID string, carrier Carrier) (*ShippingCarrierOverride, error)
	Upsert(ctx context.Context, o *ShippingCarrierOverride) error
}
