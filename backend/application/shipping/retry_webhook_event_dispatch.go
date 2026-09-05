package shipping

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	webhooksDomain "github.com/adirkuhn/commerceos/domain/webhooks"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// RetryWebhookEventDispatchUseCase is the Messages tab's per-row "Retry"
// action -- re-run dispatch for one webhook stuck Failed or still Received
// (never got a chance, e.g. this tenant's copy predates dispatch-status
// tracking) right now, on demand, instead of waiting for the next backfill
// run. Shares BackfillShipmentWebhookDispatchUseCase's own resolution path
// (shipmentWebhookDispatcher) so a retry can never behave differently than
// a live webhook or a bulk backfill would for the same row.
type RetryWebhookEventDispatchUseCase struct {
	dispatcher         *shipmentWebhookDispatcher
	orderStatusHandler myParcelWebhookHandler
}

func NewRetryWebhookEventDispatchUseCase(
	db *gorm.DB,
	webhookEventRepo webhooksDomain.WebhookEventRepository,
	shipmentRepo shippingDomain.ShipmentRepository,
	statusEventRepo shippingDomain.ShipmentStatusEventRepository,
	connRepo integrationsDomain.ConnectionRepository,
	salesOrderRepo omsDomain.SalesOrderRepository,
	myParcelFactory integrationsDomain.MyParcelClientFactory,
	encryptionSecret string,
	fulfilmentOrderRepo shippingDomain.MyParcelFulfilmentOrderRepository,
	updateSalesOrder SalesOrderStatusUpdater,
	whatsappNotifier ShipmentDeliveryNotifier,
) *RetryWebhookEventDispatchUseCase {
	uc := &RetryWebhookEventDispatchUseCase{dispatcher: &shipmentWebhookDispatcher{
		db: db, webhookEventRepo: webhookEventRepo, shipmentRepo: shipmentRepo, statusEventRepo: statusEventRepo,
		connRepo: connRepo, salesOrderRepo: salesOrderRepo, myParcelFactory: myParcelFactory, encSecret: encryptionSecret,
		updateSalesOrder: updateSalesOrder, whatsappNotifier: whatsappNotifier,
	}}
	// fulfilmentOrderRepo may be nil (a deployment that hasn't wired
	// order_status_change dispatch yet) -- order_status_change retries then
	// fall through Execute's default case as not-dispatchable, same as
	// before this feature existed.
	if fulfilmentOrderRepo != nil {
		uc.orderStatusHandler = NewHandleOrderStatusWebhookUseCase(db, fulfilmentOrderRepo, salesOrderRepo, updateSalesOrder)
	}
	return uc
}

// Execute returns webhooksDomain.ErrWebhookEventNotFound if id doesn't
// exist, ErrWebhookEventNotDispatchable if its provider/topic has no
// dispatch handler at all (retrying it could never do anything), or the
// event's own updated row (status/dispatch_error now current) on success
// -- "success" here just means the retry ran to completion; the row may
// still come back Failed if the provider call failed again.
func (uc *RetryWebhookEventDispatchUseCase) Execute(ctx context.Context, eventID string) (*webhooksDomain.WebhookEvent, error) {
	var event webhooksDomain.WebhookEvent
	err := tenant.WithTenant(ctx, uc.dispatcher.db, func(ctx context.Context) error {
		e, err := uc.dispatcher.webhookEventRepo.FindByID(ctx, eventID)
		if err != nil {
			return err
		}
		event = *e
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("retryWebhookEventDispatch: %w", err)
	}
	switch {
	case event.Provider != "myparcel":
		return nil, fmt.Errorf("retryWebhookEventDispatch: %w", webhooksDomain.ErrWebhookEventNotDispatchable)
	case event.Topic == "shipment_status_change":
		if _, err := uc.dispatcher.processEvent(ctx, event, map[int64]*shipmentDispatchOutcome{}); err != nil {
			return nil, fmt.Errorf("retryWebhookEventDispatch: %w", err)
		}
	case event.Topic == "order_status_change" && uc.orderStatusHandler != nil:
		if err := uc.dispatchOrderStatus(ctx, event); err != nil {
			return nil, fmt.Errorf("retryWebhookEventDispatch: %w", err)
		}
	default:
		return nil, fmt.Errorf("retryWebhookEventDispatch: %w", webhooksDomain.ErrWebhookEventNotDispatchable)
	}

	var updated webhooksDomain.WebhookEvent
	err = tenant.WithTenant(ctx, uc.dispatcher.db, func(ctx context.Context) error {
		e, err := uc.dispatcher.webhookEventRepo.FindByID(ctx, eventID)
		if err != nil {
			return err
		}
		updated = *e
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("retryWebhookEventDispatch: refetch: %w", err)
	}
	return &updated, nil
}

// dispatchOrderStatus is order_status_change's retry path -- simpler than
// shipmentWebhookDispatcher.processEvent (no per-shipment tracktrace-refetch
// caching needed, HandleOrderStatusWebhookUseCase trusts the stored payload
// directly). A dispatch failure is recorded on the row (Failed,
// dispatch_error set) and NOT returned as this method's own error -- same
// "a retry that fails again is a normal, retryable outcome" convention as
// shipmentWebhookDispatcher.processEvent; only a failure to persist that
// outcome itself is returned.
func (uc *RetryWebhookEventDispatchUseCase) dispatchOrderStatus(ctx context.Context, event webhooksDomain.WebhookEvent) error {
	envelope := webhooksDomain.Envelope{TenantID: event.TenantID, Provider: event.Provider, RawPayload: event.Payload}
	status, dispatchError := webhooksDomain.WebhookEventStatusProcessed, ""
	if err := uc.orderStatusHandler.Execute(ctx, envelope); err != nil {
		status, dispatchError = webhooksDomain.WebhookEventStatusFailed, err.Error()
	}
	return tenant.WithTenant(ctx, uc.dispatcher.db, func(ctx context.Context) error {
		return uc.dispatcher.webhookEventRepo.MarkDispatchResult(ctx, event.ID, status, dispatchError)
	})
}
