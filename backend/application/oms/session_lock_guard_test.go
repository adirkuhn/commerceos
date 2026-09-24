package oms_test

import (
	"context"
	"testing"
	"time"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// After a takeover the previous holder must not keep working the session:
// every write checks the caller still holds (or can hold) the lock.

func lockedBy(user string) (*string, *time.Time) {
	now := time.Now()
	return &user, &now
}

func TestScanPackItem_rejectsCallerWhoLostTheLock(t *testing.T) {
	db := newMockGormDB(t)
	holder, at := lockedBy("admin-1")
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{
		ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusInProgress, LockedBy: holder, LockedAt: at,
	}}
	scanRepo := &stubPackScanRepo{}
	uc := appoms.NewScanPackItemUseCase(db, sessionRepo, scanRepo, &stubSalesOrderLineItemRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubScaleCodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	_, err := uc.Execute(context.Background(), appoms.ScanPackItemInput{SessionID: "session-1", SKUID: "sku-1", Quantity: 1, ScannedBy: "user-1"})

	assert.ErrorIs(t, err, omsDomain.ErrPackingSessionLocked)
	assert.Empty(t, scanRepo.created)
}

func TestCompletePacking_rejectsCallerWhoLostTheLock(t *testing.T) {
	db := newMockGormDB(t)
	holder, at := lockedBy("admin-1")
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{
		ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusInProgress, LockedBy: holder, LockedAt: at,
	}}
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1"}}
	applier := appoms.NewOrderChangeApplier(db, salesOrderRepo, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubAddLineItemForPickCompletion{},
		&stubRecordMovement{}, &stubReverseMovement{}, &stubStockAvailabilityChecker{}, &stubSKUFinderForShortfall{}, &stubOutboxRepo{}, &stubPaymentRefundClientResolver{})
	uc := appoms.NewCompletePackingUseCase(db, sessionRepo, salesOrderRepo, &stubSalesOrderLineItemRepo{}, &stubPackScanRepo{}, applier)

	_, err := uc.Execute(context.Background(), appoms.CompletePackingInput{SessionID: "session-1", CompletedBy: "user-1"})

	assert.ErrorIs(t, err, omsDomain.ErrPackingSessionLocked)
	assert.Empty(t, sessionRepo.updated)
	assert.Empty(t, salesOrderRepo.gotUpdateID)
}

func TestScanPickItem_rejectsCallerWhoLostTheLock(t *testing.T) {
	db := newMockGormDB(t)
	holder, at := lockedBy("admin-1")
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{
		ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress, LockedBy: holder, LockedAt: at,
	}}
	scanRepo := &stubPickScanRepo{}
	uc := appoms.NewScanPickItemUseCase(db, sessionRepo, scanRepo, &stubSalesOrderLineItemRepo{}, &stubSalesOrderRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubScaleCodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	_, err := uc.Execute(context.Background(), appoms.ScanPickItemInput{SessionID: "session-1", SKUID: "sku-1", Quantity: 1, ScannedBy: "user-1"})

	assert.ErrorIs(t, err, omsDomain.ErrPickingSessionLocked)
	assert.Empty(t, scanRepo.created)
}

func TestCompletePicking_rejectsCallerWhoLostTheLock(t *testing.T) {
	db := newMockGormDB(t)
	holder, at := lockedBy("admin-1")
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{
		ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress, LockedBy: holder, LockedAt: at,
	}}
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1"}}
	uc := appoms.NewCompletePickingUseCase(db, sessionRepo, salesOrderRepo, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubPickScanRepo{}, &stubAddLineItemForPickCompletion{}, &stubRecordMovement{}, &stubReverseMovement{}, &stubStockAvailabilityChecker{}, &stubSKUFinderForShortfall{}, &stubOutboxRepo{}, &stubPaymentRefundClientResolver{})

	_, err := uc.Execute(context.Background(), appoms.CompletePickingInput{SessionID: "session-1", CompletedBy: "user-1"})

	assert.ErrorIs(t, err, omsDomain.ErrPickingSessionLocked)
	assert.Empty(t, sessionRepo.updated)
}

