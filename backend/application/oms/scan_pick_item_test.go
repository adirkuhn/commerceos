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

func TestScanPickItem_picksBySKUIDAndMatchesLineItem(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	scanRepo := &stubPickScanRepo{}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SalesOrderID: "order-1", SKUID: strPtr("sku-1")}}}
	uc := appoms.NewScanPickItemUseCase(db, sessionRepo, scanRepo, lineItemRepo, &stubSalesOrderRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	result, err := uc.Execute(context.Background(), appoms.ScanPickItemInput{
		SessionID: "session-1", SKUID: "sku-1", Quantity: 3, ScannedBy: "user-1",
	})

	require.NoError(t, err)
	require.Len(t, scanRepo.created, 1)
	assert.Equal(t, omsDomain.PickScanActionPick, result.Scan.ActionType)
	require.NotNil(t, result.Scan.LineItemID)
	assert.Equal(t, "li-1", *result.Scan.LineItemID)
	assert.Equal(t, int64(3), result.Scan.Quantity)
}

func TestScanPickItem_pickRejectsSKUNotOnOrder(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SalesOrderID: "order-1", SKUID: strPtr("sku-other")}}}
	uc := appoms.NewScanPickItemUseCase(db, sessionRepo, &stubPickScanRepo{}, lineItemRepo, &stubSalesOrderRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	_, err := uc.Execute(context.Background(), appoms.ScanPickItemInput{SessionID: "session-1", SKUID: "sku-1", Quantity: 1})

	assert.ErrorIs(t, err, omsDomain.ErrLineItemSKUMismatch)
}

func TestScanPickItem_resolvesByBarcodeThenAliasThenCodeThenLineItem(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SalesOrderID: "order-1", SKUID: strPtr("sku-1")}}}
	skuRepo := &stubSKURepo{byBarcode: map[string]*pmsDomain.SKU{"BAR-1": {ID: "sku-1"}}}
	uc := appoms.NewScanPickItemUseCase(db, sessionRepo, &stubPickScanRepo{}, lineItemRepo, &stubSalesOrderRepo{}, skuRepo, &stubBarcodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	result, err := uc.Execute(context.Background(), appoms.ScanPickItemInput{SessionID: "session-1", Code: "BAR-1", Quantity: 1})

	require.NoError(t, err)
	assert.Equal(t, "sku-1", result.Scan.SKUID)
}

func TestScanPickItem_resolvesViaBarcodeAlias(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SalesOrderID: "order-1", SKUID: strPtr("sku-1")}}}
	aliasRepo := &stubBarcodeAliasRepo{byCode: map[string]*pmsDomain.BarcodeAlias{"CASE-1": {SKUID: "sku-1", UnitsPerScan: 12}}}
	uc := appoms.NewScanPickItemUseCase(db, sessionRepo, &stubPickScanRepo{}, lineItemRepo, &stubSalesOrderRepo{}, &stubSKURepo{}, aliasRepo, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	result, err := uc.Execute(context.Background(), appoms.ScanPickItemInput{SessionID: "session-1", Code: "CASE-1", Quantity: 1})

	require.NoError(t, err)
	assert.Equal(t, "sku-1", result.Scan.SKUID)
}

func TestScanPickItem_resolvesViaOrdersOwnLineItemSKUCode(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SalesOrderID: "order-1", SKUID: strPtr("sku-1"), SKUCode: "UNMATCHED-CODE"}}}
	uc := appoms.NewScanPickItemUseCase(db, sessionRepo, &stubPickScanRepo{}, lineItemRepo, &stubSalesOrderRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	result, err := uc.Execute(context.Background(), appoms.ScanPickItemInput{SessionID: "session-1", Code: "UNMATCHED-CODE", Quantity: 1})

	require.NoError(t, err)
	assert.Equal(t, "sku-1", result.Scan.SKUID)
}

