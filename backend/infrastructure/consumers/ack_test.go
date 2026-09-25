package consumers

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
)

// captureLog redirects the std logger for the duration of fn.
func captureLog(t *testing.T, fn func()) string {
	t.Helper()
	var buf bytes.Buffer
	orig := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(orig)
	fn()
	return buf.String()
}

// An unbound nats.Msg (no subscription) fails Ack/Nak with ErrMsgNotBound --
// a stand-in for any real ack failure (connection dropped, AckWait elapsed).
func TestAckOrLog_LogsFailure(t *testing.T) {
	out := captureLog(t, func() { ackOrLog(&nats.Msg{Subject: "x"}, "testConsumer") })
	assert.True(t, strings.Contains(out, "testConsumer: ack failed"), out)
}

func TestNakOrLog_LogsFailure(t *testing.T) {
	out := captureLog(t, func() { nakOrLog(&nats.Msg{Subject: "x"}, "testConsumer") })
	assert.True(t, strings.Contains(out, "testConsumer: nak failed"), out)
}

func TestNakWithDelayOrLog_LogsFailure(t *testing.T) {
	out := captureLog(t, func() { nakWithDelayOrLog(&nats.Msg{Subject: "x"}, 0, "testConsumer") })
	assert.True(t, strings.Contains(out, "testConsumer: nak failed"), out)
}
