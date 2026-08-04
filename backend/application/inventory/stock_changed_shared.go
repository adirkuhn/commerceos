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

// enqueueStockChanged writes an inventory.stock.changed outbox row in the
// caller's current transaction — must be called from inside the same
// tenant.WithTenant block as the movement write, so the event and the
// write commit atomically. Consumers (see infrastructure/consumers) use
// this to re-push the SKU's current Syncable stock to every channel its
// product is already linked to.
func enqueueStockChanged(ctx context.Context, outboxRepo outbox.Repository, skuID string) error {
	event := events.InventoryStockChanged{
		Base:     events.Base{EventID: idgen.NewV4(), OccurredAt: time.Now()},
		TenantID: tenant.IDFromContext(ctx),
		SKUID:    skuID,
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return outboxRepo.Enqueue(ctx, event.EventName(), event.EventID, payload, event.OccurredAt)
}
