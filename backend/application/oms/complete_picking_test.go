package oms_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubAddLineItemForPickCompletion struct {
	calls  []appoms.AddSalesOrderLineItemInput
	err    error
	nextID int
}

func (s *stubAddLineItemForPickCompletion) Execute(_ context.Context, in appoms.AddSalesOrderLineItemInput) (*omsDomain.SalesOrderLineItem, error) {
	if s.err != nil {
		return nil, s.err
	}
	s.calls = append(s.calls, in)
	s.nextID++
	id := fmt.Sprintf("new-line-%d", s.nextID)
	sku := in.SKUID
	return &omsDomain.SalesOrderLineItem{
		ID: id, SalesOrderID: in.SalesOrderID, SKUID: &sku, Name: in.Name,
		Quantity: in.Quantity, PriceMinorUnits: in.PriceMinorUnits,
	}, nil
}

func completablePickingFixtures() (*stubPickingSessionRepo, *stubSalesOrderRepo) {
	return &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}},
		&stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", WarehouseID: "warehouse-1"}}
}

func TestCompletePicking_allPickCleanCompletion(t *testing.T) {
	db := newMockGormDBN(t, 3) // read + mark-replayed(1 scan) + mark-completed
	sessionRepo, salesOrderRepo := completablePickingFixtures()
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SKUID: strPtr("sku-1"), Name: "Widget", Quantity: 5}}}
	scanRepo := &stubPickScanRepo{listBySession: []omsDomain.PickScan{
		{ID: "scan-1", LineItemID: strPtr("li-1"), SKUID: "sku-1", ActionType: omsDomain.PickScanActionPick, Quantity: 5},
	}}
	addLineItem := &stubAddLineItemForPickCompletion{}
	recordMovement := &stubRecordMovement{}
	reverseMovement := &stubReverseMovement{}
	uc := appoms.NewCompletePickingUseCase(db, sessionRepo, salesOrderRepo, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, scanRepo, addLineItem, recordMovement, reverseMovement, &stubStockAvailabilityChecker{}, &stubSKUFinderForShortfall{}, &stubOutboxRepo{}, &stubPaymentRefundClientResolver{})

	result, err := uc.Execute(context.Background(), appoms.CompletePickingInput{SessionID: "session-1", CompletedBy: "user-1"})

	require.NoError(t, err)
	assert.Empty(t, recordMovement.calls, "a plain pick has no stock effect -- stock already left the ledger at order-processing time")
	assert.Empty(t, reverseMovement.calls)
	assert.Empty(t, addLineItem.calls)
	require.Len(t, scanRepo.markReplayed, 1)
	assert.Equal(t, "scan-1", scanRepo.markReplayed[0])
	assert.Empty(t, result.MissingItems)
	assert.Equal(t, omsDomain.PickingSessionStatusCompleted, result.Session.Status)
	assert.Equal(t, "user-1", result.Session.CompletedBy)
	require.Len(t, salesOrderRepo.updateCalls, 1)
	assert.Equal(t, "order-1", salesOrderRepo.updateCalls[0].ID)
	assert.Equal(t, true, salesOrderRepo.updateCalls[0].Fields["fulfilled"])
}

func TestCompletePicking_reportsMissingItemsButStillCompletes(t *testing.T) {
	db := newMockGormDBN(t, 2) // read + mark-completed (nothing to replay)
	sessionRepo, salesOrderRepo := completablePickingFixtures()
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SKUID: strPtr("sku-1"), Name: "Widget", Quantity: 5}}}
	scanRepo := &stubPickScanRepo{}
	uc := appoms.NewCompletePickingUseCase(db, sessionRepo, salesOrderRepo, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, scanRepo, &stubAddLineItemForPickCompletion{}, &stubRecordMovement{}, &stubReverseMovement{}, &stubStockAvailabilityChecker{}, &stubSKUFinderForShortfall{}, &stubOutboxRepo{}, &stubPaymentRefundClientResolver{})

	result, err := uc.Execute(context.Background(), appoms.CompletePickingInput{SessionID: "session-1", CompletedBy: "user-1"})

	require.NoError(t, err)
	require.Len(t, result.MissingItems, 1)
	assert.Equal(t, "li-1", result.MissingItems[0].LineItemID)
	assert.Equal(t, int64(5), result.MissingItems[0].ExpectedQty)
	assert.Equal(t, int64(0), result.MissingItems[0].PickedQty)
	assert.Equal(t, omsDomain.PickingSessionStatusCompleted, result.Session.Status, "completion always succeeds regardless of gaps")
	require.Len(t, salesOrderRepo.updateCalls, 1)
	assert.Equal(t, true, salesOrderRepo.updateCalls[0].Fields["fulfilled"])
}

