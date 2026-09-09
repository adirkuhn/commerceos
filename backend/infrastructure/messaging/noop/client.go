// Package noop implements domain/messaging.Client for when no real
// provider is configured on a channel (e.g. SMTP_HOST unset).
package noop

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/messaging"
)

// Client is used when no provider is configured for a channel -- the
// storefront auth routes still exist (oapi-codegen wires the whole
// ServerInterface uniformly, so individual operations can't be
// conditionally unmounted the way api/pos's routes are), but every send
// fails loudly with a clear error rather than the app silently pretending
// messaging works.
type Client struct{}

var _ messaging.Client = Client{}

func (Client) Send(_ context.Context, msg messaging.Message) error {
	return fmt.Errorf("messaging: %s sending is not configured", msg.Channel)
}
