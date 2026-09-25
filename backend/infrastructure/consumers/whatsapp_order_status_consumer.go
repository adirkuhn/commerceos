package consumers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// orderStatusNotifier is application/whatsapp.NotifyOrderStatusUseCase
// narrowed to point of use (ISP) -- lets this consumer be unit tested
// without a real DB or Evolution API client.
type orderStatusNotifier interface {
	Execute(ctx context.Context, salesOrderID string) error
}

const whatsAppOrderStatusConsumerName = "whatsapp-order-status-notify"

// WhatsAppOrderStatusConsumer reacts to the same oms.sales_order.status_changed
// event as SalesOrderStatusChangedConsumer (see that file's doc comment) --
// a second, independent subscriber on the same subject, its own durable
// name and its own ProcessedEventRepository key, so this consumer's
// success/failure/redelivery never interacts with the channel-push
// consumer's. Sends the customer a WhatsApp status update (see
// NotifyOrderStatusUseCase for the "only if already a known customer with a
// verified phone" rule).
type WhatsAppOrderStatusConsumer struct {
	processed events.ProcessedEventRepository
	notifier  orderStatusNotifier
	dlq       dlqPublisher
}

func NewWhatsAppOrderStatusConsumer(processed events.ProcessedEventRepository, notifier orderStatusNotifier, dlq dlqPublisher) *WhatsAppOrderStatusConsumer {
	return &WhatsAppOrderStatusConsumer{processed: processed, notifier: notifier, dlq: dlq}
}

func (c *WhatsAppOrderStatusConsumer) Handle(msg *nats.Msg) {
	var event events.SalesOrderStatusChanged
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("whatsAppOrderStatusConsumer: invalid payload, dropping (not retryable): %v", err)
		ackOrLog(msg, "whatsAppOrderStatusConsumer")
		return
	}

	ctx := tenant.WithTenantID(context.Background(), event.TenantID)

	alreadyDone, err := c.processed.IsProcessed(ctx, event.EventID, whatsAppOrderStatusConsumerName)
	if err != nil {
		nakOrDeadLetter(c.dlq, msg, "whatsAppOrderStatusConsumer: check processed "+event.EventID, err)
		return
	}
	if alreadyDone {
		ackOrLog(msg, "whatsAppOrderStatusConsumer")
		return
	}

	if err := c.notifier.Execute(ctx, event.SalesOrderID); err != nil {
		nakOrDeadLetter(c.dlq, msg, "whatsAppOrderStatusConsumer: notify order "+event.SalesOrderID, err)
		return
	}

	if err := c.processed.MarkProcessed(ctx, event.EventID, whatsAppOrderStatusConsumerName); err != nil {
		// safe to retry: notifier.Execute re-sends the same message on
		// redelivery -- an occasional duplicate WhatsApp notification is a
		// much smaller harm than silently dropping one, same tradeoff every
		// other consumer in this package makes.
		nakOrDeadLetter(c.dlq, msg, "whatsAppOrderStatusConsumer: mark processed "+event.EventID, err)
		return
	}

	ackOrLog(msg, "whatsAppOrderStatusConsumer")
}
