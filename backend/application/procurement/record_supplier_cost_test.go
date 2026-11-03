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
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
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

type stubTenantCurrency struct {
	settings *settingsDomain.TenantSettings
	err      error
}

func (s stubTenantCurrency) Get(context.Context) (*settingsDomain.TenantSettings, error) {
	return s.settings, s.err
}

func TestGetSupplierCostSummary_summarizesTheSKUsHistoryInTheTenantCurrency(t *testing.T) {
	at := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	costs := &stubSupplierCosts{entries: []procurementDomain.SupplierCostEntry{
		{SKUID: "sku-1", PurchaseOrderLineItemID: "l1", Kind: procurementDomain.SupplierCostKindReceipt, Quantity: 10, LandedUnitCostMinorUnits: 500, CurrencyCode: "EUR", ReceivedAt: at},
		{SKUID: "sku-1", PurchaseOrderLineItemID: "l2", Kind: procurementDomain.SupplierCostKindReceipt, Quantity: 10, LandedUnitCostMinorUnits: 900, CurrencyCode: "USD", ReceivedAt: at.Add(time.Hour)},
		{SKUID: "sku-2", PurchaseOrderLineItemID: "l3", Kind: procurementDomain.SupplierCostKindReceipt, Quantity: 1, LandedUnitCostMinorUnits: 1, CurrencyCode: "EUR", ReceivedAt: at},
	}}
	uc := appprocurement.NewGetSupplierCostSummaryUseCase(newMockGormDB(t), costs,
		stubTenantCurrency{settings: &settingsDomain.TenantSettings{DefaultCurrency: "EUR"}})

	report, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	assert.Equal(t, "EUR", report.Currency)
	assert.Len(t, report.Entries, 2, "the whole history, every currency")
	require.NotNil(t, report.Summary.Last)
	assert.Equal(t, int64(500), report.Summary.Last.LandedUnitCostMinorUnits, "a USD receipt is left out")
	assert.Equal(t, 1, report.Summary.ExcludedOtherCurrency)
}

func TestGetSupplierCostSummary_propagatesErrors(t *testing.T) {
	boom := errors.New("boom")
	uc := appprocurement.NewGetSupplierCostSummaryUseCase(newMockGormDB(t), &stubSupplierCosts{listErr: boom}, stubTenantCurrency{settings: &settingsDomain.TenantSettings{}})
	_, err := uc.Execute(context.Background(), "sku-1")
	require.ErrorIs(t, err, boom)

	uc = appprocurement.NewGetSupplierCostSummaryUseCase(newMockGormDB(t), &stubSupplierCosts{}, stubTenantCurrency{err: boom})
	_, err = uc.Execute(context.Background(), "sku-1")
	require.ErrorIs(t, err, boom)
}

func TestCorrectSupplierCost_repricesTheLineAndRaisesTheEvent(t *testing.T) {
	at := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	costs := &stubSupplierCosts{entries: []procurementDomain.SupplierCostEntry{{
		SKUID: "sku-1", SupplierID: strp("acme"), PurchaseOrderID: "po-1", PurchaseOrderLineItemID: "line-1",
		Kind: procurementDomain.SupplierCostKindReceipt, Quantity: 10, LandedUnitCostMinorUnits: 500, CurrencyCode: "EUR", ReceivedAt: at,
	}}}
	outbox := &eventrecordtest.Outbox{}
	uc := appprocurement.NewCorrectSupplierCostUseCase(newMockGormDB(t), costs, outbox)

	err := uc.Execute(context.Background(), appprocurement.CorrectSupplierCostInput{
		EventID: "evt-9", SKUID: "sku-1", PurchaseOrderLineItemID: "line-1", AdjustmentValueMinorUnits: 1000, At: at.Add(time.Hour),
	})

	require.NoError(t, err)
	require.Len(t, costs.appended, 1)
	pair := costs.appended[0]
	require.Len(t, pair, 2)
	assert.Equal(t, "correction:evt-9:out", pair[0].SourceKey)
	assert.Equal(t, "correction:evt-9:in", pair[1].SourceKey)
	assert.Equal(t, int64(600), pair[1].LandedUnitCostMinorUnits)
	ev, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "procurement.supplier_cost.recorded", ev.Subject)
	assert.Equal(t, "correction", ev.Data()["kind"])
	assert.Equal(t, float64(600), ev.Data()["landed_unit_cost_minor_units"])
}

