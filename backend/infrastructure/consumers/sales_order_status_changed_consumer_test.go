package consumers_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/infrastructure/consumers"
)

type stubSalesOrderStatusPusher struct {
	calls []string
	err   error
}

func (s *stubSalesOrderStatusPusher) Execute(_ context.Context, salesOrderID string) error {
	s.calls = append(s.calls, salesOrderID)
	return s.err
}

func newSalesOrderStatusChangedMsg(t *testing.T, event events.SalesOrderStatusChanged) *nats.Msg {
	t.Helper()
	payload, err := json.Marshal(event)
	require.NoError(t, err)
	return &nats.Msg{Subject: event.EventName(), Data: payload}
}

func TestSalesOrderStatusChangedConsumer_pushesAndMarksProcessedOnSuccess(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	pusher := &stubSalesOrderStatusPusher{}
	c := consumers.NewSalesOrderStatusChangedConsumer(processed, pusher, &stubDLQPublisher{})
	event := events.SalesOrderStatusChanged{
		Base:     events.Base{EventID: "evt-1", OccurredAt: time.Now()},
		TenantID: "tenant-1", SalesOrderID: "so-1",
	}

	c.Handle(newSalesOrderStatusChangedMsg(t, event))

	require.Equal(t, []string{"so-1"}, pusher.calls)
	require.Equal(t, []string{"evt-1|sales-order-status-changed-push"}, processed.marked)
}

func TestSalesOrderStatusChangedConsumer_skipsAlreadyProcessedEvents(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{"evt-1|sales-order-status-changed-push": true}}
	pusher := &stubSalesOrderStatusPusher{}
	c := consumers.NewSalesOrderStatusChangedConsumer(processed, pusher, &stubDLQPublisher{})
	event := events.SalesOrderStatusChanged{Base: events.Base{EventID: "evt-1", OccurredAt: time.Now()}, TenantID: "tenant-1", SalesOrderID: "so-1"}

	c.Handle(newSalesOrderStatusChangedMsg(t, event))

	require.Empty(t, pusher.calls, "already-processed event must not re-run the push")
}

func TestSalesOrderStatusChangedConsumer_doesNotMarkProcessedWhenPushFails(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	pusher := &stubSalesOrderStatusPusher{err: errors.New("woo unreachable")}
	c := consumers.NewSalesOrderStatusChangedConsumer(processed, pusher, &stubDLQPublisher{})
	event := events.SalesOrderStatusChanged{Base: events.Base{EventID: "evt-1", OccurredAt: time.Now()}, TenantID: "tenant-1", SalesOrderID: "so-1"}

	c.Handle(newSalesOrderStatusChangedMsg(t, event))

	require.Len(t, pusher.calls, 1)
	require.Empty(t, processed.marked, "a failed push must stay eligible for redelivery, not look already-handled")
}

// COM-15: an order with no channel to send to is a failure the consumer
// redelivers (then dead-letters), not a success to ack.
func TestSalesOrderStatusChangedConsumer_anUnresolvedChannelStaysEligibleForRedelivery(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	pusher := &stubSalesOrderStatusPusher{err: fmt.Errorf("pushSalesOrderStatusToChannels: %w", channelsDomain.ErrOrderChannelUnresolved)}
	c := consumers.NewSalesOrderStatusChangedConsumer(processed, pusher, &stubDLQPublisher{})
	event := events.SalesOrderStatusChanged{Base: events.Base{EventID: "evt-1", OccurredAt: time.Now()}, TenantID: "tenant-1", SalesOrderID: "so-1"}

	c.Handle(newSalesOrderStatusChangedMsg(t, event))

	require.Empty(t, processed.marked)
}

func TestSalesOrderStatusChangedConsumer_dropsMalformedPayloadWithoutCallingPusher(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	pusher := &stubSalesOrderStatusPusher{}
	c := consumers.NewSalesOrderStatusChangedConsumer(processed, pusher, &stubDLQPublisher{})

	c.Handle(&nats.Msg{Subject: "oms.sales_order.status_changed", Data: []byte("not json")})

	require.Empty(t, pusher.calls)
}
