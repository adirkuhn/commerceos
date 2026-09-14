package oms_test

import (
	"context"
	"testing"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubstituteLineItem_rejectsEmptyLines(t *testing.T) {
	db := newMockGormDB(t)
	uc := appoms.NewSubstituteLineItemUseCase(db, &stubPickingSessionRepo{}, &stubPickScanRepo{}, &stubSalesOrderLineItemRepo{})

	_, err := uc.Execute(context.Background(), appoms.SubstituteLineItemInput{SessionID: "session-1", LineItemID: "li-1"})

	assert.ErrorIs(t, err, omsDomain.ErrSubstituteLinesRequired)
}

func TestSubstituteLineItem_rejectsNonPositiveSubstituteQuantity(t *testing.T) {
	db := newMockGormDB(t)
	uc := appoms.NewSubstituteLineItemUseCase(db, &stubPickingSessionRepo{}, &stubPickScanRepo{}, &stubSalesOrderLineItemRepo{})

	_, err := uc.Execute(context.Background(), appoms.SubstituteLineItemInput{
		SessionID: "session-1", LineItemID: "li-1",
		Lines: []appoms.SubstituteLineInput{{SKUID: "sku-a", SubstituteQuantity: 0, OriginalQtyCovered: 1}},
	})

	assert.ErrorIs(t, err, omsDomain.ErrInvalidSubstituteQuantity)
}

func TestSubstituteLineItem_rejectsNonPositiveOriginalQtyCovered(t *testing.T) {
	db := newMockGormDB(t)
	uc := appoms.NewSubstituteLineItemUseCase(db, &stubPickingSessionRepo{}, &stubPickScanRepo{}, &stubSalesOrderLineItemRepo{})

	_, err := uc.Execute(context.Background(), appoms.SubstituteLineItemInput{
		SessionID: "session-1", LineItemID: "li-1",
		Lines: []appoms.SubstituteLineInput{{SKUID: "sku-a", SubstituteQuantity: 1, OriginalQtyCovered: 0}},
	})

	assert.ErrorIs(t, err, omsDomain.ErrInvalidSubstituteQuantity)
}

func TestSubstituteLineItem_rejectsEmptySKUID(t *testing.T) {
	db := newMockGormDB(t)
	uc := appoms.NewSubstituteLineItemUseCase(db, &stubPickingSessionRepo{}, &stubPickScanRepo{}, &stubSalesOrderLineItemRepo{})

	_, err := uc.Execute(context.Background(), appoms.SubstituteLineItemInput{
		SessionID: "session-1", LineItemID: "li-1",
		Lines: []appoms.SubstituteLineInput{{SKUID: "", SubstituteQuantity: 1, OriginalQtyCovered: 1}},
	})

	assert.ErrorIs(t, err, omsDomain.ErrInvalidSubstituteQuantity)
}

func TestSubstituteLineItem_rejectsWhenSessionNotFound(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDErr: omsDomain.ErrPickingSessionNotFound}
	uc := appoms.NewSubstituteLineItemUseCase(db, sessionRepo, &stubPickScanRepo{}, &stubSalesOrderLineItemRepo{})

	_, err := uc.Execute(context.Background(), appoms.SubstituteLineItemInput{
		SessionID: "session-1", LineItemID: "li-1",
		Lines: []appoms.SubstituteLineInput{{SKUID: "sku-a", SubstituteQuantity: 1, OriginalQtyCovered: 1}},
	})

	assert.ErrorIs(t, err, omsDomain.ErrPickingSessionNotFound)
}

func TestSubstituteLineItem_rejectsWhenSessionNotActive(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusPaused}}
	uc := appoms.NewSubstituteLineItemUseCase(db, sessionRepo, &stubPickScanRepo{}, &stubSalesOrderLineItemRepo{})

	_, err := uc.Execute(context.Background(), appoms.SubstituteLineItemInput{
		SessionID: "session-1", LineItemID: "li-1",
		Lines: []appoms.SubstituteLineInput{{SKUID: "sku-a", SubstituteQuantity: 1, OriginalQtyCovered: 1}},
	})

	assert.ErrorIs(t, err, omsDomain.ErrPickingSessionNotActive)
}

