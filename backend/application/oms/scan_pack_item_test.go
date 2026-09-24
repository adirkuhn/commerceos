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
	uc := appoms.NewScanPackItemUseCase(db, sessionRepo, scanRepo, lineItemRepo, skuRepo, &stubBarcodeAliasRepo{}, &stubScaleCodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	result, err := uc.Execute(context.Background(), appoms.ScanPackItemInput{SessionID: "session-1", Code: "BARCODE-1", Quantity: 1, ScannedBy: "user-1"})

	require.NoError(t, err)
	require.NotNil(t, result.Scan.LineItemID)
	assert.Equal(t, "li-1", *result.Scan.LineItemID)
	assert.Equal(t, omsDomain.PackScanActionPack, result.Scan.ActionType)
	assert.Equal(t, "sku-1", result.Scan.SKUID)
	require.Len(t, scanRepo.created, 1)
}

func TestScanPackItem_rejectsSKUNotOnOrder(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{}
	skuRepo := &stubSKURepo{}
	uc := appoms.NewScanPackItemUseCase(db, sessionRepo, &stubPackScanRepo{}, lineItemRepo, skuRepo, &stubBarcodeAliasRepo{}, &stubScaleCodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	_, err := uc.Execute(context.Background(), appoms.ScanPackItemInput{SessionID: "session-1", SKUID: "sku-unknown", Quantity: 1})

	assert.ErrorIs(t, err, omsDomain.ErrPackScanSKUMismatch)
}

func TestScanPackItem_rejectsQuantityExceedingOrdered(t *testing.T) {
	db := newMockGormDB(t)
	skuID := "sku-1"
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusInProgress}}
	scanRepo := &stubPackScanRepo{listBySession: []omsDomain.PackScan{{LineItemID: strPtrLI("li-1"), SKUID: skuID, ActionType: omsDomain.PackScanActionPack, Quantity: 2}}}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SKUID: &skuID, Quantity: 2}}}
	uc := appoms.NewScanPackItemUseCase(db, sessionRepo, scanRepo, lineItemRepo, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubScaleCodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	_, err := uc.Execute(context.Background(), appoms.ScanPackItemInput{SessionID: "session-1", SKUID: skuID, Quantity: 1})

	assert.ErrorIs(t, err, omsDomain.ErrPackQuantityExceedsOrdered)
	assert.Empty(t, scanRepo.created)
}

