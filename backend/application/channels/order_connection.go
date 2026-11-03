package channels

import (
	"context"
	"errors"
	"fmt"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

type salesChannelFinder interface {
	FindByID(ctx context.Context, id string) (*channelsDomain.SalesChannel, error)
}

// orderConnectionResolver is the port the outbound order paths (Woo status
// push, Bol shipment confirmation, Woo status repush) use to find where an
// order goes.
type orderConnectionResolver interface {
	ConnectionIDFor(ctx context.Context, order omsDomain.SalesOrder) (string, error)
}

// OrderConnectionResolver answers "which connection does this order go to
// now" (COM-15): the order's sales channel decides, at send time, so
// reconnecting or rotating an integration never needs the order touched.
// Orders imported before they carried a sales channel fall back to the
// connection recorded on them. Nothing to resolve is
// ErrOrderChannelUnresolved -- never a silent skip, so the event is retried
// and then dead-lettered. Runs inside the caller's tenant.WithTenant.
type OrderConnectionResolver struct {
	channels salesChannelFinder
}

func NewOrderConnectionResolver(channels salesChannelFinder) *OrderConnectionResolver {
	return &OrderConnectionResolver{channels: channels}
}

func (r *OrderConnectionResolver) ConnectionIDFor(ctx context.Context, order omsDomain.SalesOrder) (string, error) {
	if order.SalesChannelID != nil {
		ch, err := r.channels.FindByID(ctx, *order.SalesChannelID)
		switch {
		case err == nil && ch.ConnectionID != nil:
			return *ch.ConnectionID, nil
		case err != nil && !errors.Is(err, channelsDomain.ErrSalesChannelNotFound):
			return "", fmt.Errorf("order %s: sales channel %s: %w", order.ID, *order.SalesChannelID, err)
		}
	}
	if order.ConnectionID != nil {
		return *order.ConnectionID, nil
	}
	return "", fmt.Errorf("order %s: %w", order.ID, channelsDomain.ErrOrderChannelUnresolved)
}
