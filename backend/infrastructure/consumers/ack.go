package consumers

import (
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

// ackOrLog, nakOrLog and nakWithDelayOrLog replace bare `_ = msg.Ack()`
// calls. A failed ack is not something a handler can recover from (the
// message is simply redelivered after AckWait), but it must not vanish
// silently -- otherwise a redelivery loop has no trace of why it happened.
// logPrefix identifies the calling consumer, matching each consumer's log style.
func ackOrLog(msg *nats.Msg, logPrefix string) {
	if err := msg.Ack(); err != nil {
		log.Printf("%s: ack failed (message will be redelivered): %v", logPrefix, err)
	}
}

func nakOrLog(msg *nats.Msg, logPrefix string) {
	if err := msg.Nak(); err != nil {
		log.Printf("%s: nak failed (message will be redelivered after AckWait): %v", logPrefix, err)
	}
}

func nakWithDelayOrLog(msg *nats.Msg, delay time.Duration, logPrefix string) {
	if err := msg.NakWithDelay(delay); err != nil {
		log.Printf("%s: nak failed (message will be redelivered after AckWait): %v", logPrefix, err)
	}
}