func TestCompletePicking_substituteReversesOriginalAndDecrementsSubstitute(t *testing.T) {
	db := newMockGormDBN(t, 6) // read + listLineItemMovements + linkLineItemMovement(1) + mark-replayed + freezeCostAtSale + mark-completed
	sessionRepo, salesOrderRepo := completablePickingFixtures()
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SKUID: strPtr("sku-orig"), Name: "Widget", Quantity: 5}}}
	batchID := "batch-1"
	linkRepo := &stubSalesOrderLineItemStockMovementRepo{byLineItem: map[string][]omsDomain.SalesOrderLineItemStockMovement{
		"li-1": {{SalesOrderLineItemID: "li-1", BatchID: &batchID, QuantityDelta: -5}},
	}}
	qtyCovered := int64(5)
	scanRepo := &stubPickScanRepo{listBySession: []omsDomain.PickScan{
		{ID: "scan-1", LineItemID: strPtr("li-1"), SKUID: "sku-sub", ActionType: omsDomain.PickScanActionSubstitute, Quantity: 5, OriginalQtyCovered: &qtyCovered, ReturnToStock: true},
	}}
	recordMovement := &stubRecordMovement{}
	reverseMovement := &stubReverseMovement{}
	uc := appoms.NewCompletePickingUseCase(db, sessionRepo, salesOrderRepo, lineItemRepo, linkRepo, scanRepo, &stubAddLineItemForPickCompletion{}, recordMovement, reverseMovement, &stubStockAvailabilityChecker{}, &stubSKUFinderForShortfall{}, &stubOutboxRepo{}, &stubPaymentRefundClientResolver{})

	result, err := uc.Execute(context.Background(), appoms.CompletePickingInput{SessionID: "session-1", CompletedBy: "user-1"})

	require.NoError(t, err)
	require.Len(t, reverseMovement.calls, 1)
	assert.Equal(t, "sku-orig", reverseMovement.calls[0].SKUID)
	require.NotNil(t, reverseMovement.calls[0].BatchID)
	assert.Equal(t, "batch-1", *reverseMovement.calls[0].BatchID)
	assert.Equal(t, int64(5), reverseMovement.calls[0].QuantityDelta, "reversal must be positive -- giving stock back")
	assert.Equal(t, invDomain.MovementTypeAdjustment, reverseMovement.calls[0].Type)

	require.Len(t, recordMovement.calls, 1)
	assert.Equal(t, "sku-sub", recordMovement.calls[0].SKUID)
	assert.Equal(t, int64(5), recordMovement.calls[0].Quantity)
	assert.Equal(t, invDomain.MovementTypeSale, recordMovement.calls[0].Type)

	require.Len(t, linkRepo.created, 1)
	assert.Equal(t, "li-1", linkRepo.created[0].SalesOrderLineItemID, "the substitute's new movement must link to the ORIGINAL line item")
	assert.Empty(t, result.MissingItems, "a substitute fully covers the original line's expected quantity")
	assert.Equal(t, omsDomain.PickingSessionStatusCompleted, result.Session.Status)
}

func TestCompletePicking_substituteSkipsReversalWhenOriginalLineHasNoSKU(t *testing.T) {
	db := newMockGormDBN(t, 5) // read + linkLineItemMovement(1) + mark-replayed + freezeCostAtSale + mark-completed (no listLineItemMovements)
	sessionRepo, salesOrderRepo := completablePickingFixtures()
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SKUID: nil, Name: "Unmatched import line", Quantity: 5}}}
	scanRepo := &stubPickScanRepo{listBySession: []omsDomain.PickScan{
		{ID: "scan-1", LineItemID: strPtr("li-1"), SKUID: "sku-sub", ActionType: omsDomain.PickScanActionSubstitute, Quantity: 5},
	}}
	recordMovement := &stubRecordMovement{}
	reverseMovement := &stubReverseMovement{}
	uc := appoms.NewCompletePickingUseCase(db, sessionRepo, salesOrderRepo, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, scanRepo, &stubAddLineItemForPickCompletion{}, recordMovement, reverseMovement, &stubStockAvailabilityChecker{}, &stubSKUFinderForShortfall{}, &stubOutboxRepo{}, &stubPaymentRefundClientResolver{})

	_, err := uc.Execute(context.Background(), appoms.CompletePickingInput{SessionID: "session-1", CompletedBy: "user-1"})

	require.NoError(t, err)
	assert.Empty(t, reverseMovement.calls, "nothing was ever decremented for a SKU-less line, so nothing to reverse")
	require.Len(t, recordMovement.calls, 1)
}

