package integrations

import "context"

type ConnectionRepository interface {
	Create(ctx context.Context, c *Connection) error
	FindByID(ctx context.Context, id string) (*Connection, error)
	ListByTenant(ctx context.Context) ([]Connection, error)
	Update(ctx context.Context, c *Connection) error
	// FindDisconnected returns the tenant's most recently disconnected
	// connection to provider at storeURL (compared without case or a
	// trailing slash), or ErrConnectionNotFound. FindByID and ListByTenant
	// never return a disconnected connection.
	FindDisconnected(ctx context.Context, provider Provider, storeURL StoreURL) (*Connection, error)
}

// AIConnectionRepository mirrors settings.Repository's shape (Get/Upsert,
// no explicit tenant ID -- RLS + ctx scope every call to the current
// tenant) since AIConnection is likewise unique per tenant: one row, always
// upserted in place, never a second row per provider switch.
type AIConnectionRepository interface {
	// Get returns nil, nil (not an error) when the tenant hasn't configured
	// an AI provider yet -- that's the common/default state, not a fault.
	Get(ctx context.Context) (*AIConnection, error)
	Upsert(ctx context.Context, c *AIConnection) error
	Delete(ctx context.Context) error
}

// SalesOrderOriginRepository stores the one origin row per sales order
// (COM-15). Every Find returns ErrSalesOrderOriginNotFound when no row
// matches.
type SalesOrderOriginRepository interface {
	// Save writes the order's origin; saving again for the same order
	// replaces it (one row per order).
	Save(ctx context.Context, o *SalesOrderOrigin) error
	FindBySalesOrderID(ctx context.Context, salesOrderID string) (*SalesOrderOrigin, error)
	// FindByConnectionExternalID finds a shop order by the connection it
	// came from and that shop's own order id -- the re-import lookup.
	FindByConnectionExternalID(ctx context.Context, connectionID, externalOrderID string) (*SalesOrderOrigin, error)
	// FindByOriginExternalID finds an order from a system without a
	// connection row (POS) by that system's own id.
	FindByOriginExternalID(ctx context.Context, origin Origin, externalOrderID string) (*SalesOrderOrigin, error)
}
