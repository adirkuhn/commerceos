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
	// FindByMyParcelShipmentID correlates an inbound webhook's shipment_id
	// back to our own Shipment row. Returns ErrShipmentNotFound (not a
	// generic error) when none matches.
	FindByMyParcelShipmentID(ctx context.Context, myParcelShipmentID int64) (*Shipment, error)
	// FindBySalesOrderID returns every shipment created for a SalesOrder,
	// newest first -- normally one, but nothing stops staff from creating a
	// second after a failed/lost first attempt.
	FindBySalesOrderID(ctx context.Context, salesOrderID string) ([]Shipment, error)
	// FindStaleNonTerminal returns every non-terminal shipment whose
	// updated_at is older than olderThan -- ReconcileShipmentsUseCase's
	// "missed webhook" scan.
	FindStaleNonTerminal(ctx context.Context, olderThan time.Time, limit int) ([]Shipment, error)
	UpdateStatus(ctx context.Context, id string, status int, statusMain string) error
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