func TestCompletePicking_substituteWithReturnToStockFalseDoesNotReverseButStillCoversMissingItems(t *testing.T) {
	db := newMockGormDBN(t, 5) // read + linkLineItemMovement(1) + mark-replayed + mark-completed (no listLineItemMovements -- ReturnToStock is false, plus freezeCostAtSale)
	sessionRepo, salesOrderRepo := completablePickingFixtures()
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SKUID: strPtr("sku-orig"), Name: "Widget", Quantity: 5}}}
	linkRepo := &stubSalesOrderLineItemStockMovementRepo{}
	qtyCovered := int64(5)
	scanRepo := &stubPickScanRepo{listBySession: []omsDomain.PickScan{
		{ID: "scan-1", LineItemID: strPtr("li-1"), SKUID: "sku-sub", ActionType: omsDomain.PickScanActionSubstitute, Quantity: 5, OriginalQtyCovered: &qtyCovered, ReturnToStock: false},
	}}
	recordMovement := &stubRecordMovement{}
	reverseMovement := &stubReverseMovement{}
	uc := appoms.NewCompletePickingUseCase(db, sessionRepo, salesOrderRepo, lineItemRepo, linkRepo, scanRepo, &stubAddLineItemForPickCompletion{}, recordMovement, reverseMovement, &stubStockAvailabilityChecker{}, &stubSKUFinderForShortfall{}, &stubOutboxRepo{}, &stubPaymentRefundClientResolver{})

	result, err := uc.Execute(context.Background(), appoms.CompletePickingInput{SessionID: "session-1", CompletedBy: "user-1"})

	require.NoError(t, err)
	assert.Empty(t, reverseMovement.calls, "return_to_stock false means the original stock is never given back -- the substitution happened because the stock figure was wrong in the first place")
	require.Len(t, recordMovement.calls, 1, "the substitute SKU is still decremented for real")
	assert.Equal(t, "sku-sub", recordMovement.calls[0].SKUID)
	assert.Empty(t, result.MissingItems, "the original line's expected quantity is still fully covered by the substitute")
}

// One original line, quantity 2, split across two Substitute rows -- one
// per unit, one opted into return_to_stock and one not. Must reverse only
// the opted-in row's share, decrement both substitute SKUs independently,
// and never double-reverse the original line's stock.
func TestCompletePicking_multipleSubstituteRowsOnSameLine_reversesOnlyOptedInShare(t *testing.T) {
	db := newMockGormDBN(t, 9) // read + listLineItemMovements + linkLineItemMovement(2) + mark-replayed(2) + freezeCostAtSale(2) + mark-completed
	sessionRepo, salesOrderRepo := completablePickingFixtures()
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SKUID: strPtr("sku-orig"), Name: "Widget", Quantity: 2}}}
	batchID := "batch-1"
	linkRepo := &stubSalesOrderLineItemStockMovementRepo{byLineItem: map[string][]omsDomain.SalesOrderLineItemStockMovement{
		"li-1": {{SalesOrderLineItemID: "li-1", BatchID: &batchID, QuantityDelta: -2}},
	}}
	qtyOne := int64(1)
	scanRepo := &stubPickScanRepo{listBySession: []omsDomain.PickScan{
		{ID: "scan-1", LineItemID: strPtr("li-1"), SKUID: "sku-sub-a", ActionType: omsDomain.PickScanActionSubstitute, Quantity: 3, OriginalQtyCovered: &qtyOne, ReturnToStock: true},
		{ID: "scan-2", LineItemID: strPtr("li-1"), SKUID: "sku-sub-b", ActionType: omsDomain.PickScanActionSubstitute, Quantity: 4, OriginalQtyCovered: &qtyOne, ReturnToStock: false},
	}}
	recordMovement := &stubRecordMovement{}
	reverseMovement := &stubReverseMovement{}
	uc := appoms.NewCompletePickingUseCase(db, sessionRepo, salesOrderRepo, lineItemRepo, linkRepo, scanRepo, &stubAddLineItemForPickCompletion{}, recordMovement, reverseMovement, &stubStockAvailabilityChecker{}, &stubSKUFinderForShortfall{}, &stubOutboxRepo{}, &stubPaymentRefundClientResolver{})

	result, err := uc.Execute(context.Background(), appoms.CompletePickingInput{SessionID: "session-1", CompletedBy: "user-1"})

	require.NoError(t, err)
	require.Len(t, reverseMovement.calls, 1, "reversal fires exactly once for the line, not once per substitute row")
	assert.Equal(t, int64(1), reverseMovement.calls[0].QuantityDelta, "only scan-1's covered qty (1) is reversed -- scan-2 opted out")

	require.Len(t, recordMovement.calls, 2)
	assert.Equal(t, "sku-sub-a", recordMovement.calls[0].SKUID)
	assert.Equal(t, int64(3), recordMovement.calls[0].Quantity)
	assert.Equal(t, "sku-sub-b", recordMovement.calls[1].SKUID)
	assert.Equal(t, int64(4), recordMovement.calls[1].Quantity)

	require.Len(t, scanRepo.markReplayed, 2)
	assert.ElementsMatch(t, []string{"scan-1", "scan-2"}, scanRepo.markReplayed)
	assert.Empty(t, result.MissingItems, "both rows together cover the original line's full quantity (1+1=2)")
}

