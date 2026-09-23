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

func TestStartOrResumePacking_createsNewSessionWhenNoneActive(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", Fulfilled: true}}
	sessionRepo := &stubPackingSessionRepo{}
	uc := appoms.NewStartOrResumePackingUseCase(db, salesOrderRepo, sessionRepo)

	session, err := uc.Execute(context.Background(), "order-1", "user-1")

	require.NoError(t, err)
	require.Len(t, sessionRepo.created, 1)
	assert.Equal(t, "order-1", session.SalesOrderID)
	assert.Equal(t, omsDomain.PackingSessionStatusInProgress, session.Status)
	require.NotNil(t, session.LockedBy)
	assert.Equal(t, "user-1", *session.LockedBy)
}

func TestStartOrResumePacking_reusesInProgressSession(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", Fulfilled: true}}
	sessionRepo := &stubPackingSessionRepo{activeResult: &omsDomain.PackingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusInProgress}}
	uc := appoms.NewStartOrResumePackingUseCase(db, salesOrderRepo, sessionRepo)

	session, err := uc.Execute(context.Background(), "order-1", "user-1")

	require.NoError(t, err)
	assert.Empty(t, sessionRepo.created, "must not create a duplicate session")
	assert.Equal(t, "session-1", session.ID)
	assert.Equal(t, []string{"session-1"}, sessionRepo.acquiredLocks)
}

func TestStartOrResumePacking_rejectsOrderNotYetPicked(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", Fulfilled: false}}
	sessionRepo := &stubPackingSessionRepo{}
	uc := appoms.NewStartOrResumePackingUseCase(db, salesOrderRepo, sessionRepo)

	_, err := uc.Execute(context.Background(), "order-1", "user-1")

	assert.ErrorIs(t, err, omsDomain.ErrOrderNotYetPicked)
	assert.Empty(t, sessionRepo.created)
}

func TestStartOrResumePacking_rejectsAlreadyPackedOrder(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", Fulfilled: true, Packed: true}}
	sessionRepo := &stubPackingSessionRepo{}
	uc := appoms.NewStartOrResumePackingUseCase(db, salesOrderRepo, sessionRepo)

	_, err := uc.Execute(context.Background(), "order-1", "user-1")

	assert.ErrorIs(t, err, omsDomain.ErrOrderAlreadyPacked)
	assert.Empty(t, sessionRepo.created)
}

func TestStartOrResumePacking_rejectsWhenLockedByAnotherFreshUser(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", Fulfilled: true}}
	otherUser := "user-2"
	lockedAt := time.Now()
	sessionRepo := &stubPackingSessionRepo{activeResult: &omsDomain.PackingSession{
		ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusInProgress,
		LockedBy: &otherUser, LockedAt: &lockedAt,
	}}
	uc := appoms.NewStartOrResumePackingUseCase(db, salesOrderRepo, sessionRepo)

	_, err := uc.Execute(context.Background(), "order-1", "user-1")

	assert.ErrorIs(t, err, omsDomain.ErrPackingSessionLocked)
	assert.Empty(t, sessionRepo.acquiredLocks)
}

func TestStartOrResumePacking_allowsTakeoverWhenLockStale(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", Fulfilled: true}}
	otherUser := "user-2"
	staleAt := time.Now().Add(-10 * time.Minute)
	sessionRepo := &stubPackingSessionRepo{activeResult: &omsDomain.PackingSession{
		ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusInProgress,
		LockedBy: &otherUser, LockedAt: &staleAt,
	}}
	uc := appoms.NewStartOrResumePackingUseCase(db, salesOrderRepo, sessionRepo)

	session, err := uc.Execute(context.Background(), "order-1", "user-1")

	require.NoError(t, err)
	assert.Equal(t, []string{"session-1"}, sessionRepo.acquiredLocks)
	assert.Equal(t, "session-1", session.ID)
}
