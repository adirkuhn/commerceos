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

func TestResolvePickScanCode_resolvesByBarcode(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	skuRepo := &stubSKURepo{byBarcode: map[string]*pmsDomain.SKU{"BAR-1": {ID: "sku-1", VariantID: "var-1"}}}
	uc := appoms.NewResolvePickScanCodeUseCase(db, sessionRepo, &stubSalesOrderLineItemRepo{}, skuRepo, &stubBarcodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	result, err := uc.Execute(context.Background(), "session-1", "BAR-1")

	require.NoError(t, err)
	assert.Equal(t, "sku-1", result.SKUID)
	assert.NotEmpty(t, result.ProductName)
}

func TestResolvePickScanCode_resolvesByAlias(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	aliasRepo := &stubBarcodeAliasRepo{byCode: map[string]*pmsDomain.BarcodeAlias{"CASE-1": {SKUID: "sku-1", UnitsPerScan: 6}}}
	uc := appoms.NewResolvePickScanCodeUseCase(db, sessionRepo, &stubSalesOrderLineItemRepo{}, &stubSKURepo{}, aliasRepo, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	result, err := uc.Execute(context.Background(), "session-1", "CASE-1")

	require.NoError(t, err)
	assert.Equal(t, "sku-1", result.SKUID)
}

func TestResolvePickScanCode_resolvesBySKUCode(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	skuRepo := &stubSKURepo{byCode: map[string]*pmsDomain.SKU{"SKU-CODE": {ID: "sku-1"}}}
	uc := appoms.NewResolvePickScanCodeUseCase(db, sessionRepo, &stubSalesOrderLineItemRepo{}, skuRepo, &stubBarcodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	result, err := uc.Execute(context.Background(), "session-1", "SKU-CODE")

	require.NoError(t, err)
	assert.Equal(t, "sku-1", result.SKUID)
}

func TestResolvePickScanCode_resolvesViaOrdersOwnLineItem(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SKUID: strPtr("sku-1"), Barcode: "ORDER-BARCODE"}}}
	uc := appoms.NewResolvePickScanCodeUseCase(db, sessionRepo, lineItemRepo, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	result, err := uc.Execute(context.Background(), "session-1", "ORDER-BARCODE")

	require.NoError(t, err)
	assert.Equal(t, "sku-1", result.SKUID)
}

func TestResolvePickScanCode_notFoundAtEndOfChain(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	uc := appoms.NewResolvePickScanCodeUseCase(db, sessionRepo, &stubSalesOrderLineItemRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	_, err := uc.Execute(context.Background(), "session-1", "NOWHERE")

	assert.ErrorIs(t, err, pmsDomain.ErrSKUNotFound)
}

func TestResolvePickScanCode_rejectsWhenSessionNotActive(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", Status: omsDomain.PickingSessionStatusPaused}}
	uc := appoms.NewResolvePickScanCodeUseCase(db, sessionRepo, &stubSalesOrderLineItemRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	_, err := uc.Execute(context.Background(), "session-1", "ANY")

	assert.ErrorIs(t, err, omsDomain.ErrPickingSessionNotActive)
}

func TestResolvePickScanCode_rejectsEmptyCode(t *testing.T) {
	db := newMockGormDB(t)
	uc := appoms.NewResolvePickScanCodeUseCase(db, &stubPickingSessionRepo{}, &stubSalesOrderLineItemRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	_, err := uc.Execute(context.Background(), "session-1", "   ")

	assert.ErrorIs(t, err, omsDomain.ErrPickScanCodeOrSKURequired)
}
