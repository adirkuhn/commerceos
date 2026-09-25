package consumers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/domain/webhooks"
)

// webhookProcessor is application/webhooks.ProcessWebhookEventUseCase
// narrowed to point of use (ISP) -- lets WebhookConsumer be unit tested
// without a real DB.
type webhookProcessor interface {
	Execute(ctx context.Context, envelope webhooks.Envelope) error
}

const webhookConsumerName = "webhook-processor"

// WebhookConsumer is the cold-path handler for every message on
// COMMERCEOS_WEBHOOKS (subject "webhooks.received.*", one subscription
// covers every provider) -- decode the envelope, dedupe via
// ProcessedEventRepository (same idempotency convention as every other
// consumer in this package: JetStream is at-least-once delivery), hand it
// to the processor, ack only after success.
type WebhookConsumer struct {
	processed events.ProcessedEventRepository
	processor webhookProcessor
	dlq       dlqPublisher
}

func NewWebhookConsumer(processed events.ProcessedEventRepository, processor webhookProcessor, dlq dlqPublisher) *WebhookConsumer {
	return &WebhookConsumer{processed: processed, processor: processor, dlq: dlq}
}

func (c *WebhookConsumer) Handle(msg *nats.Msg) {
	var envelope webhooks.Envelope
	if err := json.Unmarshal(msg.Data, &envelope); err != nil {
		log.Printf("webhookConsumer: invalid envelope, dropping (not retryable): %v", err)
		ackOrLog(msg, "webhookConsumer")
		return
	}

	ctx := context.Background()

	// processed_events.event_id is a UUID column (shared with every other
	// consumer in this package, all of which raise proper UUID EventIDs) --
	// a webhook's own EventID is a raw-body hash (see
	// infrastructure/webhookproviders.WooCommerceVerifier's doc comment),
	// not a UUID, so it can't be inserted there directly. NewSHA1 derives a
	// deterministic, valid UUID from it instead of altering that shared
	// table's column type.
	dedupeID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(envelope.EventID)).String()

	alreadyDone, err := c.processed.IsProcessed(ctx, dedupeID, webhookConsumerName)
	if err != nil {
		nakOrDeadLetter(c.dlq, msg, "webhookConsumer: check processed "+envelope.EventID, err)
		return
	}
	if alreadyDone {
		ackOrLog(msg, "webhookConsumer")
		return
	}

	if err := c.processor.Execute(ctx, envelope); err != nil {
		nakOrDeadLetter(c.dlq, msg, "webhookConsumer: process "+envelope.EventID, err)
		return
	}

	if err := c.processed.MarkProcessed(ctx, dedupeID, webhookConsumerName); err != nil {
		// safe to retry: ProcessWebhookEventUseCase's Save is itself
		// idempotent (upsert on tenant_id/provider/event_id), so re-running
		// it on redelivery is harmless.
		nakOrDeadLetter(c.dlq, msg, "webhookConsumer: mark processed "+envelope.EventID, err)
		return
	}

	ackOrLog(msg, "webhookConsumer")
}