func TestScanPackItem_rejectsInactiveSession(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", Status: omsDomain.PackingSessionStatusCompleted}}
	uc := appoms.NewScanPackItemUseCase(db, sessionRepo, &stubPackScanRepo{}, &stubSalesOrderLineItemRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubScaleCodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	_, err := uc.Execute(context.Background(), appoms.ScanPackItemInput{SessionID: "session-1", SKUID: "sku-1", Quantity: 1})

	assert.ErrorIs(t, err, omsDomain.ErrPackingSessionNotActive)
}

func TestScanPackItem_rejectsMissingCodeAndSKU(t *testing.T) {
	db := newMockGormDB(t)
	uc := appoms.NewScanPackItemUseCase(db, &stubPackingSessionRepo{}, &stubPackScanRepo{}, &stubSalesOrderLineItemRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubScaleCodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	_, err := uc.Execute(context.Background(), appoms.ScanPackItemInput{SessionID: "session-1", Quantity: 1})

	assert.ErrorIs(t, err, omsDomain.ErrPackScanCodeOrSKURequired)
}

func TestScanPackItem_rejectsNonPositiveQuantity(t *testing.T) {
	db := newMockGormDB(t)
	uc := appoms.NewScanPackItemUseCase(db, &stubPackingSessionRepo{}, &stubPackScanRepo{}, &stubSalesOrderLineItemRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubScaleCodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	_, err := uc.Execute(context.Background(), appoms.ScanPackItemInput{SessionID: "session-1", SKUID: "sku-1", Quantity: 0})

	assert.ErrorIs(t, err, invDomain.ErrInvalidQuantity)
}

func TestScanPackItem_freebieNeedNotBeOnTheOrder(t *testing.T) {
	db := newMockGormDB(t)
	skuID := "sku-1"
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SKUID: &skuID, Quantity: 1}}}
	scanRepo := &stubPackScanRepo{}
	uc := appoms.NewScanPackItemUseCase(db, sessionRepo, scanRepo, lineItemRepo, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubScaleCodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	result, err := uc.Execute(context.Background(), appoms.ScanPackItemInput{
		SessionID: "session-1", SKUID: "sku-extra", Quantity: 2, ActionType: omsDomain.PackScanActionFreebie, Reason: "Sample", ScannedBy: "user-1",
	})

	require.NoError(t, err)
	assert.Nil(t, result.Scan.LineItemID)
	assert.Equal(t, omsDomain.PackScanActionFreebie, result.Scan.ActionType)
	assert.Equal(t, "sku-extra", result.Scan.SKUID)
	assert.Equal(t, "Sample", result.Scan.Reason)
	require.Len(t, scanRepo.created, 1)
}

func TestScanPackItem_newItemRequiresPrice(t *testing.T) {
	db := newMockGormDB(t)
	uc := appoms.NewScanPackItemUseCase(db, &stubPackingSessionRepo{}, &stubPackScanRepo{}, &stubSalesOrderLineItemRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubScaleCodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	_, err := uc.Execute(context.Background(), appoms.ScanPackItemInput{SessionID: "s", SKUID: "sku", Quantity: 1, ActionType: omsDomain.PackScanActionNewItem})

	assert.ErrorIs(t, err, omsDomain.ErrPriceRequiredForNewItem)
}

func TestScanPackItem_newItemRecordsPrice(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusInProgress}}
	scanRepo := &stubPackScanRepo{}
	uc := appoms.NewScanPackItemUseCase(db, sessionRepo, scanRepo, &stubSalesOrderLineItemRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubScaleCodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})
	price := int64(399)

	result, err := uc.Execute(context.Background(), appoms.ScanPackItemInput{
		SessionID: "session-1", SKUID: "sku-extra", Quantity: 1, ActionType: omsDomain.PackScanActionNewItem, PriceMinorUnits: &price, Reason: "Extra jar",
	})

	require.NoError(t, err)
	require.NotNil(t, result.Scan.PriceMinorUnits)
	assert.Equal(t, int64(399), *result.Scan.PriceMinorUnits)
}

func TestScanPackItem_rejectsSubstituteOrRefundThroughScan(t *testing.T) {
	db := newMockGormDB(t)
	uc := appoms.NewScanPackItemUseCase(db, &stubPackingSessionRepo{}, &stubPackScanRepo{}, &stubSalesOrderLineItemRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubScaleCodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	for _, a := range []omsDomain.PackScanActionType{omsDomain.PackScanActionSubstitute, omsDomain.PackScanActionRefund, "bogus"} {
		_, err := uc.Execute(context.Background(), appoms.ScanPackItemInput{SessionID: "s", SKUID: "sku", Quantity: 1, ActionType: a})
		assert.ErrorIs(t, err, omsDomain.ErrInvalidPackScanActionType, a)
	}
}

func TestScanPackItem_quantityCheckCountsSubstitutedUnits(t *testing.T) {
	db := newMockGormDB(t)
	skuID := "sku-1"
	covered := int64(1)
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SKUID: &skuID, Quantity: 2}}}
	scanRepo := &stubPackScanRepo{listBySession: []omsDomain.PackScan{
		{LineItemID: strPtrLI("li-1"), SKUID: "sku-sub", ActionType: omsDomain.PackScanActionSubstitute, Quantity: 4, OriginalQtyCovered: &covered},
	}}
	uc := appoms.NewScanPackItemUseCase(db, sessionRepo, scanRepo, lineItemRepo, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubScaleCodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	_, err := uc.Execute(context.Background(), appoms.ScanPackItemInput{SessionID: "session-1", SKUID: skuID, Quantity: 2})

	assert.ErrorIs(t, err, omsDomain.ErrPackQuantityExceedsOrdered)
}

