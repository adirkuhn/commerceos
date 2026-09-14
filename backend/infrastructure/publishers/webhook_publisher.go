package publishers

import (
	"fmt"

	"github.com/nats-io/nats.go"

	"github.com/adirkuhn/commerceos/domain/webhooks"
)

// JetStreamPublisher implements application/webhooks.Publisher over a real
// JetStream connection. js.Publish blocks for the broker's ack (unlike
// PublishAsync), so a nil error here means JetStream durably has the
// message before the HTTP handler responds to the webhook caller.
type JetStreamPublisher struct {
	nc *nats.Conn
	js nats.JetStreamContext
}

func NewJetStreamPublisher(nc *nats.Conn, js nats.JetStreamContext) *JetStreamPublisher {
	return &JetStreamPublisher{nc: nc, js: js}
}

// Publish rejects a payload larger than the broker's negotiated max_payload
// before ever calling js.Publish -- NATS would reject it with the same
// "maximum payload exceeded" error anyway, but as an opaque string with no
// way for a caller (the webhook fallback retry action in particular) to
// tell "will never succeed, don't retry" apart from any other transient
// publish failure.
func (p *JetStreamPublisher) Publish(subject string, data []byte) error {
	if err := checkPayloadSize(int64(len(data)), p.nc.MaxPayload()); err != nil {
		return err
	}
	_, err := p.js.Publish(subject, data)
	return err
}

// checkPayloadSize is Publish's size guard, pulled out as a pure function so
// it's unit-testable without a live NATS connection (*nats.Conn.MaxPayload
// only has a real value once connected to a server). maxPayload <= 0 means
// "unknown/not negotiated yet" -- never block on that, let the real
// js.Publish call be the judge.
func checkPayloadSize(dataLen, maxPayload int64) error {
	if maxPayload > 0 && dataLen > maxPayload {
		return fmt.Errorf("%w: %d bytes exceeds server max_payload %d", webhooks.ErrPayloadTooLarge, dataLen, maxPayload)
	}
	return nil
}
