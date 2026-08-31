package consumers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// salesOrderStatusPusher is the one method this consumer needs from
// application/channels.PushSalesOrderStatusToChannelsUseCase (ISP) -- kept
// as an interface here so the consumer can be unit tested without a real
// DB or WooClient.
type salesOrderStatusPusher interface {
	Execute(ctx context.Context, salesOrderID string) error
}

const salesOrderStatusChangedConsumerName = "sales-order-status-changed-push"

// SalesOrderStatusChangedConsumer reacts to oms.sales_order.status_changed
// by pushing the order's current status to whatever channel it came from
// (see PushSalesOrderStatusToChannelsUseCase, "the channel decides what to
// do"). Idempotent via ProcessedEventRepository: JetStream is at-least-once
// delivery, so redelivery of an event already fully handled must be a safe
// no-op.
type SalesOrderStatusChangedConsumer struct {
	processed events.ProcessedEventRepository
	pusher    salesOrderStatusPusher
	dlq       dlqPublisher
}

func NewSalesOrderStatusChangedConsumer(processed events.ProcessedEventRepository, pusher salesOrderStatusPusher, dlq dlqPublisher) *SalesOrderStatusChangedConsumer {
	return &SalesOrderStatusChangedConsumer{processed: processed, pusher: pusher, dlq: dlq}
}

// Handle is the NATS message handler -- wire it up via a JetStream durable
// subscription (nc.QueueSubscribe/ChanSubscribe with AckExplicit). It never
// panics or crashes the caller: failures are logged and the message is
// nacked so JetStream retries delivery; only a message that could never
// succeed (malformed payload) is acked-and-dropped.
func (c *SalesOrderStatusChangedConsumer) Handle(msg *nats.Msg) {
	var event events.SalesOrderStatusChanged
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("salesOrderStatusChangedConsumer: invalid payload, dropping (not retryable): %v", err)
		_ = msg.Ack()
		return
	}

	ctx := tenant.WithTenantID(context.Background(), event.TenantID)

	alreadyDone, err := c.processed.IsProcessed(ctx, event.EventID, salesOrderStatusChangedConsumerName)
	if err != nil {
		nakOrDeadLetter(c.dlq, msg, "salesOrderStatusChangedConsumer: check processed "+event.EventID, err)
		return
	}
	if alreadyDone {
		_ = msg.Ack()
		return
	}

	if err := c.pusher.Execute(ctx, event.SalesOrderID); err != nil {
		nakOrDeadLetter(c.dlq, msg, "salesOrderStatusChangedConsumer: push order "+event.SalesOrderID, err)
		return
	}

	// Marked *after* the push succeeds -- see ProcessedEventRepository's doc
	// comment for why the order matters.
	if err := c.processed.MarkProcessed(ctx, event.EventID, salesOrderStatusChangedConsumerName); err != nil {
		// safe to retry: pusher.Execute re-pushes current state, so re-running it is harmless
		nakOrDeadLetter(c.dlq, msg, "salesOrderStatusChangedConsumer: mark processed "+event.EventID, err)
		return
	}

	_ = msg.Ack()
}
