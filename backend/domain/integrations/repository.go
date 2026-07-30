package integrations

import "context"

type ConnectionRepository interface {
	Create(ctx context.Context, c *Connection) error
	FindByID(ctx context.Context, id string) (*Connection, error)
	ListByTenant(ctx context.Context) ([]Connection, error)
	Update(ctx context.Context, c *Connection) error
	Delete(ctx context.Context, id string) error
}
