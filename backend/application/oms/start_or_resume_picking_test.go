package oms_test

import (
	"context"
	"testing"

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

	session, err := uc.Execute(context.Background(), "order-1")

	require.NoError(t, err)
	require.Len(t, sessionRepo.created, 1)
	assert.Equal(t, "order-1", session.SalesOrderID)
	assert.Equal(t, omsDomain.PickingSessionStatusInProgress, session.Status)
}

func TestStartOrResumePicking_reusesInProgressSession(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1"}}
	sessionRepo := &stubPickingSessionRepo{activeResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, sessionRepo)

	session, err := uc.Execute(context.Background(), "order-1")

	require.NoError(t, err)
	assert.Empty(t, sessionRepo.created, "must not create a duplicate session")
	assert.Equal(t, "session-1", session.ID)
}

func TestStartOrResumePicking_resumesPausedSession(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1"}}
	sessionRepo := &stubPickingSessionRepo{activeResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusPaused}}
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, sessionRepo)

	session, err := uc.Execute(context.Background(), "order-1")

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

	_, err := uc.Execute(context.Background(), "order-1")

	assert.ErrorIs(t, err, omsDomain.ErrOrderAlreadyFulfilled)
	assert.Empty(t, sessionRepo.created)
}

func TestStartOrResumePicking_propagatesSalesOrderNotFound(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDErr: omsDomain.ErrSalesOrderNotFound}
	uc := appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, &stubPickingSessionRepo{})

	_, err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, omsDomain.ErrSalesOrderNotFound)
}
