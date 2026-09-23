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

func TestStartOrResumePicking_createsNewSessionWhenNoneActive(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1"}}
	sessionRepo := &stubPickingSessionRepo{}
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, sessionRepo, &stubPackingSessionRepo{})

	session, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "user-1"})

	require.NoError(t, err)
	require.Len(t, sessionRepo.created, 1)
	assert.Equal(t, "order-1", session.SalesOrderID)
	assert.Equal(t, omsDomain.PickingSessionStatusInProgress, session.Status)
	require.NotNil(t, session.LockedBy)
	assert.Equal(t, "user-1", *session.LockedBy)
}

func TestStartOrResumePicking_reusesInProgressSession(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1"}}
	sessionRepo := &stubPickingSessionRepo{activeResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, sessionRepo, &stubPackingSessionRepo{})

	session, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "user-1"})

	require.NoError(t, err)
	assert.Empty(t, sessionRepo.created, "must not create a duplicate session")
	assert.Equal(t, "session-1", session.ID)
	assert.Equal(t, []string{"session-1"}, sessionRepo.acquiredLocks)
}

func TestStartOrResumePicking_resumesPausedSession(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1"}}
	sessionRepo := &stubPickingSessionRepo{activeResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusPaused}}
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, sessionRepo, &stubPackingSessionRepo{})

	session, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "user-1"})

	require.NoError(t, err)
	require.Len(t, sessionRepo.updated, 1)
	assert.Equal(t, omsDomain.PickingSessionStatusInProgress, sessionRepo.updated[0].Status)
	assert.Nil(t, sessionRepo.updated[0].PausedAt)
	assert.Equal(t, omsDomain.PickingSessionStatusInProgress, session.Status)
}

func TestStartOrResumePicking_rejectsAlreadyFulfilledOrder(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", Fulfilled: true}}
	sessionRepo := &stubPickingSessionRepo{}
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, sessionRepo, &stubPackingSessionRepo{})

	_, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "user-1"})

	assert.ErrorIs(t, err, omsDomain.ErrOrderAlreadyFulfilled)
	assert.Empty(t, sessionRepo.created)
}

func TestStartOrResumePicking_rejectsAlreadyPackedOrder(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", Packed: true}}
	sessionRepo := &stubPickingSessionRepo{}
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, sessionRepo, &stubPackingSessionRepo{})

	_, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "user-1"})

	assert.ErrorIs(t, err, omsDomain.ErrOrderAlreadyPacked)
	assert.Empty(t, sessionRepo.created)
}

func TestStartOrResumePicking_rejectsWhenPackingSessionInProgress(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", Status: omsDomain.SalesOrderStatusProcessing}}
	sessionRepo := &stubPickingSessionRepo{}
	packingRepo := &stubPackingSessionRepo{activeResult: &omsDomain.PackingSession{ID: "packing-1", Status: omsDomain.PackingSessionStatusInProgress}}
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, sessionRepo, packingRepo)

	_, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "user-1"})

	assert.ErrorIs(t, err, omsDomain.ErrOrderPackingInProgress)
	assert.Empty(t, sessionRepo.created)
}

func TestStartOrResumePicking_propagatesSalesOrderNotFound(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDErr: omsDomain.ErrSalesOrderNotFound}
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, &stubPickingSessionRepo{}, &stubPackingSessionRepo{})

	_, err := uc.Execute(context.Background(), "missing", omsDomain.LockActor{UserID: "user-1"})

	assert.ErrorIs(t, err, omsDomain.ErrSalesOrderNotFound)
}

func TestStartOrResumePicking_rejectsWhenLockedByAnotherFreshUser(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1"}}
	otherUser := "user-2"
	lockedAt := time.Now()
	sessionRepo := &stubPickingSessionRepo{activeResult: &omsDomain.PickingSession{
		ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress,
		LockedBy: &otherUser, LockedAt: &lockedAt,
	}}
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, sessionRepo, &stubPackingSessionRepo{})

	_, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "user-1"})

	assert.ErrorIs(t, err, omsDomain.ErrPickingSessionLocked)
	assert.Empty(t, sessionRepo.acquiredLocks)
}

func TestStartOrResumePicking_allowsTakeoverWhenLockStale(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1"}}
	otherUser := "user-2"
	staleAt := time.Now().Add(-10 * time.Minute)
	sessionRepo := &stubPickingSessionRepo{activeResult: &omsDomain.PickingSession{
		ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress,
		LockedBy: &otherUser, LockedAt: &staleAt,
	}}
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, sessionRepo, &stubPackingSessionRepo{})

	session, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "user-1"})

	require.NoError(t, err)
	assert.Equal(t, []string{"session-1"}, sessionRepo.acquiredLocks)
	assert.Equal(t, "session-1", session.ID)
}

func TestStartOrResumePicking_sameUserReopeningOwnLockSucceeds(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1"}}
	sameUser := "user-1"
	lockedAt := time.Now()
	sessionRepo := &stubPickingSessionRepo{activeResult: &omsDomain.PickingSession{
		ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress,
		LockedBy: &sameUser, LockedAt: &lockedAt,
	}}
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, sessionRepo, &stubPackingSessionRepo{})

	session, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "user-1"})

	require.NoError(t, err)
	assert.Equal(t, "session-1", session.ID)
}

func TestStartOrResumePicking_adminTakesOverFreshLock(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1"}}
	otherUser := "user-2"
	lockedAt := time.Now()
	sessionRepo := &stubPickingSessionRepo{activeResult: &omsDomain.PickingSession{
		ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress,
		LockedBy: &otherUser, LockedAt: &lockedAt, LockedSince: &lockedAt,
	}}
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, sessionRepo, &stubPackingSessionRepo{})

	session, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "admin-1", IsAdmin: true})

	require.NoError(t, err)
	assert.Equal(t, []string{"session-1"}, sessionRepo.acquiredLocks)
	require.NotNil(t, session.LockedBy)
	assert.Equal(t, "admin-1", *session.LockedBy)
	require.NotNil(t, session.LockedSince)
	assert.WithinDuration(t, time.Now(), *session.LockedSince, time.Second, "a new holder's lock starts now")
}

func TestStartOrResumePicking_sameHolderKeepsLockedSince(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1"}}
	user := "user-1"
	lockedAt := time.Now()
	since := lockedAt.Add(-20 * time.Minute)
	sessionRepo := &stubPickingSessionRepo{activeResult: &omsDomain.PickingSession{
		ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress,
		LockedBy: &user, LockedAt: &lockedAt, LockedSince: &since,
	}}
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, sessionRepo, &stubPackingSessionRepo{})

	session, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "user-1"})

	require.NoError(t, err)
	assert.Equal(t, since, *session.LockedSince)
}

func TestStartOrResumePicking_newSessionLockedSinceNow(t *testing.T) {
	db := newMockGormDB(t)
	uc := appoms.NewStartOrResumePickingUseCase(db, &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1"}}, &stubPickingSessionRepo{}, &stubPackingSessionRepo{})

	session, err := uc.Execute(context.Background(), "order-1", omsDomain.LockActor{UserID: "user-1"})

	require.NoError(t, err)
	require.NotNil(t, session.LockedSince)
	assert.Equal(t, *session.LockedAt, *session.LockedSince)
}
