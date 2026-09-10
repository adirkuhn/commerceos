package shipping

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	"github.com/adirkuhn/commerceos/domain/events"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/domain/outbox"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/idgen"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// fetchBarcodeIfMissing calls the carrier's "fetch this shipment" endpoint
// (the only source for a barcode outside the shipment_status_change webhook
// payload, which GetTrackTrace never carries -- see
// domain/integrations.CarrierClient.GetShipmentBarcode's doc) when the
// shipment doesn't already have one -- skipped entirely otherwise, since
// it's an extra API call for nothing once a shipment already has its
// barcode. Best-effort: a failure here is logged, not returned -- this is
// a backfill nicety, it must never fail the poll/refresh it's called from.
func fetchBarcodeIfMissing(ctx context.Context, client integrationsDomain.CarrierClient, shipment shippingDomain.Shipment) string {
	if shipment.Barcode != "" {
		return ""
	}
	barcode, err := client.GetShipmentBarcode(ctx, shipment.ExternalShipmentID)
	if err != nil {
		log.Printf("fetchBarcodeIfMissing: shipment %s: %v", shipment.ID, err)
		return ""
	}
	return barcode
}

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
// (webhook-triggered), ReconcileShipmentsUseCase (poll-triggered),
// RefreshShipmentTrackingUseCase (staff-triggered) and
// shipmentWebhookDispatcher (backfill) so none of them ever drift on what a
// status change actually does: record the new Shipment status + a
// ShipmentStatusEvent, and (outboxRepo non-nil) raise a
// ShipmentFulfillmentSignal event in the same transaction -- what that
// signal means for SalesOrder is entirely
// ApplyShipmentFulfillmentSignalUseCase's decision, not this function's or
// its callers'. outboxRepo is nil for RefreshShipmentTrackingUseCase (a
// staff-triggered "refresh from provider" that has never driven order
// status forward, kept that way) and for shipmentWebhookDispatcher's
// one-time historical backfill (raising a fresh event for years-old
// redelivered webhooks would be surprising, not useful) -- both keep their
// pre-existing direct-call behavior for whatever they do with the result.
// barcode is the webhook payload's own `barcode` field (only the
// shipment_status_change webhook carries it -- MyParcel's tracktrace GET,
// what the poll/refresh paths use, doesn't) -- empty when the caller has
// none to offer, in which case the shipment's existing barcode (set at
// label-creation time, or nothing yet if MyParcel hadn't assigned one) is
// left untouched, never blanked out.
func applyStatusUpdate(
	ctx context.Context, db *gorm.DB,
	shipmentRepo shippingDomain.ShipmentRepository,
	statusEventRepo shippingDomain.ShipmentStatusEventRepository,
	salesOrderRepo omsDomain.SalesOrderRepository,
	shipmentID, salesOrderID string,
	tt integrationsDomain.TrackTraceResult,
	source shippingDomain.StatusEventSource,
	barcode string,
	outboxRepo outbox.Repository,
) (shippingDomain.ShipmentStatusEvent, error) {
	event := shippingDomain.ShipmentStatusEvent{
		ShipmentID: shipmentID, Status: tt.StatusCode, StatusMain: tt.StatusMain, Source: source, OccurredAt: time.Now(),
	}
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		if err := shipmentRepo.UpdateStatus(ctx, shipmentID, tt.StatusCode, tt.StatusMain, tt.TrackingURL, barcode); err != nil {
			return err
		}
		event.TenantID = tenant.IDFromContext(ctx)
		if err := statusEventRepo.Append(ctx, &event); err != nil {
			return err
		}
		if outboxRepo == nil {
			return nil
		}
		return enqueueShipmentFulfillmentSignal(ctx, outboxRepo, salesOrderID,
			events.ShipmentFulfillmentSignalKindShipmentStatus, tt.StatusMain, tt.StatusCode)
	})
	return event, err
}

