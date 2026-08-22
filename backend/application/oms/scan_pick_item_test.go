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
	uc := appoms.NewScanPickItemUseCase(db, sessionRepo, scanRepo, lineItemRepo, &stubSKURepo{}, &stubBarcodeAliasRepo{})

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
	uc := appoms.NewScanPickItemUseCase(db, sessionRepo, &stubPickScanRepo{}, lineItemRepo, &stubSKURepo{}, &stubBarcodeAliasRepo{})

	_, err := uc.Execute(context.Background(), appoms.ScanPickItemInput{SessionID: "session-1", SKUID: "sku-1", Quantity: 1})

	assert.ErrorIs(t, err, omsDomain.ErrLineItemSKUMismatch)
}

func TestScanPickItem_resolvesByBarcodeThenAliasThenCodeThenLineItem(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SalesOrderID: "order-1", SKUID: strPtr("sku-1")}}}
	skuRepo := &stubSKURepo{byBarcode: map[string]*pmsDomain.SKU{"BAR-1": {ID: "sku-1"}}}
	uc := appoms.NewScanPickItemUseCase(db, sessionRepo, &stubPickScanRepo{}, lineItemRepo, skuRepo, &stubBarcodeAliasRepo{})

	result, err := uc.Execute(context.Background(), appoms.ScanPickItemInput{SessionID: "session-1", Code: "BAR-1", Quantity: 1})

	require.NoError(t, err)
	assert.Equal(t, "sku-1", result.Scan.SKUID)
}

func TestScanPickItem_resolvesViaBarcodeAlias(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SalesOrderID: "order-1", SKUID: strPtr("sku-1")}}}
	aliasRepo := &stubBarcodeAliasRepo{byCode: map[string]*pmsDomain.BarcodeAlias{"CASE-1": {SKUID: "sku-1", UnitsPerScan: 12}}}
	uc := appoms.NewScanPickItemUseCase(db, sessionRepo, &stubPickScanRepo{}, lineItemRepo, &stubSKURepo{}, aliasRepo)

	result, err := uc.Execute(context.Background(), appoms.ScanPickItemInput{SessionID: "session-1", Code: "CASE-1", Quantity: 1})

	require.NoError(t, err)
	assert.Equal(t, "sku-1", result.Scan.SKUID)
}

func TestScanPickItem_resolvesViaOrdersOwnLineItemSKUCode(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SalesOrderID: "order-1", SKUID: strPtr("sku-1"), SKUCode: "UNMATCHED-CODE"}}}
	uc := appoms.NewScanPickItemUseCase(db, sessionRepo, &stubPickScanRepo{}, lineItemRepo, &stubSKURepo{}, &stubBarcodeAliasRepo{})

	result, err := uc.Execute(context.Background(), appoms.ScanPickItemInput{SessionID: "session-1", Code: "UNMATCHED-CODE", Quantity: 1})

	require.NoError(t, err)
	assert.Equal(t, "sku-1", result.Scan.SKUID)
}

func TestScanPickItem_substituteRequiresLineItemID(t *testing.T) {
	db := newMockGormDB(t)
	uc := appoms.NewScanPickItemUseCase(db, &stubPickingSessionRepo{}, &stubPickScanRepo{}, &stubSalesOrderLineItemRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{})

	_, err := uc.Execute(context.Background(), appoms.ScanPickItemInput{SessionID: "session-1", SKUID: "sku-2", ActionType: omsDomain.PickScanActionSubstitute})

	assert.ErrorIs(t, err, omsDomain.ErrPickLineItemRequired)
}

func TestScanPickItem_substituteUsesOriginalLineItemQuantityNotCallerQuantity(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{findByIDResult: &omsDomain.SalesOrderLineItem{ID: "li-1", SalesOrderID: "order-1", SKUID: strPtr("sku-1"), Quantity: 7}}
	uc := appoms.NewScanPickItemUseCase(db, sessionRepo, &stubPickScanRepo{}, lineItemRepo, &stubSKURepo{}, &stubBarcodeAliasRepo{})

	result, err := uc.Execute(context.Background(), appoms.ScanPickItemInput{
		SessionID: "session-1", SKUID: "sku-substitute", LineItemID: "li-1", ActionType: omsDomain.PickScanActionSubstitute, Quantity: 999,
	})

	require.NoError(t, err)
	assert.Equal(t, int64(7), result.Scan.Quantity, "must use the original line item's own quantity, not the caller's")
	require.NotNil(t, result.Scan.LineItemID)
	assert.Equal(t, "li-1", *result.Scan.LineItemID)
	assert.Equal(t, "sku-substitute", result.Scan.SKUID)
}

