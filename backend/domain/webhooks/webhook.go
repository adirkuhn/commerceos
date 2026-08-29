// Package webhooks holds the domain types for inbound third-party webhook
// ingestion: the envelope published to JetStream on the hot path, and the
// fallback-event record written when publishing that envelope fails after
// every in-request retry (see application/webhooks.ReceiveWebhookUseCase).
package webhooks

import "time"

// Envelope is the payload published to the COMMERCEOS_WEBHOOKS stream (and
// stored verbatim in a FallbackEvent on publish failure). RawPayload is the
// exact bytes the provider sent -- never re-serialized -- so a downstream
// consumer can re-verify or re-parse it exactly as received.
type Envelope struct {
	EventID    string
	TenantID   string
	Provider   string
	ReceivedAt time.Time
	RawPayload []byte
	Headers    map[string]string
}

// FallbackStatus is a FallbackEvent's lifecycle: PendingRepublish until an
// operator (or the retry action) succeeds in publishing it, at which point
// it's Republished; Abandoned is a manual "give up on this one" action.
type FallbackStatus string

const (
	FallbackStatusPendingRepublish FallbackStatus = "pending_republish"
	FallbackStatusRepublished      FallbackStatus = "republished"
	FallbackStatusAbandoned        FallbackStatus = "abandoned"
)

// WebhookEvent is the cold-path landing record: written once
// cmd/webhook-processor has successfully consumed an Envelope off
// COMMERCEOS_WEBHOOKS -- the durable, queryable proof the event was
// received and accepted. Per-topic business-rule dispatch exists only for
// myparcel (see application/webhooks.ProcessWebhookEventUseCase) -- Status
// records what happened with it, if anything.
type WebhookEvent struct {
	ID          string
	TenantID    string
	Provider    string
	EventID     string
	Topic       string
	Payload     []byte
	ReceivedAt  time.Time
	ProcessedAt time.Time
	// Status defaults to Received. Only a provider/topic with an actual
	// dispatch handler (myparcel today) ever moves it to Processed/Failed --
	// every other event stays Received forever, which is its normal,
	// successful terminal state (nothing more was ever going to happen to
	// it), not an unprocessed one.
	Status WebhookEventStatus
	// DispatchError is the dispatch handler's error message when Status is
	// Failed, empty otherwise.
	DispatchError string
}

// WebhookEventStatus is WebhookEvent's per-topic dispatch outcome -- see
// WebhookEvent.Status's doc comment.
type WebhookEventStatus string

const (
	WebhookEventStatusReceived  WebhookEventStatus = "received"
	WebhookEventStatusProcessed WebhookEventStatus = "processed"
	WebhookEventStatusFailed    WebhookEventStatus = "failed"
)

// TopicRef is one (provider, topic) pair this tenant has actually received
// at least one WebhookEvent for -- see
// WebhookEventRepository.DistinctTopics's doc comment.
type TopicRef struct {
	Provider string
	Topic    string
}

// FallbackEvent is the durable record of a webhook ReceiveWebhookUseCase
// could not hand off to JetStream after exhausting its publish retries --
// the "don't miss it" safety net for a NATS outage. Requires manual
// triage: republish (retry) or discard (abandon), see
// application/webhooks.RetryFallbackEventUseCase /
// AbandonFallbackEventUseCase.
type FallbackEvent struct {
	ID         string
	TenantID   string
	Provider   string
	EventID    string
	Payload    []byte
	Headers    map[string]string
	Status     FallbackStatus
	ReceivedAt time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
