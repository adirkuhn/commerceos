package webhooks

import (
	"context"
	"fmt"
	"log"

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

// DispatchHandler is one provider's per-topic business-rule dispatch,
// invoked after ProcessWebhookEventUseCase's own persist+forward step has
// durably landed the envelope. Exported (unlike a narrowed
// point-of-use interface) because the composition root (cmd/webhook-processor)
// needs to name this type to build the provider->handler map it hands to
// NewProcessWebhookEventUseCase -- e.g.
// application/shipping.HandleShipmentWebhookUseCase for "myparcel",
// application/whatsapp.HandleInboundMessageUseCase for "whatsapp". Adding a
// new provider's dispatch is a new map entry, never a new
// `if envelope.Provider == "x"` branch here (Open/Closed).
type DispatchHandler interface {
	Execute(ctx context.Context, envelope webhooks.Envelope) error
}

// ProcessWebhookEventUseCase is the cold-path landing point: durably
// persist the envelope cmd/webhook-processor consumed off
// COMMERCEOS_WEBHOOKS, then enqueue a ForwardDelivery for every active
// ForwardRule matching this provider/topic (the tenant's own configured
// fan-out -- see domain/webhooks.ForwardRule's doc comment) in the SAME
// transaction, so an event once durably received can never silently fail
// to be queued for forwarding. Per-topic business-rule dispatch beyond
// that is opt-in per provider -- see dispatchHandlers/DispatchHandler.
//
//events:none -- stores the inbound webhook (already the webhooks.received.<provider> event) and queues forward deliveries; the business change a dispatch handler makes raises its own events
type ProcessWebhookEventUseCase struct {
	db               *gorm.DB
	eventRepo        webhooks.WebhookEventRepository
	dispatchHandlers map[string]DispatchHandler
	forwardRuleRepo  webhooks.ForwardRuleRepository
	forwardRepo      webhooks.ForwardDeliveryRepository
}

// NewProcessWebhookEventUseCase's dispatchHandlers may be nil, or missing
// an entry for a given provider -- either way that provider's events are
// persisted like any other's, with no further dispatch (e.g. a deployment
// that hasn't wired that concern yet). forwardRuleRepo/forwardRepo may also
// both be nil -- when they are, no fan-out is attempted (a deployment that
// hasn't wired the forwarding feature yet); either both are nil or both are
// set, never one alone.
func NewProcessWebhookEventUseCase(
	db *gorm.DB, eventRepo webhooks.WebhookEventRepository, dispatchHandlers map[string]DispatchHandler,
	forwardRuleRepo webhooks.ForwardRuleRepository, forwardRepo webhooks.ForwardDeliveryRepository,
) *ProcessWebhookEventUseCase {
	return &ProcessWebhookEventUseCase{
		db: db, eventRepo: eventRepo, dispatchHandlers: dispatchHandlers,
		forwardRuleRepo: forwardRuleRepo, forwardRepo: forwardRepo,
	}
}

func (uc *ProcessWebhookEventUseCase) Execute(ctx context.Context, envelope webhooks.Envelope) error {
	ctx = tenant.WithTenantID(ctx, envelope.TenantID)
	topic := envelope.Headers[topicHeaderByProvider[envelope.Provider]]
	var event webhooks.WebhookEvent
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		event = webhooks.WebhookEvent{
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

	handler, ok := uc.dispatchHandlers[envelope.Provider]
	if !ok || handler == nil {
		return nil
	}

	if dispatchErr := handler.Execute(ctx, envelope); dispatchErr != nil {
		// Best-effort: the dispatch failure is what actually needs the
		// caller to nack/retry (Save above is idempotent, so redelivery is
		// safe) -- a failure recording that fact must not hide or replace
		// it.
		if markErr := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
			return uc.eventRepo.MarkDispatchResult(ctx, event.ID, webhooks.WebhookEventStatusFailed, dispatchErr.Error())
		}); markErr != nil {
			log.Printf("processWebhookEvent: record dispatch failure for %s: %v", event.ID, markErr)
		}
		return fmt.Errorf("processWebhookEvent: shipment dispatch: %w", dispatchErr)
	}

	if err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.eventRepo.MarkDispatchResult(ctx, event.ID, webhooks.WebhookEventStatusProcessed, "")
	}); err != nil {
		return fmt.Errorf("processWebhookEvent: mark processed: %w", err)
	}
	return nil
}