func TestCompletePicking_freebieCreatesZeroPricedLineItemAndDecrementsStock(t *testing.T) {
	db := newMockGormDBN(t, 5) // read + linkLineItemMovement(1) + mark-replayed + freezeCostAtSale + mark-completed
	sessionRepo, salesOrderRepo := completablePickingFixtures()
	lineItemRepo := &stubSalesOrderLineItemRepo{}
	scanRepo := &stubPickScanRepo{listBySession: []omsDomain.PickScan{
		{ID: "scan-1", SKUID: "sku-free", ActionType: omsDomain.PickScanActionFreebie, Quantity: 2, Reason: "Sample gift", PriceMinorUnits: nil},
	}}
	addLineItem := &stubAddLineItemForPickCompletion{}
	recordMovement := &stubRecordMovement{}
	linkRepo := &stubSalesOrderLineItemStockMovementRepo{}
	uc := appoms.NewCompletePickingUseCase(db, sessionRepo, salesOrderRepo, lineItemRepo, linkRepo, scanRepo, addLineItem, recordMovement, &stubReverseMovement{}, &stubStockAvailabilityChecker{}, &stubSKUFinderForShortfall{}, &stubOutboxRepo{}, &stubPaymentRefundClientResolver{})

	_, err := uc.Execute(context.Background(), appoms.CompletePickingInput{SessionID: "session-1", CompletedBy: "user-1"})

	require.NoError(t, err)
	require.Len(t, addLineItem.calls, 1)
	assert.Equal(t, "order-1", addLineItem.calls[0].SalesOrderID)
	assert.Equal(t, "sku-free", addLineItem.calls[0].SKUID)
	assert.Equal(t, "Sample gift", addLineItem.calls[0].Name)
	assert.Equal(t, int64(0), addLineItem.calls[0].PriceMinorUnits, "freebie must always be zero-priced")

	require.Len(t, recordMovement.calls, 1)
	assert.Equal(t, "sku-free", recordMovement.calls[0].SKUID)
	assert.Equal(t, int64(2), recordMovement.calls[0].Quantity)

	require.Len(t, linkRepo.created, 1)
	assert.Equal(t, "new-line-1", linkRepo.created[0].SalesOrderLineItemID, "must link to the NEW line item, not any existing one")
}

func TestCompletePicking_newItemUsesPickerEnteredPrice(t *testing.T) {
	db := newMockGormDBN(t, 5)
	sessionRepo, salesOrderRepo := completablePickingFixtures()
	price := int64(1500)
	scanRepo := &stubPickScanRepo{listBySession: []omsDomain.PickScan{
		{ID: "scan-1", SKUID: "sku-new", ActionType: omsDomain.PickScanActionNewItem, Quantity: 1, PriceMinorUnits: &price},
	}}
	addLineItem := &stubAddLineItemForPickCompletion{}
	uc := appoms.NewCompletePickingUseCase(db, sessionRepo, salesOrderRepo, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, scanRepo, addLineItem, &stubRecordMovement{}, &stubReverseMovement{}, &stubStockAvailabilityChecker{}, &stubSKUFinderForShortfall{}, &stubOutboxRepo{}, &stubPaymentRefundClientResolver{})

	_, err := uc.Execute(context.Background(), appoms.CompletePickingInput{SessionID: "session-1", CompletedBy: "user-1"})

	require.NoError(t, err)
	require.Len(t, addLineItem.calls, 1)
	assert.Equal(t, int64(1500), addLineItem.calls[0].PriceMinorUnits)
}

func TestCompletePicking_crashSafetySkipsAlreadyReplayedRow(t *testing.T) {
	db := newMockGormDBN(t, 3) // read + mark-replayed(1 unreplayed) + mark-completed
	sessionRepo, salesOrderRepo := completablePickingFixtures()
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{
		{ID: "li-1", SKUID: strPtr("sku-1"), Quantity: 1}, {ID: "li-2", SKUID: strPtr("sku-2"), Quantity: 1},
	}}
	replayedAt := time.Now().Add(-time.Hour)
	scanRepo := &stubPickScanRepo{listBySession: []omsDomain.PickScan{
		{ID: "scan-1", LineItemID: strPtr("li-1"), SKUID: "sku-1", ActionType: omsDomain.PickScanActionPick, Quantity: 1, ReplayedAt: &replayedAt},
		{ID: "scan-2", LineItemID: strPtr("li-2"), SKUID: "sku-2", ActionType: omsDomain.PickScanActionPick, Quantity: 1},
	}}
	uc := appoms.NewCompletePickingUseCase(db, sessionRepo, salesOrderRepo, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, scanRepo, &stubAddLineItemForPickCompletion{}, &stubRecordMovement{}, &stubReverseMovement{}, &stubStockAvailabilityChecker{}, &stubSKUFinderForShortfall{}, &stubOutboxRepo{}, &stubPaymentRefundClientResolver{})

	_, err := uc.Execute(context.Background(), appoms.CompletePickingInput{SessionID: "session-1", CompletedBy: "user-1"})

	require.NoError(t, err)
	require.Len(t, scanRepo.markReplayed, 1, "only the not-yet-replayed row gets marked -- the already-replayed one is untouched")
	assert.Equal(t, "scan-2", scanRepo.markReplayed[0])
}

