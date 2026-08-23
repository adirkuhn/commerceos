package publishers

import "github.com/nats-io/nats.go"

// JetStreamPublisher implements application/webhooks.Publisher over a real
// JetStream connection. js.Publish blocks for the broker's ack (unlike
// PublishAsync), so a nil error here means JetStream durably has the
// message before the HTTP handler responds to the webhook caller.
type JetStreamPublisher struct {
	js nats.JetStreamContext
}

func NewJetStreamPublisher(js nats.JetStreamContext) *JetStreamPublisher {
	return &JetStreamPublisher{js: js}
}

func (p *JetStreamPublisher) Publish(subject string, data []byte) error {
	_, err := p.js.Publish(subject, data)
	return err
}
