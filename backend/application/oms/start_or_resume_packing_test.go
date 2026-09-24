package oms_test

import (
	"context"
	"errors"
	"testing"
	"time"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartOrResumePacking_createsNewSessionWhenNoneActive(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", Fulfilled: true, Status: omsDomain.SalesOrderStatusProcessing}}
	sessionRepo := &stubPackingSessionRepo{}
	uc := appoms.NewStartOrResumePackingUseCase(db, salesOrderRepo, sessionRepo, &stubPickingSessionRepo{}, &stubPickScanRepo{})

	session, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "user-1"})

	require.NoError(t, err)
	require.Len(t, sessionRepo.created, 1)
	assert.Equal(t, "order-1", session.SalesOrderID)
	assert.Equal(t, omsDomain.PackingSessionStatusInProgress, session.Status)
	assert.Nil(t, session.LockedBy, "opening only views -- the first action takes the lock")
}

func TestStartOrResumePacking_reusesInProgressSession(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", Fulfilled: true, Status: omsDomain.SalesOrderStatusProcessing}}
	sessionRepo := &stubPackingSessionRepo{activeResult: &omsDomain.PackingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusInProgress}}
	uc := appoms.NewStartOrResumePackingUseCase(db, salesOrderRepo, sessionRepo, &stubPickingSessionRepo{}, &stubPickScanRepo{})

	session, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "user-1"})

	require.NoError(t, err)
	assert.Empty(t, sessionRepo.created, "must not create a duplicate session")
	assert.Equal(t, "session-1", session.ID)
	assert.Empty(t, sessionRepo.acquiredLocks, "opening only views")
}

func TestStartOrResumePacking_allowsUnpickedOrderInPickableStatus(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", Status: omsDomain.SalesOrderStatusProcessing}}
	sessionRepo := &stubPackingSessionRepo{}
	uc := appoms.NewStartOrResumePackingUseCase(db, salesOrderRepo, sessionRepo, &stubPickingSessionRepo{}, &stubPickScanRepo{})

	session, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "user-1"})

	require.NoError(t, err)
	require.Len(t, sessionRepo.created, 1)
	assert.Equal(t, "order-1", session.SalesOrderID)
}

func TestStartOrResumePacking_rejectsUnpickedOrderInTerminalStatus(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", Status: omsDomain.SalesOrderStatusCancelled}}
	sessionRepo := &stubPackingSessionRepo{}
	uc := appoms.NewStartOrResumePackingUseCase(db, salesOrderRepo, sessionRepo, &stubPickingSessionRepo{}, &stubPickScanRepo{})

	_, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "user-1"})

	assert.ErrorIs(t, err, omsDomain.ErrOrderNotPackable)
	assert.Empty(t, sessionRepo.created)
}

func TestStartOrResumePacking_rejectsWhenPickingLockHeldByAnotherUser(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", Status: omsDomain.SalesOrderStatusProcessing}}
	sessionRepo := &stubPackingSessionRepo{}
	picker, now := "picker", time.Now()
	pickingRepo := &stubPickingSessionRepo{activeResult: &omsDomain.PickingSession{
		ID: "picking-1", Status: omsDomain.PickingSessionStatusInProgress, LockedBy: &picker, LockedAt: &now,
	}}
	uc := appoms.NewStartOrResumePackingUseCase(db, salesOrderRepo, sessionRepo, pickingRepo, &stubPickScanRepo{})

	_, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "user-1"})

	assert.ErrorIs(t, err, omsDomain.ErrOrderPickingInProgress)
	assert.Empty(t, sessionRepo.created)
	assert.Empty(t, pickingRepo.updated)
}

func TestStartOrResumePacking_abandonsStalePickingSessionAndStarts(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", Status: omsDomain.SalesOrderStatusProcessing}}
	sessionRepo := &stubPackingSessionRepo{}
	picker, stale := "picker", time.Now().Add(-omsDomain.SessionLockTTL-time.Minute)
	pickingRepo := &stubPickingSessionRepo{activeResult: &omsDomain.PickingSession{
		ID: "picking-1", Status: omsDomain.PickingSessionStatusPaused, LockedBy: &picker, LockedAt: &stale,
	}}
	scans := &stubPickScanRepo{listBySession: []omsDomain.PickScan{{ActionType: omsDomain.PickScanActionPick}}}
	uc := appoms.NewStartOrResumePackingUseCase(db, salesOrderRepo, sessionRepo, pickingRepo, scans)

	_, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "user-1"})

	require.NoError(t, err)
	require.Len(t, pickingRepo.updated, 1)
	abandoned := pickingRepo.updated[0]
	assert.Equal(t, omsDomain.PickingSessionStatusAbandoned, abandoned.Status)
	assert.Equal(t, "user-1", abandoned.CompletedBy)
	assert.NotNil(t, abandoned.CompletedAt)
	assert.Nil(t, abandoned.LockedBy)
	assert.Equal(t, []string{"picking-1"}, pickingRepo.released)
	require.Len(t, sessionRepo.created, 1)
}

