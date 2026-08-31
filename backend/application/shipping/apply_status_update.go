package shipping

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// SalesOrderStatusUpdater is *application/oms.UpdateSalesOrderUseCase
// narrowed to point of use (ISP) -- lets a Completed transition go through
// the same stock decrement/restock rules a manual status edit would (see
// UpdateSalesOrderUseCase's own doc), instead of a raw repo write that would
// silently skip them.
type SalesOrderStatusUpdater interface {
	Execute(ctx context.Context, id string, in appoms.UpdateSalesOrderInput) (*omsDomain.SalesOrder, error)
}

// applyStatusUpdate is the one place a fetched MyParcel TrackTraceResult
// turns into local writes -- shared by HandleShipmentWebhookUseCase
// (webhook-triggered) and ReconcileShipmentsUseCase (poll-triggered) so the
// two never drift on what a status change actually does (record history +
// the one Fulfilled-on-"shipped" order effect).
func applyStatusUpdate(
	ctx context.Context, db *gorm.DB,
	shipmentRepo shippingDomain.ShipmentRepository,
	statusEventRepo shippingDomain.ShipmentStatusEventRepository,
	salesOrderRepo omsDomain.SalesOrderRepository,
	shipmentID, salesOrderID string,
	tt integrationsDomain.TrackTraceResult,
	source shippingDomain.StatusEventSource,
) (shippingDomain.ShipmentStatusEvent, error) {
	event := shippingDomain.ShipmentStatusEvent{
		ShipmentID: shipmentID, Status: tt.StatusCode, StatusMain: tt.StatusMain, Source: source, OccurredAt: time.Now(),
	}
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		if err := shipmentRepo.UpdateStatus(ctx, shipmentID, tt.StatusCode, tt.StatusMain); err != nil {
			return err
		}
		event.TenantID = tenant.IDFromContext(ctx)
		if err := statusEventRepo.Append(ctx, &event); err != nil {
			return err
		}
		if tt.StatusMain == shippingDomain.StatusMainShipped {
			if err := salesOrderRepo.Update(ctx, salesOrderID, map[string]any{"fulfilled": true, "updated_at": time.Now()}); err != nil {
				return err
			}
		}
		return nil
	})
	return event, err
}

// completeOrderOnDelivery moves a SalesOrder to Completed once its shipment
// is actually delivered (StatusMainDelivered) -- deliberately a separate
// top-level call after applyStatusUpdate's own transaction commits, not a
// branch inside it, matching this codebase's established two-phase
// composition (see UpdateSalesOrderUseCase's doc: header write is one
// transaction, this use case's own stock-movement side effects are separate
// calls after) rather than nesting one tenant.WithTenant inside another.
// updater may be nil (deployment hasn't wired it yet) -- then this is a
// no-op, same convention as ProcessWebhookEventUseCase's optional
// shipmentHandler.
func completeOrderOnDelivery(ctx context.Context, updater SalesOrderStatusUpdater, statusMain, salesOrderID string) error {
	if updater == nil || statusMain != shippingDomain.StatusMainDelivered {
		return nil
	}
	completed := omsDomain.SalesOrderStatusCompleted
	if _, err := updater.Execute(ctx, salesOrderID, appoms.UpdateSalesOrderInput{Status: &completed}); err != nil {
		return fmt.Errorf("completeOrderOnDelivery: %w", err)
	}
	return nil
}

// MarkOrderShippedOnFulfillment moves a SalesOrder out of Pending/Processing
// into its carrier-handoff state once it's known to be fulfilled -- Shipped
// for every source except WooCommerce, which has no native "shipped" status
// of its own, so a Woo order goes straight to OnHold instead (see
// application/channels.PushSalesOrderStatusToChannelsUseCase's
// wooStatusBySalesOrderStatus map, where Shipped is deliberately absent).
// No-ops (not an error) if the order has already moved past
// Pending/Processing -- already applied via a different path, or diverged --
// never moves status backward or reapplies.
//
// Shared by HandleOrderStatusWebhookUseCase.applyProcessed (order_status_change
// webhook), HandleShipmentWebhookUseCase/shipmentWebhookDispatcher's
// post-applyStatusUpdate call (shipment_status_change webhook), and
// application/scheduler.ReconcileFulfilledOrdersUseCase (the 24h sweep for
// orders that missed both webhook paths) so none of them ever drift on this
// decision. Callers decide for themselves whether/when this should run
// (e.g. only on StatusMainShipped, only when updater != nil) -- this
// function itself always attempts the transition for whatever salesOrderID
// it's given.
func MarkOrderShippedOnFulfillment(ctx context.Context, db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, updater SalesOrderStatusUpdater, salesOrderID string) error {
	var order omsDomain.SalesOrder
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		o, err := salesOrderRepo.FindByID(ctx, salesOrderID)
		if err != nil {
			return err
		}
		order = *o
		return nil
	})
	if err != nil {
		return fmt.Errorf("markOrderShippedOnFulfillment: %w", err)
	}

	if order.Status != omsDomain.SalesOrderStatusPending && order.Status != omsDomain.SalesOrderStatusProcessing {
		return nil
	}

	target := omsDomain.SalesOrderStatusShipped
	if order.Source == omsDomain.SalesOrderSourceWooCommerce {
		target = omsDomain.SalesOrderStatusOnHold
	}
	if _, err := updater.Execute(ctx, order.ID, appoms.UpdateSalesOrderInput{Status: &target}); err != nil {
		return fmt.Errorf("markOrderShippedOnFulfillment: %w", err)
	}
	return nil
}
