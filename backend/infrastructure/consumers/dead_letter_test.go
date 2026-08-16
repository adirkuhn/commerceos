package consumers

import (
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
)

func TestShouldDeadLetter(t *testing.T) {
	cases := []struct {
		name    string
		meta    *nats.MsgMetadata
		metaErr error
		want    bool
	}{
		{"below the threshold", &nats.MsgMetadata{NumDelivered: 1}, nil, false},
		{"one below the threshold", &nats.MsgMetadata{NumDelivered: MaxDeliverBeforeDeadLetter - 1}, nil, false},
		{"exactly at the threshold", &nats.MsgMetadata{NumDelivered: MaxDeliverBeforeDeadLetter}, nil, true},
		{"past the threshold", &nats.MsgMetadata{NumDelivered: MaxDeliverBeforeDeadLetter + 10}, nil, true},
		{"metadata unreadable -- not a real JetStream message (test harness)", nil, errors.New("nats: not a jetstream message"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, shouldDeadLetter(tc.meta, tc.metaErr))
		})
	}
}

// TestBackoffFor is the exact bug caught live: a plain msg.Nak() ignores
// the consumer's configured BackOff and redelivers immediately, so pacing
// has to come from an explicit NakWithDelay(backoffFor(...)) instead.
func TestBackoffFor(t *testing.T) {
	cases := []struct {
		name         string
		numDelivered uint64
		want         time.Duration
	}{
		{"first delivery -- delay before the 2nd attempt", 1, BackoffSchedule[0]},
		{"second delivery -- delay before the 3rd attempt", 2, BackoffSchedule[1]},
		{"last scheduled entry", uint64(len(BackoffSchedule)), BackoffSchedule[len(BackoffSchedule)-1]},
		{"beyond the schedule -- reuses the last entry", uint64(len(BackoffSchedule)) + 10, BackoffSchedule[len(BackoffSchedule)-1]},
		{"zero (shouldn't happen -- JetStream numbers deliveries from 1) -- clamped, not a panic", 0, BackoffSchedule[0]},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, backoffFor(tc.numDelivered))
		})
	}
}

func TestDeadLetterSubject(t *testing.T) {
	assert.Equal(t, "dlq.pms.product.changed", deadLetterSubject("pms.product.changed"))
	assert.Equal(t, "dlq.inventory.stock.changed", deadLetterSubject("inventory.stock.changed"))
}

// TestNakOrDeadLetter_fallsBackToNakWhenNotARealJetStreamMessage documents
// the safe fallback every existing consumer test relies on: a bare
// *nats.Msg{} built by hand in a test (no real Sub/Reply from a live
// JetStream subscription) always falls through to a plain Nak, exactly like
// the pre-DLQ behavior -- the dead-letter branch is only reachable with a
// real JetStream-delivered message, which none of this package's unit
// tests construct (that would require either a live NATS server or faking
// *nats.Subscription internals deep enough to risk a nil-pointer panic
// inside nats.go's own Ack/Nak implementation).
func TestNakOrDeadLetter_fallsBackToNakWhenNotARealJetStreamMessage(t *testing.T) {
	dlq := &stubDLQPublisher{}
	msg := &nats.Msg{Subject: "pms.product.changed", Data: []byte(`{}`)}

	nakOrDeadLetter(dlq, msg, "test", errors.New("boom"))

	assert.Empty(t, dlq.calls, "must not dead-letter when delivery count can't be determined")
}

type stubDLQPublisher struct {
	calls []string
}

func (s *stubDLQPublisher) Publish(subj string, _ []byte, _ ...nats.PubOpt) (*nats.PubAck, error) {
	s.calls = append(s.calls, subj)
	return &nats.PubAck{}, nil
}
