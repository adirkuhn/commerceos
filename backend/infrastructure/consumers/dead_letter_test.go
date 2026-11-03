package consumers

import (
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/infrastructure/dlq"
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

// TestDeadLetterMsg pins what COM-14 needs to triage a dead letter without
// the logs: the data stays byte-identical, and the headers say which
// consumer gave up, why, after how many deliveries and when.
func TestDeadLetterMsg(t *testing.T) {
	data := []byte(`{"event_id":"e-1","tenant_id":"t-1"}`)
	orig := &nats.Msg{Subject: "oms.sales_order.status_changed", Data: data}
	meta := &nats.MsgMetadata{Consumer: "sales-order-status-changed-scheduler", NumDelivered: 5}
	at := time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)

	got := deadLetterMsg(orig, meta, errors.New("woo: 403"), at)

	assert.Equal(t, "dlq.oms.sales_order.status_changed", got.Subject)
	assert.Equal(t, data, got.Data)
	assert.Equal(t, "sales-order-status-changed-scheduler", got.Header.Get(dlq.HeaderConsumer))
	assert.Equal(t, "oms.sales_order.status_changed", got.Header.Get(dlq.HeaderOriginalSubject))
	assert.Equal(t, "woo: 403", got.Header.Get(dlq.HeaderError))
	assert.Equal(t, "5", got.Header.Get(dlq.HeaderNumDelivered))
	assert.Equal(t, "2026-11-03T10:00:00Z", got.Header.Get(dlq.HeaderDeadLetteredAt))
}

// TestDeadLetterMsg_dropsRetryTarget: a retried event that fails again is
// dead-lettered as a fresh failure of the consumer that just failed it.
func TestDeadLetterMsg_dropsRetryTarget(t *testing.T) {
	orig := &nats.Msg{Subject: "s", Data: []byte(`{}`), Header: nats.Header{}}
	orig.Header.Set(dlq.HeaderRetryConsumer, "c")

	got := deadLetterMsg(orig, &nats.MsgMetadata{Consumer: "c", NumDelivered: 5}, errors.New("boom"), time.Now())

	assert.Equal(t, "", got.Header.Get(dlq.HeaderRetryConsumer))
}

// TestOnlyRetriesFor is the single place a targeted retry is kept away
// from consumers that already handled the event (COM-14): every
// subscription wraps its handler with it.
func TestOnlyRetriesFor(t *testing.T) {
	retryFor := func(consumer string) *nats.Msg {
		m := &nats.Msg{Subject: "oms.sales_order.status_changed", Data: []byte(`{}`), Header: nats.Header{}}
		m.Header.Set(dlq.HeaderRetryConsumer, consumer)
		return m
	}
	cases := []struct {
		name       string
		msg        *nats.Msg
		wantCalled bool
	}{
		{"normal delivery", &nats.Msg{Subject: "s", Data: []byte(`{}`)}, true},
		{"retry for this consumer", retryFor("whatsapp-order-status-scheduler"), true},
		{"retry for another consumer -- acked and skipped", retryFor("sales-order-status-changed-scheduler"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			h := OnlyRetriesFor("whatsapp-order-status-scheduler", func(*nats.Msg) { called = true })

			h(tc.msg)

			assert.Equal(t, tc.wantCalled, called)
		})
	}
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

func (s *stubDLQPublisher) PublishMsg(m *nats.Msg, _ ...nats.PubOpt) (*nats.PubAck, error) {
	s.calls = append(s.calls, m.Subject)
	return &nats.PubAck{}, nil
}
