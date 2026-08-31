package shipping

import (
	"context"

	"github.com/adirkuhn/commerceos/domain/webhooks"
)

// myParcelWebhookHandler is the narrow Execute(ctx, envelope) error shape
// both HandleShipmentWebhookUseCase and HandleOrderStatusWebhookUseCase
// satisfy -- lets MyParcelWebhookRouter hold either without depending on
// their concrete types.
type myParcelWebhookHandler interface {
	Execute(ctx context.Context, envelope webhooks.Envelope) error
}

// MyParcelWebhookRouter dispatches an inbound myparcel webhook envelope to
// the handler for its own hook type (the X-Myparcel-Hook header) instead of
// one handler running -- and mis-decoding -- every myparcel topic
// unconditionally, which is what silently failed order_status_change
// webhooks before this router existed (HandleShipmentWebhookUseCase's
// int-typed Status field chokes decoding order_status_change's own string
// status). Extend via a new field + case here for a future topic, rather
// than branching inside either handler. A nil handler for a topic (or an
// unrecognized one) is a no-op -- same "every other myparcel hook type
// stays received, nothing more was ever going to happen to it" convention
// as every other provider gets by default.
type MyParcelWebhookRouter struct {
	ShipmentHandler    myParcelWebhookHandler
	OrderStatusHandler myParcelWebhookHandler
}

func NewMyParcelWebhookRouter(shipmentHandler, orderStatusHandler myParcelWebhookHandler) *MyParcelWebhookRouter {
	return &MyParcelWebhookRouter{ShipmentHandler: shipmentHandler, OrderStatusHandler: orderStatusHandler}
}

func (r *MyParcelWebhookRouter) Execute(ctx context.Context, envelope webhooks.Envelope) error {
	switch envelope.Headers["X-Myparcel-Hook"] {
	case "shipment_status_change":
		if r.ShipmentHandler == nil {
			return nil
		}
		return r.ShipmentHandler.Execute(ctx, envelope)
	case "order_status_change":
		if r.OrderStatusHandler == nil {
			return nil
		}
		return r.OrderStatusHandler.Execute(ctx, envelope)
	default:
		return nil
	}
}