func TestStartOrResumePacking_adminTakesOverActivePicking(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", Status: omsDomain.SalesOrderStatusProcessing}}
	sessionRepo := &stubPackingSessionRepo{}
	picker, now := "picker", time.Now()
	pickingRepo := &stubPickingSessionRepo{activeResult: &omsDomain.PickingSession{
		ID: "picking-1", Status: omsDomain.PickingSessionStatusInProgress, LockedBy: &picker, LockedAt: &now,
	}}
	uc := appoms.NewStartOrResumePackingUseCase(db, salesOrderRepo, sessionRepo, pickingRepo, &stubPickScanRepo{})

	_, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "admin-1", IsAdmin: true})

	require.NoError(t, err)
	require.Len(t, pickingRepo.updated, 1)
	assert.Equal(t, omsDomain.PickingSessionStatusAbandoned, pickingRepo.updated[0].Status)
	require.Len(t, sessionRepo.created, 1)
}

func TestStartOrResumePacking_refusesToAbandonPickingWithPendingChanges(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", Status: omsDomain.SalesOrderStatusProcessing}}
	sessionRepo := &stubPackingSessionRepo{}
	pickingRepo := &stubPickingSessionRepo{activeResult: &omsDomain.PickingSession{ID: "picking-1", Status: omsDomain.PickingSessionStatusPaused}}
	scans := &stubPickScanRepo{listBySession: []omsDomain.PickScan{{ActionType: omsDomain.PickScanActionRefund}}}
	uc := appoms.NewStartOrResumePackingUseCase(db, salesOrderRepo, sessionRepo, pickingRepo, scans)

	_, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "user-1"})

	assert.ErrorIs(t, err, omsDomain.ErrPickingSessionHasPendingChanges)
	assert.Empty(t, pickingRepo.updated)
	assert.Empty(t, sessionRepo.created)
}

func TestStartOrResumePacking_propagatesPickingSessionLookupError(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", Status: omsDomain.SalesOrderStatusProcessing}}
	boom := errors.New("boom")
	uc := appoms.NewStartOrResumePackingUseCase(db, salesOrderRepo, &stubPackingSessionRepo{}, &stubPickingSessionRepo{activeErr: boom}, &stubPickScanRepo{})

	_, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "user-1"})

	assert.ErrorIs(t, err, boom)
}

func TestStartOrResumePacking_rejectsAlreadyPackedOrder(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", Fulfilled: true, Packed: true}}
	sessionRepo := &stubPackingSessionRepo{}
	uc := appoms.NewStartOrResumePackingUseCase(db, salesOrderRepo, sessionRepo, &stubPickingSessionRepo{}, &stubPickScanRepo{})

	_, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "user-1"})

	assert.ErrorIs(t, err, omsDomain.ErrOrderAlreadyPacked)
	assert.Empty(t, sessionRepo.created)
}

func TestStartOrResumePacking_opensAnotherUsersSessionReadOnly(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", Fulfilled: true, Status: omsDomain.SalesOrderStatusProcessing}}
	otherUser := "user-2"
	lockedAt := time.Now()
	sessionRepo := &stubPackingSessionRepo{activeResult: &omsDomain.PackingSession{
		ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusInProgress,
		LockedBy: &otherUser, LockedAt: &lockedAt,
	}}
	uc := appoms.NewStartOrResumePackingUseCase(db, salesOrderRepo, sessionRepo, &stubPickingSessionRepo{}, &stubPickScanRepo{})

	session, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "user-1"})

	require.NoError(t, err, "anyone may look at an order someone else is packing")
	assert.Empty(t, sessionRepo.acquiredLocks)
	require.NotNil(t, session.LockedBy)
	assert.Equal(t, "user-2", *session.LockedBy, "the lock is reported so the screen can show who has it")
}

func TestStartOrResumePacking_rejectsPickedOrderThatAlreadyShipped(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", Fulfilled: true, Status: omsDomain.SalesOrderStatusShipped}}
	sessionRepo := &stubPackingSessionRepo{}
	uc := appoms.NewStartOrResumePackingUseCase(db, salesOrderRepo, sessionRepo, &stubPickingSessionRepo{}, &stubPickScanRepo{})

	_, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "user-1"})

	assert.ErrorIs(t, err, omsDomain.ErrOrderNotPackable)
	assert.Empty(t, sessionRepo.created)
}

func TestStartOrResumePacking_rejectsPOSOrder(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", Source: omsDomain.SalesOrderSourcePOS, Status: omsDomain.SalesOrderStatusProcessing}}
	sessionRepo := &stubPackingSessionRepo{}
	uc := appoms.NewStartOrResumePackingUseCase(db, salesOrderRepo, sessionRepo, &stubPickingSessionRepo{}, &stubPickScanRepo{})

	_, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "user-1"})

	assert.ErrorIs(t, err, omsDomain.ErrPOSOrderNotFulfillable)
	assert.Empty(t, sessionRepo.created)
}
