package main

import (
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
)

type stubJetStreamSubscriber struct {
	sub *nats.Subscription
	err error
}

func (s *stubJetStreamSubscriber) Subscribe(_ string, _ nats.MsgHandler, _ ...nats.SubOpt) (*nats.Subscription, error) {
	return s.sub, s.err
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