func TestCompletePicking_hardStopsOnReplayFailure(t *testing.T) {
	db := newMockGormDBN(t, 1) // only the initial read -- replay fails before anything else
	sessionRepo, salesOrderRepo := completablePickingFixtures()
	scanRepo := &stubPickScanRepo{listBySession: []omsDomain.PickScan{
		{ID: "scan-1", SKUID: "sku-free", ActionType: omsDomain.PickScanActionFreebie, Quantity: 1},
	}}
	recordMovement := &stubRecordMovement{err: invDomain.ErrInsufficientStock}
	uc := appoms.NewCompletePickingUseCase(db, sessionRepo, salesOrderRepo, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, scanRepo, &stubAddLineItemForPickCompletion{}, recordMovement, &stubReverseMovement{}, &stubStockAvailabilityChecker{}, &stubSKUFinderForShortfall{}, &stubOutboxRepo{}, &stubPaymentRefundClientResolver{})

	_, err := uc.Execute(context.Background(), appoms.CompletePickingInput{SessionID: "session-1", CompletedBy: "user-1"})

	require.Error(t, err)
	assert.ErrorIs(t, err, invDomain.ErrInsufficientStock)
	assert.Empty(t, scanRepo.markReplayed)
	assert.Empty(t, sessionRepo.updated, "the session must stay incomplete on a hard replay failure")
}

func TestCompletePicking_rejectsAlreadyCompletedSession(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", Status: omsDomain.PickingSessionStatusCompleted}}
	uc := appoms.NewCompletePickingUseCase(db, sessionRepo, &stubSalesOrderRepo{}, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubPickScanRepo{}, &stubAddLineItemForPickCompletion{}, &stubRecordMovement{}, &stubReverseMovement{}, &stubStockAvailabilityChecker{}, &stubSKUFinderForShortfall{}, &stubOutboxRepo{}, &stubPaymentRefundClientResolver{})

	_, err := uc.Execute(context.Background(), appoms.CompletePickingInput{SessionID: "session-1", CompletedBy: "user-1"})

	assert.ErrorIs(t, err, omsDomain.ErrPickingSessionNotActive)
}

func TestCompletePicking_propagatesNotFound(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDErr: omsDomain.ErrPickingSessionNotFound}
	uc := appoms.NewCompletePickingUseCase(db, sessionRepo, &stubSalesOrderRepo{}, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubPickScanRepo{}, &stubAddLineItemForPickCompletion{}, &stubRecordMovement{}, &stubReverseMovement{}, &stubStockAvailabilityChecker{}, &stubSKUFinderForShortfall{}, &stubOutboxRepo{}, &stubPaymentRefundClientResolver{})

	_, err := uc.Execute(context.Background(), appoms.CompletePickingInput{SessionID: "missing", CompletedBy: "user-1"})

	assert.ErrorIs(t, err, omsDomain.ErrPickingSessionNotFound)
}

func TestCompletePicking_stockShortfall_blocksCompletionAndReportsIt(t *testing.T) {
	db := newMockGormDBN(t, 2) // read + shortfall-name lookup (nothing replayed/completed)
	sessionRepo, salesOrderRepo := completablePickingFixtures()
	scanRepo := &stubPickScanRepo{listBySession: []omsDomain.PickScan{
		{ID: "scan-1", SKUID: "sku-sub", ActionType: omsDomain.PickScanActionSubstitute, LineItemID: strPtr("li-1"), Quantity: 5},
	}}
	stockChecker := &stubStockAvailabilityChecker{shortBySKU: map[string]int64{"sku-sub": 3}}
	recordMovement := &stubRecordMovement{}
	uc := appoms.NewCompletePickingUseCase(db, sessionRepo, salesOrderRepo, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, scanRepo, &stubAddLineItemForPickCompletion{}, recordMovement, &stubReverseMovement{}, stockChecker, &stubSKUFinderForShortfall{}, &stubOutboxRepo{}, &stubPaymentRefundClientResolver{})

	result, err := uc.Execute(context.Background(), appoms.CompletePickingInput{SessionID: "session-1", CompletedBy: "user-1"})

	require.NoError(t, err)
	assert.False(t, result.Completed)
	require.Len(t, result.StockShortfalls, 1)
	assert.Equal(t, "scan-1", result.StockShortfalls[0].ScanID)
	assert.Equal(t, "sku-sub", result.StockShortfalls[0].SKUID)
	assert.Equal(t, "sku-sub", result.StockShortfalls[0].Name, "substitute falls back to the SKU's own SKUCode")
	assert.Equal(t, int64(5), result.StockShortfalls[0].Needed)
	assert.Equal(t, int64(3), result.StockShortfalls[0].Short)
	assert.Empty(t, recordMovement.calls, "nothing is replayed while a shortfall is unresolved")
	assert.Empty(t, scanRepo.markReplayed)
}

