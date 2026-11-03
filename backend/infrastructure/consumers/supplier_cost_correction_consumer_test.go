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

type stubSupplierCostCorrector struct {
	calls []appprocurement.CorrectSupplierCostInput
	err   error
}

func (s *stubSupplierCostCorrector) Execute(_ context.Context, in appprocurement.CorrectSupplierCostInput) error {
	s.calls = append(s.calls, in)
	return s.err
}

func inventoryCostChangedMsg(t *testing.T, at time.Time, data map[string]any) *nats.Msg {
	t.Helper()
	payload, err := json.Marshal(events.EntityAction{
		Base: events.Base{EventID: "evt-1", OccurredAt: at}, TenantID: "tenant-1", EntityID: "sku-1", Data: data,
	})
	require.NoError(t, err)
	return &nats.Msg{Subject: events.SKUInventoryCostChanged.Subject, Data: payload}
}

func TestSupplierCostOnCostCorrected_repricesTheReferencedLine(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	corrector := &stubSupplierCostCorrector{}
	at := time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)
	c := consumers.NewSupplierCostOnCostCorrectedConsumer(processed, corrector, &stubDLQPublisher{})

	c.Handle(inventoryCostChangedMsg(t, at, map[string]any{
		"reason": "correction", "reference_type": "purchase_order_line_item", "reference_id": "line-1",
		"adjustment_value_minor_units": -1500,
	}))

	assert.Equal(t, []appprocurement.CorrectSupplierCostInput{{
		EventID: "evt-1", SKUID: "sku-1", PurchaseOrderLineItemID: "line-1", AdjustmentValueMinorUnits: -1500, At: at,
	}}, corrector.calls)
	assert.Equal(t, []string{"evt-1|" + consumers.SupplierCostOnCostCorrectedConsumerName}, processed.marked)
}

func TestSupplierCostOnCostCorrected_ignoresEveryOtherInventoryCostChange(t *testing.T) {
	for name, data := range map[string]map[string]any{
		"a receipt":                 {"reason": "receipt", "reference_type": "purchase_order_line_item", "reference_id": "line-1"},
		"a correction without a PO": {"reason": "correction", "adjustment_value_minor_units": 100},
		"a correction of a batch":   {"reason": "correction", "reference_type": "batch", "reference_id": "b-1", "adjustment_value_minor_units": 100},
	} {
		t.Run(name, func(t *testing.T) {
			processed := &stubProcessedEvents{processed: map[string]bool{}}
			corrector := &stubSupplierCostCorrector{}
			c := consumers.NewSupplierCostOnCostCorrectedConsumer(processed, corrector, &stubDLQPublisher{})

			c.Handle(inventoryCostChangedMsg(t, time.Now(), data))

			assert.Empty(t, corrector.calls)
		})
	}
}
