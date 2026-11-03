package consumers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// supplierCostCorrector is application/procurement.CorrectSupplierCostUseCase (ISP).
type supplierCostCorrector interface {
	Execute(ctx context.Context, in appprocurement.CorrectSupplierCostInput) error
}

const SupplierCostOnCostCorrectedConsumerName = "supplier-cost-on-cost-corrected"

// SupplierCostOnCostCorrectedConsumer reprices a purchase order line in the
// supplier cost history when a cost correction references it (COM-11):
// handles pms.sku.inventory_cost_changed, acting only on reason
// "correction" with reference_type purchase_order_line_item -- every other
// inventory cost change is acked untouched.
type SupplierCostOnCostCorrectedConsumer struct {
	processed events.ProcessedEventRepository
	corrector supplierCostCorrector
	dlq       dlqPublisher
}

func NewSupplierCostOnCostCorrectedConsumer(processed events.ProcessedEventRepository, corrector supplierCostCorrector, dlq dlqPublisher) *SupplierCostOnCostCorrectedConsumer {
	return &SupplierCostOnCostCorrectedConsumer{processed: processed, corrector: corrector, dlq: dlq}
}

// Name is the durable consumer name (and dedup key).
func (c *SupplierCostOnCostCorrectedConsumer) Name() string {
	return SupplierCostOnCostCorrectedConsumerName
}

// correctionOf reads the correction a payload describes; ok is false for
// any other inventory cost change.
func correctionOf(e events.EntityAction) (appprocurement.CorrectSupplierCostInput, bool) {
	refType, _ := e.Data["reference_type"].(string)
	lineID, _ := e.Data["reference_id"].(string)
	adjustment, hasAdjustment := e.Data[apppms.CorrectionAdjustmentDataKey].(float64)
	if e.Data["reason"] != apppms.CostPriceChangeReasonCorrection || refType != apppms.ReceiptReferencePurchaseOrderLine ||
		lineID == "" || !hasAdjustment {
		return appprocurement.CorrectSupplierCostInput{}, false
	}
	return appprocurement.CorrectSupplierCostInput{
		EventID: e.EventID, SKUID: e.EntityID, PurchaseOrderLineItemID: lineID,
		AdjustmentValueMinorUnits: int64(adjustment), At: e.OccurredAt,
	}, true
}

// Handle never panics: a malformed payload is acked and dropped, a failure
// is nacked for JetStream to redeliver.
func (c *SupplierCostOnCostCorrectedConsumer) Handle(msg *nats.Msg) {
	name := SupplierCostOnCostCorrectedConsumerName
	var e events.EntityAction
	if err := json.Unmarshal(msg.Data, &e); err != nil || e.EntityID == "" {
		log.Printf("%s: invalid payload, dropping (not retryable): %v", name, err)
		ackOrLog(msg, name)
		return
	}
	in, ok := correctionOf(e)
	if !ok {
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
	if err := c.corrector.Execute(ctx, in); err != nil {
		nakOrDeadLetter(c.dlq, msg, name+": correct "+e.EventID, err)
		return
	}
	if err := c.processed.MarkProcessed(ctx, e.EventID, name); err != nil {
		// safe to retry: the rows are keyed by the event
		nakOrDeadLetter(c.dlq, msg, name+": mark processed "+e.EventID, err)
		return
	}
	ackOrLog(msg, name)
}
