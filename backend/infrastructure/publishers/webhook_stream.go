package publishers

import (
	"errors"
	"fmt"

	"github.com/nats-io/nats.go"
)

// WebhookStreamName is a SEPARATE stream from StreamName (COMMERCEOS_EVENTS)
// -- inbound third-party webhook volume can be large and bursty and
// shouldn't compete for that stream's retention/limits with internal
// domain events. It must stay separate at the subject level too: JetStream
// refuses to let two streams claim overlapping subjects, and StreamName
// already owns "integrations.>", so webhook subjects live under their own
// "webhooks.>" prefix rather than "integrations.webhook.>" as originally
// sketched.
const WebhookStreamName = "COMMERCEOS_WEBHOOKS"

var webhookSubjects = []string{"webhooks.>"}

// EnsureWebhookStream mirrors EnsureStream for WebhookStreamName. Called by
// every process that publishes or consumes webhook envelopes (cmd/api,
// cmd/webhook-processor).
func EnsureWebhookStream(js nats.JetStreamContext) error {
	info, err := js.StreamInfo(WebhookStreamName)
	if err != nil && !errors.Is(err, nats.ErrStreamNotFound) {
		return fmt.Errorf("publishers: check stream %s: %w", WebhookStreamName, err)
	}

	if info == nil {
		if _, err := js.AddStream(&nats.StreamConfig{
			Name: WebhookStreamName, Subjects: webhookSubjects, Storage: nats.FileStorage,
		}); err != nil {
			return fmt.Errorf("publishers: create stream %s: %w", WebhookStreamName, err)
		}
		return nil
	}

	if !hasAllSubjects(info.Config.Subjects, webhookSubjects) {
		cfg := info.Config
		cfg.Subjects = webhookSubjects
		if _, err := js.UpdateStream(&cfg); err != nil {
			return fmt.Errorf("publishers: update stream %s subjects: %w", WebhookStreamName, err)
		}
	}
	return nil
}

// WebhookDLQStreamName is WebhookStreamName's dead-letter counterpart, same
// reasoning as DLQStreamName.
const WebhookDLQStreamName = "COMMERCEOS_WEBHOOKS_DLQ"

var webhookDLQSubjects = []string{"dlq.webhooks.>"}

// EnsureWebhookDLQStream mirrors EnsureDLQStream for WebhookDLQStreamName.
// Called from cmd/webhook-processor, where webhook consumers run.
func EnsureWebhookDLQStream(js nats.JetStreamContext) error {
	info, err := js.StreamInfo(WebhookDLQStreamName)
	if err != nil && !errors.Is(err, nats.ErrStreamNotFound) {
		return fmt.Errorf("publishers: check stream %s: %w", WebhookDLQStreamName, err)
	}

	if info == nil {
		if _, err := js.AddStream(&nats.StreamConfig{
			Name: WebhookDLQStreamName, Subjects: webhookDLQSubjects, Storage: nats.FileStorage,
		}); err != nil {
			return fmt.Errorf("publishers: create stream %s: %w", WebhookDLQStreamName, err)
		}
		return nil
	}

	if !hasAllSubjects(info.Config.Subjects, webhookDLQSubjects) {
		cfg := info.Config
		cfg.Subjects = webhookDLQSubjects
		if _, err := js.UpdateStream(&cfg); err != nil {
			return fmt.Errorf("publishers: update stream %s subjects: %w", WebhookDLQStreamName, err)
		}
	}
	return nil
}