func TestSubstituteLineItem_rejectsLineItemNotFound(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{findByIDErr: omsDomain.ErrSalesOrderLineItemNotFound}
	uc := appoms.NewSubstituteLineItemUseCase(db, sessionRepo, &stubPickScanRepo{}, lineItemRepo)

	_, err := uc.Execute(context.Background(), appoms.SubstituteLineItemInput{
		SessionID: "session-1", LineItemID: "li-1",
		Lines: []appoms.SubstituteLineInput{{SKUID: "sku-a", SubstituteQuantity: 1, OriginalQtyCovered: 1}},
	})

	assert.ErrorIs(t, err, omsDomain.ErrSalesOrderLineItemNotFound)
}

func TestSubstituteLineItem_rejectsLineItemFromAnotherOrder(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{findByIDResult: &omsDomain.SalesOrderLineItem{ID: "li-1", SalesOrderID: "order-2", Quantity: 1}}
	uc := appoms.NewSubstituteLineItemUseCase(db, sessionRepo, &stubPickScanRepo{}, lineItemRepo)

	_, err := uc.Execute(context.Background(), appoms.SubstituteLineItemInput{
		SessionID: "session-1", LineItemID: "li-1",
		Lines: []appoms.SubstituteLineInput{{SKUID: "sku-a", SubstituteQuantity: 1, OriginalQtyCovered: 1}},
	})

	assert.ErrorIs(t, err, omsDomain.ErrLineItemNotOnOrder)
}

func TestSubstituteLineItem_rejectsCoverageTooLow(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{findByIDResult: &omsDomain.SalesOrderLineItem{ID: "li-1", SalesOrderID: "order-1", Quantity: 5}}
	uc := appoms.NewSubstituteLineItemUseCase(db, sessionRepo, &stubPickScanRepo{}, lineItemRepo)

	_, err := uc.Execute(context.Background(), appoms.SubstituteLineItemInput{
		SessionID: "session-1", LineItemID: "li-1",
		Lines: []appoms.SubstituteLineInput{{SKUID: "sku-a", SubstituteQuantity: 3, OriginalQtyCovered: 4}},
	})

	assert.ErrorIs(t, err, omsDomain.ErrSubstitutionCoverageMismatch)
}

func TestSubstituteLineItem_rejectsCoverageTooHigh(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{findByIDResult: &omsDomain.SalesOrderLineItem{ID: "li-1", SalesOrderID: "order-1", Quantity: 5}}
	uc := appoms.NewSubstituteLineItemUseCase(db, sessionRepo, &stubPickScanRepo{}, lineItemRepo)

	_, err := uc.Execute(context.Background(), appoms.SubstituteLineItemInput{
		SessionID: "session-1", LineItemID: "li-1",
		Lines: []appoms.SubstituteLineInput{{SKUID: "sku-a", SubstituteQuantity: 3, OriginalQtyCovered: 6}},
	})

	assert.ErrorIs(t, err, omsDomain.ErrSubstitutionCoverageMismatch)
}

func TestSubstituteLineItem_happyPathSingleLine(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{findByIDResult: &omsDomain.SalesOrderLineItem{ID: "li-1", SalesOrderID: "order-1", Quantity: 5}}
	scanRepo := &stubPickScanRepo{}
	uc := appoms.NewSubstituteLineItemUseCase(db, sessionRepo, scanRepo, lineItemRepo)

	result, err := uc.Execute(context.Background(), appoms.SubstituteLineItemInput{
		SessionID: "session-1", LineItemID: "li-1", ScannedBy: "user-1", Reason: "out of stock",
		Lines: []appoms.SubstituteLineInput{{SKUID: "sku-sub", SubstituteQuantity: 5, OriginalQtyCovered: 5, ReturnToStock: true}},
	})

	require.NoError(t, err)
	require.Len(t, scanRepo.created, 1)
	require.Len(t, result.Scans, 1)
	s := result.Scans[0]
	assert.Equal(t, omsDomain.PickScanActionSubstitute, s.ActionType)
	require.NotNil(t, s.LineItemID)
	assert.Equal(t, "li-1", *s.LineItemID)
	assert.Equal(t, "sku-sub", s.SKUID)
	assert.Equal(t, int64(5), s.Quantity)
	require.NotNil(t, s.OriginalQtyCovered)
	assert.Equal(t, int64(5), *s.OriginalQtyCovered)
	assert.True(t, s.ReturnToStock)
	assert.Equal(t, "user-1", s.ScannedBy)
	assert.Equal(t, "out of stock", s.Reason)
}

