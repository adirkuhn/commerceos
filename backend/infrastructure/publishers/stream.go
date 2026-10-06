// Package publishers holds NATS JetStream infrastructure shared by every
// process that publishes or consumes domain events (cmd/relay publishes,
// cmd/scheduler consumes) — there's exactly one JetStream deployment for
// this app, so one stream covering every event subject is enough.
package publishers

import (
	"errors"
	"fmt"

	"github.com/nats-io/nats.go"
)

// StreamName is the single JetStream stream every domain event is
// published to and consumed from.
const StreamName = "COMMERCEOS_EVENTS"

// subjects lists every domain prefix an event gets published under. A bare
// ">" wildcard would also capture NATS' own reserved subjects ($JS.>,
// _INBOX.>, etc.), which JetStream refuses unless acks are disabled
// entirely — so each domain prefix is listed explicitly instead. Add a new
// one here whenever a new domain starts publishing events (enforced by
// TestSubjectsCoverEveryDomainEvent); EnsureStream
// updates an already-existing stream's subject list too, so this doesn't
// require dropping the stream in a live deployment.
var subjects = []string{"integrations.>", "pms.>", "inventory.>", "oms.>", "shipping.>", "settings.>", "notifications.>", "payments.>", "procurement.>", "storefront.>", "collections.>", "newsletter.>", "suppliers.>",
	"apikeys.>", "channels.>", "coupons.>", "customers.>", "fx.>", "iam.>", "messagecenter.>", "promotions.>",
	"reviews.>", "scheduler.>", "tax.>", "theme.>", "webhook_forwarding.>", "whatsapp.>"}

// EnsureStream creates the stream if it doesn't exist yet, or widens its
// subject list if it exists but is missing one of `subjects` (e.g. a new
// domain started publishing events after the stream was first created).
// Safe to call from every process at startup (relay and scheduler both
// need it to exist before they can publish/subscribe).
func EnsureStream(js nats.JetStreamContext) error {
	info, err := js.StreamInfo(StreamName)
	if err != nil && !errors.Is(err, nats.ErrStreamNotFound) {
		return fmt.Errorf("publishers: check stream %s: %w", StreamName, err)
	}

	if info == nil {
		if _, err := js.AddStream(&nats.StreamConfig{
			Name: StreamName, Subjects: subjects, Storage: nats.FileStorage,
		}); err != nil {
			return fmt.Errorf("publishers: create stream %s: %w", StreamName, err)
		}
		return nil
	}

	if !hasAllSubjects(info.Config.Subjects, subjects) {
		cfg := info.Config
		cfg.Subjects = subjects
		if _, err := js.UpdateStream(&cfg); err != nil {
			return fmt.Errorf("publishers: update stream %s subjects: %w", StreamName, err)
		}
	}
	return nil
}

// DLQStreamName is the dedicated stream every consumer's exhausted
// (dead-lettered) message lands on -- separate from StreamName so a
// backlog of dead letters (which an operator triages manually, see
// infrastructure/consumers.nakOrDeadLetter) doesn't compete with live
// traffic for that stream's limits, and can be purged/inspected on its own.
const DLQStreamName = "COMMERCEOS_EVENTS_DLQ"

// dlqSubjects mirrors StreamName's subjects under the "dlq." prefix that
// infrastructure/consumers.deadLetterSubject publishes under.
// Derived, not hand-listed: a hand-kept copy had drifted (oms, shipping and
// settings were missing), so dead-letters from those consumers had no
// stream to land on.
var dlqSubjects = func() []string {
	out := make([]string, len(subjects))
	for i, s := range subjects {
		out[i] = "dlq." + s
	}
	return out
}()

// EnsureDLQStream is EnsureStream's counterpart for DLQStreamName. Called
// from cmd/scheduler, which is where every consumer that can dead-letter a
// message runs -- the stream must exist before any of them can publish to
// it.
func EnsureDLQStream(js nats.JetStreamContext) error {
	info, err := js.StreamInfo(DLQStreamName)
	if err != nil && !errors.Is(err, nats.ErrStreamNotFound) {
		return fmt.Errorf("publishers: check stream %s: %w", DLQStreamName, err)
	}

	if info == nil {
		if _, err := js.AddStream(&nats.StreamConfig{
			Name: DLQStreamName, Subjects: dlqSubjects, Storage: nats.FileStorage,
		}); err != nil {
			return fmt.Errorf("publishers: create stream %s: %w", DLQStreamName, err)
		}
		return nil
	}

	if !hasAllSubjects(info.Config.Subjects, dlqSubjects) {
		cfg := info.Config
		cfg.Subjects = dlqSubjects
		if _, err := js.UpdateStream(&cfg); err != nil {
			return fmt.Errorf("publishers: update stream %s subjects: %w", DLQStreamName, err)
		}
	}
	return nil
}

func hasAllSubjects(existing, want []string) bool {
	have := make(map[string]bool, len(existing))
	for _, s := range existing {
		have[s] = true
	}
	for _, s := range want {
		if !have[s] {
			return false
		}
	}
	return true
}
