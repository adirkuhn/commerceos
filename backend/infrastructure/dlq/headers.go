// Package dlq is the JetStream side of dead letters: the headers a failing
// consumer stamps on a dead letter, the rule that keeps a targeted retry
// away from consumers that already handled the event, and the adapter the
// dead-letter admin reads, retries and discards through.
package dlq

import (
	"strconv"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

// Headers a dead letter carries next to its byte-identical original data.
// Messages dead-lettered before these existed have none of them.
const (
	HeaderConsumer        = "Commerceos-Consumer"
	HeaderOriginalSubject = "Commerceos-Original-Subject"
	HeaderError           = "Commerceos-Error"
	HeaderNumDelivered    = "Commerceos-Num-Delivered"
	HeaderDeadLetteredAt  = "Commerceos-Dead-Lettered-At"

	// HeaderRetryConsumer marks a retried event: only the consumer named
	// here processes it, every other consumer on the subject skips it.
	HeaderRetryConsumer = "Commerceos-Retry-Consumer"
)

// MaxErrorLen keeps a dead letter's error header readable and well under
// NATS' header size limits.
const MaxErrorLen = 1000

// subjectPrefix is what a dead letter's subject starts with, on both DLQ
// streams (see infrastructure/publishers).
const subjectPrefix = "dlq."

// Subject is the dead-letter subject for an event's original subject.
func Subject(originalSubject string) string {
	return subjectPrefix + originalSubject
}

// OriginalSubject reverses Subject, for dead letters that predate
// HeaderOriginalSubject.
func OriginalSubject(deadLetterSubject string) string {
	return strings.TrimPrefix(deadLetterSubject, subjectPrefix)
}

// FailureHeader records which consumer gave up on a message, why, after how
// many deliveries and when. consumer is the JetStream durable name.
func FailureHeader(consumer, originalSubject string, cause error, numDelivered uint64, at time.Time) nats.Header {
	h := nats.Header{}
	h.Set(HeaderConsumer, consumer)
	h.Set(HeaderOriginalSubject, originalSubject)
	h.Set(HeaderError, errorText(cause))
	h.Set(HeaderNumDelivered, strconv.FormatUint(numDelivered, 10))
	h.Set(HeaderDeadLetteredAt, at.UTC().Format(time.RFC3339))
	return h
}

// errorText is cause as a single header-safe line of at most MaxErrorLen
// bytes -- a CR or LF would end the header early on the wire, and a cut
// through a multi-byte character is dropped.
func errorText(cause error) string {
	if cause == nil {
		return ""
	}
	text := strings.NewReplacer("\r", " ", "\n", " ").Replace(cause.Error())
	if len(text) > MaxErrorLen {
		text = strings.ToValidUTF8(text[:MaxErrorLen], "")
	}
	return text
}

// SkipsRetry reports whether consumer must skip this delivery: it is a
// retry aimed at a different consumer. Normal deliveries, and retries with
// no target, reach every consumer.
func SkipsRetry(h nats.Header, consumer string) bool {
	target := h.Get(HeaderRetryConsumer)
	return target != "" && target != consumer
}
