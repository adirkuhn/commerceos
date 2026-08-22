package oms_test

import (
	"context"
	"testing"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPickingSession_assemblesSessionLineItemsAndScans(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SalesOrderID: "order-1"}}}
	scanRepo := &stubPickScanRepo{listBySession: []omsDomain.PickScan{{ID: "scan-1", SessionID: "session-1"}}}
	uc := appoms.NewGetPickingSessionUseCase(db, sessionRepo, lineItemRepo, scanRepo, &stubSKURepo{}, &stubVariantRepoForPicking{})

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
	uc := appoms.NewGetPickingSessionUseCase(db, sessionRepo, &stubSalesOrderLineItemRepo{}, &stubPickScanRepo{}, &stubSKURepo{}, &stubVariantRepoForPicking{})

	_, err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, omsDomain.ErrPickingSessionNotFound)
}

func TestGetPickingSession_attachesProductIDForLinkedLineItems(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1"}}
	skuID := "sku-1"
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{
		{ID: "li-1", SalesOrderID: "order-1", SKUID: &skuID},
		{ID: "li-2", SalesOrderID: "order-1", SKUID: nil}, // unmatched Woo-import line -- no SKU to resolve
	}}
	skuRepo := &stubSKURepo{byID: map[string]*pmsDomain.SKU{"sku-1": {ID: "sku-1", VariantID: "variant-1"}}}
	variantRepo := &stubVariantRepoForPicking{byID: map[string]*pmsDomain.Variant{"variant-1": {ID: "variant-1", ProductID: "product-1"}}}
	uc := appoms.NewGetPickingSessionUseCase(db, sessionRepo, lineItemRepo, &stubPickScanRepo{}, skuRepo, variantRepo)

	detail, err := uc.Execute(context.Background(), "session-1")

	require.NoError(t, err)
	require.Len(t, detail.LineItems, 2)
	require.NotNil(t, detail.LineItems[0].ProductID)
	assert.Equal(t, "product-1", *detail.LineItems[0].ProductID)
	assert.Nil(t, detail.LineItems[1].ProductID)
}

func TestGetPickingSession_deletedSKULeavesProductIDNilInsteadOfFailing(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1"}}
	skuID := "sku-deleted"
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SalesOrderID: "order-1", SKUID: &skuID}}}
	skuRepo := &stubSKURepo{findErr: pmsDomain.ErrSKUNotFound}
	uc := appoms.NewGetPickingSessionUseCase(db, sessionRepo, lineItemRepo, &stubPickScanRepo{}, skuRepo, &stubVariantRepoForPicking{})

	detail, err := uc.Execute(context.Background(), "session-1")

	require.NoError(t, err)
	require.Len(t, detail.LineItems, 1)
	assert.Nil(t, detail.LineItems[0].ProductID)
}