func TestCompletePicking_freebieShortfall_usesReasonAsName(t *testing.T) {
	db := newMockGormDBN(t, 1) // read only -- Freebie/NewItem never needs the SKU lookup
	sessionRepo, salesOrderRepo := completablePickingFixtures()
	scanRepo := &stubPickScanRepo{listBySession: []omsDomain.PickScan{
		{ID: "scan-1", SKUID: "sku-free", ActionType: omsDomain.PickScanActionFreebie, Quantity: 2, Reason: "Sample gift"},
	}}
	stockChecker := &stubStockAvailabilityChecker{shortBySKU: map[string]int64{"sku-free": 2}}
	uc := appoms.NewCompletePickingUseCase(db, sessionRepo, salesOrderRepo, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, scanRepo, &stubAddLineItemForPickCompletion{}, &stubRecordMovement{}, &stubReverseMovement{}, stockChecker, &stubSKUFinderForShortfall{}, &stubOutboxRepo{}, &stubPaymentRefundClientResolver{})

	result, err := uc.Execute(context.Background(), appoms.CompletePickingInput{SessionID: "session-1", CompletedBy: "user-1"})

	require.NoError(t, err)
	require.Len(t, result.StockShortfalls, 1)
	assert.Equal(t, "Sample gift", result.StockShortfalls[0].Name)
}

func TestCompletePicking_alreadyResolvedScan_skipsShortfallCheck(t *testing.T) {
	db := newMockGormDBN(t, 5) // read + linkLineItemMovement(1) + mark-replayed + freezeCostAtSale + mark-completed
	sessionRepo, salesOrderRepo := completablePickingFixtures()
	scanRepo := &stubPickScanRepo{listBySession: []omsDomain.PickScan{
		{ID: "scan-1", SKUID: "sku-free", ActionType: omsDomain.PickScanActionFreebie, Quantity: 2, Reason: "Sample gift"},
	}}
	stockChecker := &stubStockAvailabilityChecker{shortBySKU: map[string]int64{"sku-free": 2}}
	addLineItem := &stubAddLineItemForPickCompletion{}
	recordMovement := &stubRecordMovement{}
	uc := appoms.NewCompletePickingUseCase(db, sessionRepo, salesOrderRepo, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, scanRepo, addLineItem, recordMovement, &stubReverseMovement{}, stockChecker, &stubSKUFinderForShortfall{}, &stubOutboxRepo{}, &stubPaymentRefundClientResolver{})

	result, err := uc.Execute(context.Background(), appoms.CompletePickingInput{
		SessionID: "session-1", CompletedBy: "user-1",
		Resolutions: map[string]appoms.StockShortfallResolution{"scan-1": appoms.StockShortfallWithItem},
	})

	require.NoError(t, err)
	assert.Empty(t, stockChecker.calls, "an already-resolved scan must not be checked again")
	assert.True(t, result.Completed)
	require.Len(t, recordMovement.calls, 1)
	assert.True(t, recordMovement.calls[0].ReconcileShortfall, "with_item must reconcile the shortfall")
}

