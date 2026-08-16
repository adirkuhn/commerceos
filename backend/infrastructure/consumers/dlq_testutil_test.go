package consumers_test

import "github.com/nats-io/nats.go"

// stubDLQPublisher is shared by every consumer's tests -- none of them
// reach the dead-letter branch (that requires a real JetStream-delivered
// message, see dead_letter_test.go's fallback-to-Nak test in the internal
// package), so this only needs to satisfy the constructor signature.
type stubDLQPublisher struct {
	calls []string
}

func (s *stubDLQPublisher) Publish(subj string, _ []byte, _ ...nats.PubOpt) (*nats.PubAck, error) {
	s.calls = append(s.calls, subj)
	return &nats.PubAck{}, nil
}
