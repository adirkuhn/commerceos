package procurement_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

func TestListBookingDiscrepancies_returnsEmptyWhenNoCompletedBookings(t *testing.T) {
	db := newMockGormDB(t)
	uc := appprocurement.NewListBookingDiscrepanciesUseCase(db, &stubBookingRepo{}, &stubLineItemRepo{}, &stubBookingScanRepo{}, &stubSKURepo{}, &stubVariantRepo{}, &stubProductRepo{})

	report, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.Empty(t, report.MissingItems)
	require.Empty(t, report.DamagedLost)
}

func TestListBookingDiscrepancies_reportsUnderScannedLineItemsEnrichedWithProductDisplayNames(t *testing.T) {
	db := newMockGormDB(t)
	completedAt := time.Now()
	skuID := "sku-1"
	bookingRepo := &stubBookingRepo{completed: []procurementDomain.Booking{
		{ID: "booking-1", PurchaseOrderID: "po-1", Status: procurementDomain.BookingStatusCompleted, CompletedAt: &completedAt},
	}}
	lineItemRepo := &stubLineItemRepo{byPOMap: map[string][]procurementDomain.PurchaseOrderLineItem{
		"po-1": {{ID: "line-1", PurchaseOrderID: "po-1", SKUID: &skuID, RawDescription: "Widget 12pk", Quantity: 10, LineType: procurementDomain.LineItemTypeProduct}},
	}}
	scanRepo := &stubBookingScanRepo{byBookingMap: map[string][]procurementDomain.BookingScan{
		"booking-1": {{ID: "scan-1", BookingID: "booking-1", SKUID: skuID, Quantity: 4, ActionType: procurementDomain.BookingScanActionReceive}},
	}}
	skuRepo := &stubSKURepo{findByIDs: map[string]*pmsDomain.SKU{"sku-1": {ID: "sku-1", VariantID: "variant-1", SKUCode: "WGT-1"}}}
	variantRepo := &stubVariantRepo{byID: map[string]*pmsDomain.Variant{"variant-1": {ID: "variant-1", ProductID: "product-1"}}}
	productRepo := &stubProductRepo{byID: map[string]*pmsDomain.Product{"product-1": {ID: "product-1", Name: "Widget"}}}
	uc := appprocurement.NewListBookingDiscrepanciesUseCase(db, bookingRepo, lineItemRepo, scanRepo, skuRepo, variantRepo, productRepo)

	report, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.Len(t, report.MissingItems, 1)
	item := report.MissingItems[0]
	require.Equal(t, "booking-1", item.BookingID)
	require.Equal(t, "po-1", item.PurchaseOrderID)
	require.Equal(t, "line-1", item.LineItemID)
	require.Equal(t, int64(10), item.ExpectedQty)
	require.Equal(t, int64(4), item.ScannedQty)
	require.Equal(t, int64(6), item.MissingQty)
	require.NotNil(t, item.SKUCode)
	require.Equal(t, "WGT-1", *item.SKUCode)
	require.NotNil(t, item.ProductName)
	require.Equal(t, "Widget", *item.ProductName)
	require.Empty(t, report.DamagedLost)
}

func TestListBookingDiscrepancies_excludesFullyScannedLineItems(t *testing.T) {
	db := newMockGormDB(t)
	skuID := "sku-1"
	bookingRepo := &stubBookingRepo{completed: []procurementDomain.Booking{
		{ID: "booking-1", PurchaseOrderID: "po-1", Status: procurementDomain.BookingStatusCompleted},
	}}
	lineItemRepo := &stubLineItemRepo{byPOMap: map[string][]procurementDomain.PurchaseOrderLineItem{
		"po-1": {{ID: "line-1", PurchaseOrderID: "po-1", SKUID: &skuID, Quantity: 10, LineType: procurementDomain.LineItemTypeProduct}},
	}}
	scanRepo := &stubBookingScanRepo{byBookingMap: map[string][]procurementDomain.BookingScan{
		"booking-1": {{ID: "scan-1", BookingID: "booking-1", SKUID: skuID, Quantity: 10, ActionType: procurementDomain.BookingScanActionReceive}},
	}}
	uc := appprocurement.NewListBookingDiscrepanciesUseCase(db, bookingRepo, lineItemRepo, scanRepo, &stubSKURepo{}, &stubVariantRepo{}, &stubProductRepo{})

	report, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.Empty(t, report.MissingItems)
}

