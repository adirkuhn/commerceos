package oms_test

import (
	"context"
	"testing"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type completePackingFixture struct {
	sessions  *stubPackingSessionRepo
	orders    *stubSalesOrderRepo
	lineItems *stubSalesOrderLineItemRepo
	scans     *stubPackScanRepo
	addLine   *stubAddLineItemForPickCompletion
	movements *stubRecordMovement
	stock     *stubStockAvailabilityChecker
}

func newCompletePackingFixture(lineItems []omsDomain.SalesOrderLineItem, scans []omsDomain.PackScan) completePackingFixture {
	return completePackingFixture{
		sessions:  &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusInProgress}},
		orders:    &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", WarehouseID: "wh-1"}},
		lineItems: &stubSalesOrderLineItemRepo{listBySalesOrder: lineItems}, scans: &stubPackScanRepo{listBySession: scans},
		addLine: &stubAddLineItemForPickCompletion{}, movements: &stubRecordMovement{}, stock: &stubStockAvailabilityChecker{},
	}
}

func (f completePackingFixture) useCase(t *testing.T) *appoms.CompletePackingUseCase {
	// Loading, each applied change, and completing are separate transactions.
	db := newMockGormDBN(t, 10)
	applier := appoms.NewOrderChangeApplier(db, f.orders, f.lineItems, &stubSalesOrderLineItemStockMovementRepo{}, f.addLine,
		f.movements, &stubReverseMovement{}, f.stock, &stubSKUFinderForShortfall{}, &stubOutboxRepo{}, &stubPaymentRefundClientResolver{})
	return appoms.NewCompletePackingUseCase(db, f.sessions, f.orders, f.lineItems, f.scans, applier)
}

func widget() omsDomain.SalesOrderLineItem {
	sku := "sku-1"
	return omsDomain.SalesOrderLineItem{ID: "li-1", SKUID: &sku, Quantity: 2, Name: "Widget"}
}

func TestCompletePacking_completesAndMarksOrderPacked(t *testing.T) {
	f := newCompletePackingFixture([]omsDomain.SalesOrderLineItem{widget()},
		[]omsDomain.PackScan{{LineItemID: strPtrLI("li-1"), SKUID: "sku-1", ActionType: omsDomain.PackScanActionPack, Quantity: 2}})

	result, err := f.useCase(t).Execute(context.Background(), appoms.CompletePackingInput{SessionID: "session-1", CompletedBy: "user-1"})

	require.NoError(t, err)
	assert.True(t, result.Completed)
	assert.Empty(t, result.MissingItems)
	assert.Equal(t, omsDomain.PackingSessionStatusCompleted, result.Session.Status)
	require.Len(t, f.sessions.updated, 1)
	assert.Equal(t, "user-1", f.sessions.updated[0].CompletedBy)
	assert.Equal(t, []string{"session-1"}, f.sessions.released)
	assert.Equal(t, "order-1", f.orders.gotUpdateID)
	assert.Equal(t, map[string]any{"packed": true, "fulfilled": true}, f.orders.gotUpdateFields,
		"packing also marks the order fulfilled -- picking is optional")
}

func TestCompletePacking_reportsMissingItemsButStillCompletes(t *testing.T) {
	f := newCompletePackingFixture([]omsDomain.SalesOrderLineItem{widget()},
		[]omsDomain.PackScan{{LineItemID: strPtrLI("li-1"), SKUID: "sku-1", ActionType: omsDomain.PackScanActionPack, Quantity: 1}})

	result, err := f.useCase(t).Execute(context.Background(), appoms.CompletePackingInput{SessionID: "session-1", CompletedBy: "user-1"})

	require.NoError(t, err)
	require.Len(t, result.MissingItems, 1)
	assert.Equal(t, int64(1), result.MissingItems[0].PackedQty)
	assert.Equal(t, int64(2), result.MissingItems[0].ExpectedQty)
	assert.Equal(t, omsDomain.PackingSessionStatusCompleted, result.Session.Status)
}

func TestCompletePacking_rejectsAlreadyCompletedSession(t *testing.T) {
	f := newCompletePackingFixture(nil, nil)
	f.sessions.findByIDResult.Status = omsDomain.PackingSessionStatusCompleted

	_, err := f.useCase(t).Execute(context.Background(), appoms.CompletePackingInput{SessionID: "session-1"})

	assert.ErrorIs(t, err, omsDomain.ErrPackingSessionNotActive)
}

