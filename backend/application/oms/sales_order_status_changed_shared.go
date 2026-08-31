package oms

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

// enqueueSalesOrderStatusChanged writes an oms.sales_order.status_changed
// outbox row in the caller's current transaction -- must be called from
// inside the same tenant.WithTenant block as the write that changed the
// order's Status, so the event and the write commit atomically. Consumers
// (see infrastructure/consumers) use this to push the order's new status to
// whatever channel it came from (WooCommerce today).
func enqueueSalesOrderStatusChanged(ctx context.Context, outboxRepo outbox.Repository, salesOrderID string) error {
	event := events.SalesOrderStatusChanged{
		Base:         events.Base{EventID: idgen.NewV4(), OccurredAt: time.Now()},
		TenantID:     tenant.IDFromContext(ctx),
		SalesOrderID: salesOrderID,
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return outboxRepo.Enqueue(ctx, event.EventName(), event.EventID, payload, event.OccurredAt)
}
