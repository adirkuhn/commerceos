package oms_test

import (
	"context"
	"testing"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPackingSession_returnsSessionLineItemsAndScans(t *testing.T) {
	db := newMockGormDB(t)
	skuID := "sku-1"
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", SalesOrderID: "order-1"}}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SKUID: &skuID, Quantity: 2}}}
	scanRepo := &stubPackScanRepo{listBySession: []omsDomain.PackScan{{ID: "scan-1", LineItemID: strPtrLI("li-1"), SKUID: skuID, Quantity: 1}}}
	uc := appoms.NewGetPackingSessionUseCase(db, sessionRepo, lineItemRepo, scanRepo, &stubSKURepo{}, &stubVariantRepoForPicking{})

	detail, err := uc.Execute(context.Background(), "session-1")

	require.NoError(t, err)
	assert.Equal(t, "session-1", detail.Session.ID)
	require.Len(t, detail.LineItems, 1)
	require.Len(t, detail.Scans, 1)
	assert.Equal(t, "scan-1", detail.Scans[0].ID)
}

func TestGetPackingSession_propagatesNotFound(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPackingSessionRepo{findByIDErr: omsDomain.ErrPackingSessionNotFound}
	uc := appoms.NewGetPackingSessionUseCase(db, sessionRepo, &stubSalesOrderLineItemRepo{}, &stubPackScanRepo{}, &stubSKURepo{}, &stubVariantRepoForPicking{})

	_, err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, omsDomain.ErrPackingSessionNotFound)
}
