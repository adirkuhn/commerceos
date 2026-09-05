package consumers_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/infrastructure/consumers"
)

type stubOrderStatusNotifier struct {
	calls []string
	err   error
}

func (s *stubOrderStatusNotifier) Execute(_ context.Context, salesOrderID string) error {
	s.calls = append(s.calls, salesOrderID)
	return s.err
}

func TestWhatsAppOrderStatusConsumer_notifiesAndMarksProcessedOnSuccess(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	notifier := &stubOrderStatusNotifier{}
	c := consumers.NewWhatsAppOrderStatusConsumer(processed, notifier, &stubDLQPublisher{})
	event := events.SalesOrderStatusChanged{
		Base:     events.Base{EventID: "evt-1", OccurredAt: time.Now()},
		TenantID: "tenant-1", SalesOrderID: "so-1",
	}

	c.Handle(newSalesOrderStatusChangedMsg(t, event))

	require.Equal(t, []string{"so-1"}, notifier.calls)
	require.Equal(t, []string{"evt-1|whatsapp-order-status-notify"}, processed.marked)
}

func TestWhatsAppOrderStatusConsumer_skipsAlreadyProcessedEvents(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{"evt-1|whatsapp-order-status-notify": true}}
	notifier := &stubOrderStatusNotifier{}
	c := consumers.NewWhatsAppOrderStatusConsumer(processed, notifier, &stubDLQPublisher{})
	event := events.SalesOrderStatusChanged{Base: events.Base{EventID: "evt-1", OccurredAt: time.Now()}, TenantID: "tenant-1", SalesOrderID: "so-1"}

	c.Handle(newSalesOrderStatusChangedMsg(t, event))

	require.Empty(t, notifier.calls, "already-processed event must not re-notify")
}

func TestWhatsAppOrderStatusConsumer_doesNotMarkProcessedWhenNotifyFails(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	notifier := &stubOrderStatusNotifier{err: errors.New("evolution api unreachable")}
	c := consumers.NewWhatsAppOrderStatusConsumer(processed, notifier, &stubDLQPublisher{})
	event := events.SalesOrderStatusChanged{Base: events.Base{EventID: "evt-1", OccurredAt: time.Now()}, TenantID: "tenant-1", SalesOrderID: "so-1"}

	c.Handle(newSalesOrderStatusChangedMsg(t, event))

	require.Len(t, notifier.calls, 1)
	require.Empty(t, processed.marked, "a failed notify must stay eligible for redelivery, not look already-handled")
}

func TestWhatsAppOrderStatusConsumer_dropsMalformedPayloadWithoutCallingNotifier(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	notifier := &stubOrderStatusNotifier{}
	c := consumers.NewWhatsAppOrderStatusConsumer(processed, notifier, &stubDLQPublisher{})

	c.Handle(&nats.Msg{Subject: "oms.sales_order.status_changed", Data: []byte("not json")})

	require.Empty(t, notifier.calls)
}

// TestWhatsAppOrderStatusConsumer_independentFromSalesOrderStatusChangedConsumer
// proves the two consumers use distinct processed-event keys -- one being
// already marked done must never make the other skip.
func TestWhatsAppOrderStatusConsumer_independentFromSalesOrderStatusChangedConsumer(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{"evt-1|sales-order-status-changed-push": true}}
	notifier := &stubOrderStatusNotifier{}
	c := consumers.NewWhatsAppOrderStatusConsumer(processed, notifier, &stubDLQPublisher{})
	event := events.SalesOrderStatusChanged{Base: events.Base{EventID: "evt-1", OccurredAt: time.Now()}, TenantID: "tenant-1", SalesOrderID: "so-1"}

	c.Handle(newSalesOrderStatusChangedMsg(t, event))

	require.Equal(t, []string{"so-1"}, notifier.calls, "a different consumer's processed marker must not suppress this one")
}
