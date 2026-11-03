package consumers_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/infrastructure/consumers"
)

type stubGoodsReceivedCostReverser struct {
	calls []appprocurement.ReverseGoodsReceivedCostInput
}

func (s *stubGoodsReceivedCostReverser) Execute(_ context.Context, in appprocurement.ReverseGoodsReceivedCostInput) error {
	s.calls = append(s.calls, in)
	return nil
}

func TestSupplierCostOnReceiptCorrected_reversesTheLine(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	reverser := &stubGoodsReceivedCostReverser{}
	at := time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)
	payload, err := json.Marshal(events.EntityAction{
		Base: events.Base{EventID: "evt-1", OccurredAt: at}, TenantID: "tenant-1", EntityID: "sku-1",
		Data: map[string]any{"purchase_order_line_item_id": "line-1", "booking_id": "b-1", "scan_id": "s-2", "quantity": -3},
	})
	require.NoError(t, err)
	c := consumers.NewSupplierCostOnReceiptCorrectedConsumer(processed, reverser, &stubDLQPublisher{})

	c.Handle(&nats.Msg{Subject: events.GoodsReceiptCorrected.Subject, Data: payload})

	assert.Equal(t, []appprocurement.ReverseGoodsReceivedCostInput{{
		EventID: "evt-1", SKUID: "sku-1", PurchaseOrderLineItemID: "line-1", Quantity: -3, At: at,
	}}, reverser.calls)
	assert.Equal(t, []string{"evt-1|" + consumers.SupplierCostOnReceiptCorrectedConsumerName}, processed.marked)
}

func TestSupplierCostOnReceiptCorrected_dropsAPayloadWithoutALine(t *testing.T) {
	reverser := &stubGoodsReceivedCostReverser{}
	c := consumers.NewSupplierCostOnReceiptCorrectedConsumer(&stubProcessedEvents{processed: map[string]bool{}}, reverser, &stubDLQPublisher{})

	c.Handle(entityActionMsg(t, events.GoodsReceiptCorrected.Subject, "evt-1", "sku-1"))

	assert.Empty(t, reverser.calls)
}
