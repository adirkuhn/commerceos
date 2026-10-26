package oms_test

import (
	"context"
	"testing"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type editFixture struct {
	orders   *stubSalesOrderRepo
	lines    *stubSalesOrderLineItemRepo
	links    *stubSalesOrderLineItemStockMovementRepo
	allocate *stubRecordMovement
	reverse  *stubReverseMovement
	picking  *stubPickingSessionRepo
	packing  *stubPackingSessionRepo
	outbox   *eventrecordtest.Outbox
}

func newEditFixture(order omsDomain.SalesOrder, lines ...omsDomain.SalesOrderLineItem) *editFixture {
	return &editFixture{
		orders: &stubSalesOrderRepo{findByIDResult: &order}, lines: &stubSalesOrderLineItemRepo{listBySalesOrder: lines},
		links: &stubSalesOrderLineItemStockMovementRepo{}, allocate: &stubRecordMovement{}, reverse: &stubReverseMovement{},
		picking: &stubPickingSessionRepo{}, packing: &stubPackingSessionRepo{}, outbox: &eventrecordtest.Outbox{},
	}
}

func (f *editFixture) run(t *testing.T, transactions int, in appoms.EditSalesOrderInput) (*appoms.CreateSalesOrderResult, error) {
	return appoms.NewEditSalesOrderUseCase(newMockGormDBN(t, transactions), f.orders, f.lines, f.links, &stubSKURepo{},
		f.allocate, f.reverse, f.picking, f.packing, f.outbox).Execute(context.Background(), in)
}

func editInput(lines ...appoms.CreateSalesOrderLineItemInput) appoms.EditSalesOrderInput {
	return appoms.EditSalesOrderInput{
		ID: "so-1", WarehouseID: "wh-1", OccurredAt: time.Now(),
		Billing:                 omsDomain.Address{Company: "Claus Hoofddorp B.V.", City: "Hoofddorp"},
		Shipping:                omsDomain.Address{Company: "Claus Hoofddorp B.V.", City: "Hoofddorp"},
		ShippingTotalMinorUnits: 799, TaxTotalMinorUnits: 489, TotalMinorUnits: 5882, CustomerNote: "Back door",
		LineItems: lines,
	}
}

var mateLine = appoms.CreateSalesOrderLineItemInput{SKUID: "sku-1", Name: "Chá Mate", Quantity: 17, PriceMinorUnits: 299,
	SubtotalMinorUnits: 5083, SubtotalTaxMinorUnits: 420, TotalMinorUnits: 5083, TotalTaxMinorUnits: 420, TotalIncludesTax: true}

func TestEditSalesOrder_replacesLinesAndTotalsOfAPendingOrder(t *testing.T) {
	f := newEditFixture(omsDomain.SalesOrder{ID: "so-1", Source: omsDomain.SalesOrderSourceManual, Status: omsDomain.SalesOrderStatusPending, Number: "CO-1001", WarehouseID: "wh-1"},
		omsDomain.SalesOrderLineItem{ID: "old-line"})

	res, err := f.run(t, 2, editInput(mateLine))

	require.NoError(t, err)
	assert.Equal(t, "so-1", f.lines.gotDeleteSalesOrderID)
	require.Len(t, f.lines.created, 1)
	assert.Equal(t, int64(420), f.lines.created[0].TotalTaxMinorUnits)
	assert.True(t, f.lines.created[0].TotalIncludesTax)
	assert.Equal(t, int64(489), f.orders.gotUpdateFields["tax_total_minor_units"])
	assert.Equal(t, int64(5882), f.orders.gotUpdateFields["total_minor_units"])
	assert.Equal(t, "Back door", f.orders.gotUpdateFields["customer_note"])
	assert.Equal(t, "Claus Hoofddorp B.V.", f.orders.gotUpdateFields["billing_company"])
	assert.NotContains(t, f.orders.gotUpdateFields, "number", "an empty number keeps the order's own")
	assert.Equal(t, "CO-1001", res.SalesOrder.Number)
	assert.Equal(t, int64(5882), res.SalesOrder.TotalMinorUnits)
	assert.Empty(t, f.allocate.calls, "a pending order holds no stock")
	assert.Contains(t, f.outbox.Subjects(), "oms.sales_order.edited")
}

func TestEditSalesOrder_givesBackTheOldStockAndTakesTheNew(t *testing.T) {
	sku := "sku-1"
	f := newEditFixture(omsDomain.SalesOrder{ID: "so-1", Source: omsDomain.SalesOrderSourceManual, Status: omsDomain.SalesOrderStatusProcessing, StockDecremented: true, WarehouseID: "wh-1"},
		omsDomain.SalesOrderLineItem{ID: "old-line", SKUID: &sku, Quantity: 10, StockStatus: omsDomain.StockStatusDecremented})
	f.links.byLineItem = map[string][]omsDomain.SalesOrderLineItemStockMovement{"old-line": {{QuantityDelta: -10}}}

	_, err := f.run(t, 10, editInput(mateLine))

	require.NoError(t, err)
	require.Len(t, f.reverse.calls, 1)
	assert.Equal(t, int64(10), f.reverse.calls[0].QuantityDelta)
	require.Len(t, f.allocate.calls, 1)
	assert.Equal(t, int64(17), f.allocate.calls[0].Quantity)
}

func TestEditSalesOrder_refusesAnOrderThatCannotBeEdited(t *testing.T) {
	f := newEditFixture(omsDomain.SalesOrder{ID: "so-1", Source: omsDomain.SalesOrderSourceWooCommerce, Status: omsDomain.SalesOrderStatusProcessing})

	_, err := f.run(t, 1, editInput(mateLine))

	assert.ErrorIs(t, err, omsDomain.ErrSalesOrderNotEditable)
	assert.Empty(t, f.lines.created)
}

func TestEditSalesOrder_refusesAnOrderBeingPicked(t *testing.T) {
	f := newEditFixture(omsDomain.SalesOrder{ID: "so-1", Source: omsDomain.SalesOrderSourceManual, Status: omsDomain.SalesOrderStatusProcessing})
	f.picking.activeResult = &omsDomain.PickingSession{ID: "ps-1"}

	_, err := f.run(t, 1, editInput(mateLine))

	assert.ErrorIs(t, err, omsDomain.ErrOrderPickingInProgress)
}

func TestEditSalesOrder_refusesAnOrderBeingPacked(t *testing.T) {
	f := newEditFixture(omsDomain.SalesOrder{ID: "so-1", Source: omsDomain.SalesOrderSourceManual, Status: omsDomain.SalesOrderStatusProcessing})
	f.packing.activeResult = &omsDomain.PackingSession{ID: "pk-1"}

	_, err := f.run(t, 1, editInput(mateLine))

	assert.ErrorIs(t, err, omsDomain.ErrOrderPackingInProgress)
}

func TestEditSalesOrder_needsAtLeastOneLine(t *testing.T) {
	f := newEditFixture(omsDomain.SalesOrder{ID: "so-1", Source: omsDomain.SalesOrderSourceManual, Status: omsDomain.SalesOrderStatusPending})

	_, err := f.run(t, 0, editInput())

	assert.ErrorIs(t, err, omsDomain.ErrEmptyLineItems)
}

func TestEditSalesOrder_keepsATypedNumber(t *testing.T) {
	f := newEditFixture(omsDomain.SalesOrder{ID: "so-1", Source: omsDomain.SalesOrderSourceManual, Status: omsDomain.SalesOrderStatusPending, Number: "CO-1001"})
	in := editInput(mateLine)
	in.Number = "2026-17"

	res, err := f.run(t, 2, in)

	require.NoError(t, err)
	assert.Equal(t, "2026-17", f.orders.gotUpdateFields["number"])
	assert.Equal(t, "2026-17", res.SalesOrder.Number)
}
