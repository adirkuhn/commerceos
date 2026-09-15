package inventory

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/domain/outbox"
	"github.com/adirkuhn/commerceos/internal/idgen"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// enqueueGoodsReceived writes a procurement.goods.received outbox row in
// the caller's current transaction -- must be called from inside the same
// tenant.WithTenant block as the batch/movement write, so the event and
// the write commit atomically (mandatory outbox pattern -- never publish
// without an outbox row in the same transaction). See
// events.GoodsReceived's doc comment for when this fires.
func enqueueGoodsReceived(ctx context.Context, outboxRepo outbox.Repository, skuID, warehouseID, batchID, purchaseOrderLineItemID string, quantity, landedUnitCostMinorUnits int64) error {
	event := events.GoodsReceived{
		Base:     events.Base{EventID: idgen.NewV4(), OccurredAt: time.Now()},
		TenantID: tenant.IDFromContext(ctx), SKUID: skuID, WarehouseID: warehouseID, BatchID: batchID,
		PurchaseOrderLineItemID: purchaseOrderLineItemID, Quantity: quantity, LandedUnitCostMinorUnits: landedUnitCostMinorUnits,
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return outboxRepo.Enqueue(ctx, event.EventName(), event.EventID, payload, event.OccurredAt)
}
