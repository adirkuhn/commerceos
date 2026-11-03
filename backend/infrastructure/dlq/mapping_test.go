package dlq

import (
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/domain/deadletter"
)

func TestToDeadLetter_withFailureHeaders(t *testing.T) {
	at := time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)
	stored := at.Add(time.Second)
	data := []byte(`{"event_id":"e-1","tenant_id":"t-1"}`)
	raw := &jetstream.RawStreamMsg{
		Subject: "dlq.oms.sales_order.status_changed", Sequence: 7, Data: data, Time: stored,
		Header: FailureHeader("sales-order-status-changed-scheduler", "oms.sales_order.status_changed", assertErr("woo: 403"), 5, at),
	}

	got := toDeadLetter(deadletter.StreamEvents, raw)

	assert.Equal(t, deadletter.DeadLetter{
		Stream: deadletter.StreamEvents, Sequence: 7, Subject: "oms.sales_order.status_changed",
		Consumer: "sales-order-status-changed-scheduler", EventID: "e-1", TenantID: "t-1",
		Error: "woo: 403", NumDelivered: 5, DeadLetteredAt: &at, StoredAt: stored, Data: data,
	}, got)
}

// Dead letters from before COM-14 carry no headers: still listed, subject
// taken from the dead-letter subject, consumer unknown.
func TestToDeadLetter_withoutHeaders(t *testing.T) {
	raw := &jetstream.RawStreamMsg{Subject: "dlq.pms.product.changed", Sequence: 3, Data: []byte(`{"event_id":"e-2","tenant_id":"t-2"}`)}

	got := toDeadLetter(deadletter.StreamEvents, raw)

	assert.Equal(t, "pms.product.changed", got.Subject)
	assert.False(t, got.ConsumerKnown())
	assert.Equal(t, "e-2", got.EventID)
	assert.Equal(t, 0, got.NumDelivered)
	assert.Nil(t, got.DeadLetteredAt)
}

func TestRetryMsg_targetsTheFailedConsumer(t *testing.T) {
	d := deadletter.DeadLetter{Subject: "oms.sales_order.status_changed", Consumer: "c1", Data: []byte(`{}`)}

	got := retryMsg(d)

	assert.Equal(t, "oms.sales_order.status_changed", got.Subject)
	assert.Equal(t, d.Data, got.Data)
	assert.Equal(t, "c1", got.Header.Get(HeaderRetryConsumer))
}

func TestRetryMsg_unknownConsumerReachesEveryone(t *testing.T) {
	got := retryMsg(deadletter.DeadLetter{Subject: "s", Data: []byte(`{}`)})

	assert.Equal(t, nats.Header{}, got.Header)
}

type assertErr string

func (e assertErr) Error() string { return string(e) }
