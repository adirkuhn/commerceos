package consumers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// clearanceSKUFinisher is the one method this consumer needs from
// application/promotions.FinishClearanceLinesUseCase (ISP).
type clearanceSKUFinisher interface {
	ExecuteForSKU(ctx context.Context, skuID string) error
}

const clearancePromotionStockConsumerName = "inventory-stock-changed-clearance-promotions"

// ClearancePromotionStockConsumer reacts to inventory.stock.changed by
// ending any clearance promotion line on that SKU whose batch just sold out,
// was fully discarded/written off, or expired -- restoring the SKU's price
// within seconds instead of waiting for the promotion_activation sweep.
// Its own consumer name (and durable) keeps dedup independent of
// InventoryStockChangedConsumer, which handles the same subject.
type ClearancePromotionStockConsumer struct {
	processed events.ProcessedEventRepository
	finisher  clearanceSKUFinisher
	dlq       dlqPublisher
}

func NewClearancePromotionStockConsumer(processed events.ProcessedEventRepository, finisher clearanceSKUFinisher, dlq dlqPublisher) *ClearancePromotionStockConsumer {
	return &ClearancePromotionStockConsumer{processed: processed, finisher: finisher, dlq: dlq}
}

// Handle is the NATS message handler -- same ack/nack contract as
// InventoryStockChangedConsumer.Handle.
func (c *ClearancePromotionStockConsumer) Handle(msg *nats.Msg) {
	var event events.InventoryStockChanged
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("clearancePromotionStockConsumer: invalid payload, dropping (not retryable): %v", err)
		_ = msg.Ack()
		return
	}

	ctx := tenant.WithTenantID(context.Background(), event.TenantID)

	alreadyDone, err := c.processed.IsProcessed(ctx, event.EventID, clearancePromotionStockConsumerName)
	if err != nil {
		nakOrDeadLetter(c.dlq, msg, "clearancePromotionStockConsumer: check processed "+event.EventID, err)
		return
	}
	if alreadyDone {
		_ = msg.Ack()
		return
	}

	if err := c.finisher.ExecuteForSKU(ctx, event.SKUID); err != nil {
		nakOrDeadLetter(c.dlq, msg, "clearancePromotionStockConsumer: check sku "+event.SKUID, err)
		return
	}

	if err := c.processed.MarkProcessed(ctx, event.EventID, clearancePromotionStockConsumerName); err != nil {
		// safe to retry: ExecuteForSKU only looks at still-open lines, so a
		// line it already finished is simply not seen again
		nakOrDeadLetter(c.dlq, msg, "clearancePromotionStockConsumer: mark processed "+event.EventID, err)
		return
	}

	_ = msg.Ack()
}
