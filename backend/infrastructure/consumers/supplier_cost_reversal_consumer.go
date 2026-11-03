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

// goodsReceivedCostReverser is application/procurement.ReverseGoodsReceivedCostUseCase (ISP).
type goodsReceivedCostReverser interface {
	Execute(ctx context.Context, in appprocurement.ReverseGoodsReceivedCostInput) error
}

const SupplierCostOnReceiptCorrectedConsumerName = "supplier-cost-on-receipt-corrected"

// SupplierCostOnReceiptCorrectedConsumer takes received stock back off a
// purchase order line in the supplier cost history when a booking
// correction does (procurement.goods.receipt_corrected, COM-11).
type SupplierCostOnReceiptCorrectedConsumer struct {
	processed events.ProcessedEventRepository
	reverser  goodsReceivedCostReverser
	dlq       dlqPublisher
}

func NewSupplierCostOnReceiptCorrectedConsumer(processed events.ProcessedEventRepository, reverser goodsReceivedCostReverser, dlq dlqPublisher) *SupplierCostOnReceiptCorrectedConsumer {
	return &SupplierCostOnReceiptCorrectedConsumer{processed: processed, reverser: reverser, dlq: dlq}
}

// Name is the durable consumer name (and dedup key).
func (c *SupplierCostOnReceiptCorrectedConsumer) Name() string {
	return SupplierCostOnReceiptCorrectedConsumerName
}

// Handle never panics: a malformed payload is acked and dropped, a failure
// is nacked for JetStream to redeliver.
func (c *SupplierCostOnReceiptCorrectedConsumer) Handle(msg *nats.Msg) {
	name := SupplierCostOnReceiptCorrectedConsumerName
	var e events.EntityAction
	err := json.Unmarshal(msg.Data, &e)
	lineID, _ := e.Data["purchase_order_line_item_id"].(string)
	qty, hasQty := e.Data["quantity"].(float64)
	if err != nil || e.EntityID == "" || lineID == "" || !hasQty {
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
	if err := c.reverser.Execute(ctx, appprocurement.ReverseGoodsReceivedCostInput{
		EventID: e.EventID, SKUID: e.EntityID, PurchaseOrderLineItemID: lineID, Quantity: int64(qty), At: e.OccurredAt,
	}); err != nil {
		nakOrDeadLetter(c.dlq, msg, name+": reverse "+e.EventID, err)
		return
	}
	if err := c.processed.MarkProcessed(ctx, e.EventID, name); err != nil {
		// safe to retry: the row is keyed by the event
		nakOrDeadLetter(c.dlq, msg, name+": mark processed "+e.EventID, err)
		return
	}
	ackOrLog(msg, name)
}
