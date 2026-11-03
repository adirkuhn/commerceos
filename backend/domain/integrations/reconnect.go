package integrations

import (
	"context"
	"errors"
	"time"
)

// Disconnect keeps the connection row but drops every credential (COM-15).
// Deleting the row cascaded into sales channels, listings, schedules and
// category links and nulled sales_orders.connection_id; a disconnected row
// changes none of them.
func (c *Connection) Disconnect(now time.Time) {
	c.Status = StatusDisconnected
	c.ConsumerKey = ""
	c.ConsumerSecretEncrypted = ""
	c.WebhookSecretEncrypted = ""
	c.UpdatedAt = now
}

func (c Connection) IsDisconnected() bool { return c.Status == StatusDisconnected }

// connectionSaver is what SaveConnection needs from a ConnectionRepository.
type connectionSaver interface {
	FindDisconnected(ctx context.Context, provider Provider, storeURL StoreURL) (*Connection, error)
	Create(ctx context.Context, c *Connection) error
	Update(ctx context.Context, c *Connection) error
}

// SaveConnection stores a freshly connected c. When the tenant disconnected
// a connection to the same provider and store before, that row is taken
// back -- same id, new credentials -- so everything that pointed at it keeps
// working; otherwise c is created. reconnected reports which happened; c.ID
// holds the id in both cases.
func SaveConnection(ctx context.Context, repo connectionSaver, c *Connection) (reconnected bool, err error) {
	previous, err := repo.FindDisconnected(ctx, c.Provider, c.StoreURL)
	switch {
	case errors.Is(err, ErrConnectionNotFound):
		return false, repo.Create(ctx, c)
	case err != nil:
		return false, err
	}
	c.ID, c.CreatedAt = previous.ID, previous.CreatedAt
	return true, repo.Update(ctx, c)
}