// A Woo-import line item that never matched a catalog SKU (SKUID nil) can
// still be picked -- scanning the physical item's real barcode/code
// resolves it to a genuine catalog SKU, and if that SKU's product name
// exactly matches (normalized) this order's own unlinked line's stored
// Name, ScanPickItemUseCase links the two (same effect as
// LinkLineItemSKUUseCase from the Unmatched SKUs page) and picks it in one
// step, instead of forcing staff to that separate page first.
func TestScanPickItem_autoLinksAnUnmatchedLineItemWhenProductNameMatches(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	scanRepo := &stubPickScanRepo{}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{
		{ID: "li-1", SalesOrderID: "order-1", Name: "Carne Seca ~414g", SKUID: nil},
	}}
	salesOrderRepo := &stubSalesOrderRepo{}
	skuRepo := &stubSKURepo{byID: map[string]*pmsDomain.SKU{"sku-1": {ID: "sku-1", VariantID: "variant-1"}}}
	variantRepo := &stubVariantRepoForPicking{byID: map[string]*pmsDomain.Variant{"variant-1": {ID: "variant-1", ProductID: "product-1"}}}
	productRepo := &stubProductRepoForPicking{byID: map[string]*pmsDomain.Product{"product-1": {ID: "product-1", Name: "Carne Seca ~414g"}}}
	uc := appoms.NewScanPickItemUseCase(db, sessionRepo, scanRepo, lineItemRepo, salesOrderRepo, skuRepo, &stubBarcodeAliasRepo{}, variantRepo, productRepo)

	result, err := uc.Execute(context.Background(), appoms.ScanPickItemInput{SessionID: "session-1", SKUID: "sku-1", Quantity: 1})

	require.NoError(t, err)
	require.NotNil(t, result.Scan.LineItemID)
	assert.Equal(t, "li-1", *result.Scan.LineItemID)
	require.Len(t, lineItemRepo.updateCalls, 1, "the line item must actually be linked, not just picked in memory")
	assert.Equal(t, "li-1", lineItemRepo.updateCalls[0].ID)
	assert.Equal(t, "sku-1", lineItemRepo.updateCalls[0].Fields["sku_id"])
	require.Len(t, salesOrderRepo.updateCalls, 1, "needs_sku_review must be recomputed once the last unlinked line resolves")
	assert.Equal(t, false, salesOrderRepo.updateCalls[0].Fields["needs_sku_review"])
}

// Normalization tolerates case/punctuation/whitespace noise (same
// normalizedNameSQL convention the Woo-import fallback uses) -- not a
// fuzzy/substring match.
func TestScanPickItem_autoLinkNormalizesNameComparison(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{
		{ID: "li-1", SalesOrderID: "order-1", Name: "  CARNE, SECA ~414G.  ", SKUID: nil},
	}}
	skuRepo := &stubSKURepo{byID: map[string]*pmsDomain.SKU{"sku-1": {ID: "sku-1", VariantID: "variant-1"}}}
	variantRepo := &stubVariantRepoForPicking{byID: map[string]*pmsDomain.Variant{"variant-1": {ID: "variant-1", ProductID: "product-1"}}}
	productRepo := &stubProductRepoForPicking{byID: map[string]*pmsDomain.Product{"product-1": {ID: "product-1", Name: "Carne Seca ~414g"}}}
	uc := appoms.NewScanPickItemUseCase(db, sessionRepo, &stubPickScanRepo{}, lineItemRepo, &stubSalesOrderRepo{}, skuRepo, &stubBarcodeAliasRepo{}, variantRepo, productRepo)

	result, err := uc.Execute(context.Background(), appoms.ScanPickItemInput{SessionID: "session-1", SKUID: "sku-1", Quantity: 1})

	require.NoError(t, err)
	require.NotNil(t, result.Scan.LineItemID)
	assert.Equal(t, "li-1", *result.Scan.LineItemID)
}

// Two unlinked line items sharing the same product name is ambiguous --
// must not guess which one the scan belongs to.
func TestScanPickItem_autoLinkAmbiguousWhenMultipleUnlinkedLinesMatch(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{
		{ID: "li-1", SalesOrderID: "order-1", Name: "Carne Seca ~414g", SKUID: nil},
		{ID: "li-2", SalesOrderID: "order-1", Name: "Carne Seca ~414g", SKUID: nil},
	}}
	skuRepo := &stubSKURepo{byID: map[string]*pmsDomain.SKU{"sku-1": {ID: "sku-1", VariantID: "variant-1"}}}
	variantRepo := &stubVariantRepoForPicking{byID: map[string]*pmsDomain.Variant{"variant-1": {ID: "variant-1", ProductID: "product-1"}}}
	productRepo := &stubProductRepoForPicking{byID: map[string]*pmsDomain.Product{"product-1": {ID: "product-1", Name: "Carne Seca ~414g"}}}
	uc := appoms.NewScanPickItemUseCase(db, sessionRepo, &stubPickScanRepo{}, lineItemRepo, &stubSalesOrderRepo{}, skuRepo, &stubBarcodeAliasRepo{}, variantRepo, productRepo)

	_, err := uc.Execute(context.Background(), appoms.ScanPickItemInput{SessionID: "session-1", SKUID: "sku-1", Quantity: 1})

	assert.ErrorIs(t, err, omsDomain.ErrLineItemSKUMismatch)
}

