package consumers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// variantChannelsDeleter is the one method this consumer needs from
// application/channels.DeleteVariantFromChannelsUseCase (ISP) — kept as an
// interface here so the consumer can be unit tested without a real DB or
// WooClient.
type variantChannelsDeleter interface {
	Execute(ctx context.Context, listings []events.VariantUnlistedListing) error
}

const variantUnlistedConsumerName = "variant-unlisted-cleanup"

// VariantUnlistedConsumer reacts to pms.variant.unlisted by taking the
// variant off every external channel it was listed on (e.g. its Woo
// product or variation), using the listing snapshot carried on the event
// -- the listing rows are already gone (see application/pms.
// DeleteVariantUseCase/DeleteSKUUseCase). Idempotent via
// ProcessedEventRepository, same as ProductDeletedConsumer.
type VariantUnlistedConsumer struct {
	processed events.ProcessedEventRepository
	deleter   variantChannelsDeleter
	dlq       dlqPublisher
}

func NewVariantUnlistedConsumer(processed events.ProcessedEventRepository, deleter variantChannelsDeleter, dlq dlqPublisher) *VariantUnlistedConsumer {
	return &VariantUnlistedConsumer{processed: processed, deleter: deleter, dlq: dlq}
}

// Handle is the NATS message handler — wire it up via a JetStream durable
// subscription (nc.QueueSubscribe/ChanSubscribe with AckExplicit). It never
// panics or crashes the caller: failures are logged and the message is
// nacked so JetStream retries delivery; only a message that could never
// succeed (malformed payload) is acked-and-dropped.
func (c *VariantUnlistedConsumer) Handle(msg *nats.Msg) {
	var event events.VariantUnlisted
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("variantUnlistedConsumer: invalid payload, dropping (not retryable): %v", err)
		ackOrLog(msg, "variantUnlistedConsumer")
		return
	}

	ctx := tenant.WithTenantID(context.Background(), event.TenantID)

	alreadyDone, err := c.processed.IsProcessed(ctx, event.EventID, variantUnlistedConsumerName)
	if err != nil {
		nakOrDeadLetter(c.dlq, msg, "variantUnlistedConsumer: check processed "+event.EventID, err)
		return
	}
	if alreadyDone {
		ackOrLog(msg, "variantUnlistedConsumer")
		return
	}

	if err := c.deleter.Execute(ctx, event.Listings); err != nil {
		nakOrDeadLetter(c.dlq, msg, "variantUnlistedConsumer: delete from channels "+event.VariantID, err)
		return
	}

	if err := c.processed.MarkProcessed(ctx, event.EventID, variantUnlistedConsumerName); err != nil {
		// safe to retry: DeleteVariantFromChannelsUseCase is itself best-effort per listing
		nakOrDeadLetter(c.dlq, msg, "variantUnlistedConsumer: mark processed "+event.EventID, err)
		return
	}

	ackOrLog(msg, "variantUnlistedConsumer")
}