func TestListBookingDiscrepancies_reportsDamagedAndLostScansEnriched(t *testing.T) {
	db := newMockGormDB(t)
	scannedAt := time.Now()
	bookingRepo := &stubBookingRepo{completed: []procurementDomain.Booking{
		{ID: "booking-1", PurchaseOrderID: "po-1", Status: procurementDomain.BookingStatusCompleted},
	}}
	scanRepo := &stubBookingScanRepo{byBookingMap: map[string][]procurementDomain.BookingScan{
		"booking-1": {
			{ID: "scan-1", BookingID: "booking-1", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionDamaged, WriteOffQuantity: 2, Reason: "box crushed", ScannedAt: scannedAt, ScannedBy: "user-1"},
			{ID: "scan-2", BookingID: "booking-1", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionLost, WriteOffQuantity: 1, ScannedAt: scannedAt},
			// A receive row must never show up as a discrepancy.
			{ID: "scan-3", BookingID: "booking-1", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionReceive, Quantity: 5},
		},
	}}
	skuRepo := &stubSKURepo{findByIDs: map[string]*pmsDomain.SKU{"sku-1": {ID: "sku-1", VariantID: "variant-1", SKUCode: "WGT-1"}}}
	variantRepo := &stubVariantRepo{byID: map[string]*pmsDomain.Variant{"variant-1": {ID: "variant-1", ProductID: "product-1"}}}
	productRepo := &stubProductRepo{byID: map[string]*pmsDomain.Product{"product-1": {ID: "product-1", Name: "Widget"}}}
	uc := appprocurement.NewListBookingDiscrepanciesUseCase(db, bookingRepo, &stubLineItemRepo{}, scanRepo, skuRepo, variantRepo, productRepo)

	report, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.Len(t, report.DamagedLost, 2)
	require.Equal(t, procurementDomain.BookingScanActionDamaged, report.DamagedLost[0].ActionType)
	require.Equal(t, int64(2), report.DamagedLost[0].Quantity)
	require.Equal(t, "box crushed", report.DamagedLost[0].Reason)
	require.Equal(t, "WGT-1", *report.DamagedLost[0].SKUCode)
	require.Equal(t, "Widget", *report.DamagedLost[0].ProductName)
	require.Equal(t, procurementDomain.BookingScanActionLost, report.DamagedLost[1].ActionType)
	require.Equal(t, int64(1), report.DamagedLost[1].Quantity)
}

func TestListBookingDiscrepancies_toleratesAStaleSKUReferenceRatherThanFailingTheWholeReport(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo := &stubBookingRepo{completed: []procurementDomain.Booking{
		{ID: "booking-1", PurchaseOrderID: "po-1", Status: procurementDomain.BookingStatusCompleted},
	}}
	scanRepo := &stubBookingScanRepo{byBookingMap: map[string][]procurementDomain.BookingScan{
		"booking-1": {{ID: "scan-1", BookingID: "booking-1", SKUID: "deleted-sku", ActionType: procurementDomain.BookingScanActionLost, WriteOffQuantity: 1}},
	}}
	uc := appprocurement.NewListBookingDiscrepanciesUseCase(db, bookingRepo, &stubLineItemRepo{}, scanRepo, &stubSKURepo{}, &stubVariantRepo{}, &stubProductRepo{})

	report, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.Len(t, report.DamagedLost, 1)
	require.Nil(t, report.DamagedLost[0].SKUCode)
	require.Nil(t, report.DamagedLost[0].ProductName)
}

func TestListBookingDiscrepancies_aggregatesAcrossMultipleCompletedBookings(t *testing.T) {
	db := newMockGormDB(t)
	sku1, sku2 := "sku-1", "sku-2"
	bookingRepo := &stubBookingRepo{completed: []procurementDomain.Booking{
		{ID: "booking-1", PurchaseOrderID: "po-1", Status: procurementDomain.BookingStatusCompleted},
		{ID: "booking-2", PurchaseOrderID: "po-2", Status: procurementDomain.BookingStatusCompleted},
	}}
	lineItemRepo := &stubLineItemRepo{byPOMap: map[string][]procurementDomain.PurchaseOrderLineItem{
		"po-1": {{ID: "line-1", PurchaseOrderID: "po-1", SKUID: &sku1, Quantity: 5, LineType: procurementDomain.LineItemTypeProduct}},
		"po-2": {{ID: "line-2", PurchaseOrderID: "po-2", SKUID: &sku2, Quantity: 3, LineType: procurementDomain.LineItemTypeProduct}},
	}}
	scanRepo := &stubBookingScanRepo{byBookingMap: map[string][]procurementDomain.BookingScan{
		"booking-1": {}, // fully missing
		"booking-2": {}, // fully missing
	}}
	uc := appprocurement.NewListBookingDiscrepanciesUseCase(db, bookingRepo, lineItemRepo, scanRepo, &stubSKURepo{}, &stubVariantRepo{}, &stubProductRepo{})

	report, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.Len(t, report.MissingItems, 2)
	bookingIDs := []string{report.MissingItems[0].BookingID, report.MissingItems[1].BookingID}
	require.ElementsMatch(t, []string{"booking-1", "booking-2"}, bookingIDs)
}