// Substitute must go through SubstituteLineItemUseCase's batch endpoint --
// see ErrSubstituteViaBatchEndpointOnly's doc.
func TestScanPickItem_rejectsSubstituteActionType(t *testing.T) {
	db := newMockGormDB(t)
	uc := appoms.NewScanPickItemUseCase(db, &stubPickingSessionRepo{}, &stubPickScanRepo{}, &stubSalesOrderLineItemRepo{}, &stubSalesOrderRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	_, err := uc.Execute(context.Background(), appoms.ScanPickItemInput{SessionID: "session-1", SKUID: "sku-2", ActionType: omsDomain.PickScanActionSubstitute, Quantity: 1})

	assert.ErrorIs(t, err, omsDomain.ErrSubstituteViaBatchEndpointOnly)
}

func TestScanPickItem_freebieForcesZeroPriceRegardlessOfInput(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	uc := appoms.NewScanPickItemUseCase(db, sessionRepo, &stubPickScanRepo{}, &stubSalesOrderLineItemRepo{}, &stubSalesOrderRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	price := int64(5000)
	result, err := uc.Execute(context.Background(), appoms.ScanPickItemInput{
		SessionID: "session-1", SKUID: "sku-1", ActionType: omsDomain.PickScanActionFreebie, Quantity: 1, PriceMinorUnits: &price,
	})

	require.NoError(t, err)
	require.NotNil(t, result.Scan.PriceMinorUnits)
	assert.Equal(t, int64(0), *result.Scan.PriceMinorUnits)
	assert.Nil(t, result.Scan.LineItemID)
}

func TestScanPickItem_newItemRequiresPrice(t *testing.T) {
	db := newMockGormDB(t)
	uc := appoms.NewScanPickItemUseCase(db, &stubPickingSessionRepo{}, &stubPickScanRepo{}, &stubSalesOrderLineItemRepo{}, &stubSalesOrderRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	_, err := uc.Execute(context.Background(), appoms.ScanPickItemInput{SessionID: "session-1", SKUID: "sku-1", ActionType: omsDomain.PickScanActionNewItem, Quantity: 1})

	assert.ErrorIs(t, err, omsDomain.ErrPriceRequiredForNewItem)
}

func TestScanPickItem_newItemUsesGivenPrice(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	uc := appoms.NewScanPickItemUseCase(db, sessionRepo, &stubPickScanRepo{}, &stubSalesOrderLineItemRepo{}, &stubSalesOrderRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	price := int64(1250)
	result, err := uc.Execute(context.Background(), appoms.ScanPickItemInput{
		SessionID: "session-1", SKUID: "sku-1", ActionType: omsDomain.PickScanActionNewItem, Quantity: 2, PriceMinorUnits: &price,
	})

	require.NoError(t, err)
	require.NotNil(t, result.Scan.PriceMinorUnits)
	assert.Equal(t, int64(1250), *result.Scan.PriceMinorUnits)
	assert.Nil(t, result.Scan.LineItemID)
}

func TestScanPickItem_rejectsInvalidActionType(t *testing.T) {
	db := newMockGormDB(t)
	uc := appoms.NewScanPickItemUseCase(db, &stubPickingSessionRepo{}, &stubPickScanRepo{}, &stubSalesOrderLineItemRepo{}, &stubSalesOrderRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	_, err := uc.Execute(context.Background(), appoms.ScanPickItemInput{SessionID: "session-1", SKUID: "sku-1", ActionType: "bogus", Quantity: 1})

	assert.ErrorIs(t, err, omsDomain.ErrInvalidPickScanActionType)
}

func TestScanPickItem_rejectsNonPositiveQuantity(t *testing.T) {
	db := newMockGormDB(t)
	uc := appoms.NewScanPickItemUseCase(db, &stubPickingSessionRepo{}, &stubPickScanRepo{}, &stubSalesOrderLineItemRepo{}, &stubSalesOrderRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	_, err := uc.Execute(context.Background(), appoms.ScanPickItemInput{SessionID: "session-1", SKUID: "sku-1", Quantity: 0})

	assert.ErrorIs(t, err, invDomain.ErrInvalidQuantity)
}

func TestScanPickItem_rejectsMissingCodeAndSKUID(t *testing.T) {
	db := newMockGormDB(t)
	uc := appoms.NewScanPickItemUseCase(db, &stubPickingSessionRepo{}, &stubPickScanRepo{}, &stubSalesOrderLineItemRepo{}, &stubSalesOrderRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	_, err := uc.Execute(context.Background(), appoms.ScanPickItemInput{SessionID: "session-1", Quantity: 1})

	assert.ErrorIs(t, err, omsDomain.ErrPickScanCodeOrSKURequired)
}

func TestScanPickItem_rejectsWhenSessionNotActive(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", Status: omsDomain.PickingSessionStatusPaused}}
	uc := appoms.NewScanPickItemUseCase(db, sessionRepo, &stubPickScanRepo{}, &stubSalesOrderLineItemRepo{}, &stubSalesOrderRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})

	_, err := uc.Execute(context.Background(), appoms.ScanPickItemInput{SessionID: "session-1", SKUID: "sku-1", Quantity: 1})

	assert.ErrorIs(t, err, omsDomain.ErrPickingSessionNotActive)
}