func TestCompletePicking_resolutionWithoutItem_substituteShipsWithoutReplacementAndShowsMissing(t *testing.T) {
	db := newMockGormDBN(t, 4) // read + listLineItemMovements + mark-replayed + mark-completed (no recordMovement tx since it's a fake)
	sessionRepo, salesOrderRepo := completablePickingFixtures()
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SKUID: strPtr("sku-orig"), Name: "Widget", Quantity: 5}}}
	batchID := "batch-1"
	linkRepo := &stubSalesOrderLineItemStockMovementRepo{byLineItem: map[string][]omsDomain.SalesOrderLineItemStockMovement{
		"li-1": {{SalesOrderLineItemID: "li-1", BatchID: &batchID, QuantityDelta: -5}},
	}}
	qtyCovered := int64(5)
	scanRepo := &stubPickScanRepo{listBySession: []omsDomain.PickScan{
		{ID: "scan-1", LineItemID: strPtr("li-1"), SKUID: "sku-sub", ActionType: omsDomain.PickScanActionSubstitute, Quantity: 5, OriginalQtyCovered: &qtyCovered, ReturnToStock: true},
	}}
	recordMovement := &stubRecordMovement{}
	reverseMovement := &stubReverseMovement{}
	uc := appoms.NewCompletePickingUseCase(db, sessionRepo, salesOrderRepo, lineItemRepo, linkRepo, scanRepo, &stubAddLineItemForPickCompletion{}, recordMovement, reverseMovement, &stubStockAvailabilityChecker{}, &stubSKUFinderForShortfall{}, &stubOutboxRepo{}, &stubPaymentRefundClientResolver{})

	result, err := uc.Execute(context.Background(), appoms.CompletePickingInput{
		SessionID: "session-1", CompletedBy: "user-1",
		Resolutions: map[string]appoms.StockShortfallResolution{"scan-1": appoms.StockShortfallWithoutItem},
	})

	require.NoError(t, err)
	assert.True(t, result.Completed)
	require.Len(t, reverseMovement.calls, 1, "the original line's stock is still given back")
	assert.Empty(t, recordMovement.calls, "without_item never decrements the substitute SKU")
	require.Len(t, result.MissingItems, 1, "the line ships with nothing, so it's a gap")
	assert.Equal(t, "li-1", result.MissingItems[0].LineItemID)
}

func TestCompletePicking_resolutionWithoutItem_freebieNeverCreatedOrDecremented(t *testing.T) {
	db := newMockGormDBN(t, 3) // read + mark-replayed + mark-completed (no addLineItem/recordMovement)
	sessionRepo, salesOrderRepo := completablePickingFixtures()
	scanRepo := &stubPickScanRepo{listBySession: []omsDomain.PickScan{
		{ID: "scan-1", SKUID: "sku-free", ActionType: omsDomain.PickScanActionFreebie, Quantity: 2, Reason: "Sample gift"},
	}}
	addLineItem := &stubAddLineItemForPickCompletion{}
	recordMovement := &stubRecordMovement{}
	uc := appoms.NewCompletePickingUseCase(db, sessionRepo, salesOrderRepo, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, scanRepo, addLineItem, recordMovement, &stubReverseMovement{}, &stubStockAvailabilityChecker{}, &stubSKUFinderForShortfall{}, &stubOutboxRepo{}, &stubPaymentRefundClientResolver{})

	result, err := uc.Execute(context.Background(), appoms.CompletePickingInput{
		SessionID: "session-1", CompletedBy: "user-1",
		Resolutions: map[string]appoms.StockShortfallResolution{"scan-1": appoms.StockShortfallWithoutItem},
	})

	require.NoError(t, err)
	assert.True(t, result.Completed)
	assert.Empty(t, addLineItem.calls)
	assert.Empty(t, recordMovement.calls)
	require.Len(t, scanRepo.markReplayed, 1, "the scan is still marked replayed so it's not re-prompted")
}

func TestCompletePicking_refundWithKnownProviderIssuesRealRefund(t *testing.T) {
	db := newMockGormDBN(t, 4) // read + replayRefund's own txn + mark-replayed + mark-completed
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{
		ID: "order-1", WarehouseID: "warehouse-1", PaymentProvider: "stripe", PaymentExternalID: "cs_123",
	}}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{
		{ID: "li-1", SKUID: strPtr("sku-1"), Name: "Widget", Quantity: 2, PriceMinorUnits: 500},
	}}
	scanRepo := &stubPickScanRepo{listBySession: []omsDomain.PickScan{
		{ID: "scan-1", LineItemID: strPtr("li-1"), SKUID: "sku-1", ActionType: omsDomain.PickScanActionRefund, Quantity: 2, Reason: "damaged"},
	}}
	refundClient := &stubPaymentLinkClient{refundRef: "re_123"}
	resolver := &stubPaymentRefundClientResolver{resolveResult: refundClient, resolveOK: true}
	outboxRepo := &stubOutboxRepo{}
	uc := appoms.NewCompletePickingUseCase(db, sessionRepo, salesOrderRepo, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, scanRepo, &stubAddLineItemForPickCompletion{}, &stubRecordMovement{}, &stubReverseMovement{}, &stubStockAvailabilityChecker{}, &stubSKUFinderForShortfall{}, outboxRepo, resolver)

	result, err := uc.Execute(context.Background(), appoms.CompletePickingInput{SessionID: "session-1", CompletedBy: "user-1"})

	require.NoError(t, err)
	assert.True(t, result.Completed)
	assert.Equal(t, "stripe", resolver.gotProvider)
	require.Equal(t, 1, refundClient.refundCalls)
	assert.Equal(t, "cs_123", refundClient.gotExternalPaymentID)
	assert.Equal(t, int64(1000), refundClient.gotAmountMinorUnits, "quantity 2 * price 500")
	assert.Equal(t, "damaged", refundClient.gotReason)

	require.Len(t, lineItemRepo.updateCalls, 1)
	assert.Equal(t, "li-1", lineItemRepo.updateCalls[0].ID)
	assert.Equal(t, int64(2), lineItemRepo.updateCalls[0].Fields["refunded_quantity"])
	assert.Equal(t, int64(1000), lineItemRepo.updateCalls[0].Fields["refunded_amount_minor_units"])

	require.Len(t, salesOrderRepo.updateCalls, 2, "refund status write + the always-on fulfilled write")
	assert.Equal(t, "partially_refunded", salesOrderRepo.updateCalls[0].Fields["status"])

	require.Len(t, outboxRepo.calls, 1)
	assert.Equal(t, "payments.refund.issued", outboxRepo.calls[0])
	require.Empty(t, result.MissingItems, "a refunded line counts as resolved, not missing")
}

