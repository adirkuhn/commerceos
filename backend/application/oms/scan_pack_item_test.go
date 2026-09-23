package oms_test

import (
	"context"
	"testing"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanPackItem_resolvesByBarcodeAndRecordsScan(t *testing.T) {
	db := newMockGormDB(t)
	skuID := "sku-1"
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusInProgress}}
	scanRepo := &stubPackScanRepo{}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SKUID: &skuID, Quantity: 3}}}
	skuRepo := &stubSKURepo{byBarcode: map[string]*pmsDomain.SKU{"BARCODE-1": {ID: "sku-1"}}}
	uc := appoms.NewScanPackItemUseCase(db, sessionRepo, scanRepo, lineItemRepo, skuRepo, &stubBarcodeAliasRepo{})

	result, err := uc.Execute(context.Background(), appoms.ScanPackItemInput{SessionID: "session-1", Code: "BARCODE-1", Quantity: 1, ScannedBy: "user-1"})

	require.NoError(t, err)
	assert.Equal(t, "li-1", result.Scan.LineItemID)
	assert.Equal(t, "sku-1", result.Scan.SKUID)
	require.Len(t, scanRepo.created, 1)
}

func TestScanPackItem_rejectsSKUNotOnOrder(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{}
	skuRepo := &stubSKURepo{}
	uc := appoms.NewScanPackItemUseCase(db, sessionRepo, &stubPackScanRepo{}, lineItemRepo, skuRepo, &stubBarcodeAliasRepo{})

	_, err := uc.Execute(context.Background(), appoms.ScanPackItemInput{SessionID: "session-1", SKUID: "sku-unknown", Quantity: 1})

	assert.ErrorIs(t, err, omsDomain.ErrPackScanSKUMismatch)
}

func TestScanPackItem_rejectsQuantityExceedingOrdered(t *testing.T) {
	db := newMockGormDB(t)
	skuID := "sku-1"
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusInProgress}}
	scanRepo := &stubPackScanRepo{listBySession: []omsDomain.PackScan{{LineItemID: "li-1", SKUID: skuID, Quantity: 2}}}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SKUID: &skuID, Quantity: 2}}}
	uc := appoms.NewScanPackItemUseCase(db, sessionRepo, scanRepo, lineItemRepo, &stubSKURepo{}, &stubBarcodeAliasRepo{})

	_, err := uc.Execute(context.Background(), appoms.ScanPackItemInput{SessionID: "session-1", SKUID: skuID, Quantity: 1})

	assert.ErrorIs(t, err, omsDomain.ErrPackQuantityExceedsOrdered)
	assert.Empty(t, scanRepo.created)
}

func TestScanPackItem_rejectsInactiveSession(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", Status: omsDomain.PackingSessionStatusCompleted}}
	uc := appoms.NewScanPackItemUseCase(db, sessionRepo, &stubPackScanRepo{}, &stubSalesOrderLineItemRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{})

	_, err := uc.Execute(context.Background(), appoms.ScanPackItemInput{SessionID: "session-1", SKUID: "sku-1", Quantity: 1})

	assert.ErrorIs(t, err, omsDomain.ErrPackingSessionNotActive)
}

func TestScanPackItem_rejectsMissingCodeAndSKU(t *testing.T) {
	db := newMockGormDB(t)
	uc := appoms.NewScanPackItemUseCase(db, &stubPackingSessionRepo{}, &stubPackScanRepo{}, &stubSalesOrderLineItemRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{})

	_, err := uc.Execute(context.Background(), appoms.ScanPackItemInput{SessionID: "session-1", Quantity: 1})

	assert.ErrorIs(t, err, omsDomain.ErrPackScanCodeOrSKURequired)
}

func TestScanPackItem_rejectsNonPositiveQuantity(t *testing.T) {
	db := newMockGormDB(t)
	uc := appoms.NewScanPackItemUseCase(db, &stubPackingSessionRepo{}, &stubPackScanRepo{}, &stubSalesOrderLineItemRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{})

	_, err := uc.Execute(context.Background(), appoms.ScanPackItemInput{SessionID: "session-1", SKUID: "sku-1", Quantity: 0})

	assert.ErrorIs(t, err, invDomain.ErrInvalidQuantity)
}
