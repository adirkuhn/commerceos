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

// ProcessWebhookEventUseCase is the cold-path landing point: durably
// persist the envelope cmd/webhook-processor consumed off
// COMMERCEOS_WEBHOOKS. Per-topic business-rule dispatch (e.g. actually
// updating a sales order from an order.updated webhook) is deliberately
// out of scope for this pass -- this use case's only job is "prove this
// event was received and accepted," idempotently.
type ProcessWebhookEventUseCase struct {
	db        *gorm.DB
	eventRepo webhooks.WebhookEventRepository
}

func NewProcessWebhookEventUseCase(db *gorm.DB, eventRepo webhooks.WebhookEventRepository) *ProcessWebhookEventUseCase {
	return &ProcessWebhookEventUseCase{db: db, eventRepo: eventRepo}
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
	return nil
}
