package consumers

import (
	"log"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/adirkuhn/commerceos/infrastructure/dlq"
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
	PublishMsg(m *nats.Msg, opts ...nats.PubOpt) (*nats.PubAck, error)
}

// deadLetterMsg is what a message that exhausted its redelivery attempts
// is republished as: the original data, byte-identical, under the
// dead-letter subject (stored on the dedicated DLQ stream, see
// infrastructure/publishers.EnsureDLQStream), with headers saying which
// consumer gave up, why, after how many deliveries and when -- so a dead
// letter can be triaged from the admin without searching the logs. Any
// retry target the message carried is not copied: a retry that fails again
// is a fresh failure of the consumer named in the new headers.
func deadLetterMsg(msg *nats.Msg, meta *nats.MsgMetadata, cause error, now time.Time) *nats.Msg {
	return &nats.Msg{
		Subject: dlq.Subject(msg.Subject),
		Data:    msg.Data,
		Header:  dlq.FailureHeader(meta.Consumer, msg.Subject, cause, meta.NumDelivered, now),
	}
}

// OnlyRetriesFor wraps a subscription's handler so a retried dead letter
// (see dlq.HeaderRetryConsumer) only reaches the consumer that failed it:
// every other consumer on the subject acks and skips it, so nothing that
// already handled the event runs again. consumer is the subscription's
// JetStream durable name -- the same name nakOrDeadLetter records. Every
// subscription wraps its handler with this, so no consumer repeats the check.
func OnlyRetriesFor(consumer string, handler nats.MsgHandler) nats.MsgHandler {
	return func(msg *nats.Msg) {
		if dlq.SkipsRetry(msg.Header, consumer) {
			ackOrLog(msg, consumer+": skipping retry for "+msg.Header.Get(dlq.HeaderRetryConsumer))
			return
		}
		handler(msg)
	}
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
func nakOrDeadLetter(publisher dlqPublisher, msg *nats.Msg, logPrefix string, cause error) {
	meta, metaErr := msg.Metadata()
	if metaErr != nil {
		// Not a real JetStream message (only happens in tests) -- no
		// delivery count to pace or dead-letter by, so just Nak.
		log.Printf("%s: %v", logPrefix, cause)
		nakOrLog(msg, logPrefix)
		return
	}
	if shouldDeadLetter(meta, metaErr) {
		log.Printf("%s: giving up after %d deliveries, dead-lettering: %v", logPrefix, meta.NumDelivered, cause)
		if _, err := publisher.PublishMsg(deadLetterMsg(msg, meta, cause, time.Now())); err != nil {
			log.Printf("%s: dead-letter publish failed, retrying instead: %v", logPrefix, err)
			nakOrLog(msg, logPrefix)
			return
		}
		ackOrLog(msg, logPrefix)
		return
	}
	log.Printf("%s: %v", logPrefix, cause)
	nakWithDelayOrLog(msg, backoffFor(meta.NumDelivered), logPrefix)
}
