package consumers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// productSyncer is the one method this consumer needs from
// application/channels.SyncProductToChannelsUseCase (ISP) — kept as an
// interface here so the consumer can be unit tested without a real DB or
// WooClient.
type productSyncer interface {
	Execute(ctx context.Context, productID string) error
}

const productChangedConsumerName = "product-changed-sync"

// ProductChangedConsumer reacts to pms.product.changed in two steps: first
// enroller creates a ProductChannelListing for every SalesChannel this
// product newly matches (the *first* link -- see
// application/channels.EnrollProductInChannelsUseCase, which is what makes
// a brand-new product actually reach WooCommerce instead of only ever
// being pushed by a manual action), then syncer re-pushes current field
// values to every listing that (now) exists. Idempotent via
// ProcessedEventRepository: JetStream is at-least-once delivery, so
// redelivery of an event already fully handled must be a safe no-op.
type ProductChangedConsumer struct {
	processed events.ProcessedEventRepository
	enroller  productSyncer
	syncer    productSyncer
	dlq       dlqPublisher
}

func NewProductChangedConsumer(processed events.ProcessedEventRepository, enroller, syncer productSyncer, dlq dlqPublisher) *ProductChangedConsumer {
	return &ProductChangedConsumer{processed: processed, enroller: enroller, syncer: syncer, dlq: dlq}
}

// Handle is the NATS message handler — wire it up via a JetStream durable
// subscription (nc.QueueSubscribe/ChanSubscribe with AckExplicit). It never
// panics or crashes the caller: failures are logged and the message is
// nacked so JetStream retries delivery; only a message that could never
// succeed (malformed payload) is acked-and-dropped.
func (c *ProductChangedConsumer) Handle(msg *nats.Msg) {
	var event events.ProductChanged
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("productChangedConsumer: invalid payload, dropping (not retryable): %v", err)
		_ = msg.Ack()
		return
	}

	ctx := tenant.WithTenantID(context.Background(), event.TenantID)

	alreadyDone, err := c.processed.IsProcessed(ctx, event.EventID, productChangedConsumerName)
	if err != nil {
		nakOrDeadLetter(c.dlq, msg, "productChangedConsumer: check processed "+event.EventID, err)
		return
	}
	if alreadyDone {
		_ = msg.Ack()
		return
	}

	if err := c.enroller.Execute(ctx, event.ProductID); err != nil {
		nakOrDeadLetter(c.dlq, msg, "productChangedConsumer: enroll product "+event.ProductID, err)
		return
	}

	if err := c.syncer.Execute(ctx, event.ProductID); err != nil {
		nakOrDeadLetter(c.dlq, msg, "productChangedConsumer: sync product "+event.ProductID, err)
		return
	}

	// Marked *after* the sync succeeds — see ProcessedEventRepository's
	// doc comment for why the order matters.
	if err := c.processed.MarkProcessed(ctx, event.EventID, productChangedConsumerName); err != nil {
		// safe to retry: syncer.Execute re-pushes current state, so re-running it is harmless
		nakOrDeadLetter(c.dlq, msg, "productChangedConsumer: mark processed "+event.EventID, err)
		return
	}

	_ = msg.Ack()
}
