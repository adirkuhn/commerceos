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

// EnsureStream creates the stream if it doesn't exist yet. Safe to call
// from every process at startup (relay and scheduler both need it to
// exist before they can publish/subscribe) — a stream that already exists
// is left untouched.
func EnsureStream(js nats.JetStreamContext) error {
	_, err := js.StreamInfo(StreamName)
	if err == nil {
		return nil
	}
	if !errors.Is(err, nats.ErrStreamNotFound) {
		return fmt.Errorf("publishers: check stream %s: %w", StreamName, err)
	}

	_, err = js.AddStream(&nats.StreamConfig{
		Name: StreamName,
		// Every domain event subject is "{domain}.{entity}.{verb}" (see
		// CLAUDE.md's naming convention) — a bare ">" wildcard would also
		// capture NATS' own reserved subjects ($JS.>, _INBOX.>, etc.),
		// which JetStream refuses unless acks are disabled entirely. List
		// each domain prefix explicitly instead; add a new one here
		// whenever a new domain starts publishing events.
		Subjects: []string{"integrations.>", "pms.>"},
		Storage:  nats.FileStorage,
	})
	if err != nil {
		return fmt.Errorf("publishers: create stream %s: %w", StreamName, err)
	}
	return nil
}
