package consumers

import (
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

// MaxDeliverBeforeDeadLetter caps how many times JetStream redelivers a
// message to any of this package's consumers before nakOrDeadLetter gives
// up on it. Must be passed as nats.MaxDeliver(MaxDeliverBeforeDeadLetter)
// when each durable consumer is created (see cmd/scheduler/main.go) --
// JetStream itself stops redelivering past that point regardless, so a
// mismatch here just means dead-lettering happens a delivery or two early
// (harmless) rather than a message JetStream has already given up on
// silently disappearing with nowhere for it to land.
const MaxDeliverBeforeDeadLetter = 5

// BackoffSchedule paces redelivery of a failing message -- flat, immediate
// redelivery (the previous behavior, effectively AckWait's ~30s default)
// just hammers a downstream dependency that's already erroring. One entry
// per allowed delivery; JetStream reuses the last entry for any delivery
// beyond len(BackoffSchedule), so this only needs len <= MaxDeliverBeforeDeadLetter.
var BackoffSchedule = []time.Duration{
	5 * time.Second,
	30 * time.Second,
	2 * time.Minute,
	10 * time.Minute,
	30 * time.Minute,
}

// dlqPublisher is nats.JetStreamContext narrowed to point of use (ISP) --
// lets nakOrDeadLetter be unit tested without a real NATS connection.
type dlqPublisher interface {
	Publish(subj string, data []byte, opts ...nats.PubOpt) (*nats.PubAck, error)
}

// deadLetterSubject is where a message goes once it has exhausted its
// redelivery attempts, republished on the same COMMERCEOS_EVENTS stream
// under a filterable prefix (see infrastructure/publishers.EnsureDLQStream
// for the dedicated stream that actually stores these).
func deadLetterSubject(originalSubject string) string {
	return "dlq." + originalSubject
}

// shouldDeadLetter decides, from a message's own JetStream delivery count,
// whether a consumer should give up and dead-letter it instead of nacking
// for another redelivery. A message whose Metadata couldn't be read (e.g.
// not a real JetStream message, which only happens in tests) is treated as
// not yet exhausted -- the same as today's plain-Nak behavior -- so it's
// always safe to fall through to Nak.
func shouldDeadLetter(meta *nats.MsgMetadata, metaErr error) bool {
	return metaErr == nil && meta.NumDelivered >= MaxDeliverBeforeDeadLetter
}

// backoffFor picks BackoffSchedule's delay for a message's next redelivery
// attempt from how many times it's already been delivered (1 on the first
// delivery). Plain msg.Nak() asks JetStream to redeliver immediately,
// ignoring the consumer's configured BackOff entirely -- BackOff only
// governs redelivery timing for a message that's never acked/nacked and
// simply times out (AckWait elapses). Since every consumer here always
// explicitly Naks, pacing has to be applied by hand via NakWithDelay, or a
// failing handler hammers its downstream dependency at whatever rate
// messages arrive, immediately re-delivering all the way up to
// MaxDeliverBeforeDeadLetter within the same instant -- confirmed live: an
// unpaced consumer burned all 5 deliveries in well under a second.
func backoffFor(numDelivered uint64) time.Duration {
	idx := int(numDelivered) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(BackoffSchedule) {
		idx = len(BackoffSchedule) - 1
	}
	return BackoffSchedule[idx]
}

// nakOrDeadLetter is what every consumer in this package calls instead of a
// bare msg.Nak() on a retryable failure. Past MaxDeliverBeforeDeadLetter
// redeliveries -- meaning this handler has already proven, repeatedly, that
// it can't process this message -- it republishes the raw payload to this
// event's dead-letter subject and Acks the original (removing it from the
// main stream) instead of nacking it into a redelivery that's already known
// to fail every time. Below that, it nacks with an explicit delay from
// BackoffSchedule (see backoffFor) rather than a bare Nak, which would
// otherwise redeliver immediately. logPrefix identifies the calling
// consumer in the log line, matching each consumer's existing log style.
func nakOrDeadLetter(dlq dlqPublisher, msg *nats.Msg, logPrefix string, cause error) {
	meta, metaErr := msg.Metadata()
	if metaErr != nil {
		// Not a real JetStream message (only happens in tests) -- no
		// delivery count to pace or dead-letter by, so just Nak.
		log.Printf("%s: %v", logPrefix, cause)
		_ = msg.Nak()
		return
	}
	if shouldDeadLetter(meta, metaErr) {
		log.Printf("%s: giving up after %d deliveries, dead-lettering: %v", logPrefix, meta.NumDelivered, cause)
		if _, err := dlq.Publish(deadLetterSubject(msg.Subject), msg.Data); err != nil {
			log.Printf("%s: dead-letter publish failed, retrying instead: %v", logPrefix, err)
			_ = msg.Nak()
			return
		}
		_ = msg.Ack()
		return
	}
	log.Printf("%s: %v", logPrefix, cause)
	_ = msg.NakWithDelay(backoffFor(meta.NumDelivered))
}
