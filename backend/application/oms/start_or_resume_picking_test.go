package oms_test

import (
	"context"
	"github.com/adirkuhn/commerceos/domain/integrations"
	"testing"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartOrResumePicking_createsNewSessionWhenNoneActive(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1"}}
	sessionRepo := &stubPickingSessionRepo{}
	outbox := &eventrecordtest.Outbox{}
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, &stubOrigins{}, sessionRepo, &stubPackingSessionRepo{}, &stubPackScanRepo{}, outbox)

	session, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "user-1"})

	require.NoError(t, err)
	require.Len(t, sessionRepo.created, 1)
	assert.Equal(t, "order-1", session.SalesOrderID)
	assert.Equal(t, omsDomain.PickingSessionStatusInProgress, session.Status)
	assert.Nil(t, session.LockedBy, "opening only views -- the first scan takes the lock")

	assert.Contains(t, outbox.Subjects(), "oms.picking_session.started")

}

func TestStartOrResumePicking_reusesInProgressSession(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1"}}
	sessionRepo := &stubPickingSessionRepo{activeResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, &stubOrigins{}, sessionRepo, &stubPackingSessionRepo{}, &stubPackScanRepo{}, &eventrecordtest.Outbox{})

	session, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "user-1"})

	require.NoError(t, err)
	assert.Empty(t, sessionRepo.created, "must not create a duplicate session")
	assert.Equal(t, "session-1", session.ID)
	assert.Empty(t, sessionRepo.acquiredLocks, "opening only views")
}

func TestStartOrResumePicking_resumesPausedSession(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1"}}
	sessionRepo := &stubPickingSessionRepo{activeResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusPaused}}
	outbox := &eventrecordtest.Outbox{}
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, &stubOrigins{}, sessionRepo, &stubPackingSessionRepo{}, &stubPackScanRepo{}, outbox)

	session, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "user-1"})

	require.NoError(t, err)
	require.Len(t, sessionRepo.updated, 1)
	assert.Equal(t, omsDomain.PickingSessionStatusInProgress, sessionRepo.updated[0].Status)
	assert.Nil(t, sessionRepo.updated[0].PausedAt)
	assert.Equal(t, omsDomain.PickingSessionStatusInProgress, session.Status)

	assert.Contains(t, outbox.Subjects(), "oms.picking_session.resumed")

}

func TestStartOrResumePicking_rejectsAlreadyFulfilledOrder(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", Fulfilled: true}}
	sessionRepo := &stubPickingSessionRepo{}
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, &stubOrigins{}, sessionRepo, &stubPackingSessionRepo{}, &stubPackScanRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "user-1"})

	assert.ErrorIs(t, err, omsDomain.ErrOrderAlreadyFulfilled)
	assert.Empty(t, sessionRepo.created)
}

func TestStartOrResumePicking_rejectsAlreadyPackedOrder(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", Packed: true}}
	sessionRepo := &stubPickingSessionRepo{}
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, &stubOrigins{}, sessionRepo, &stubPackingSessionRepo{}, &stubPackScanRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "user-1"})

	assert.ErrorIs(t, err, omsDomain.ErrOrderAlreadyPacked)
	assert.Empty(t, sessionRepo.created)
}

func TestStartOrResumePicking_rejectsWhenPackingSessionInProgress(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", Status: omsDomain.SalesOrderStatusProcessing}}
	sessionRepo := &stubPickingSessionRepo{}
	packer, now := "packer", time.Now()
	packingRepo := &stubPackingSessionRepo{activeResult: &omsDomain.PackingSession{ID: "packing-1", Status: omsDomain.PackingSessionStatusInProgress, LockedBy: &packer, LockedAt: &now}}
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, &stubOrigins{}, sessionRepo, packingRepo, &stubPackScanRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "user-1"})

	assert.ErrorIs(t, err, omsDomain.ErrOrderPackingInProgress)
	assert.Empty(t, sessionRepo.created)
}

func TestStartOrResumePicking_rejectsWhenPackingAlreadyRecordedSomething(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", Status: omsDomain.SalesOrderStatusProcessing}}
	packingRepo := &stubPackingSessionRepo{activeResult: &omsDomain.PackingSession{ID: "packing-1", Status: omsDomain.PackingSessionStatusInProgress}}
	scans := &stubPackScanRepo{sessionsWithScans: []string{"packing-1"}}
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, &stubOrigins{}, &stubPickingSessionRepo{}, packingRepo, scans, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "user-1"})

	assert.ErrorIs(t, err, omsDomain.ErrOrderPackingInProgress)
	assert.Equal(t, []string{"packing-1"}, scans.gotSessionIDs)
}

func TestStartOrResumePicking_aPackingSessionMerelyOpenedDoesNotBlock(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", Status: omsDomain.SalesOrderStatusProcessing}}
	sessionRepo := &stubPickingSessionRepo{}
	packingRepo := &stubPackingSessionRepo{activeResult: &omsDomain.PackingSession{ID: "packing-1", Status: omsDomain.PackingSessionStatusInProgress}}
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, &stubOrigins{}, sessionRepo, packingRepo, &stubPackScanRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "user-1"})

	require.NoError(t, err)
	require.Len(t, sessionRepo.created, 1)
}

func TestStartOrResumePicking_rejectsPOSOrder(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", Status: omsDomain.SalesOrderStatusProcessing}}
	sessionRepo := &stubPickingSessionRepo{}
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, &stubOrigins{source: integrations.OriginPOS}, sessionRepo, &stubPackingSessionRepo{}, &stubPackScanRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "user-1"})

	assert.ErrorIs(t, err, omsDomain.ErrPOSOrderNotFulfillable)
	assert.Empty(t, sessionRepo.created)
}

func TestStartOrResumePicking_propagatesSalesOrderNotFound(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDErr: omsDomain.ErrSalesOrderNotFound}
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, &stubOrigins{}, &stubPickingSessionRepo{}, &stubPackingSessionRepo{}, &stubPackScanRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "missing", omsDomain.LockActor{UserID: "user-1"})

	assert.ErrorIs(t, err, omsDomain.ErrSalesOrderNotFound)
}

func TestStartOrResumePicking_opensAnotherUsersSessionReadOnly(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1"}}
	otherUser := "user-2"
	lockedAt := time.Now()
	sessionRepo := &stubPickingSessionRepo{activeResult: &omsDomain.PickingSession{
		ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress,
		LockedBy: &otherUser, LockedAt: &lockedAt,
	}}
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, &stubOrigins{}, sessionRepo, &stubPackingSessionRepo{}, &stubPackScanRepo{}, &eventrecordtest.Outbox{})

	session, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "user-1"})

	require.NoError(t, err)
	assert.Empty(t, sessionRepo.acquiredLocks)
	assert.Equal(t, "user-2", *session.LockedBy)
}
