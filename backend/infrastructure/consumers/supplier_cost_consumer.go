package consumers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// goodsReceivedCostRecorder is application/procurement.RecordGoodsReceivedCostUseCase (ISP).
type goodsReceivedCostRecorder interface {
	Execute(ctx context.Context, in appprocurement.RecordGoodsReceivedCostInput) error
}

const SupplierCostOnGoodsReceivedConsumerName = "supplier-cost-on-goods-received"

// SupplierCostOnGoodsReceivedConsumer writes each procurement.goods.received
// into the SKU's cost history by supplier (COM-11). Idempotent twice over:
// ProcessedEventRepository, and the history row is keyed by the event.
type SupplierCostOnGoodsReceivedConsumer struct {
	processed events.ProcessedEventRepository
	recorder  goodsReceivedCostRecorder
	dlq       dlqPublisher
}

func NewSupplierCostOnGoodsReceivedConsumer(processed events.ProcessedEventRepository, recorder goodsReceivedCostRecorder, dlq dlqPublisher) *SupplierCostOnGoodsReceivedConsumer {
	return &SupplierCostOnGoodsReceivedConsumer{processed: processed, recorder: recorder, dlq: dlq}
}

// Name is the durable consumer name (and dedup key).
func (c *SupplierCostOnGoodsReceivedConsumer) Name() string {
	return SupplierCostOnGoodsReceivedConsumerName
}

// Handle never panics: a malformed payload is acked and dropped, a failure
// is nacked for JetStream to redeliver.
func (c *SupplierCostOnGoodsReceivedConsumer) Handle(msg *nats.Msg) {
	name := SupplierCostOnGoodsReceivedConsumerName
	var e events.GoodsReceived
	if err := json.Unmarshal(msg.Data, &e); err != nil || e.EventID == "" || e.SKUID == "" || e.PurchaseOrderLineItemID == "" {
		log.Printf("%s: invalid payload, dropping (not retryable): %v", name, err)
		ackOrLog(msg, name)
		return
	}
	ctx := tenant.WithTenantID(context.Background(), e.TenantID)

	done, err := c.processed.IsProcessed(ctx, e.EventID, name)
	if err != nil {
		nakOrDeadLetter(c.dlq, msg, name+": check processed "+e.EventID, err)
		return
	}
	if done {
		ackOrLog(msg, name)
		return
	}
	if err := c.recorder.Execute(ctx, appprocurement.RecordGoodsReceivedCostInput{
		EventID: e.EventID, SKUID: e.SKUID, PurchaseOrderLineItemID: e.PurchaseOrderLineItemID, BatchID: e.BatchID,
		Quantity: e.Quantity, LandedUnitCostMinorUnits: e.LandedUnitCostMinorUnits, ReceivedAt: e.OccurredAt,
	}); err != nil {
		nakOrDeadLetter(c.dlq, msg, name+": record "+e.EventID, err)
		return
	}
	if err := c.processed.MarkProcessed(ctx, e.EventID, name); err != nil {
		// safe to retry: the history row is keyed by the event
		nakOrDeadLetter(c.dlq, msg, name+": mark processed "+e.EventID, err)
		return
	}
	ackOrLog(msg, name)
}
