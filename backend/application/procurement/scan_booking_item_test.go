package procurement_test

import (
	"context"
	"testing"

	appinventory "github.com/adirkuhn/commerceos/application/inventory"
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

func TestScanBookingItem_resolvesUnitBarcodeAndBooksStock(t *testing.T) {
	db := newMockGormDBN(t, 2)
	bookingRepo, poRepo := activeBookingFixtures()
	scanRepo := &stubBookingScanRepo{}
	skuRepo := &stubSKURepo{byBarcode: map[string]*pmsDomain.SKU{"BARCODE-1": {ID: "sku-1"}}}
	aliasRepo := &stubBarcodeAliasRepo{}
	receiveStock := &stubReceiveStockForBooking{}
	uc := appprocurement.NewScanBookingItemUseCase(db, bookingRepo, poRepo, scanRepo, skuRepo, aliasRepo, receiveStock)

	result, err := uc.Execute(context.Background(), appprocurement.ScanBookingItemInput{
		BookingID: "booking-1", Code: "BARCODE-1", Quantity: 3, ScannedBy: "user-1",
	})

	require.NoError(t, err)
	assert.Equal(t, "sku-1", receiveStock.gotInput.SKUID)
	assert.Equal(t, "warehouse-1", receiveStock.gotInput.WarehouseID)
	assert.Equal(t, int64(3), receiveStock.gotInput.Quantity)
	require.Len(t, scanRepo.created, 1)
	assert.Equal(t, int64(3), scanRepo.created[0].Quantity)
	assert.Equal(t, "sku-1", result.Scan.SKUID)
}

func TestScanBookingItem_resolvesCaseBarcodeAliasWithMultiplier(t *testing.T) {
	db := newMockGormDBN(t, 2)
	bookingRepo, poRepo := activeBookingFixtures()
	scanRepo := &stubBookingScanRepo{}
	skuRepo := &stubSKURepo{}
	aliasRepo := &stubBarcodeAliasRepo{byCode: map[string]*pmsDomain.BarcodeAlias{"CASE-1": {SKUID: "sku-1", UnitsPerScan: 12}}}
	receiveStock := &stubReceiveStockForBooking{}
	uc := appprocurement.NewScanBookingItemUseCase(db, bookingRepo, poRepo, scanRepo, skuRepo, aliasRepo, receiveStock)

	_, err := uc.Execute(context.Background(), appprocurement.ScanBookingItemInput{
		BookingID: "booking-1", Code: "CASE-1", Quantity: 2, ScannedBy: "user-1",
	})

	require.NoError(t, err)
	assert.Equal(t, "sku-1", receiveStock.gotInput.SKUID)
	assert.Equal(t, int64(24), receiveStock.gotInput.Quantity, "2 cases x 12 units per case")
}

func TestScanBookingItem_rejectsNonPositiveQuantity(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo, poRepo := activeBookingFixtures()
	uc := appprocurement.NewScanBookingItemUseCase(db, bookingRepo, poRepo, &stubBookingScanRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubReceiveStockForBooking{})

	_, err := uc.Execute(context.Background(), appprocurement.ScanBookingItemInput{BookingID: "booking-1", Code: "X", Quantity: 0})

	assert.ErrorIs(t, err, invDomain.ErrInvalidQuantity)
}

func TestScanBookingItem_rejectsWhenBookingNotActive(t *testing.T) {
	db := newMockGormDBN(t, 1)
	bookingRepo := &stubBookingRepo{findResult: &procurementDomain.Booking{ID: "booking-1", Status: procurementDomain.BookingStatusPaused}}
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1"}}
	uc := appprocurement.NewScanBookingItemUseCase(db, bookingRepo, poRepo, &stubBookingScanRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubReceiveStockForBooking{})

	_, err := uc.Execute(context.Background(), appprocurement.ScanBookingItemInput{BookingID: "booking-1", Code: "X", Quantity: 1})

	assert.ErrorIs(t, err, procurementDomain.ErrBookingNotActive)
}

func TestScanBookingItem_returnsSKUNotFoundWhenNeitherBarcodeNorAliasMatch(t *testing.T) {
	db := newMockGormDBN(t, 1)
	bookingRepo, poRepo := activeBookingFixtures()
	uc := appprocurement.NewScanBookingItemUseCase(db, bookingRepo, poRepo, &stubBookingScanRepo{}, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubReceiveStockForBooking{})

	_, err := uc.Execute(context.Background(), appprocurement.ScanBookingItemInput{BookingID: "booking-1", Code: "UNKNOWN", Quantity: 1})

	assert.ErrorIs(t, err, pmsDomain.ErrSKUNotFound)
}

func TestScanBookingItem_propagatesReceiveStockError(t *testing.T) {
	db := newMockGormDBN(t, 1)
	bookingRepo, poRepo := activeBookingFixtures()
	skuRepo := &stubSKURepo{byBarcode: map[string]*pmsDomain.SKU{"BARCODE-1": {ID: "sku-1"}}}
	receiveStock := &stubReceiveStockForBooking{err: invDomain.ErrInvalidQuantity}
	uc := appprocurement.NewScanBookingItemUseCase(db, bookingRepo, poRepo, &stubBookingScanRepo{}, skuRepo, &stubBarcodeAliasRepo{}, receiveStock)

	_, err := uc.Execute(context.Background(), appprocurement.ScanBookingItemInput{BookingID: "booking-1", Code: "BARCODE-1", Quantity: 1})

	require.Error(t, err)
}

func TestScanBookingItem_propagatesScanCreateError(t *testing.T) {
	db := newMockGormDBN(t, 2)
	bookingRepo, poRepo := activeBookingFixtures()
	skuRepo := &stubSKURepo{byBarcode: map[string]*pmsDomain.SKU{"BARCODE-1": {ID: "sku-1"}}}
	scanRepo := &stubBookingScanRepo{createErr: assert.AnError}
	uc := appprocurement.NewScanBookingItemUseCase(db, bookingRepo, poRepo, scanRepo, skuRepo, &stubBarcodeAliasRepo{}, &stubReceiveStockForBooking{})

	_, err := uc.Execute(context.Background(), appprocurement.ScanBookingItemInput{BookingID: "booking-1", Code: "BARCODE-1", Quantity: 1})

	require.Error(t, err)
}

func TestScanBookingItem_usesReceiptBatchIDWhenNewBatchCreated(t *testing.T) {
	db := newMockGormDBN(t, 2)
	bookingRepo, poRepo := activeBookingFixtures()
	scanRepo := &stubBookingScanRepo{}
	skuRepo := &stubSKURepo{byBarcode: map[string]*pmsDomain.SKU{"BARCODE-1": {ID: "sku-1"}}}
	receiveStock := &stubReceiveStockForBooking{out: &appinventory.ReceiveStockResult{
		Batch:    &invDomain.Batch{ID: "new-batch-1"},
		Movement: &invDomain.StockMovement{ID: "movement-1"},
	}}
	uc := appprocurement.NewScanBookingItemUseCase(db, bookingRepo, poRepo, scanRepo, skuRepo, &stubBarcodeAliasRepo{}, receiveStock)

	_, err := uc.Execute(context.Background(), appprocurement.ScanBookingItemInput{BookingID: "booking-1", Code: "BARCODE-1", Quantity: 1})

	require.NoError(t, err)
	require.NotNil(t, scanRepo.created[0].BatchID)
	assert.Equal(t, "new-batch-1", *scanRepo.created[0].BatchID)
}