// TestSubstituteLineItem_partialPick_substituteJustTheRemainder is a
// regression test for a real bug: an order line for 4x Beans, 3 already
// picked (staged Pick scans), and a picker wants to substitute only the
// last unit. The coverage check used to require covering the line's full
// original Quantity (4) regardless of what was already picked, making a
// partial substitute impossible. It must now require covering exactly
// what's left (1).
func TestSubstituteLineItem_partialPick_substituteJustTheRemainder(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{findByIDResult: &omsDomain.SalesOrderLineItem{ID: "li-beans", SalesOrderID: "order-1", Quantity: 4}}
	scanRepo := &stubPickScanRepo{listBySession: []omsDomain.PickScan{
		{ID: "scan-1", LineItemID: strPtr("li-beans"), ActionType: omsDomain.PickScanActionPick, Quantity: 3},
	}}
	uc := appoms.NewSubstituteLineItemUseCase(db, sessionRepo, scanRepo, lineItemRepo)

	result, err := uc.Execute(context.Background(), appoms.SubstituteLineItemInput{
		SessionID: "session-1", LineItemID: "li-beans", ScannedBy: "user-1", Reason: "out of stock",
		Lines: []appoms.SubstituteLineInput{{SKUID: "sku-canned-beans", SubstituteQuantity: 1, OriginalQtyCovered: 1}},
	})

	require.NoError(t, err)
	require.Len(t, result.Scans, 1)
	assert.Equal(t, int64(1), *result.Scans[0].OriginalQtyCovered)
}

// TestSubstituteLineItem_partialPick_rejectsSubstitutingTheFullOriginalQuantity
// is the flip side: after 3 of 4 are already picked, trying to substitute
// all 4 (the old, buggy requirement) must now be rejected -- only 1 unit is
// actually available to substitute.
func TestSubstituteLineItem_partialPick_rejectsSubstitutingTheFullOriginalQuantity(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{findByIDResult: &omsDomain.SalesOrderLineItem{ID: "li-beans", SalesOrderID: "order-1", Quantity: 4}}
	scanRepo := &stubPickScanRepo{listBySession: []omsDomain.PickScan{
		{ID: "scan-1", LineItemID: strPtr("li-beans"), ActionType: omsDomain.PickScanActionPick, Quantity: 3},
	}}
	uc := appoms.NewSubstituteLineItemUseCase(db, sessionRepo, scanRepo, lineItemRepo)

	_, err := uc.Execute(context.Background(), appoms.SubstituteLineItemInput{
		SessionID: "session-1", LineItemID: "li-beans",
		Lines: []appoms.SubstituteLineInput{{SKUID: "sku-canned-beans", SubstituteQuantity: 4, OriginalQtyCovered: 4}},
	})

	assert.ErrorIs(t, err, omsDomain.ErrSubstitutionCoverageMismatch)
}

