package shipping

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"gorm.io/gorm"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/domain/webhooks"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ErrUnrecognizedMyParcelOrderStatus is returned (and surfaces the webhook
// event as Failed/needs-attention) for any MyParcel order_status_change
// status other than "open"/"processed" -- MyParcel's docs don't enumerate
// the full set (only "open" and "partially_processed" are documented, and
// "processed" -- the only other value actually observed in production --
// isn't even among those), so an unrecognized value is treated as "needs a
// human to look at this and extend the mapping," not silently ignored.
var ErrUnrecognizedMyParcelOrderStatus = errors.New("shipping: unrecognized myparcel order status")

type myParcelOrderStatusHook struct {
	Order  string `json:"order"`
	Status string `json:"status"`
}

type myParcelOrderWebhookPayload struct {
	Data struct {
		Hooks []myParcelOrderStatusHook `json:"hooks"`
	} `json:"data"`
}

// HandleOrderStatusWebhookUseCase is the MyParcel order_status_change branch
// of the webhook dispatch (see MyParcelWebhookRouter) -- a different topic
// than shipment_status_change, and a different MyParcel API (Orders/
// Fulfilment, not Shipments), so the payload shape and the reference it
// carries (MyParcel's own order UUID, not a shipment id) are both distinct
// from HandleShipmentWebhookUseCase. Unlike that handler, this one trusts
// the webhook's own status field -- MyParcel's Orders API exposes no GET to
// re-fetch authoritative current status the way GetTrackTrace does for
// shipments.
//
// Status mapping (approved design):
//   - "open": no-op -- the order's initial state; by the time it was
//     exported it's already past pending, so nothing here should move it
//     backward.
//   - "processed": SalesOrder moves out of Pending/Processing -- OnHold for
//     a WooCommerce-sourced order (Woo has no "shipped" concept; OnHold
//     keeps the status round-trippable against Woo's own vocabulary, see
//     SalesOrderStatus's doc), Shipped for every other source. Guarded so an
//     order already past Pending/Processing (already applied, or diverged
//     via a different path) is left alone -- not reapplied, not moved
//     backward.
//   - anything else: ErrUnrecognizedMyParcelOrderStatus.
//
// An unknown MyParcel order UUID (no myparcel_fulfilment_orders row -- a
// tenant that never exported this specific order) is a no-op, same
// convention as HandleShipmentWebhookUseCase's unknown shipment_id.
type HandleOrderStatusWebhookUseCase struct {
	db                  *gorm.DB
	fulfilmentOrderRepo shippingDomain.MyParcelFulfilmentOrderRepository
	salesOrderRepo      omsDomain.SalesOrderRepository
	updateSalesOrder    SalesOrderStatusUpdater
}

func NewHandleOrderStatusWebhookUseCase(
	db *gorm.DB,
	fulfilmentOrderRepo shippingDomain.MyParcelFulfilmentOrderRepository,
	salesOrderRepo omsDomain.SalesOrderRepository,
	updateSalesOrder SalesOrderStatusUpdater,
) *HandleOrderStatusWebhookUseCase {
	return &HandleOrderStatusWebhookUseCase{
		db: db, fulfilmentOrderRepo: fulfilmentOrderRepo, salesOrderRepo: salesOrderRepo, updateSalesOrder: updateSalesOrder,
	}
}

func (uc *HandleOrderStatusWebhookUseCase) Execute(ctx context.Context, envelope webhooks.Envelope) error {
	var payload myParcelOrderWebhookPayload
	if err := json.Unmarshal(envelope.RawPayload, &payload); err != nil {
		return fmt.Errorf("handleOrderStatusWebhook: decode payload: %w", err)
	}

	ctx = tenant.WithTenantID(ctx, envelope.TenantID)
	for _, hook := range payload.Data.Hooks {
		if err := uc.handleOne(ctx, hook); err != nil {
			return fmt.Errorf("handleOrderStatusWebhook: order %s: %w", hook.Order, err)
		}
	}
	return nil
}

func (uc *HandleOrderStatusWebhookUseCase) handleOne(ctx context.Context, hook myParcelOrderStatusHook) error {
	var fulfilment *shippingDomain.MyParcelFulfilmentOrder
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		f, err := uc.fulfilmentOrderRepo.FindByMyParcelUUID(ctx, hook.Order)
		if err != nil {
			return err
		}
		fulfilment = f
		return nil
	})
	if err != nil {
		return err
	}
	if fulfilment == nil {
		// A MyParcel order uuid this tenant never exported (or exported for
		// a different sales order since deleted) -- nothing to update.
		return nil
	}

	switch hook.Status {
	case "open":
		return nil
	case "processed":
		return uc.applyProcessed(ctx, *fulfilment)
	default:
		return fmt.Errorf("%w: %q", ErrUnrecognizedMyParcelOrderStatus, hook.Status)
	}
}

func (uc *HandleOrderStatusWebhookUseCase) applyProcessed(ctx context.Context, fulfilment shippingDomain.MyParcelFulfilmentOrder) error {
	if err := MarkOrderShippedOnFulfillment(ctx, uc.db, uc.salesOrderRepo, uc.updateSalesOrder, fulfilment.SalesOrderID); err != nil {
		return fmt.Errorf("update sales order status: %w", err)
	}
	return nil
}
