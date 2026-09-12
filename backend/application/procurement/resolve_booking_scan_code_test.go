package procurement_test

import (
	"context"
	"testing"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resolveBookingScanCodeFixtures() (*stubBookingRepo, *stubPurchaseOrderRepo, *stubVariantRepo, *stubProductRepo) {
	bookingRepo, poRepo := activeBookingFixtures()
	variantRepo := &stubVariantRepo{byID: map[string]*pmsDomain.Variant{"variant-1": {ID: "variant-1", ProductID: "product-1"}}}
	productRepo := &stubProductRepo{byID: map[string]*pmsDomain.Product{"product-1": {ID: "product-1", Name: "Widget"}}}
	return bookingRepo, poRepo, variantRepo, productRepo
}

func TestResolveBookingScanCode_resolvesUnitBarcodeWithoutStagingAScan(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo, poRepo, variantRepo, productRepo := resolveBookingScanCodeFixtures()
	skuRepo := &stubSKURepo{byBarcode: map[string]*pmsDomain.SKU{"BARCODE-1": {ID: "sku-1", VariantID: "variant-1"}}}
	aliasRepo := &stubBarcodeAliasRepo{}
	uc := appprocurement.NewResolveBookingScanCodeUseCase(db, bookingRepo, poRepo, skuRepo, aliasRepo, &stubScaleCodeAliasRepo{}, &stubLineItemRepo{}, variantRepo, productRepo)

	result, err := uc.Execute(context.Background(), "booking-1", "BARCODE-1")

	require.NoError(t, err)
	assert.Equal(t, "sku-1", result.SKUID)
	assert.Equal(t, "Widget", result.ProductName)
	assert.Equal(t, int64(1), result.UnitsPerScan)
}

func TestResolveBookingScanCode_resolvesCaseBarcodeAliasWithMultiplier(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo, poRepo, variantRepo, productRepo := resolveBookingScanCodeFixtures()
	skuRepo := &stubSKURepo{findByIDs: map[string]*pmsDomain.SKU{"sku-1": {ID: "sku-1", VariantID: "variant-1"}}}
	aliasRepo := &stubBarcodeAliasRepo{byCode: map[string]*pmsDomain.BarcodeAlias{"CASE-1": {SKUID: "sku-1", UnitsPerScan: 12}}}
	uc := appprocurement.NewResolveBookingScanCodeUseCase(db, bookingRepo, poRepo, skuRepo, aliasRepo, &stubScaleCodeAliasRepo{}, &stubLineItemRepo{}, variantRepo, productRepo)

	result, err := uc.Execute(context.Background(), "booking-1", "CASE-1")

	require.NoError(t, err)
	assert.Equal(t, "sku-1", result.SKUID)
	assert.Equal(t, int64(12), result.UnitsPerScan)
}

func TestResolveBookingScanCode_resolvesWeightEmbeddedBarcodeAndReturnsWeightGrams(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo, poRepo, variantRepo, productRepo := resolveBookingScanCodeFixtures()
	skuRepo := &stubSKURepo{byScaleCode: map[string]*pmsDomain.SKU{"2994721": {ID: "sku-1", VariantID: "variant-1", PricingMode: pmsDomain.SKUPricingModePerWeight}}}
	uc := appprocurement.NewResolveBookingScanCodeUseCase(db, bookingRepo, poRepo, skuRepo, &stubBarcodeAliasRepo{}, &stubScaleCodeAliasRepo{}, &stubLineItemRepo{}, variantRepo, productRepo)

	// Real dry-meat label: scale code 94721, weight 462g.
	result, err := uc.Execute(context.Background(), "booking-1", "2994721004622")

	require.NoError(t, err)
	assert.Equal(t, "sku-1", result.SKUID)
	require.NotNil(t, result.WeightGrams)
	assert.Equal(t, int64(462), *result.WeightGrams)
}

func TestResolveBookingScanCode_resolvesViaScaleCodeAliasForASecondScale(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo, poRepo, variantRepo, productRepo := resolveBookingScanCodeFixtures()
	scaleAliasRepo := &stubScaleCodeAliasRepo{byCode: map[string]*pmsDomain.ScaleCodeAlias{"2994722": {SKUID: "sku-1"}}}
	skuRepo := &stubSKURepo{findByIDs: map[string]*pmsDomain.SKU{"sku-1": {ID: "sku-1", VariantID: "variant-1"}}}
	uc := appprocurement.NewResolveBookingScanCodeUseCase(db, bookingRepo, poRepo, skuRepo, &stubBarcodeAliasRepo{}, scaleAliasRepo, &stubLineItemRepo{}, variantRepo, productRepo)

	result, err := uc.Execute(context.Background(), "booking-1", "2994722004621")

	require.NoError(t, err)
	assert.Equal(t, "sku-1", result.SKUID)
	require.NotNil(t, result.WeightGrams)
}

func TestResolveBookingScanCode_rejectsSKUCodeMatchForAPerWeightSKU(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo, poRepo, variantRepo, productRepo := resolveBookingScanCodeFixtures()
	skuRepo := &stubSKURepo{byCode: map[string]*pmsDomain.SKU{"DRYMEAT-1": {ID: "sku-1", PricingMode: pmsDomain.SKUPricingModePerWeight}}}
	uc := appprocurement.NewResolveBookingScanCodeUseCase(db, bookingRepo, poRepo, skuRepo, &stubBarcodeAliasRepo{}, &stubScaleCodeAliasRepo{}, &stubLineItemRepo{}, variantRepo, productRepo)

	_, err := uc.Execute(context.Background(), "booking-1", "DRYMEAT-1")

	assert.ErrorIs(t, err, pmsDomain.ErrWeightRequiredForPerWeightSKU)
}

func TestResolveBookingScanCode_rejectsCorruptedWeightEmbeddedCheckDigit(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo, poRepo, variantRepo, productRepo := resolveBookingScanCodeFixtures()
	uc := appprocurement.NewResolveBookingScanCodeUseCase(db, bookingRepo, poRepo, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubScaleCodeAliasRepo{}, &stubLineItemRepo{}, variantRepo, productRepo)

	_, err := uc.Execute(context.Background(), "booking-1", "2900633003339")

	assert.ErrorIs(t, err, pmsDomain.ErrInvalidCheckDigit)
}

func TestResolveBookingScanCode_resolvesTypedSKUCode(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo, poRepo, variantRepo, productRepo := resolveBookingScanCodeFixtures()
	skuRepo := &stubSKURepo{byCode: map[string]*pmsDomain.SKU{"SKU-1": {ID: "sku-1", VariantID: "variant-1"}}}
	aliasRepo := &stubBarcodeAliasRepo{}
	uc := appprocurement.NewResolveBookingScanCodeUseCase(db, bookingRepo, poRepo, skuRepo, aliasRepo, &stubScaleCodeAliasRepo{}, &stubLineItemRepo{}, variantRepo, productRepo)

	result, err := uc.Execute(context.Background(), "booking-1", "SKU-1")

	require.NoError(t, err)
	assert.Equal(t, "sku-1", result.SKUID)
	assert.Equal(t, int64(1), result.UnitsPerScan)
}

func TestResolveBookingScanCode_fallsBackToThisPOsLineItemRawSKUWhenCatalogWideLookupsAllMiss(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo, poRepo, variantRepo, productRepo := resolveBookingScanCodeFixtures()
	skuRepo := &stubSKURepo{findByIDs: map[string]*pmsDomain.SKU{"sku-1": {ID: "sku-1", VariantID: "variant-1"}}}
	aliasRepo := &stubBarcodeAliasRepo{}
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{
		{ID: "li-1", PurchaseOrderID: "po-1", RawSKU: " raw-code-1 ", SKUID: strPtr("sku-1")},
	}}
	uc := appprocurement.NewResolveBookingScanCodeUseCase(db, bookingRepo, poRepo, skuRepo, aliasRepo, &stubScaleCodeAliasRepo{}, lineItemRepo, variantRepo, productRepo)

	result, err := uc.Execute(context.Background(), "booking-1", "raw-code-1")

	require.NoError(t, err)
	assert.Equal(t, "sku-1", result.SKUID)
}

