// Package eventrecordtest is a fake outbox for use-case tests: it keeps
// every recorded event, decoded, so a test can assert what was raised.
package eventrecordtest

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Event is one recorded event: its subject and decoded JSON payload.
type Event struct {
	Subject string
	Payload map[string]any
}

// Outbox implements eventrecord.Enqueuer. Err, when set, fails every enqueue.
type Outbox struct {
	Events []Event
	Err    error
}

func (o *Outbox) Enqueue(_ context.Context, subject, _ string, payload []byte, _ time.Time) error {
	if o.Err != nil {
		return o.Err
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return fmt.Errorf("eventrecordtest: %w", err)
	}
	o.Events = append(o.Events, Event{Subject: subject, Payload: decoded})
	return nil
}

// Subjects lists the recorded subjects in order.
func (o *Outbox) Subjects() []string {
	out := make([]string, len(o.Events))
	for i, e := range o.Events {
		out[i] = e.Subject
	}
	return out
}

// Only returns the single recorded event, or an error naming what was recorded.
func (o *Outbox) Only() (Event, error) {
	if len(o.Events) != 1 {
		return Event{}, fmt.Errorf("want exactly 1 event, got %v", o.Subjects())
	}
	return o.Events[0], nil
}

// Changes is the event's "changes" as field -> [before, after].
func (e Event) Changes() map[string][2]any {
	out := map[string][2]any{}
	raw, _ := e.Payload["changes"].(map[string]any)
	for field, v := range raw {
		c, _ := v.(map[string]any)
		out[field] = [2]any{c["before"], c["after"]}
	}
	return out
}

// Data is the event's "data" object.
func (e Event) Data() map[string]any {
	d, _ := e.Payload["data"].(map[string]any)
	return d
}
