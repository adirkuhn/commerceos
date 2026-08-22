package oms_test

import (
	"context"
	"testing"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPickingSession_assemblesSessionLineItemsAndScans(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SalesOrderID: "order-1"}}}
	scanRepo := &stubPickScanRepo{listBySession: []omsDomain.PickScan{{ID: "scan-1", SessionID: "session-1"}}}
	uc := appoms.NewGetPickingSessionUseCase(db, sessionRepo, lineItemRepo, scanRepo)

	detail, err := uc.Execute(context.Background(), "session-1")

	require.NoError(t, err)
	assert.Equal(t, "session-1", detail.Session.ID)
	require.Len(t, detail.LineItems, 1)
	assert.Equal(t, "li-1", detail.LineItems[0].ID)
	require.Len(t, detail.Scans, 1)
	assert.Equal(t, "scan-1", detail.Scans[0].ID)
}

func TestGetPickingSession_propagatesNotFound(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDErr: omsDomain.ErrPickingSessionNotFound}
	uc := appoms.NewGetPickingSessionUseCase(db, sessionRepo, &stubSalesOrderLineItemRepo{}, &stubPickScanRepo{})

	_, err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, omsDomain.ErrPickingSessionNotFound)
}
