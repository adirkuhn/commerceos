package pms

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

// enqueueProductChanged writes a pms.product.changed outbox row in the
// caller's current transaction — must be called from inside the same
// tenant.WithTenant block as the write that changed the product, so the
// event and the write commit atomically. Consumers (see
// infrastructure/consumers) use this to re-push the product to every
// channel it's already linked to.
func enqueueProductChanged(ctx context.Context, outboxRepo outbox.Repository, productID string) error {
	event := events.ProductChanged{
		Base:      events.Base{EventID: idgen.NewV4(), OccurredAt: time.Now()},
		TenantID:  tenant.IDFromContext(ctx),
		ProductID: productID,
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return outboxRepo.Enqueue(ctx, event.EventName(), event.EventID, payload, event.OccurredAt)
}
