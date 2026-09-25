package consumers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// shipmentFulfillmentSignalApplier is the one method this consumer needs
// from application/shipping.ApplyShipmentFulfillmentSignalUseCase (ISP) --
// kept as an interface here so the consumer can be unit tested without a
// real DB.
type shipmentFulfillmentSignalApplier interface {
	Execute(ctx context.Context, salesOrderID, kind, statusMain string, statusCode int) error
}

const shipmentFulfillmentSignalConsumerName = "shipment-fulfillment-signal-apply"

// ShipmentFulfillmentSignalConsumer reacts to
// shipping.shipment.fulfillment_signal by deciding what it means for the
// SalesOrder it names (see ApplyShipmentFulfillmentSignalUseCase, "the
// event manager forwards it, the consumer decides if it's relevant").
// Idempotent via ProcessedEventRepository: JetStream is at-least-once
// delivery, so redelivery of an event already fully handled must be a safe
// no-op.
type ShipmentFulfillmentSignalConsumer struct {
	processed events.ProcessedEventRepository
	applier   shipmentFulfillmentSignalApplier
	dlq       dlqPublisher
}

func NewShipmentFulfillmentSignalConsumer(processed events.ProcessedEventRepository, applier shipmentFulfillmentSignalApplier, dlq dlqPublisher) *ShipmentFulfillmentSignalConsumer {
	return &ShipmentFulfillmentSignalConsumer{processed: processed, applier: applier, dlq: dlq}
}

// Handle is the NATS message handler -- wire it up via a JetStream durable
// subscription (nc.Subscribe with AckExplicit). It never panics or crashes
// the caller: failures are logged and the message is nacked so JetStream
// retries delivery; only a message that could never succeed (malformed
// payload) is acked-and-dropped.
func (c *ShipmentFulfillmentSignalConsumer) Handle(msg *nats.Msg) {
	var event events.ShipmentFulfillmentSignal
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("shipmentFulfillmentSignalConsumer: invalid payload, dropping (not retryable): %v", err)
		ackOrLog(msg, "shipmentFulfillmentSignalConsumer")
		return
	}

	ctx := tenant.WithTenantID(context.Background(), event.TenantID)

	alreadyDone, err := c.processed.IsProcessed(ctx, event.EventID, shipmentFulfillmentSignalConsumerName)
	if err != nil {
		nakOrDeadLetter(c.dlq, msg, "shipmentFulfillmentSignalConsumer: check processed "+event.EventID, err)
		return
	}
	if alreadyDone {
		ackOrLog(msg, "shipmentFulfillmentSignalConsumer")
		return
	}

	if err := c.applier.Execute(ctx, event.SalesOrderID, event.Kind, event.StatusMain, event.StatusCode); err != nil {
		nakOrDeadLetter(c.dlq, msg, "shipmentFulfillmentSignalConsumer: apply "+event.SalesOrderID, err)
		return
	}

	// Marked *after* the apply succeeds -- see ProcessedEventRepository's
	// doc comment for why the order matters.
	if err := c.processed.MarkProcessed(ctx, event.EventID, shipmentFulfillmentSignalConsumerName); err != nil {
		nakOrDeadLetter(c.dlq, msg, "shipmentFulfillmentSignalConsumer: mark processed "+event.EventID, err)
		return
	}

	ackOrLog(msg, "shipmentFulfillmentSignalConsumer")
}