func TestCompletePacking_appliesAnExtraItemAndMarksItApplied(t *testing.T) {
	price := int64(399)
	f := newCompletePackingFixture([]omsDomain.SalesOrderLineItem{widget()}, []omsDomain.PackScan{
		{LineItemID: strPtrLI("li-1"), SKUID: "sku-1", ActionType: omsDomain.PackScanActionPack, Quantity: 2},
		{ID: "extra-1", SKUID: "sku-extra", ActionType: omsDomain.PackScanActionNewItem, Quantity: 1, PriceMinorUnits: &price, Reason: "Extra jar"},
	})

	result, err := f.useCase(t).Execute(context.Background(), appoms.CompletePackingInput{SessionID: "session-1", CompletedBy: "user-1"})

	require.NoError(t, err)
	assert.True(t, result.Completed)
	require.Len(t, f.addLine.calls, 1)
	assert.Equal(t, appoms.AddSalesOrderLineItemInput{SalesOrderID: "order-1", SKUID: "sku-extra", Name: "Extra jar", Quantity: 1, PriceMinorUnits: 399}, f.addLine.calls[0])
	require.Len(t, f.movements.calls, 1)
	assert.Equal(t, "packing session session-1 new_item", f.movements.calls[0].Reason)
	assert.Equal(t, []string{"extra-1"}, f.scans.markReplayed)
}

func TestCompletePacking_stopsOnStockShortfallUntilResolved(t *testing.T) {
	f := newCompletePackingFixture([]omsDomain.SalesOrderLineItem{widget()}, []omsDomain.PackScan{
		{ID: "free-1", SKUID: "sku-extra", ActionType: omsDomain.PackScanActionFreebie, Quantity: 3, Reason: "Sample"},
	})
	f.stock.shortBySKU = map[string]int64{"sku-extra": 2}

	result, err := f.useCase(t).Execute(context.Background(), appoms.CompletePackingInput{SessionID: "session-1", CompletedBy: "user-1"})

	require.NoError(t, err)
	assert.False(t, result.Completed)
	require.Len(t, result.StockShortfalls, 1)
	assert.Equal(t, appoms.StockShortfall{ScanID: "free-1", SKUID: "sku-extra", Name: "Sample", Needed: 3, Short: 2}, result.StockShortfalls[0])
	assert.Empty(t, f.sessions.updated, "nothing completes while a shortfall is undecided")
	assert.Empty(t, f.addLine.calls)

	result, err = f.useCase(t).Execute(context.Background(), appoms.CompletePackingInput{
		SessionID: "session-1", CompletedBy: "user-1",
		Resolutions: map[string]appoms.StockShortfallResolution{"free-1": appoms.StockShortfallWithoutItem},
	})

	require.NoError(t, err)
	assert.True(t, result.Completed)
	assert.Empty(t, f.addLine.calls, "shipping without the item never creates it")
	assert.Equal(t, []string{"free-1"}, f.scans.markReplayed)
}

func TestCompletePacking_substituteCoversTheOriginalLine(t *testing.T) {
	covered := int64(2)
	f := newCompletePackingFixture([]omsDomain.SalesOrderLineItem{widget()}, []omsDomain.PackScan{
		{ID: "sub-1", LineItemID: strPtrLI("li-1"), SKUID: "sku-sub", ActionType: omsDomain.PackScanActionSubstitute, Quantity: 4, OriginalQtyCovered: &covered},
	})

	result, err := f.useCase(t).Execute(context.Background(), appoms.CompletePackingInput{SessionID: "session-1", CompletedBy: "user-1"})

	require.NoError(t, err)
	assert.Empty(t, result.MissingItems)
	assert.Equal(t, []string{"sub-1"}, f.scans.markReplayed)
}

func TestCompletePacking_substituteShippedWithoutIsStillMissing(t *testing.T) {
	covered := int64(2)
	f := newCompletePackingFixture([]omsDomain.SalesOrderLineItem{widget()}, []omsDomain.PackScan{
		{ID: "sub-1", LineItemID: strPtrLI("li-1"), SKUID: "sku-sub", ActionType: omsDomain.PackScanActionSubstitute, Quantity: 4, OriginalQtyCovered: &covered},
	})

	result, err := f.useCase(t).Execute(context.Background(), appoms.CompletePackingInput{
		SessionID: "session-1", CompletedBy: "user-1",
		Resolutions: map[string]appoms.StockShortfallResolution{"sub-1": appoms.StockShortfallWithoutItem},
	})

	require.NoError(t, err)
	require.Len(t, result.MissingItems, 1)
	assert.Equal(t, int64(0), result.MissingItems[0].PackedQty)
}

func TestCompletePacking_propagatesOrderLookupError(t *testing.T) {
	f := newCompletePackingFixture(nil, nil)
	f.orders.findByIDResult = nil

	_, err := f.useCase(t).Execute(context.Background(), appoms.CompletePackingInput{SessionID: "session-1"})

	assert.ErrorIs(t, err, omsDomain.ErrSalesOrderNotFound)
}