func TestScanPickItem_substituteRejectsLineItemFromAnotherOrder(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{findByIDResult: &omsDomain.SalesOrderLineItem{ID: "li-1", SalesOrderID: "order-2", Quantity: 1}}
	uc := appoms.NewScanPickItemUseCase(db, sessionRepo, &stubPickScanRepo{}, lineItemRepo, &stubSKURepo{}, &stubBarcodeAliasRepo{})

	_, err := uc.Execute(context.Background(), appoms.ScanPickItemInput{
		SessionID: "session-1", SKUID: "sku-1", LineItemID: "li-1", ActionType: omsDomain.PickScanActionSubstitute,
	})

	assert.ErrorIs(t, err, omsDomain.ErrLineItemNotOnOrder)
}

func TestScanPickItem_freebieForcesZeroPriceRegardlessOfInput(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	uc := appoms.NewScanPickItemUseCase(db, sessionRepo, &stubPickScanRepo{}, &stubSalesOrderLineItemRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{})

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
	uc := appoms.NewScanPickItemUseCase(db, &stubPickingSessionRepo{}, &stubPickScanRepo{}, &stubSalesOrderLineItemRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{})

	_, err := uc.Execute(context.Background(), appoms.ScanPickItemInput{SessionID: "session-1", SKUID: "sku-1", ActionType: omsDomain.PickScanActionNewItem, Quantity: 1})

	assert.ErrorIs(t, err, omsDomain.ErrPriceRequiredForNewItem)
}

func TestScanPickItem_newItemUsesGivenPrice(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	uc := appoms.NewScanPickItemUseCase(db, sessionRepo, &stubPickScanRepo{}, &stubSalesOrderLineItemRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{})

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
	uc := appoms.NewScanPickItemUseCase(db, &stubPickingSessionRepo{}, &stubPickScanRepo{}, &stubSalesOrderLineItemRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{})

	_, err := uc.Execute(context.Background(), appoms.ScanPickItemInput{SessionID: "session-1", SKUID: "sku-1", ActionType: "bogus", Quantity: 1})

	assert.ErrorIs(t, err, omsDomain.ErrInvalidPickScanActionType)
}

func TestScanPickItem_rejectsNonPositiveQuantity(t *testing.T) {
	db := newMockGormDB(t)
	uc := appoms.NewScanPickItemUseCase(db, &stubPickingSessionRepo{}, &stubPickScanRepo{}, &stubSalesOrderLineItemRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{})

	_, err := uc.Execute(context.Background(), appoms.ScanPickItemInput{SessionID: "session-1", SKUID: "sku-1", Quantity: 0})

	assert.ErrorIs(t, err, invDomain.ErrInvalidQuantity)
}

func TestScanPickItem_rejectsMissingCodeAndSKUID(t *testing.T) {
	db := newMockGormDB(t)
	uc := appoms.NewScanPickItemUseCase(db, &stubPickingSessionRepo{}, &stubPickScanRepo{}, &stubSalesOrderLineItemRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{})

	_, err := uc.Execute(context.Background(), appoms.ScanPickItemInput{SessionID: "session-1", Quantity: 1})

	assert.ErrorIs(t, err, omsDomain.ErrPickScanCodeOrSKURequired)
}

func TestScanPickItem_rejectsWhenSessionNotActive(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", Status: omsDomain.PickingSessionStatusPaused}}
	uc := appoms.NewScanPickItemUseCase(db, sessionRepo, &stubPickScanRepo{}, &stubSalesOrderLineItemRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{})

	_, err := uc.Execute(context.Background(), appoms.ScanPickItemInput{SessionID: "session-1", SKUID: "sku-1", Quantity: 1})

	assert.ErrorIs(t, err, omsDomain.ErrPickingSessionNotActive)
}
