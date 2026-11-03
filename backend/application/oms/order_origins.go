package oms

import (
	"context"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

// originRecorder is application/integrations.SalesOrderOrigins narrowed to
// what creating an order needs (COM-15): the order's origin is written in
// the order's own transaction, the order itself holds no integration data.
type originRecorder interface {
	Record(ctx context.Context, in integrationsDomain.SalesOrderOriginInput) error
}

// posOrigins adds the POS idempotency lookup: a POS retry of a sale already
// received is found by the POS's own id.
type posOrigins interface {
	originRecorder
	FindByExternalID(ctx context.Context, origin integrationsDomain.Origin, externalOrderID string) (*integrationsDomain.SalesOrderOrigin, error)
}
