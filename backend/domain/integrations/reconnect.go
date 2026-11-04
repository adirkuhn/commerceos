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
	FindByStore(ctx context.Context, provider Provider, storeURL StoreURL) (*Connection, error)
	Create(ctx context.Context, c *Connection) error
	Update(ctx context.Context, c *Connection) error
}

// SaveConnection stores a freshly connected c. The tenant's row for that
// provider and store is the store's identity: whatever its status, it is
// taken back -- same id, new credentials, its other settings kept -- so
// everything that points at it keeps working. Rotated keys are applied by
// connecting again. Otherwise c is created. reconnected reports which
// happened; c holds the saved connection in both cases.
func SaveConnection(ctx context.Context, repo connectionSaver, c *Connection) (reconnected bool, err error) {
	previous, err := repo.FindByStore(ctx, c.Provider, c.StoreURL)
	switch {
	case errors.Is(err, ErrConnectionNotFound):
		return false, repo.Create(ctx, c)
	case err != nil:
		return false, err
	}
	c.takeOver(*previous)
	return true, repo.Update(ctx, c)
}

// takeOver makes c the continuation of previous: its id and creation date,
// and every setting a connect form doesn't carry. Update writes every
// column, so without this a key change would blank them. What the form did
// supply wins. PostNL's account fields are always on its form, sandbox
// included, so they are not listed here.
func (c *Connection) takeOver(previous Connection) {
	c.ID, c.CreatedAt = previous.ID, previous.CreatedAt
	if c.WebhookSecretEncrypted == "" {
		c.WebhookSecretEncrypted = previous.WebhookSecretEncrypted
	}
	if c.DefaultCurrency == "" {
		c.DefaultCurrency = previous.DefaultCurrency
	}
	if c.DefaultWarehouseID == nil {
		c.DefaultWarehouseID = previous.DefaultWarehouseID
	}
	if c.BolEconomicOperatorID == nil {
		c.BolEconomicOperatorID = previous.BolEconomicOperatorID
	}
	if c.BolDeliveryCode == nil {
		c.BolDeliveryCode = previous.BolDeliveryCode
	}
	if c.BolShippingCostMinorUnits == 0 {
		c.BolShippingCostMinorUnits = previous.BolShippingCostMinorUnits
	}
	if c.SMTPPort == nil {
		c.SMTPPort = previous.SMTPPort
	}
	if c.EmailFrom == nil {
		c.EmailFrom = previous.EmailFrom
	}
	c.WhatsAppTestModeEnabled = previous.WhatsAppTestModeEnabled
	c.WhatsAppTestPhoneE164 = previous.WhatsAppTestPhoneE164
	c.WhatsAppAIParsingEnabled = previous.WhatsAppAIParsingEnabled
}