func TestResolveBookingScanCode_lineItemFallbackIgnoresAnUnlinkedLineItem(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo, poRepo, variantRepo, productRepo := resolveBookingScanCodeFixtures()
	skuRepo := &stubSKURepo{}
	aliasRepo := &stubBarcodeAliasRepo{}
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{
		{ID: "li-1", PurchaseOrderID: "po-1", RawSKU: "raw-code-1", SKUID: nil},
	}}
	uc := appprocurement.NewResolveBookingScanCodeUseCase(db, bookingRepo, poRepo, skuRepo, aliasRepo, &stubScaleCodeAliasRepo{}, lineItemRepo, variantRepo, productRepo)

	_, err := uc.Execute(context.Background(), "booking-1", "raw-code-1")

	require.Error(t, err)
	assert.ErrorIs(t, err, pmsDomain.ErrSKUNotFound)
}

func TestResolveBookingScanCode_returnsNotFoundWhenNothingMatches(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo, poRepo, variantRepo, productRepo := resolveBookingScanCodeFixtures()
	uc := appprocurement.NewResolveBookingScanCodeUseCase(db, bookingRepo, poRepo, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubScaleCodeAliasRepo{}, &stubLineItemRepo{}, variantRepo, productRepo)

	_, err := uc.Execute(context.Background(), "booking-1", "unknown-code")

	require.Error(t, err)
	assert.ErrorIs(t, err, pmsDomain.ErrSKUNotFound)
}

func TestResolveBookingScanCode_rejectsAnEmptyCode(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo, poRepo, variantRepo, productRepo := resolveBookingScanCodeFixtures()
	uc := appprocurement.NewResolveBookingScanCodeUseCase(db, bookingRepo, poRepo, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubScaleCodeAliasRepo{}, &stubLineItemRepo{}, variantRepo, productRepo)

	_, err := uc.Execute(context.Background(), "booking-1", "   ")

	require.Error(t, err)
	assert.ErrorIs(t, err, procurementDomain.ErrScanCodeOrSKURequired)
}

func TestResolveBookingScanCode_rejectsANonActiveBooking(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo := &stubBookingRepo{findResult: &procurementDomain.Booking{ID: "booking-1", PurchaseOrderID: "po-1", Status: procurementDomain.BookingStatusPaused}}
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1"}}
	_, _, variantRepo, productRepo := resolveBookingScanCodeFixtures()
	uc := appprocurement.NewResolveBookingScanCodeUseCase(db, bookingRepo, poRepo, &stubSKURepo{}, &stubBarcodeAliasRepo{}, &stubScaleCodeAliasRepo{}, &stubLineItemRepo{}, variantRepo, productRepo)

	_, err := uc.Execute(context.Background(), "booking-1", "some-code")

	require.Error(t, err)
	assert.ErrorIs(t, err, procurementDomain.ErrBookingNotActive)
}
