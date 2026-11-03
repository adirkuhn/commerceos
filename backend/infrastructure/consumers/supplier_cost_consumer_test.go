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

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/infrastructure/consumers"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type stubGoodsReceivedCostRecorder struct {
	calls   []appprocurement.RecordGoodsReceivedCostInput
	tenants []string
	err     error
}

func (s *stubGoodsReceivedCostRecorder) Execute(ctx context.Context, in appprocurement.RecordGoodsReceivedCostInput) error {
	s.calls = append(s.calls, in)
	s.tenants = append(s.tenants, tenant.IDFromContext(ctx))
	return s.err
}

func goodsReceivedMsg(t *testing.T, e events.GoodsReceived) *nats.Msg {
	t.Helper()
	payload, err := json.Marshal(e)
	require.NoError(t, err)
	return &nats.Msg{Subject: e.EventName(), Data: payload}
}

func goodsReceived(at time.Time) events.GoodsReceived {
	return events.GoodsReceived{
		Base: events.Base{EventID: "evt-1", OccurredAt: at}, TenantID: "tenant-1", SKUID: "sku-1",
		WarehouseID: "wh-1", BatchID: "batch-1", PurchaseOrderLineItemID: "line-1", Quantity: 12, LandedUnitCostMinorUnits: 500,
	}
}

func TestSupplierCostOnGoodsReceived_recordsTheReceipt(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	recorder := &stubGoodsReceivedCostRecorder{}
	at := time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)
	c := consumers.NewSupplierCostOnGoodsReceivedConsumer(processed, recorder, &stubDLQPublisher{})

	c.Handle(goodsReceivedMsg(t, goodsReceived(at)))

	require.Len(t, recorder.calls, 1)
	assert.Equal(t, appprocurement.RecordGoodsReceivedCostInput{
		EventID: "evt-1", SKUID: "sku-1", PurchaseOrderLineItemID: "line-1", BatchID: "batch-1",
		Quantity: 12, LandedUnitCostMinorUnits: 500, ReceivedAt: at,
	}, recorder.calls[0])
	assert.Equal(t, []string{"tenant-1"}, recorder.tenants)
	assert.Equal(t, []string{"evt-1|" + consumers.SupplierCostOnGoodsReceivedConsumerName}, processed.marked)
}

func TestSupplierCostOnGoodsReceived_skipsAlreadyProcessed(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{"evt-1|" + consumers.SupplierCostOnGoodsReceivedConsumerName: true}}
	recorder := &stubGoodsReceivedCostRecorder{}
	c := consumers.NewSupplierCostOnGoodsReceivedConsumer(processed, recorder, &stubDLQPublisher{})

	c.Handle(goodsReceivedMsg(t, goodsReceived(time.Now())))

	assert.Empty(t, recorder.calls)
}

func TestSupplierCostOnGoodsReceived_dropsAMalformedPayload(t *testing.T) {
	recorder := &stubGoodsReceivedCostRecorder{}
	c := consumers.NewSupplierCostOnGoodsReceivedConsumer(&stubProcessedEvents{processed: map[string]bool{}}, recorder, &stubDLQPublisher{})

	c.Handle(&nats.Msg{Subject: "procurement.goods.received", Data: []byte("{not json")})

	assert.Empty(t, recorder.calls)
}

func TestSupplierCostOnGoodsReceived_failureIsNotMarkedProcessed(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	recorder := &stubGoodsReceivedCostRecorder{err: errors.New("db down")}
	c := consumers.NewSupplierCostOnGoodsReceivedConsumer(processed, recorder, &stubDLQPublisher{})

	c.Handle(goodsReceivedMsg(t, goodsReceived(time.Now())))

	assert.Empty(t, processed.marked)
}
