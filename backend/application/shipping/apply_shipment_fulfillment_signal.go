package shipping

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	"github.com/adirkuhn/commerceos/domain/events"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ApplyShipmentFulfillmentSignalUseCase is the one place a
// ShipmentFulfillmentSignal event turns into a SalesOrder decision --
// consumed by infrastructure/consumers.ShipmentFulfillmentSignalConsumer.
// Centralizing this (instead of each webhook/poll call site deciding
// inline, the way this codebase used to) means HandleShipmentWebhookUseCase,
// ReconcileShipmentsUseCase, and HandleOrderStatusWebhookUseCase all get
// identical behavior for the same signal.
type ApplyShipmentFulfillmentSignalUseCase struct {
	db               *gorm.DB
	salesOrderRepo   omsDomain.SalesOrderRepository
	updateSalesOrder SalesOrderStatusUpdater
	whatsappNotifier ShipmentDeliveryNotifier
}

func NewApplyShipmentFulfillmentSignalUseCase(
	db *gorm.DB,
	salesOrderRepo omsDomain.SalesOrderRepository,
	updateSalesOrder SalesOrderStatusUpdater,
	whatsappNotifier ShipmentDeliveryNotifier,
) *ApplyShipmentFulfillmentSignalUseCase {
	return &ApplyShipmentFulfillmentSignalUseCase{
		db: db, salesOrderRepo: salesOrderRepo, updateSalesOrder: updateSalesOrder, whatsappNotifier: whatsappNotifier,
	}
}

// Execute mirrors exactly what every shipment_status_change call site used
// to do inline (see MarkOrderShippedOnFulfillment/completeOrderOnDelivery/
// notifyWhatsAppOnDelivery's own docs for each step's own gating), plus the
// one new step that closes the bug this event exists to fix:
// KindShipmentStatus with carrier-confirmed evidence (see
// shippingDomain.IsCarrierConfirmed) now flips SalesOrder.Fulfilled itself,
// via UpdateSalesOrderInput.Fulfilled's manual-override path, instead of
// waiting on an internal pick/pack session that may never run for an order
// fulfilled by creating a shipping label directly (e.g. many Bol orders).
func (uc *ApplyShipmentFulfillmentSignalUseCase) Execute(ctx context.Context, salesOrderID, kind, statusMain string, statusCode int) error {
	switch kind {
	case events.ShipmentFulfillmentSignalKindOrderProcessed:
		if err := MarkOrderShippedOnFulfillment(ctx, uc.db, uc.salesOrderRepo, uc.updateSalesOrder, salesOrderID, false); err != nil {
			return fmt.Errorf("applyShipmentFulfillmentSignal: %w", err)
		}
		return nil
	case events.ShipmentFulfillmentSignalKindShipmentStatus:
		if shippingDomain.IsCarrierConfirmed(statusMain) {
			if err := uc.markFulfilledIfNotAlready(ctx, salesOrderID); err != nil {
				return err
			}
		}
		if err := MarkOrderShippedOnFulfillment(ctx, uc.db, uc.salesOrderRepo, uc.updateSalesOrder, salesOrderID, true); err != nil {
			return fmt.Errorf("applyShipmentFulfillmentSignal: %w", err)
		}
		if err := completeOrderOnDelivery(ctx, uc.updateSalesOrder, statusMain, salesOrderID); err != nil {
			return err
		}
		return notifyWhatsAppOnDelivery(ctx, uc.whatsappNotifier, statusMain, statusCode, salesOrderID)
	default:
		return fmt.Errorf("applyShipmentFulfillmentSignal: unknown kind %q", kind)
	}
}

func (uc *ApplyShipmentFulfillmentSignalUseCase) markFulfilledIfNotAlready(ctx context.Context, salesOrderID string) error {
	var order omsDomain.SalesOrder
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		o, err := uc.salesOrderRepo.FindByID(ctx, salesOrderID)
		if err != nil {
			return err
		}
		order = *o
		return nil
	})
	if err != nil {
		return fmt.Errorf("applyShipmentFulfillmentSignal: %w", err)
	}
	if order.Fulfilled {
		return nil
	}
	fulfilled := true
	if _, err := uc.updateSalesOrder.Execute(ctx, salesOrderID, appoms.UpdateSalesOrderInput{Fulfilled: &fulfilled}); err != nil {
		return fmt.Errorf("applyShipmentFulfillmentSignal: mark fulfilled: %w", err)
	}
	return nil
}
