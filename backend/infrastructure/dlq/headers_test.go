package dlq_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/infrastructure/dlq"
)

func TestFailureHeader_recordsWhoFailedWhyAndWhen(t *testing.T) {
	at := time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)

	h := dlq.FailureHeader("sales-order-status-changed-scheduler", "oms.sales_order.status_changed", errors.New("woo: 403"), 5, at)

	assert.Equal(t, "sales-order-status-changed-scheduler", h.Get(dlq.HeaderConsumer))
	assert.Equal(t, "oms.sales_order.status_changed", h.Get(dlq.HeaderOriginalSubject))
	assert.Equal(t, "woo: 403", h.Get(dlq.HeaderError))
	assert.Equal(t, "5", h.Get(dlq.HeaderNumDelivered))
	assert.Equal(t, "2026-11-03T10:00:00Z", h.Get(dlq.HeaderDeadLetteredAt))
}

func TestFailureHeader_truncatesLongErrors(t *testing.T) {
	long := strings.Repeat("x", dlq.MaxErrorLen+50)

	h := dlq.FailureHeader("c", "s", errors.New(long), 5, time.Now())

	assert.Len(t, h.Get(dlq.HeaderError), dlq.MaxErrorLen)
}

func TestFailureHeader_nilCauseLeavesErrorEmpty(t *testing.T) {
	h := dlq.FailureHeader("c", "s", nil, 5, time.Now())

	assert.Equal(t, "", h.Get(dlq.HeaderError))
}

func TestFailureHeader_flattensNewlinesSoTheHeaderStaysValid(t *testing.T) {
	h := dlq.FailureHeader("c", "s", errors.New("line one\r\nline two"), 5, time.Now())

	assert.Equal(t, "line one  line two", h.Get(dlq.HeaderError))
}

func TestSkipsRetry(t *testing.T) {
	targeted := func(consumer string) nats.Header {
		h := nats.Header{}
		h.Set(dlq.HeaderRetryConsumer, consumer)
		return h
	}
	cases := []struct {
		name     string
		header   nats.Header
		consumer string
		want     bool
	}{
		{"no header -- a normal delivery reaches every consumer", nil, "a", false},
		{"no retry header -- e.g. an old dead letter retried to everyone", nats.Header{}, "a", false},
		{"retry aimed at this consumer", targeted("a"), "a", false},
		{"retry aimed at another consumer", targeted("b"), "a", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, dlq.SkipsRetry(tc.header, tc.consumer))
		})
	}
}

func TestSubject(t *testing.T) {
	assert.Equal(t, "dlq.pms.product.changed", dlq.Subject("pms.product.changed"))
}
