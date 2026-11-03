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