func TestCorrectSupplierCost_nothingToRepriceIsANoOp(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	costs := &stubSupplierCosts{}
	uc := appprocurement.NewCorrectSupplierCostUseCase(newMockGormDB(t), costs, outbox)

	err := uc.Execute(context.Background(), appprocurement.CorrectSupplierCostInput{
		EventID: "evt-9", SKUID: "sku-1", PurchaseOrderLineItemID: "line-unknown", AdjustmentValueMinorUnits: 1000, At: time.Now(),
	})

	require.NoError(t, err)
	assert.Empty(t, costs.appended)
	assert.Empty(t, outbox.Events)
}

func TestCorrectSupplierCost_ignoresAnotherSKUsLine(t *testing.T) {
	costs := &stubSupplierCosts{entries: []procurementDomain.SupplierCostEntry{{
		SKUID: "sku-2", PurchaseOrderLineItemID: "line-1", Kind: procurementDomain.SupplierCostKindReceipt, Quantity: 10, LandedUnitCostMinorUnits: 500, CurrencyCode: "EUR",
	}}}
	uc := appprocurement.NewCorrectSupplierCostUseCase(newMockGormDB(t), costs, &eventrecordtest.Outbox{})

	require.NoError(t, uc.Execute(context.Background(), appprocurement.CorrectSupplierCostInput{
		EventID: "evt-9", SKUID: "sku-1", PurchaseOrderLineItemID: "line-1", AdjustmentValueMinorUnits: 1000, At: time.Now(),
	}))

	assert.Empty(t, costs.appended)
}

func TestReverseGoodsReceivedCost_takesTheStockBackOffTheLine(t *testing.T) {
	at := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	costs := &stubSupplierCosts{entries: []procurementDomain.SupplierCostEntry{{
		SKUID: "sku-1", PurchaseOrderID: "po-1", PurchaseOrderLineItemID: "line-1",
		Kind: procurementDomain.SupplierCostKindReceipt, Quantity: 10, LandedUnitCostMinorUnits: 500, CurrencyCode: "EUR", ReceivedAt: at,
	}}}
	outbox := &eventrecordtest.Outbox{}
	uc := appprocurement.NewReverseGoodsReceivedCostUseCase(newMockGormDB(t), costs, outbox)

	err := uc.Execute(context.Background(), appprocurement.ReverseGoodsReceivedCostInput{
		EventID: "evt-5", SKUID: "sku-1", PurchaseOrderLineItemID: "line-1", Quantity: -3, At: at.Add(time.Hour),
	})

	require.NoError(t, err)
	require.Len(t, costs.appended, 1)
	e := costs.appended[0][0]
	assert.Equal(t, procurementDomain.SupplierCostKindReversal, e.Kind)
	assert.Equal(t, int64(-3), e.Quantity)
	assert.Equal(t, "reversal:evt-5", e.SourceKey)
	ev, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "reversal", ev.Data()["kind"])
}

func TestReverseGoodsReceivedCost_nothingStandingIsANoOp(t *testing.T) {
	costs := &stubSupplierCosts{}
	outbox := &eventrecordtest.Outbox{}
	uc := appprocurement.NewReverseGoodsReceivedCostUseCase(newMockGormDB(t), costs, outbox)

	require.NoError(t, uc.Execute(context.Background(), appprocurement.ReverseGoodsReceivedCostInput{
		EventID: "evt-5", SKUID: "sku-1", PurchaseOrderLineItemID: "line-1", Quantity: -3, At: time.Now(),
	}))

	assert.Empty(t, costs.appended)
	assert.Empty(t, outbox.Events)
}