// TestSubstituteLineItem_coverageCountsExistingSubstituteScansToo proves
// coveredQuantity's second scan type (Substitute, via OriginalQtyCovered)
// is honored too, not just Pick -- e.g. a prior partial substitute already
// covered 2 of 4, so a second batch must cover exactly the remaining 2.
func TestSubstituteLineItem_coverageCountsExistingSubstituteScansToo(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{findByIDResult: &omsDomain.SalesOrderLineItem{ID: "li-beans", SalesOrderID: "order-1", Quantity: 4}}
	alreadyCovered := int64(2)
	scanRepo := &stubPickScanRepo{listBySession: []omsDomain.PickScan{
		{ID: "scan-1", LineItemID: strPtr("li-beans"), ActionType: omsDomain.PickScanActionSubstitute, Quantity: 10, OriginalQtyCovered: &alreadyCovered},
	}}
	uc := appoms.NewSubstituteLineItemUseCase(db, sessionRepo, scanRepo, lineItemRepo)

	result, err := uc.Execute(context.Background(), appoms.SubstituteLineItemInput{
		SessionID: "session-1", LineItemID: "li-beans",
		Lines: []appoms.SubstituteLineInput{{SKUID: "sku-canned-beans", SubstituteQuantity: 2, OriginalQtyCovered: 2}},
	})

	require.NoError(t, err)
	require.Len(t, result.Scans, 1)
}

// TestSubstituteLineItem_ignoresOtherLineItemsScans proves coveredQuantity
// filters by LineItemID -- a Pick scan against a different line on the
// same order/session must not affect this line's remaining-quantity math.
func TestSubstituteLineItem_ignoresOtherLineItemsScans(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{findByIDResult: &omsDomain.SalesOrderLineItem{ID: "li-beans", SalesOrderID: "order-1", Quantity: 4}}
	scanRepo := &stubPickScanRepo{listBySession: []omsDomain.PickScan{
		{ID: "scan-1", LineItemID: strPtr("li-other"), ActionType: omsDomain.PickScanActionPick, Quantity: 3},
	}}
	uc := appoms.NewSubstituteLineItemUseCase(db, sessionRepo, scanRepo, lineItemRepo)

	_, err := uc.Execute(context.Background(), appoms.SubstituteLineItemInput{
		SessionID: "session-1", LineItemID: "li-beans",
		Lines: []appoms.SubstituteLineInput{{SKUID: "sku-canned-beans", SubstituteQuantity: 4, OriginalQtyCovered: 4}},
	})

	require.NoError(t, err, "the other line item's Pick scan must not reduce li-beans' own remaining quantity")
}

func TestSubstituteLineItem_happyPathMultipleLinesSplitAcrossCheaperSKUs(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	// One original unit covered jointly by two different cheaper SKUs
	// (3 of one, 4 of the other) plus a second original unit returned to
	// stock via a third substitute line -- exercises both multi-line split
	// and per-line ReturnToStock varying within the same batch.
	lineItemRepo := &stubSalesOrderLineItemRepo{findByIDResult: &omsDomain.SalesOrderLineItem{ID: "li-1", SalesOrderID: "order-1", Quantity: 2}}
	scanRepo := &stubPickScanRepo{}
	uc := appoms.NewSubstituteLineItemUseCase(db, sessionRepo, scanRepo, lineItemRepo)

	result, err := uc.Execute(context.Background(), appoms.SubstituteLineItemInput{
		SessionID: "session-1", LineItemID: "li-1",
		Lines: []appoms.SubstituteLineInput{
			{SKUID: "sku-cheap-a", SubstituteQuantity: 3, OriginalQtyCovered: 1, ReturnToStock: false},
			{SKUID: "sku-cheap-b", SubstituteQuantity: 4, OriginalQtyCovered: 1, ReturnToStock: true},
		},
	})

	require.NoError(t, err)
	require.Len(t, scanRepo.created, 2)
	require.Len(t, result.Scans, 2)
	assert.Equal(t, "sku-cheap-a", result.Scans[0].SKUID)
	assert.Equal(t, int64(3), result.Scans[0].Quantity)
	assert.False(t, result.Scans[0].ReturnToStock)
	assert.Equal(t, "sku-cheap-b", result.Scans[1].SKUID)
	assert.Equal(t, int64(4), result.Scans[1].Quantity)
	assert.True(t, result.Scans[1].ReturnToStock)
	for _, s := range result.Scans {
		require.NotNil(t, s.LineItemID)
		assert.Equal(t, "li-1", *s.LineItemID)
		require.NotNil(t, s.OriginalQtyCovered)
		assert.Equal(t, int64(1), *s.OriginalQtyCovered)
	}
}