// enqueueShipmentFulfillmentSignal writes a ShipmentFulfillmentSignal
// outbox row in the caller's current transaction -- must be called from
// inside the same tenant.WithTenant block as whatever write it reports on,
// so the event and that write commit atomically (same convention as
// application/inventory's enqueueStockChanged).
func enqueueShipmentFulfillmentSignal(ctx context.Context, outboxRepo outbox.Repository, salesOrderID, kind, statusMain string, statusCode int) error {
	event := events.ShipmentFulfillmentSignal{
		Base:         events.Base{EventID: idgen.NewV4(), OccurredAt: time.Now()},
		TenantID:     tenant.IDFromContext(ctx),
		SalesOrderID: salesOrderID,
		Kind:         kind,
		StatusMain:   statusMain,
		StatusCode:   statusCode,
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return outboxRepo.Enqueue(ctx, event.EventName(), event.EventID, payload, event.OccurredAt)
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

// ShipmentDeliveryNotifier is *application/whatsapp.NotifyShipmentDeliveryUseCase
// narrowed to point of use (ISP) -- lets this package stay decoupled from
// whatsapp's own dependencies (Evolution API client, customer/connection
// repos, etc).
type ShipmentDeliveryNotifier interface {
	Execute(ctx context.Context, salesOrderID string, statusCode int) error
}

// notifyWhatsAppOnDelivery fires at the same trigger point as
// completeOrderOnDelivery (StatusMainDelivered) but, unlike that function,
// needs the raw numeric status code -- not just status_main -- to
// distinguish "delivered to recipient" from "left at a collect point"
// (MyParcel codes 7/9 vs 8/19 all report status_main "delivered", see
// shipping.MyParcelStatusLabel). notifier may be nil (deployment hasn't
// wired WhatsApp yet) -- then this is a no-op, same convention as
// completeOrderOnDelivery's updater. Deliberately a separate call the
// caller makes alongside completeOrderOnDelivery, not folded into it --
// completing the order and notifying the customer are two independent
// side effects of the same status transition. An error here (e.g. the
// WhatsApp send failing) does propagate up like any other dispatch error,
// which means the whole webhook redelivers -- completeOrderOnDelivery's own
// write is idempotent against that, and an occasional duplicate WhatsApp
// message on retry is preferable to silently dropping the notification.
func notifyWhatsAppOnDelivery(ctx context.Context, notifier ShipmentDeliveryNotifier, statusMain string, statusCode int, salesOrderID string) error {
	if notifier == nil || statusMain != shippingDomain.StatusMainDelivered {
		return nil
	}
	if err := notifier.Execute(ctx, salesOrderID, statusCode); err != nil {
		return fmt.Errorf("notifyWhatsAppOnDelivery: %w", err)
	}
	return nil
}

// MarkOrderShippedOnFulfillment moves a SalesOrder out of Pending/Processing
// into Shipped once it's known to be fulfilled -- uniformly, regardless of
// source. No-ops (not an error) if the order has already moved past
// Pending/Processing -- already applied via a different path, or diverged
// -- never moves status backward or reapplies. WooCommerce has no native
// "shipped" status of its own, but that's handled entirely at the
// channel-push boundary (see application/channels.PushSalesOrderStatusToChannelsUseCase's
// wooStatusBySalesOrderStatus map, which maps Shipped -> Woo's "on-hold"
// and attaches a shipment-info order note) -- CommerceOS's own record of
// the order always says Shipped.
//
// requireFulfilled additionally gates on SalesOrder.Fulfilled -- true for
// every shipment_status_change-family caller (ApplyShipmentFulfillmentSignal-
// UseCase's KindShipmentStatus branch, shipmentWebhookDispatcher's backfill).
// Fulfilled itself is set two ways now: independently by
// application/oms.CompletePickingUseCase at pick/pack time, or (since the
// bug this event exists to fix: an order fulfilled purely by creating a
// MyParcel label, no picking session ever run, stuck on Processing forever)
// by ApplyShipmentFulfillmentSignalUseCase itself the moment
// shippingDomain.IsCarrierConfirmed(statusMain) is true -- see that use
// case's doc. requireFulfilled is false for
// ApplyShipmentFulfillmentSignalUseCase's KindOrderProcessed branch
// (MyParcel's order_status_change "processed" webhook): that's its own
// unambiguous "MyParcel fulfilled this order" signal, independent of
// CommerceOS's local pick/pack state -- gating it on local Fulfilled would
// wrongly block orders exported straight to MyParcel's Order API without
// ever going through a local picking session. application/scheduler.
// ReconcileFulfilledOrdersUseCase (the 24h sweep) also passes true, but its
// own query already filters on Fulfilled=true, so it's a no-op restating
// what's already guaranteed.
func MarkOrderShippedOnFulfillment(ctx context.Context, db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, updater SalesOrderStatusUpdater, salesOrderID string, requireFulfilled bool) error {
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

	if requireFulfilled && !order.Fulfilled {
		return nil
	}
	if order.Status != omsDomain.SalesOrderStatusPending && order.Status != omsDomain.SalesOrderStatusProcessing {
		return nil
	}

	target := omsDomain.SalesOrderStatusShipped
	if _, err := updater.Execute(ctx, order.ID, appoms.UpdateSalesOrderInput{Status: &target}); err != nil {
		return fmt.Errorf("markOrderShippedOnFulfillment: %w", err)
	}
	return nil
}
