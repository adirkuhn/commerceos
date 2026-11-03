package oms

import (
	"context"
	"fmt"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
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

// originFinder reads an order's origin -- the rules that depend on where an
// order came from (POS sales are never picked or packed, only orders made
// here are edited) receive it as their source.
type originFinder interface {
	Of(ctx context.Context, salesOrderID string) (*integrationsDomain.SalesOrderOrigin, error)
}

// orderKeySetter changes the order key kept on an order's origin.
type orderKeySetter interface {
	originFinder
	SetExternalKey(ctx context.Context, salesOrderID, key string) error
}

// sourceOf is the order's origin as the oms rules' source.
func sourceOf(ctx context.Context, origins originFinder, salesOrderID string) (omsDomain.SalesOrderSource, error) {
	o, err := origins.Of(ctx, salesOrderID)
	if err != nil {
		return "", fmt.Errorf("order %s: origin: %w", salesOrderID, err)
	}
	return omsDomain.SalesOrderSource(o.Origin), nil
}
