package consumers

import (
	"context"
	"log"

	"github.com/nats-io/nats.go"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// costPriceFromHistorySyncer is application/pms.SyncCostPriceFromHistoryUseCase (ISP).
type costPriceFromHistorySyncer interface {
	Execute(ctx context.Context, skuID string) error
}

const CostPriceOnSupplierCostRecordedConsumerName = "cost-price-on-supplier-cost-recorded"

// CostPriceOnSupplierCostRecordedConsumer keeps a SKU's cost price on the
// last receipt still standing in its cost history by supplier (COM-11):
// handles procurement.supplier_cost.recorded (receipts, reversals,
// corrections). Idempotent via ProcessedEventRepository, and the sync
// changes nothing when the cost price is already right.
type CostPriceOnSupplierCostRecordedConsumer struct {
	processed events.ProcessedEventRepository
	syncer    costPriceFromHistorySyncer
	dlq       dlqPublisher
}

func NewCostPriceOnSupplierCostRecordedConsumer(processed events.ProcessedEventRepository, syncer costPriceFromHistorySyncer, dlq dlqPublisher) *CostPriceOnSupplierCostRecordedConsumer {
	return &CostPriceOnSupplierCostRecordedConsumer{processed: processed, syncer: syncer, dlq: dlq}
}

// Name is the durable consumer name (and dedup key).
func (c *CostPriceOnSupplierCostRecordedConsumer) Name() string {
	return CostPriceOnSupplierCostRecordedConsumerName
}

// Handle never panics: a malformed payload is acked and dropped, a failure
// is nacked for JetStream to redeliver.
func (c *CostPriceOnSupplierCostRecordedConsumer) Handle(msg *nats.Msg) {
	name := CostPriceOnSupplierCostRecordedConsumerName
	tenantID, eventID, skuID, err := decodeEntityAction(msg.Data)
	if err != nil || skuID == "" {
		log.Printf("%s: invalid payload, dropping (not retryable): %v", name, err)
		ackOrLog(msg, name)
		return
	}
	ctx := tenant.WithTenantID(context.Background(), tenantID)

	done, err := c.processed.IsProcessed(ctx, eventID, name)
	if err != nil {
		nakOrDeadLetter(c.dlq, msg, name+": check processed "+eventID, err)
		return
	}
	if done {
		ackOrLog(msg, name)
		return
	}
	if err := c.syncer.Execute(ctx, skuID); err != nil {
		nakOrDeadLetter(c.dlq, msg, name+": sync "+skuID, err)
		return
	}
	if err := c.processed.MarkProcessed(ctx, eventID, name); err != nil {
		// safe to retry: a repeated sync changes nothing
		nakOrDeadLetter(c.dlq, msg, name+": mark processed "+eventID, err)
		return
	}
	ackOrLog(msg, name)
}
