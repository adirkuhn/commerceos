package procurement_test

import (
	"context"
	"testing"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func activeBookingFixtures() (*stubBookingRepo, *stubPurchaseOrderRepo) {
	return &stubBookingRepo{findResult: &procurementDomain.Booking{ID: "booking-1", PurchaseOrderID: "po-1", Status: procurementDomain.BookingStatusInProgress}},
		&stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1", WarehouseID: strPtr("warehouse-1")}}
}

func TestScanBookingItem_resolvesUnitBarcodeAndStagesAReceive(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo, poRepo := activeBookingFixtures()
	scanRepo := &stubBookingScanRepo{}
	skuRepo := &stubSKURepo{byBarcode: map[string]*pmsDomain.SKU{"BARCODE-1": {ID: "sku-1"}}}
	aliasRepo := &stubBarcodeAliasRepo{}
	uc := appprocurement.NewScanBookingItemUseCase(db, bookingRepo, poRepo, scanRepo, skuRepo, aliasRepo, &stubLineItemRepo{})

	result, err := uc.Execute(context.Background(), appprocurement.ScanBookingItemInput{
		BookingID: "booking-1", Code: "BARCODE-1", Quantity: 3, ScannedBy: "user-1",
	})

	require.NoError(t, err)
	require.Len(t, scanRepo.created, 1)
	staged := scanRepo.created[0]
	assert.Equal(t, "sku-1", staged.SKUID)
	assert.Equal(t, procurementDomain.BookingScanActionReceive, staged.ActionType)
	assert.Equal(t, int64(3), staged.Quantity)
	assert.Equal(t, int64(0), staged.WriteOffQuantity)
	assert.Equal(t, "sku-1", result.Scan.SKUID)
	// Nothing real happened yet -- no inventory use case was even injected,
	// so if this use case tried to call one it would nil-panic.
}

func TestScanBookingItem_resolvesCaseBarcodeAliasWithMultiplier(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo, poRepo := activeBookingFixtures()
	scanRepo := &stubBookingScanRepo{}
	skuRepo := &stubSKURepo{}
	aliasRepo := &stubBarcodeAliasRepo{byCode: map[string]*pmsDomain.BarcodeAlias{"CASE-1": {SKUID: "sku-1", UnitsPerScan: 12}}}
	uc := appprocurement.NewScanBookingItemUseCase(db, bookingRepo, poRepo, scanRepo, skuRepo, aliasRepo, &stubLineItemRepo{})

	_, err := uc.Execute(context.Background(), appprocurement.ScanBookingItemInput{
		BookingID: "booking-1", Code: "CASE-1", Quantity: 2, ScannedBy: "user-1",
	})

	require.NoError(t, err)
	require.Len(t, scanRepo.created, 1)
	assert.Equal(t, "sku-1", scanRepo.created[0].SKUID)
	assert.Equal(t, int64(24), scanRepo.created[0].Quantity, "2 cases x 12 units per case")
}

func TestScanBookingItem_booksDirectlyAgainstSKUIDSkippingBarcodeResolution(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo, poRepo := activeBookingFixtures()
	scanRepo := &stubBookingScanRepo{}
	skuRepo := &stubSKURepo{} // no barcode registered -- must not be consulted
	aliasRepo := &stubBarcodeAliasRepo{}
	uc := appprocurement.NewScanBookingItemUseCase(db, bookingRepo, poRepo, scanRepo, skuRepo, aliasRepo, &stubLineItemRepo{})

	// Tapping a line item row on the booking screen: the SKU is already
	// known (from the line item), no barcode was scanned.
	result, err := uc.Execute(context.Background(), appprocurement.ScanBookingItemInput{
		BookingID: "booking-1", SKUID: "sku-direct-1", Quantity: 4, ScannedBy: "user-1",
	})

	require.NoError(t, err)
	assert.Equal(t, "sku-direct-1", scanRepo.created[0].SKUID)
	assert.Equal(t, int64(4), scanRepo.created[0].Quantity)
	assert.Equal(t, "sku-direct-1", result.Scan.SKUID)
}

func TestScanBookingItem_stagesACorrectionWithNegativeNetQuantity(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo, poRepo := activeBookingFixtures()
	scanRepo := &stubBookingScanRepo{}
	uc := appprocurement.NewScanBookingItemUseCase(db, bookingRepo, poRepo, scanRepo, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubLineItemRepo{})

	batchID := "batch-1"
	result, err := uc.Execute(context.Background(), appprocurement.ScanBookingItemInput{
		BookingID: "booking-1", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionCorrection,
		Quantity: 3, BatchID: &batchID, Reason: "counted wrong", ScannedBy: "user-1",
	})

	require.NoError(t, err)
	require.Len(t, scanRepo.created, 1)
	staged := scanRepo.created[0]
	assert.Equal(t, procurementDomain.BookingScanActionCorrection, staged.ActionType)
	assert.Equal(t, int64(-3), staged.Quantity, "correction nets negative against the line's received progress")
	assert.Equal(t, int64(0), staged.WriteOffQuantity)
	assert.Equal(t, &batchID, staged.BatchID)
	assert.Equal(t, "counted wrong", staged.Reason)
	assert.Equal(t, int64(-3), result.Scan.Quantity)
}

func TestScanBookingItem_stagesADamagedWriteOffWithoutAffectingReceivedProgress(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo, poRepo := activeBookingFixtures()
	scanRepo := &stubBookingScanRepo{}
	uc := appprocurement.NewScanBookingItemUseCase(db, bookingRepo, poRepo, scanRepo, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubLineItemRepo{})

	result, err := uc.Execute(context.Background(), appprocurement.ScanBookingItemInput{
		BookingID: "booking-1", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionDamaged,
		Quantity: 2, Reason: "arrived crushed", ScannedBy: "user-1",
	})

	require.NoError(t, err)
	require.Len(t, scanRepo.created, 1)
	staged := scanRepo.created[0]
	assert.Equal(t, procurementDomain.BookingScanActionDamaged, staged.ActionType)
	assert.Equal(t, int64(0), staged.Quantity, "damaged/lost never reduces received progress -- the stock did arrive")
	assert.Equal(t, int64(2), staged.WriteOffQuantity)
	assert.Equal(t, "arrived crushed", staged.Reason)
	assert.Equal(t, int64(0), result.Scan.Quantity)
}

func TestScanBookingItem_stagesALostWriteOff(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo, poRepo := activeBookingFixtures()
	scanRepo := &stubBookingScanRepo{}
	uc := appprocurement.NewScanBookingItemUseCase(db, bookingRepo, poRepo, scanRepo, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubLineItemRepo{})

	_, err := uc.Execute(context.Background(), appprocurement.ScanBookingItemInput{
		BookingID: "booking-1", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionLost, Quantity: 1,
	})

	require.NoError(t, err)
	assert.Equal(t, procurementDomain.BookingScanActionLost, scanRepo.created[0].ActionType)
	assert.Equal(t, int64(1), scanRepo.created[0].WriteOffQuantity)
}

func TestScanBookingItem_stagesLotNumberAndExpiryForANewBatch(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo, poRepo := activeBookingFixtures()
	scanRepo := &stubBookingScanRepo{}
	uc := appprocurement.NewScanBookingItemUseCase(db, bookingRepo, poRepo, scanRepo, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubLineItemRepo{})

	_, err := uc.Execute(context.Background(), appprocurement.ScanBookingItemInput{
		BookingID: "booking-1", SKUID: "sku-1", Quantity: 10, LotNumber: "LOT-NEW",
	})

	require.NoError(t, err)
	assert.Equal(t, "LOT-NEW", scanRepo.created[0].LotNumber)
	assert.Nil(t, scanRepo.created[0].BatchID, "no real batch exists yet -- resolved later by CompleteBookingUseCase's replay")
}

func TestScanBookingItem_rejectsInvalidActionType(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo, poRepo := activeBookingFixtures()
	uc := appprocurement.NewScanBookingItemUseCase(db, bookingRepo, poRepo, &stubBookingScanRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubLineItemRepo{})

	_, err := uc.Execute(context.Background(), appprocurement.ScanBookingItemInput{
		BookingID: "booking-1", SKUID: "sku-1", ActionType: "bogus", Quantity: 1,
	})

	assert.ErrorIs(t, err, procurementDomain.ErrInvalidBookingScanActionType)
}

func TestScanBookingItem_rejectsWhenNeitherCodeNorSKUIDProvided(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo, poRepo := activeBookingFixtures()
	uc := appprocurement.NewScanBookingItemUseCase(db, bookingRepo, poRepo, &stubBookingScanRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubLineItemRepo{})

	_, err := uc.Execute(context.Background(), appprocurement.ScanBookingItemInput{BookingID: "booking-1", Quantity: 1})

	assert.ErrorIs(t, err, procurementDomain.ErrScanCodeOrSKURequired)
}

func TestScanBookingItem_rejectsNonPositiveQuantity(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo, poRepo := activeBookingFixtures()
	uc := appprocurement.NewScanBookingItemUseCase(db, bookingRepo, poRepo, &stubBookingScanRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubLineItemRepo{})

	_, err := uc.Execute(context.Background(), appprocurement.ScanBookingItemInput{BookingID: "booking-1", Code: "X", Quantity: 0})

	assert.ErrorIs(t, err, invDomain.ErrInvalidQuantity)
}

func TestScanBookingItem_rejectsWhenBookingNotActive(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo := &stubBookingRepo{findResult: &procurementDomain.Booking{ID: "booking-1", Status: procurementDomain.BookingStatusPaused}}
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1"}}
	uc := appprocurement.NewScanBookingItemUseCase(db, bookingRepo, poRepo, &stubBookingScanRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubLineItemRepo{})

	_, err := uc.Execute(context.Background(), appprocurement.ScanBookingItemInput{BookingID: "booking-1", Code: "X", Quantity: 1})

	assert.ErrorIs(t, err, procurementDomain.ErrBookingNotActive)
}

func TestScanBookingItem_fallsBackToSKUCodeWhenNeitherBarcodeNorAliasMatches(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo, poRepo := activeBookingFixtures()
	scanRepo := &stubBookingScanRepo{}
	// No barcode, no alias -- only a direct SKU.SKUCode match, same
	// fallback pms.ResolveBarcodeUseCase already has.
	skuRepo := &stubSKURepo{byCode: map[string]*pmsDomain.SKU{"TSHIRT-1": {ID: "sku-1"}}}
	uc := appprocurement.NewScanBookingItemUseCase(db, bookingRepo, poRepo, scanRepo, skuRepo, &stubBarcodeAliasRepo{}, &stubLineItemRepo{})

	result, err := uc.Execute(context.Background(), appprocurement.ScanBookingItemInput{
		BookingID: "booking-1", Code: "TSHIRT-1", Quantity: 3, ScannedBy: "user-1",
	})

	require.NoError(t, err)
	assert.Equal(t, "sku-1", result.Scan.SKUID)
	assert.Equal(t, int64(3), result.Scan.Quantity)
}

func TestScanBookingItem_trimsWhitespaceFromTheScannedCode(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo, poRepo := activeBookingFixtures()
	scanRepo := &stubBookingScanRepo{}
	skuRepo := &stubSKURepo{byBarcode: map[string]*pmsDomain.SKU{"BARCODE-1": {ID: "sku-1"}}}
	uc := appprocurement.NewScanBookingItemUseCase(db, bookingRepo, poRepo, scanRepo, skuRepo, &stubBarcodeAliasRepo{}, &stubLineItemRepo{})

	_, err := uc.Execute(context.Background(), appprocurement.ScanBookingItemInput{
		BookingID: "booking-1", Code: " BARCODE-1\n", Quantity: 1, ScannedBy: "user-1",
	})

	require.NoError(t, err)
	require.Len(t, scanRepo.created, 1)
	assert.Equal(t, "sku-1", scanRepo.created[0].SKUID)
}

func TestScanBookingItem_returnsSKUNotFoundWhenNeitherBarcodeNorAliasMatch(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo, poRepo := activeBookingFixtures()
	uc := appprocurement.NewScanBookingItemUseCase(db, bookingRepo, poRepo, &stubBookingScanRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubLineItemRepo{})

	_, err := uc.Execute(context.Background(), appprocurement.ScanBookingItemInput{BookingID: "booking-1", Code: "UNKNOWN", Quantity: 1})

	assert.ErrorIs(t, err, pmsDomain.ErrSKUNotFound)
}

func TestScanBookingItem_fallsBackToThisPOsLineItemRawSKUWhenCatalogWideLookupsAllMiss(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo, poRepo := activeBookingFixtures()
	scanRepo := &stubBookingScanRepo{}
	skuID := "sku-1"
	// Nothing in the tenant-wide catalog matches -- the only thing that
	// does is this PO's own line item, exactly as printed on the supplier
	// document (RawSKU), already linked to a real SKU.
	lineItemRepo := &stubLineItemRepo{byPOMap: map[string][]procurementDomain.PurchaseOrderLineItem{
		"po-1": {{ID: "line-1", PurchaseOrderID: "po-1", SKUID: &skuID, RawSKU: "  7897122604154 "}},
	}}
	uc := appprocurement.NewScanBookingItemUseCase(db, bookingRepo, poRepo, scanRepo, &stubSKURepo{}, &stubBarcodeAliasRepo{}, lineItemRepo)

	result, err := uc.Execute(context.Background(), appprocurement.ScanBookingItemInput{
		BookingID: "booking-1", Code: "7897122604154", Quantity: 1, ScannedBy: "user-1",
	})

	require.NoError(t, err)
	assert.Equal(t, "sku-1", result.Scan.SKUID)
}

func TestScanBookingItem_lineItemFallbackIgnoresAnUnlinkedLineItem(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo, poRepo := activeBookingFixtures()
	// Matches RawSKU, but this line item was never linked to a real
	// product -- nothing to book against, so this must still fail rather
	// than book against a nonexistent SKU.
	lineItemRepo := &stubLineItemRepo{byPOMap: map[string][]procurementDomain.PurchaseOrderLineItem{
		"po-1": {{ID: "line-1", PurchaseOrderID: "po-1", SKUID: nil, RawSKU: "7897122604154"}},
	}}
	uc := appprocurement.NewScanBookingItemUseCase(db, bookingRepo, poRepo, &stubBookingScanRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{}, lineItemRepo)

	_, err := uc.Execute(context.Background(), appprocurement.ScanBookingItemInput{
		BookingID: "booking-1", Code: "7897122604154", Quantity: 1,
	})

	assert.ErrorIs(t, err, pmsDomain.ErrSKUNotFound)
}

func TestScanBookingItem_propagatesScanCreateError(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo, poRepo := activeBookingFixtures()
	skuRepo := &stubSKURepo{byBarcode: map[string]*pmsDomain.SKU{"BARCODE-1": {ID: "sku-1"}}}
	scanRepo := &stubBookingScanRepo{createErr: assert.AnError}
	uc := appprocurement.NewScanBookingItemUseCase(db, bookingRepo, poRepo, scanRepo, skuRepo, &stubBarcodeAliasRepo{}, &stubLineItemRepo{})

	_, err := uc.Execute(context.Background(), appprocurement.ScanBookingItemInput{BookingID: "booking-1", Code: "BARCODE-1", Quantity: 1})

	require.Error(t, err)
}

func TestScanBookingItem_rejectsWarehouseRequired(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo := &stubBookingRepo{findResult: &procurementDomain.Booking{ID: "booking-1", PurchaseOrderID: "po-1", Status: procurementDomain.BookingStatusInProgress}}
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1"}}
	uc := appprocurement.NewScanBookingItemUseCase(db, bookingRepo, poRepo, &stubBookingScanRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubLineItemRepo{})

	_, err := uc.Execute(context.Background(), appprocurement.ScanBookingItemInput{BookingID: "booking-1", SKUID: "sku-1", Quantity: 1})

	assert.ErrorIs(t, err, procurementDomain.ErrWarehouseRequired)
}
