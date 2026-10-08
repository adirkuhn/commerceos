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
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type stubBolShipmentConfirmer struct {
	calls   []string
	tenants []string
	err     error
}

func (s *stubBolShipmentConfirmer) Execute(ctx context.Context, salesOrderID string) error {
	s.calls = append(s.calls, salesOrderID)
	s.tenants = append(s.tenants, tenant.IDFromContext(ctx))
	return s.err
}

func bolStatusChangedEvent() events.SalesOrderStatusChanged {
	return events.SalesOrderStatusChanged{
		Base:     events.Base{EventID: "evt-1", OccurredAt: time.Now()},
		TenantID: "tenant-1", SalesOrderID: "so-1",
	}
}

func TestBolShipmentConfirmationConsumer_confirmsAndMarksProcessed(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	confirmer := &stubBolShipmentConfirmer{}
	c := consumers.NewBolShipmentConfirmationConsumer(processed, confirmer, &stubDLQPublisher{})

	c.Handle(newSalesOrderStatusChangedMsg(t, bolStatusChangedEvent()))

	require.Equal(t, []string{"so-1"}, confirmer.calls)
	require.Equal(t, []string{"tenant-1"}, confirmer.tenants, "runs as the event's tenant")
	require.Equal(t, []string{"evt-1|bol-shipment-confirmation"}, processed.marked)
}

func TestBolShipmentConfirmationConsumer_skipsAlreadyProcessedEvents(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{"evt-1|bol-shipment-confirmation": true}}
	confirmer := &stubBolShipmentConfirmer{}
	c := consumers.NewBolShipmentConfirmationConsumer(processed, confirmer, &stubDLQPublisher{})

	c.Handle(newSalesOrderStatusChangedMsg(t, bolStatusChangedEvent()))

	require.Empty(t, confirmer.calls)
}

func TestBolShipmentConfirmationConsumer_failure_staysEligibleForRedelivery(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	confirmer := &stubBolShipmentConfirmer{err: errors.New("bol unavailable")}
	c := consumers.NewBolShipmentConfirmationConsumer(processed, confirmer, &stubDLQPublisher{})

	c.Handle(newSalesOrderStatusChangedMsg(t, bolStatusChangedEvent()))

	require.Len(t, confirmer.calls, 1)
	require.Empty(t, processed.marked)
}

func TestBolShipmentConfirmationConsumer_dropsMalformedPayload(t *testing.T) {
	confirmer := &stubBolShipmentConfirmer{}
	c := consumers.NewBolShipmentConfirmationConsumer(&stubProcessedEvents{processed: map[string]bool{}}, confirmer, &stubDLQPublisher{})

	c.Handle(&nats.Msg{Subject: "oms.sales_order.status_changed", Data: []byte("not json")})

	require.Empty(t, confirmer.calls)
}

func TestBolShipmentConfirmationConsumer_independentFromChannelStatusPush(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{"evt-1|sales-order-status-changed-push": true}}
	confirmer := &stubBolShipmentConfirmer{}
	c := consumers.NewBolShipmentConfirmationConsumer(processed, confirmer, &stubDLQPublisher{})

	c.Handle(newSalesOrderStatusChangedMsg(t, bolStatusChangedEvent()))

	require.Equal(t, []string{"so-1"}, confirmer.calls)
}
