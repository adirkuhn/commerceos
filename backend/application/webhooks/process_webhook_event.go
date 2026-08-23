package webhooks

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/domain/webhooks"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// topicHeader is the header WooCommerce sends its event topic under (e.g.
// "order.updated", "product.created") -- best-effort only, never verified
// or relied on for correctness (see the events themselves for the
// authoritative payload); used purely so an operator browsing WebhookEvent
// rows can tell what kind of event each one was without opening the payload.
const topicHeader = "X-Wc-Webhook-Topic"

// shipmentStatusWebhookHandler is application/shipping.HandleShipmentWebhookUseCase
// narrowed to point of use (ISP) -- lets this package stay decoupled from
// shipping's own dependencies (MyParcel client, oms repo, etc) and keeps
// this use case unit-testable without them.
type shipmentStatusWebhookHandler interface {
	Execute(ctx context.Context, envelope webhooks.Envelope) error
}

// ProcessWebhookEventUseCase is the cold-path landing point: durably
// persist the envelope cmd/webhook-processor consumed off
// COMMERCEOS_WEBHOOKS. Per-topic business-rule dispatch is deliberately out
// of scope for this pass for every provider EXCEPT "myparcel" -- see
// shipmentHandler below, the one exception, added when the MyParcel
// shipping integration needed webhook-driven order updates. Every other
// provider's event still just gets persisted, exactly as before.
type ProcessWebhookEventUseCase struct {
	db              *gorm.DB
	eventRepo       webhooks.WebhookEventRepository
	shipmentHandler shipmentStatusWebhookHandler
}

// NewProcessWebhookEventUseCase's shipmentHandler may be nil -- when it is,
// "myparcel" events are persisted like any other provider's, with no
// further dispatch (e.g. a deployment that hasn't wired shipping yet).
func NewProcessWebhookEventUseCase(db *gorm.DB, eventRepo webhooks.WebhookEventRepository, shipmentHandler shipmentStatusWebhookHandler) *ProcessWebhookEventUseCase {
	return &ProcessWebhookEventUseCase{db: db, eventRepo: eventRepo, shipmentHandler: shipmentHandler}
}

func (uc *ProcessWebhookEventUseCase) Execute(ctx context.Context, envelope webhooks.Envelope) error {
	ctx = tenant.WithTenantID(ctx, envelope.TenantID)
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.eventRepo.Save(ctx, &webhooks.WebhookEvent{
			TenantID: envelope.TenantID, Provider: envelope.Provider, EventID: envelope.EventID,
			Topic: envelope.Headers[topicHeader], Payload: envelope.RawPayload, ReceivedAt: envelope.ReceivedAt,
		})
	})
	if err != nil {
		return fmt.Errorf("processWebhookEvent: %w", err)
	}

	if envelope.Provider == "myparcel" && uc.shipmentHandler != nil {
		if err := uc.shipmentHandler.Execute(ctx, envelope); err != nil {
			return fmt.Errorf("processWebhookEvent: shipment dispatch: %w", err)
		}
	}
	return nil
}
