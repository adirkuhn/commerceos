package channels

import (
	"context"
	"errors"
	"fmt"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

// orderOrigins is application/integrations.SalesOrderOrigins narrowed to
// what the shop importers need (COM-15): an imported order's link to the
// shop is its origin row, found by connection + the shop's own order id.
type orderOrigins interface {
	Record(ctx context.Context, in integrationsDomain.SalesOrderOriginInput) error
	FindImported(ctx context.Context, origin integrationsDomain.Origin, connectionID, externalOrderID string) (*integrationsDomain.SalesOrderOrigin, error)
}

type salesOrderByID interface {
	FindByID(ctx context.Context, id string) (*omsDomain.SalesOrder, error)
}

// findImportedOrder returns the order already imported from this
// connection under externalOrderID, or nil when it never was. Runs inside
// the caller's tenant.WithTenant.
func findImportedOrder(ctx context.Context, origins orderOrigins, orders salesOrderByID, origin integrationsDomain.Origin, connectionID, externalOrderID string) (*omsDomain.SalesOrder, error) {
	o, err := origins.FindImported(ctx, origin, connectionID, externalOrderID)
	if errors.Is(err, integrationsDomain.ErrSalesOrderOriginNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return orders.FindByID(ctx, o.SalesOrderID)
}

// isDuplicateImport: another import of the same order won the create race,
// caught by its origin's unique key.
func isDuplicateImport(err error) bool {
	return errors.Is(err, integrationsDomain.ErrExternalOrderAlreadyImported)
}

// orderOriginReader is application/integrations.SalesOrderOrigins narrowed
// to what the outbound paths (Woo status push, Bol shipment confirmation,
// Woo status repush, payment backfill) need: where the order came from.
type orderOriginReader interface {
	Of(ctx context.Context, salesOrderID string) (*integrationsDomain.SalesOrderOrigin, error)
}

// shopOrderOf is the origin an outbound message for this shop order goes
// back to: its connection and the shop's own order id. Neither known is
// ErrOrderChannelUnresolved -- never a silent skip, so the event is retried
// and then dead-lettered (COM-15). Runs inside the caller's
// tenant.WithTenant.
func shopOrderOf(ctx context.Context, origins orderOriginReader, salesOrderID string) (*integrationsDomain.SalesOrderOrigin, error) {
	o, err := origins.Of(ctx, salesOrderID)
	if errors.Is(err, integrationsDomain.ErrSalesOrderOriginNotFound) {
		return nil, fmt.Errorf("order %s: %w", salesOrderID, channelsDomain.ErrOrderChannelUnresolved)
	}
	if err != nil {
		return nil, fmt.Errorf("order %s: origin: %w", salesOrderID, err)
	}
	if err := requireShopReference(*o); err != nil {
		return nil, err
	}
	return o, nil
}

// requireShopReference: the origin says which connection and shop order an
// outbound message goes to, or ErrOrderChannelUnresolved.
func requireShopReference(o integrationsDomain.SalesOrderOrigin) error {
	if o.ConnectionID == nil || o.ExternalOrderID == "" {
		return fmt.Errorf("order %s: %w", o.SalesOrderID, channelsDomain.ErrOrderChannelUnresolved)
	}
	return nil
}
