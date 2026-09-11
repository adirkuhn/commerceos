package main

import (
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

// jetStreamSubscriber is the narrow slice of nats.JetStreamContext that
// subscribeConsumer needs -- defined at point of use per this repo's
// interface-segregation rule, instead of depending on the full
// JetStreamContext (which also carries stream/KV/object-store management
// methods no caller here needs).
type jetStreamSubscriber interface {
	Subscribe(subj string, cb nats.MsgHandler, opts ...nats.SubOpt) (*nats.Subscription, error)
}

// subscribeConsumer registers one durable, manually-acked JetStream
// consumer and isolates its registration failure from every other consumer
// and from the periodic sync-schedule ticker started later in main().
//
// Before this existed, every call site did `if err != nil { log.Fatalf(...) }`
// inline -- so a single missing stream subject (see publishers.subjects)
// crash-looped the *entire* scheduler process on startup, which silently
// starved every other consumer already wired above it and, because the
// poll ticker is only reached after all consumers finish registering,
// permanently halted every scheduled Woo/Bol sync too -- not just the one
// consumer whose subject was missing. See the 2026-09-10 incident where a
// missing "shipping.>" stream subject took down order-status sync,
// stock-to-channel sync, and Woo/Bol order import simultaneously.
//
// A failed subscribe here is logged and that one consumer is disabled
// until the next process restart (which retries registration
// automatically) -- it no longer takes the rest of the process down with
// it.
func subscribeConsumer(js jetStreamSubscriber, subject, durable string, maxDeliver int, backoff []time.Duration, handler nats.MsgHandler) *nats.Subscription {
	sub, err := js.Subscribe(subject, handler,
		nats.Durable(durable), nats.ManualAck(),
		nats.MaxDeliver(maxDeliver), nats.BackOff(backoff))
	if err != nil {
		log.Printf("subscribe %s (durable=%s): %v -- consumer disabled until next restart", subject, durable, err)
		return nil
	}
	return sub
}
