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
	dispatcher *shipmentWebhookDispatcher
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
) *RetryWebhookEventDispatchUseCase {
	return &RetryWebhookEventDispatchUseCase{dispatcher: &shipmentWebhookDispatcher{
		db: db, webhookEventRepo: webhookEventRepo, shipmentRepo: shipmentRepo, statusEventRepo: statusEventRepo,
		connRepo: connRepo, salesOrderRepo: salesOrderRepo, myParcelFactory: myParcelFactory, encSecret: encryptionSecret,
	}}
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
	if event.Provider != "myparcel" || event.Topic != "shipment_status_change" {
		return nil, fmt.Errorf("retryWebhookEventDispatch: %w", webhooksDomain.ErrWebhookEventNotDispatchable)
	}

	if _, err := uc.dispatcher.processEvent(ctx, event, map[int64]*shipmentDispatchOutcome{}); err != nil {
		return nil, fmt.Errorf("retryWebhookEventDispatch: %w", err)
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
