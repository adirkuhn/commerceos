package consumers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// productChannelsDeleter is the one method this consumer needs from
// application/channels.DeleteProductFromChannelsUseCase (ISP) — kept as an
// interface here so the consumer can be unit tested without a real DB or
// WooClient.
type productChannelsDeleter interface {
	Execute(ctx context.Context, listings []events.ProductDeletedListing) error
}

const productDeletedConsumerName = "product-deleted-cleanup"

// ProductDeletedConsumer reacts to pms.product.deleted by removing the
// product from every external channel it was synced to (e.g. WooCommerce),
// using the listing snapshot carried on the event -- the local
// product_channel_listings row is already gone by DB cascade before this
// ever runs (see application/pms.DeleteProductUseCase). Idempotent via
// ProcessedEventRepository, same as ProductChangedConsumer.
type ProductDeletedConsumer struct {
	processed events.ProcessedEventRepository
	deleter   productChannelsDeleter
	dlq       dlqPublisher
}

func NewProductDeletedConsumer(processed events.ProcessedEventRepository, deleter productChannelsDeleter, dlq dlqPublisher) *ProductDeletedConsumer {
	return &ProductDeletedConsumer{processed: processed, deleter: deleter, dlq: dlq}
}

// Handle is the NATS message handler — wire it up via a JetStream durable
// subscription (nc.QueueSubscribe/ChanSubscribe with AckExplicit). It never
// panics or crashes the caller: failures are logged and the message is
// nacked so JetStream retries delivery; only a message that could never
// succeed (malformed payload) is acked-and-dropped.
func (c *ProductDeletedConsumer) Handle(msg *nats.Msg) {
	var event events.ProductDeleted
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("productDeletedConsumer: invalid payload, dropping (not retryable): %v", err)
		_ = msg.Ack()
		return
	}

	ctx := tenant.WithTenantID(context.Background(), event.TenantID)

	alreadyDone, err := c.processed.IsProcessed(ctx, event.EventID, productDeletedConsumerName)
	if err != nil {
		nakOrDeadLetter(c.dlq, msg, "productDeletedConsumer: check processed "+event.EventID, err)
		return
	}
	if alreadyDone {
		_ = msg.Ack()
		return
	}

	if err := c.deleter.Execute(ctx, event.Listings); err != nil {
		nakOrDeadLetter(c.dlq, msg, "productDeletedConsumer: delete from channels "+event.ProductID, err)
		return
	}

	if err := c.processed.MarkProcessed(ctx, event.EventID, productDeletedConsumerName); err != nil {
		// safe to retry: DeleteProductFromChannelsUseCase is itself best-effort per listing
		nakOrDeadLetter(c.dlq, msg, "productDeletedConsumer: mark processed "+event.EventID, err)
		return
	}

	_ = msg.Ack()
}