func TestCompletePicking_rejectsAbandonedSession(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusAbandoned}}
	uc := appoms.NewCompletePickingUseCase(db, sessionRepo, &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1"}}, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubPickScanRepo{}, &stubAddLineItemForPickCompletion{}, &stubRecordMovement{}, &stubReverseMovement{}, &stubStockAvailabilityChecker{}, &stubSKUFinderForShortfall{}, &stubOutboxRepo{}, &stubPaymentRefundClientResolver{})

	_, err := uc.Execute(context.Background(), appoms.CompletePickingInput{SessionID: "session-1", CompletedBy: "user-1"})

	assert.ErrorIs(t, err, omsDomain.ErrPickingSessionNotActive)
}

func TestSubstituteLineItem_rejectsCallerWhoLostTheLock(t *testing.T) {
	db := newMockGormDB(t)
	holder, at := lockedBy("admin-1")
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{
		ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusInProgress, LockedBy: holder, LockedAt: at,
	}}
	scanRepo := &stubPackScanRepo{}
	uc := appoms.NewSubstituteLineItemUseCase(db, sessionRepo, scanRepo, &stubSalesOrderLineItemRepo{})

	_, err := uc.Execute(context.Background(), appoms.SubstituteLineItemInput{
		SessionID: "session-1", LineItemID: "li-1", ScannedBy: "user-1",
		Lines: []appoms.SubstituteLineInput{{SKUID: "sku-a", SubstituteQuantity: 1, OriginalQtyCovered: 1}},
	})

	assert.ErrorIs(t, err, omsDomain.ErrPackingSessionLocked)
	assert.Empty(t, scanRepo.created)
}

func TestRefundLineItem_rejectsCallerWhoLostTheLock(t *testing.T) {
	db := newMockGormDB(t)
	holder, at := lockedBy("admin-1")
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{
		ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusInProgress, LockedBy: holder, LockedAt: at,
	}}
	scanRepo := &stubPackScanRepo{}
	uc := appoms.NewRefundLineItemUseCase(db, sessionRepo, scanRepo, &stubSalesOrderLineItemRepo{}, &stubGroupPermissionRepo{})

	_, err := uc.Execute(context.Background(), appoms.RefundLineItemInput{
		SessionID: "session-1", LineItemID: "li-1", Quantity: 1, ScannedBy: "user-1", RequesterUserID: "user-1", RequesterRole: "customer_owner",
	})

	assert.ErrorIs(t, err, omsDomain.ErrPackingSessionLocked)
	assert.Empty(t, scanRepo.created)
}

// Opening a session only views it; the first write takes the lock.

func TestScanPackItem_firstActionTakesTheLock(t *testing.T) {
	db := newMockGormDB(t)
	skuID := "sku-1"
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SKUID: &skuID, Quantity: 2}}}
	uc := appoms.NewScanPackItemUseCase(db, sessionRepo, &stubPackScanRepo{}, lineItemRepo, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubScaleCodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	_, err := uc.Execute(context.Background(), appoms.ScanPackItemInput{SessionID: "session-1", SKUID: skuID, Quantity: 1, ScannedBy: "user-1"})

	require.NoError(t, err)
	assert.Equal(t, []string{"session-1"}, sessionRepo.acquiredLocks)
}

func TestSubstituteLineItem_firstActionTakesTheLock(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{findByIDResult: &omsDomain.SalesOrderLineItem{ID: "li-1", SalesOrderID: "order-1", Quantity: 1}}
	uc := appoms.NewSubstituteLineItemUseCase(db, sessionRepo, &stubPackScanRepo{}, lineItemRepo)

	_, err := uc.Execute(context.Background(), appoms.SubstituteLineItemInput{
		SessionID: "session-1", LineItemID: "li-1", ScannedBy: "user-1",
		Lines: []appoms.SubstituteLineInput{{SKUID: "sku-a", SubstituteQuantity: 1, OriginalQtyCovered: 1}},
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"session-1"}, sessionRepo.acquiredLocks)
}

func TestScanPickItem_firstActionTakesTheLock(t *testing.T) {
	db := newMockGormDB(t)
	skuID := "sku-1"
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SKUID: &skuID, Quantity: 2}}}
	uc := appoms.NewScanPickItemUseCase(db, sessionRepo, &stubPickScanRepo{}, lineItemRepo, &stubSalesOrderRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubScaleCodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	_, err := uc.Execute(context.Background(), appoms.ScanPickItemInput{SessionID: "session-1", SKUID: skuID, Quantity: 1, ScannedBy: "user-1"})

	require.NoError(t, err)
	assert.Equal(t, []string{"session-1"}, sessionRepo.acquiredLocks)
}
