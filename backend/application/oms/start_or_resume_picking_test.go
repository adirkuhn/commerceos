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
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, sessionRepo)

	session, err := uc.Execute(context.Background(), "order-1", "user-1")

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
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, sessionRepo)

	session, err := uc.Execute(context.Background(), "order-1", "user-1")

	require.NoError(t, err)
	assert.Empty(t, sessionRepo.created, "must not create a duplicate session")
	assert.Equal(t, "session-1", session.ID)
	assert.Equal(t, []string{"session-1"}, sessionRepo.acquiredLocks)
}

func TestStartOrResumePicking_resumesPausedSession(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1"}}
	sessionRepo := &stubPickingSessionRepo{activeResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusPaused}}
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, sessionRepo)

	session, err := uc.Execute(context.Background(), "order-1", "user-1")

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
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, sessionRepo)

	_, err := uc.Execute(context.Background(), "order-1", "user-1")

	assert.ErrorIs(t, err, omsDomain.ErrOrderAlreadyFulfilled)
	assert.Empty(t, sessionRepo.created)
}

func TestStartOrResumePicking_propagatesSalesOrderNotFound(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDErr: omsDomain.ErrSalesOrderNotFound}
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, &stubPickingSessionRepo{})

	_, err := uc.Execute(context.Background(), "missing", "user-1")

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
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, sessionRepo)

	_, err := uc.Execute(context.Background(), "order-1", "user-1")

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
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, sessionRepo)

	session, err := uc.Execute(context.Background(), "order-1", "user-1")

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
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, sessionRepo)

	session, err := uc.Execute(context.Background(), "order-1", "user-1")

	require.NoError(t, err)
	assert.Equal(t, "session-1", session.ID)
}
