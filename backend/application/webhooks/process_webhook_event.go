package webhooks

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/domain/webhooks"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// topicHeaderByProvider maps provider -> the (canonical-cased) HTTP header
// that provider's webhook request sends its event topic/hook-type under --
// best-effort only, never verified or relied on for correctness (see the
// events themselves for the authoritative payload). Used both so an
// operator browsing WebhookEvent rows can tell what kind of event each one
// was without opening the payload, AND for ForwardRule.Match's per-topic
// fan-out matching -- a provider missing here (or a request missing the
// header) always lands with Topic == "", which only ever matches a
// provider-wide wildcard rule (Topic == ""), never a topic-specific one.
// WooCommerce docs: Webhooks -> Delivery -- X-Wc-Webhook-Topic. MyParcel:
// developer.myparcel.nl/api-reference/11.webhook-object-definitions.html --
// X-MyParcel-Hook, canonicalized by net/http to X-Myparcel-Hook.
var topicHeaderByProvider = map[string]string{
	"woocommerce": "X-Wc-Webhook-Topic",
	"myparcel":    "X-Myparcel-Hook",
}

// shipmentStatusWebhookHandler is application/shipping.HandleShipmentWebhookUseCase
// narrowed to point of use (ISP) -- lets this package stay decoupled from
// shipping's own dependencies (MyParcel client, oms repo, etc) and keeps
// this use case unit-testable without them.
type shipmentStatusWebhookHandler interface {
	Execute(ctx context.Context, envelope webhooks.Envelope) error
}

// ProcessWebhookEventUseCase is the cold-path landing point: durably
// persist the envelope cmd/webhook-processor consumed off
// COMMERCEOS_WEBHOOKS, then enqueue a ForwardDelivery for every active
// ForwardRule matching this provider/topic (the tenant's own configured
// fan-out -- see domain/webhooks.ForwardRule's doc comment) in the SAME
// transaction, so an event once durably received can never silently fail
// to be queued for forwarding. Per-topic business-rule dispatch is
// deliberately out of scope for this pass for every provider EXCEPT
// "myparcel" -- see shipmentHandler below, the one exception, added when
// the MyParcel shipping integration needed webhook-driven order updates.
type ProcessWebhookEventUseCase struct {
	db              *gorm.DB
	eventRepo       webhooks.WebhookEventRepository
	shipmentHandler shipmentStatusWebhookHandler
	forwardRuleRepo webhooks.ForwardRuleRepository
	forwardRepo     webhooks.ForwardDeliveryRepository
}

// NewProcessWebhookEventUseCase's shipmentHandler may be nil -- when it is,
// "myparcel" events are persisted like any other provider's, with no
// further dispatch (e.g. a deployment that hasn't wired shipping yet).
// forwardRuleRepo/forwardRepo may also both be nil -- when they are, no
// fan-out is attempted (a deployment that hasn't wired the forwarding
// feature yet); either both are nil or both are set, never one alone.
func NewProcessWebhookEventUseCase(
	db *gorm.DB, eventRepo webhooks.WebhookEventRepository, shipmentHandler shipmentStatusWebhookHandler,
	forwardRuleRepo webhooks.ForwardRuleRepository, forwardRepo webhooks.ForwardDeliveryRepository,
) *ProcessWebhookEventUseCase {
	return &ProcessWebhookEventUseCase{
		db: db, eventRepo: eventRepo, shipmentHandler: shipmentHandler,
		forwardRuleRepo: forwardRuleRepo, forwardRepo: forwardRepo,
	}
}

func (uc *ProcessWebhookEventUseCase) Execute(ctx context.Context, envelope webhooks.Envelope) error {
	ctx = tenant.WithTenantID(ctx, envelope.TenantID)
	topic := envelope.Headers[topicHeaderByProvider[envelope.Provider]]
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		event := webhooks.WebhookEvent{
			TenantID: envelope.TenantID, Provider: envelope.Provider, EventID: envelope.EventID,
			Topic: topic, Payload: envelope.RawPayload, ReceivedAt: envelope.ReceivedAt,
		}
		if err := uc.eventRepo.Save(ctx, &event); err != nil {
			return err
		}
		if uc.forwardRuleRepo == nil || uc.forwardRepo == nil {
			return nil
		}
		rules, err := uc.forwardRuleRepo.Match(ctx, envelope.Provider, topic)
		if err != nil {
			return fmt.Errorf("match forward rules: %w", err)
		}
		for _, rule := range rules {
			delivery := webhooks.ForwardDelivery{
				TenantID: envelope.TenantID, RuleID: rule.ID, WebhookEventID: event.ID,
				Provider: envelope.Provider, Topic: topic, TargetURL: rule.TargetURL, Payload: envelope.RawPayload,
			}
			if err := uc.forwardRepo.Create(ctx, &delivery); err != nil {
				return fmt.Errorf("enqueue forward delivery for rule %s: %w", rule.ID, err)
			}
		}
		return nil
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
