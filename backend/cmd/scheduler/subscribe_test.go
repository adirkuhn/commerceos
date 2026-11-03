package main

import (
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/infrastructure/dlq"
)

type stubJetStreamSubscriber struct {
	sub     *nats.Subscription
	err     error
	handler nats.MsgHandler
}

func (s *stubJetStreamSubscriber) Subscribe(_ string, cb nats.MsgHandler, _ ...nats.SubOpt) (*nats.Subscription, error) {
	s.handler = cb
	return s.sub, s.err
}

// TestSubscribeConsumer_skipsRetriesForOtherConsumers: every scheduler
// subscription goes through subscribeConsumer, so a dead letter retried for
// one consumer (COM-14) never reaches the others on the same subject.
func TestSubscribeConsumer_skipsRetriesForOtherConsumers(t *testing.T) {
	retryFor := func(durable string) *nats.Msg {
		m := &nats.Msg{Subject: "oms.sales_order.status_changed", Data: []byte(`{}`), Header: nats.Header{}}
		m.Header.Set(dlq.HeaderRetryConsumer, durable)
		return m
	}
	js := &stubJetStreamSubscriber{sub: &nats.Subscription{}}
	calls := 0

	subscribeConsumer(js, "oms.sales_order.status_changed", "whatsapp-order-status-scheduler", 5, []time.Duration{time.Second},
		func(*nats.Msg) { calls++ })
	js.handler(retryFor("sales-order-status-changed-scheduler"))
	js.handler(retryFor("whatsapp-order-status-scheduler"))
	js.handler(&nats.Msg{Subject: "oms.sales_order.status_changed", Data: []byte(`{}`)})

	assert.Equal(t, 2, calls, "only the retry for this durable and the normal delivery reach the handler")
}

func TestSubscribeConsumer_Success(t *testing.T) {
	want := &nats.Subscription{}
	js := &stubJetStreamSubscriber{sub: want}

	got := subscribeConsumer(js, "pms.product.changed", "product-changed-scheduler", 5, []time.Duration{time.Second}, nil)

	assert.Same(t, want, got)
}

// TestSubscribeConsumer_FailureDoesNotPanic is the regression this helper
// exists for: a subscribe failure (e.g. "no stream matches subject" from a
// stream missing the subject's prefix) must not crash the caller -- it used
// to be a log.Fatalf at every one of the 6 call sites in main(), which took
// the whole scheduler process down and, with it, every other consumer and
// the periodic sync-schedule ticker.
func TestSubscribeConsumer_FailureDoesNotPanic(t *testing.T) {
	js := &stubJetStreamSubscriber{err: errors.New("no stream matches subject")}

	got := subscribeConsumer(js, "shipping.shipment.fulfillment_signal", "shipment-fulfillment-signal-scheduler", 5, []time.Duration{time.Second}, nil)

	assert.Nil(t, got)
}
