package procurement_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

// stubSupplierCosts is an in-memory SupplierCostRepository.
type stubSupplierCosts struct {
	entries   []procurementDomain.SupplierCostEntry
	appended  [][]procurementDomain.SupplierCostEntry
	appendErr error
	listErr   error
}

func (s *stubSupplierCosts) Append(_ context.Context, entries []procurementDomain.SupplierCostEntry) error {
	if s.appendErr != nil {
		return s.appendErr
	}
	s.appended = append(s.appended, entries)
	s.entries = append(s.entries, entries...)
	return nil
}

func (s *stubSupplierCosts) ListBySKU(_ context.Context, skuID string) ([]procurementDomain.SupplierCostEntry, error) {
	var out []procurementDomain.SupplierCostEntry
	for _, e := range s.entries {
		if e.SKUID == skuID {
			out = append(out, e)
		}
	}
	return out, s.listErr
}

func (s *stubSupplierCosts) ListByLine(_ context.Context, lineID string) ([]procurementDomain.SupplierCostEntry, error) {
	var out []procurementDomain.SupplierCostEntry
	for _, e := range s.entries {
		if e.PurchaseOrderLineItemID == lineID {
			out = append(out, e)
		}
	}
	return out, s.listErr
}

func strp(v string) *string { return &v }

func costFixture() (*stubLineItemRepo, *stubPurchaseOrderRepo) {
	lines := &stubLineItemRepo{findResult: &procurementDomain.PurchaseOrderLineItem{
		ID: "line-1", PurchaseOrderID: "po-1", SKUID: strp("sku-1"), UnitCostMinorUnits: 450,
	}}
	orders := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{
		ID: "po-1", SupplierID: strp("acme"), CurrencyCode: "EUR",
	}}
	return lines, orders
}

func TestRecordGoodsReceivedCost_appendsAReceiptRowAndRaisesTheEvent(t *testing.T) {
	lines, orders := costFixture()
	costs := &stubSupplierCosts{}
	outbox := &eventrecordtest.Outbox{}
	at := time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)
	uc := appprocurement.NewRecordGoodsReceivedCostUseCase(newMockGormDB(t), lines, orders, costs, outbox)

	err := uc.Execute(context.Background(), appprocurement.RecordGoodsReceivedCostInput{
		EventID: "evt-1", SKUID: "sku-1", PurchaseOrderLineItemID: "line-1", BatchID: "batch-1",
		Quantity: 12, LandedUnitCostMinorUnits: 500, ReceivedAt: at,
	})

	require.NoError(t, err)
	require.Len(t, costs.entries, 1)
	e := costs.entries[0]
	assert.Equal(t, procurementDomain.SupplierCostKindReceipt, e.Kind)
	assert.Equal(t, "sku-1", e.SKUID)
	assert.Equal(t, "acme", *e.SupplierID)
	assert.Equal(t, "po-1", e.PurchaseOrderID)
	assert.Equal(t, "line-1", e.PurchaseOrderLineItemID)
	assert.Equal(t, "batch-1", *e.BatchID)
	assert.Equal(t, int64(12), e.Quantity)
	assert.Equal(t, int64(450), e.UnitCostMinorUnits, "the PO line's own price")
	assert.Equal(t, int64(500), e.LandedUnitCostMinorUnits, "fees included")
	assert.Equal(t, "EUR", e.CurrencyCode)
	assert.Equal(t, at, e.ReceivedAt)
	assert.Equal(t, "goods_received:evt-1", e.SourceKey, "a redelivered event never adds a second row")

	ev, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "procurement.supplier_cost.recorded", ev.Subject)
	assert.Equal(t, "sku-1", ev.Payload["entity_id"])
	assert.Equal(t, "receipt", ev.Data()["kind"])
	assert.Equal(t, "line-1", ev.Data()["purchase_order_line_item_id"])
}

func TestRecordGoodsReceivedCost_propagatesErrors(t *testing.T) {
	boom := errors.New("boom")
	for name, breakIt := range map[string]func(*stubLineItemRepo, *stubPurchaseOrderRepo, *stubSupplierCosts, *eventrecordtest.Outbox){
		"line": func(l *stubLineItemRepo, _ *stubPurchaseOrderRepo, _ *stubSupplierCosts, _ *eventrecordtest.Outbox) {
			l.findErr = boom
		},
		"order": func(_ *stubLineItemRepo, o *stubPurchaseOrderRepo, _ *stubSupplierCosts, _ *eventrecordtest.Outbox) {
			o.findErr = boom
		},
		"append": func(_ *stubLineItemRepo, _ *stubPurchaseOrderRepo, c *stubSupplierCosts, _ *eventrecordtest.Outbox) {
			c.appendErr = boom
		},
		"event": func(_ *stubLineItemRepo, _ *stubPurchaseOrderRepo, _ *stubSupplierCosts, o *eventrecordtest.Outbox) {
			o.Err = boom
		},
	} {
		t.Run(name, func(t *testing.T) {
			lines, orders := costFixture()
			costs, outbox := &stubSupplierCosts{}, &eventrecordtest.Outbox{}
			breakIt(lines, orders, costs, outbox)
			uc := appprocurement.NewRecordGoodsReceivedCostUseCase(newMockGormDB(t), lines, orders, costs, outbox)

			err := uc.Execute(context.Background(), appprocurement.RecordGoodsReceivedCostInput{
				EventID: "evt-1", SKUID: "sku-1", PurchaseOrderLineItemID: "line-1", Quantity: 1, LandedUnitCostMinorUnits: 1, ReceivedAt: time.Now(),
			})

			require.ErrorIs(t, err, boom)
		})
	}
}

func TestRecordGoodsReceivedCost_rejectsAnEmptyReceipt(t *testing.T) {
	lines, orders := costFixture()
	uc := appprocurement.NewRecordGoodsReceivedCostUseCase(newMockGormDB(t), lines, orders, &stubSupplierCosts{}, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), appprocurement.RecordGoodsReceivedCostInput{EventID: "evt-1", SKUID: "sku-1", PurchaseOrderLineItemID: "line-1", Quantity: 0})

	require.Error(t, err)
}
