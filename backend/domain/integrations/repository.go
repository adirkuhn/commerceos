package integrations

import "context"

type ConnectionRepository interface {
	Create(ctx context.Context, c *Connection) error
	FindByID(ctx context.Context, id string) (*Connection, error)
	ListByTenant(ctx context.Context) ([]Connection, error)
	Update(ctx context.Context, c *Connection) error
	// FindByStore returns the tenant's connection to provider at storeURL
	// (compared without scheme, www., case or a trailing slash) whatever
	// its status, a live one first, or ErrConnectionNotFound. It is the only lookup that
	// sees a disconnected connection: FindByID and ListByTenant never do.
	FindByStore(ctx context.Context, provider Provider, storeURL StoreURL) (*Connection, error)
}

// AIConnectionRepository stores the tenant's ordered AI connections (RLS
// and ctx scope every call to the current tenant).
type AIConnectionRepository interface {
	// List returns every connection in routing order (Position ascending),
	// enabled or not; empty when none is configured.
	List(ctx context.Context) ([]AIConnection, error)
	// Get returns ErrAIConnectionNotFound for an unknown id.
	Get(ctx context.Context, id string) (*AIConnection, error)
	Create(ctx context.Context, c *AIConnection) error
	Update(ctx context.Context, c *AIConnection) error
	Delete(ctx context.Context, id string) error
}

// SalesOrderOriginRepository stores the one origin row per sales order
// (COM-15). Every Find returns ErrSalesOrderOriginNotFound when no row
// matches.
type SalesOrderOriginRepository interface {
	// Save writes the order's origin; saving again for the same order
	// replaces it (one row per order).
	Save(ctx context.Context, o *SalesOrderOrigin) error
	FindBySalesOrderID(ctx context.Context, salesOrderID string) (*SalesOrderOrigin, error)
	// ListBySalesOrderIDs returns the origins of those orders (any order
	// without one is simply absent); empty ids is an empty result.
	ListBySalesOrderIDs(ctx context.Context, salesOrderIDs []string) ([]SalesOrderOrigin, error)
	// FindByConnectionExternalID finds a shop order by the connection it
	// came from and that shop's own order id -- the re-import lookup.
	FindByConnectionExternalID(ctx context.Context, connectionID, externalOrderID string) (*SalesOrderOrigin, error)
	// FindByOriginExternalID finds an order from a system without a
	// connection row (POS) by that system's own id.
	FindByOriginExternalID(ctx context.Context, origin Origin, externalOrderID string) (*SalesOrderOrigin, error)
}
