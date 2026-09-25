package consumers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// automationEvaluator is application/messagecenter.EvaluateAutomationRulesUseCase
// narrowed to point of use (ISP).
type automationEvaluator interface {
	Execute(ctx context.Context, subject, eventID string, payload map[string]any) error
}

// AutomationEngineConsumer is the one Go type behind every automation-rule
// trigger subject. Unlike every other consumer in this package (one struct
// per event type), a merchant's AutomationRule.TriggerSubject is itself the
// variable part (see domain/messagecenter.KnownTriggerSubjects) -- so the
// SOLID-O extension point here is the subject string passed to
// NewAutomationEngineConsumer, not a new Go type. Each of the known
// subjects still gets its own JetStream subscription, durable name, and
// processed_events consumer key (see cmd/scheduler/main.go's wiring, one
// subscribeConsumer call per subject) -- only the handler code itself is
// shared, not delivery tracking, matching this package's existing
// "independent consumers on the same subject" convention.
//
// Unlike every other consumer, this one unmarshals into a generic
// map[string]any rather than one event struct -- AutomationRule.Matches
// needs the whole payload keyed by JSON field name regardless of which of
// the 10 trigger subjects it came from, and event_id/tenant_id are read
// off that same map (every event's Base/TenantID both marshal under those
// exact keys, see domain/events).
type AutomationEngineConsumer struct {
	subject   string
	processed events.ProcessedEventRepository
	evaluator automationEvaluator
	dlq       dlqPublisher
}

func NewAutomationEngineConsumer(subject string, processed events.ProcessedEventRepository, evaluator automationEvaluator, dlq dlqPublisher) *AutomationEngineConsumer {
	return &AutomationEngineConsumer{subject: subject, processed: processed, evaluator: evaluator, dlq: dlq}
}

func (c *AutomationEngineConsumer) consumerName() string {
	return "automation-engine-" + c.subject
}

func (c *AutomationEngineConsumer) Handle(msg *nats.Msg) {
	var payload map[string]any
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		log.Printf("automationEngineConsumer(%s): invalid payload, dropping (not retryable): %v", c.subject, err)
		ackOrLog(msg, "automationEngineConsumer("+c.subject+")")
		return
	}

	eventID, _ := payload["event_id"].(string)
	tenantID, _ := payload["tenant_id"].(string)
	if eventID == "" || tenantID == "" {
		log.Printf("automationEngineConsumer(%s): payload missing event_id/tenant_id, dropping (not retryable)", c.subject)
		ackOrLog(msg, "automationEngineConsumer("+c.subject+")")
		return
	}

	ctx := tenant.WithTenantID(context.Background(), tenantID)

	alreadyDone, err := c.processed.IsProcessed(ctx, eventID, c.consumerName())
	if err != nil {
		nakOrDeadLetter(c.dlq, msg, "automationEngineConsumer("+c.subject+"): check processed "+eventID, err)
		return
	}
	if alreadyDone {
		ackOrLog(msg, "automationEngineConsumer("+c.subject+")")
		return
	}

	if err := c.evaluator.Execute(ctx, c.subject, eventID, payload); err != nil {
		nakOrDeadLetter(c.dlq, msg, "automationEngineConsumer("+c.subject+"): evaluate "+eventID, err)
		return
	}

	if err := c.processed.MarkProcessed(ctx, eventID, c.consumerName()); err != nil {
		// safe to retry: Execute re-sends on redelivery -- an occasional
		// duplicate is a much smaller harm than silently dropping one, same
		// tradeoff every other consumer in this package makes.
		nakOrDeadLetter(c.dlq, msg, "automationEngineConsumer("+c.subject+"): mark processed "+eventID, err)
		return
	}

	ackOrLog(msg, "automationEngineConsumer("+c.subject+")")
}
