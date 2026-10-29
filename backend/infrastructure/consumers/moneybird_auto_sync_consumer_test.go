package consumers_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/infrastructure/consumers"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type stubMoneybirdSync struct {
	orders, quotes []string
	tenants        []string
	err            error
}

func (s *stubMoneybirdSync) SalesOrder(ctx context.Context, id string) error {
	s.orders = append(s.orders, id)
	s.tenants = append(s.tenants, tenant.IDFromContext(ctx))
	return s.err
}

func (s *stubMoneybirdSync) Quote(ctx context.Context, id string) error {
	s.quotes = append(s.quotes, id)
	s.tenants = append(s.tenants, tenant.IDFromContext(ctx))
	return s.err
}

func entityActionMsg(t *testing.T, subject, eventID, entityID string) *nats.Msg {
	t.Helper()
	payload, err := json.Marshal(events.EntityAction{Base: events.Base{EventID: eventID, OccurredAt: time.Now()}, TenantID: "tenant-1", EntityID: entityID})
	require.NoError(t, err)
	return &nats.Msg{Subject: subject, Data: payload}
}

func TestMoneybirdOrderCreatedConsumer_sendsTheOrder(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	sync := &stubMoneybirdSync{}
	c := consumers.NewMoneybirdSalesOrderCreatedConsumer(processed, sync, &stubDLQPublisher{})

	c.Handle(entityActionMsg(t, events.SalesOrderCreated.Subject, "evt-1", "so-1"))

	assert.Equal(t, []string{"so-1"}, sync.orders)
	assert.Equal(t, []string{"tenant-1"}, sync.tenants)
	assert.Equal(t, []string{"evt-1|" + consumers.MoneybirdSalesOrderCreatedConsumerName}, processed.marked)
}

func TestMoneybirdOrderStatusChangedConsumer_sendsTheOrder(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	sync := &stubMoneybirdSync{}
	c := consumers.NewMoneybirdSalesOrderStatusChangedConsumer(processed, sync, &stubDLQPublisher{})
	payload, err := json.Marshal(events.SalesOrderStatusChanged{Base: events.Base{EventID: "evt-2"}, TenantID: "tenant-1", SalesOrderID: "so-2"})
	require.NoError(t, err)

	c.Handle(&nats.Msg{Subject: "oms.sales_order.status_changed", Data: payload})

	assert.Equal(t, []string{"so-2"}, sync.orders)
	assert.Len(t, processed.marked, 1)
}

func TestMoneybirdQuoteConsumers_sendTheQuote(t *testing.T) {
	for _, subject := range []string{events.QuoteCreated.Subject, events.QuoteUpdated.Subject} {
		processed := &stubProcessedEvents{processed: map[string]bool{}}
		sync := &stubMoneybirdSync{}
		c := consumers.NewMoneybirdQuoteChangedConsumer(subject, processed, sync, &stubDLQPublisher{})

		c.Handle(entityActionMsg(t, subject, "evt-q", "q-1"))

		assert.Equal(t, []string{"q-1"}, sync.quotes, subject)
	}
}

// One durable consumer per subject, so each needs its own dedup name.
func TestMoneybirdQuoteConsumers_haveDistinctNames(t *testing.T) {
	assert.NotEqual(t, consumers.MoneybirdQuoteChangedConsumerName(events.QuoteCreated.Subject),
		consumers.MoneybirdQuoteChangedConsumerName(events.QuoteUpdated.Subject))
}

func TestMoneybirdConsumer_skipsAlreadyProcessed(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{"evt-1|" + consumers.MoneybirdSalesOrderCreatedConsumerName: true}}
	sync := &stubMoneybirdSync{}
	c := consumers.NewMoneybirdSalesOrderCreatedConsumer(processed, sync, &stubDLQPublisher{})

	c.Handle(entityActionMsg(t, events.SalesOrderCreated.Subject, "evt-1", "so-1"))

	assert.Empty(t, sync.orders)
}

func TestMoneybirdConsumer_failureIsRetriedNotMarked(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	sync := &stubMoneybirdSync{err: errors.New("moneybird down")}
	c := consumers.NewMoneybirdSalesOrderCreatedConsumer(processed, sync, &stubDLQPublisher{})

	c.Handle(entityActionMsg(t, events.SalesOrderCreated.Subject, "evt-1", "so-1"))

	assert.Len(t, sync.orders, 1)
	assert.Empty(t, processed.marked)
}

func TestMoneybirdConsumer_dropsMalformedPayload(t *testing.T) {
	sync := &stubMoneybirdSync{}
	c := consumers.NewMoneybirdSalesOrderCreatedConsumer(&stubProcessedEvents{processed: map[string]bool{}}, sync, &stubDLQPublisher{})

	c.Handle(&nats.Msg{Subject: events.SalesOrderCreated.Subject, Data: []byte("not json")})

	assert.Empty(t, sync.orders)
}
