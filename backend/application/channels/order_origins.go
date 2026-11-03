package channels

import (
	"context"
	"errors"

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

// isDuplicateImport: another import of the same order won the create race
// -- caught by the order's own unique key or by its origin's.
func isDuplicateImport(err error) bool {
	return errors.Is(err, omsDomain.ErrDuplicateExternalOrder) || errors.Is(err, integrationsDomain.ErrExternalOrderAlreadyImported)
}
