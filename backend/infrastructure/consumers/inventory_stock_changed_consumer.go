package consumers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// stockSyncer is the one method this consumer needs from
// application/channels.SyncStockToChannelsUseCase (ISP) — kept as an
// interface here so the consumer can be unit tested without a real DB or
// WooClient.
type stockSyncer interface {
	Execute(ctx context.Context, skuID string) error
}

const inventoryStockChangedConsumerName = "inventory-stock-changed-sync"

// InventoryStockChangedConsumer reacts to inventory.stock.changed by
// re-pushing the SKU's current stock level to every channel its product is
// already linked to. Idempotent via ProcessedEventRepository: JetStream is
// at-least-once delivery, so redelivery of an event already fully handled
// must be a safe no-op.
type InventoryStockChangedConsumer struct {
	processed events.ProcessedEventRepository
	syncer    stockSyncer
	dlq       dlqPublisher
}

func NewInventoryStockChangedConsumer(processed events.ProcessedEventRepository, syncer stockSyncer, dlq dlqPublisher) *InventoryStockChangedConsumer {
	return &InventoryStockChangedConsumer{processed: processed, syncer: syncer, dlq: dlq}
}

// Handle is the NATS message handler — wire it up via a JetStream durable
// subscription (nc.QueueSubscribe/ChanSubscribe with AckExplicit). It never
// panics or crashes the caller: failures are logged and the message is
// nacked so JetStream retries delivery; only a message that could never
// succeed (malformed payload) is acked-and-dropped.
func (c *InventoryStockChangedConsumer) Handle(msg *nats.Msg) {
	var event events.InventoryStockChanged
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("inventoryStockChangedConsumer: invalid payload, dropping (not retryable): %v", err)
		ackOrLog(msg, "inventoryStockChangedConsumer")
		return
	}

	ctx := tenant.WithTenantID(context.Background(), event.TenantID)

	alreadyDone, err := c.processed.IsProcessed(ctx, event.EventID, inventoryStockChangedConsumerName)
	if err != nil {
		nakOrDeadLetter(c.dlq, msg, "inventoryStockChangedConsumer: check processed "+event.EventID, err)
		return
	}
	if alreadyDone {
		ackOrLog(msg, "inventoryStockChangedConsumer")
		return
	}

	if err := c.syncer.Execute(ctx, event.SKUID); err != nil {
		nakOrDeadLetter(c.dlq, msg, "inventoryStockChangedConsumer: sync sku "+event.SKUID, err)
		return
	}

	// Marked *after* the sync succeeds — see ProcessedEventRepository's
	// doc comment for why the order matters.
	if err := c.processed.MarkProcessed(ctx, event.EventID, inventoryStockChangedConsumerName); err != nil {
		// safe to retry: syncer.Execute re-pushes current state, so re-running it is harmless
		nakOrDeadLetter(c.dlq, msg, "inventoryStockChangedConsumer: mark processed "+event.EventID, err)
		return
	}

	ackOrLog(msg, "inventoryStockChangedConsumer")
}