// Scale labels -- same resolution as picking and receiving (see
// ResolvePickScanCodeUseCase): a scale code or additional scale code matches
// the SKU, one label packs one of the line.

func newScanPackForLine(t *testing.T, skuRepo *stubSKURepo, scaleAliasRepo *stubScaleCodeAliasRepo) (*appoms.ScanPackItemUseCase, *stubPackScanRepo) {
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SalesOrderID: "order-1", SKUID: strPtr("sku-1"), Quantity: 2}}}
	scanRepo := &stubPackScanRepo{}
	uc := appoms.NewScanPackItemUseCase(newMockGormDB(t), sessionRepo, scanRepo, lineItemRepo, skuRepo, &stubBarcodeAliasRepo{}, scaleAliasRepo, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})
	return uc, scanRepo
}

func TestScanPackItem_resolvesAScaleLabelViaTheScaleCode(t *testing.T) {
	uc, scans := newScanPackForLine(t, &stubSKURepo{byScaleCode: map[string]*pmsDomain.SKU{"2994721": {ID: "sku-1", PricingMode: pmsDomain.SKUPricingModePerWeight}}}, &stubScaleCodeAliasRepo{})

	// Real dry-meat label: scale code 94721, weight 462g.
	result, err := uc.Execute(context.Background(), appoms.ScanPackItemInput{SessionID: "session-1", Code: "2994721004622", Quantity: 1})

	require.NoError(t, err)
	assert.Equal(t, "sku-1", result.Scan.SKUID)
	assert.Equal(t, "li-1", *result.Scan.LineItemID)
	assert.Equal(t, int64(1), scans.created[0].Quantity, "one label packs one of the line")
}

func TestScanPackItem_resolvesAScaleLabelForAFixedPriceSKU(t *testing.T) {
	uc, _ := newScanPackForLine(t, &stubSKURepo{byScaleCode: map[string]*pmsDomain.SKU{"2994721": {ID: "sku-1", PricingMode: pmsDomain.SKUPricingModeFixed}}}, &stubScaleCodeAliasRepo{})

	result, err := uc.Execute(context.Background(), appoms.ScanPackItemInput{SessionID: "session-1", Code: "2994721004622", Quantity: 1})

	require.NoError(t, err)
	assert.Equal(t, "sku-1", result.Scan.SKUID)
}

func TestScanPackItem_resolvesAScaleLabelViaAnAdditionalScaleCode(t *testing.T) {
	uc, _ := newScanPackForLine(t, &stubSKURepo{}, &stubScaleCodeAliasRepo{byCode: map[string]*pmsDomain.ScaleCodeAlias{"2994722": {SKUID: "sku-1"}}})

	// Same dry meat weighed on the tenant's second scale (code 94722).
	result, err := uc.Execute(context.Background(), appoms.ScanPackItemInput{SessionID: "session-1", Code: "2994722004621", Quantity: 1})

	require.NoError(t, err)
	assert.Equal(t, "sku-1", result.Scan.SKUID)
}

func TestScanPackItem_rejectsACorruptedScaleLabel(t *testing.T) {
	uc, scans := newScanPackForLine(t, &stubSKURepo{}, &stubScaleCodeAliasRepo{})

	_, err := uc.Execute(context.Background(), appoms.ScanPackItemInput{SessionID: "session-1", Code: "2900633003339", Quantity: 1})

	assert.ErrorIs(t, err, pmsDomain.ErrInvalidCheckDigit)
	assert.Empty(t, scans.created)
}

func TestScanPackItem_rejectsTypingAPerWeightSKUCode(t *testing.T) {
	uc, _ := newScanPackForLine(t, &stubSKURepo{byCode: map[string]*pmsDomain.SKU{"DRYMEAT-1": {ID: "sku-1", PricingMode: pmsDomain.SKUPricingModePerWeight}}}, &stubScaleCodeAliasRepo{})

	_, err := uc.Execute(context.Background(), appoms.ScanPackItemInput{SessionID: "session-1", Code: "DRYMEAT-1", Quantity: 1})

	assert.ErrorIs(t, err, pmsDomain.ErrWeightRequiredForPerWeightSKU)
}