func TestCompletePicking_refundWithNoProviderInfoFlagsManualReview(t *testing.T) {
	db := newMockGormDBN(t, 4)
	sessionRepo, salesOrderRepo := completablePickingFixtures() // no PaymentProvider/PaymentExternalID set
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{
		{ID: "li-1", SKUID: strPtr("sku-1"), Name: "Widget", Quantity: 1, PriceMinorUnits: 500},
	}}
	scanRepo := &stubPickScanRepo{listBySession: []omsDomain.PickScan{
		{ID: "scan-1", LineItemID: strPtr("li-1"), SKUID: "sku-1", ActionType: omsDomain.PickScanActionRefund, Quantity: 1},
	}}
	resolver := &stubPaymentRefundClientResolver{} // ok=false -- no connection for an empty provider name
	outboxRepo := &stubOutboxRepo{}
	uc := appoms.NewCompletePickingUseCase(db, sessionRepo, salesOrderRepo, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, scanRepo, &stubAddLineItemForPickCompletion{}, &stubRecordMovement{}, &stubReverseMovement{}, &stubStockAvailabilityChecker{}, &stubSKUFinderForShortfall{}, outboxRepo, resolver)

	result, err := uc.Execute(context.Background(), appoms.CompletePickingInput{SessionID: "session-1", CompletedBy: "user-1"})

	require.NoError(t, err, "a refund with no known payment provider must never block completion")
	assert.True(t, result.Completed)
	assert.Equal(t, "", resolver.gotProvider, "resolver must never even be called with no provider info")

	require.Len(t, salesOrderRepo.updateCalls, 2, "manual-review flag write + the always-on fulfilled write")
	assert.Equal(t, true, salesOrderRepo.updateCalls[0].Fields["needs_manual_refund_review"])
	assert.NotContains(t, salesOrderRepo.updateCalls[0].Fields, "status", "no status change on the manual-review path")

	require.Len(t, outboxRepo.calls, 1)
	assert.Equal(t, "payments.refund.manual_review_required", outboxRepo.calls[0])
}

func TestCompletePicking_refundAPIFailureFallsBackToManualReviewWithoutBlocking(t *testing.T) {
	db := newMockGormDBN(t, 4)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{
		ID: "order-1", WarehouseID: "warehouse-1", PaymentProvider: "stripe", PaymentExternalID: "cs_123",
	}}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{
		{ID: "li-1", SKUID: strPtr("sku-1"), Name: "Widget", Quantity: 1, PriceMinorUnits: 500},
	}}
	scanRepo := &stubPickScanRepo{listBySession: []omsDomain.PickScan{
		{ID: "scan-1", LineItemID: strPtr("li-1"), SKUID: "sku-1", ActionType: omsDomain.PickScanActionRefund, Quantity: 1},
	}}
	refundClient := &stubPaymentLinkClient{refundErr: fmt.Errorf("stripe: already refunded")}
	resolver := &stubPaymentRefundClientResolver{resolveResult: refundClient, resolveOK: true}
	uc := appoms.NewCompletePickingUseCase(db, sessionRepo, salesOrderRepo, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, scanRepo, &stubAddLineItemForPickCompletion{}, &stubRecordMovement{}, &stubReverseMovement{}, &stubStockAvailabilityChecker{}, &stubSKUFinderForShortfall{}, &stubOutboxRepo{}, resolver)

	result, err := uc.Execute(context.Background(), appoms.CompletePickingInput{SessionID: "session-1", CompletedBy: "user-1"})

	require.NoError(t, err, "an API refund failure must never block picking completion")
	assert.True(t, result.Completed)
	require.Len(t, salesOrderRepo.updateCalls, 2, "manual-review flag write + the always-on fulfilled write")
	assert.Equal(t, true, salesOrderRepo.updateCalls[0].Fields["needs_manual_refund_review"])
}
