package pms_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func int64Ptr(v int64) *int64 { return &v }
func strPtr(v string) *string { return &v }

func TestCreateSKU_rejectsDimensionsOnVirtualSKU(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubSKUPriceHistoryRepo{}, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "SKU-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, IsVirtual: true, WeightGrams: int64Ptr(500),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrVirtualSKUCannotHaveDimensions))
}

func TestCreateSKU_persistsTaxClassID(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubSKUPriceHistoryRepo{}, &stubOutboxRepo{})

	sku, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "SKU-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, TaxClassID: strPtr("class-1"),
	})

	require.NoError(t, err)
	require.NotNil(t, sku.TaxClassID)
	require.Equal(t, "class-1", *sku.TaxClassID)
}

func TestCreateSKU_allowsVirtualSKUWithoutDimensions(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubSKUPriceHistoryRepo{}, &stubOutboxRepo{})

	sku, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "SKU-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, IsVirtual: true,
	})

	require.NoError(t, err)
	require.True(t, sku.IsVirtual)
}

func TestCreateSKU_allowsPhysicalSKUWithDimensions(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubSKUPriceHistoryRepo{}, &stubOutboxRepo{})

	sku, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "SKU-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, IsVirtual: false, WeightGrams: int64Ptr(500),
	})

	require.NoError(t, err)
	require.False(t, sku.IsVirtual)
	require.Equal(t, int64(500), *sku.WeightGrams)
}

func TestCreateSKU_rejectsPerWeightPricingWithoutPricePerKilo(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubSKUPriceHistoryRepo{}, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "SKU-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, PricingMode: pmsDomain.SKUPricingModePerWeight,
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrPricePerKiloRequiredForPerWeight))
}

func TestCreateSKU_rejectsPricePerKiloOnFixedPricing(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubSKUPriceHistoryRepo{}, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "SKU-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, PricingMode: pmsDomain.SKUPricingModeFixed,
		PricePerKiloMinorUnits: int64Ptr(2500),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrPricePerKiloOnlyForPerWeight))
}

func TestCreateSKU_rejectsNonPositivePricePerKilo(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubSKUPriceHistoryRepo{}, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "SKU-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, PricingMode: pmsDomain.SKUPricingModePerWeight,
		PricePerKiloMinorUnits: int64Ptr(0),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrInvalidPricePerKilo))
}

func TestCreateSKU_rejectsInvalidPricingMode(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubSKUPriceHistoryRepo{}, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "SKU-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, PricingMode: pmsDomain.SKUPricingMode("bogus"),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrInvalidSKUPricingMode))
}

func TestCreateSKU_defaultsToFixedPricingMode(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubSKUPriceHistoryRepo{}, &stubOutboxRepo{})

	sku, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "SKU-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive,
	})

	require.NoError(t, err)
	require.Equal(t, pmsDomain.SKUPricingModeFixed, sku.PricingMode)
}

func TestCreateSKU_persistsPerWeightPricingWithScaleCode(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubSKUPriceHistoryRepo{}, &stubOutboxRepo{})

	// Mirrors the "dry meat" case: price computed from a scanned package's
	// actual weight, resolved via ScaleCode (see
	// ResolveBarcodeUseCase/DecodeWeightEmbeddedBarcode) rather than an
	// exact barcode match.
	sku, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "SKU-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, PricingMode: pmsDomain.SKUPricingModePerWeight,
		PricePerKiloMinorUnits: int64Ptr(4999), ScaleCode: strPtr("94721"),
	})

	require.NoError(t, err)
	require.Equal(t, pmsDomain.SKUPricingModePerWeight, sku.PricingMode)
	require.Equal(t, int64(4999), *sku.PricePerKiloMinorUnits)
	require.Equal(t, "94721", *sku.ScaleCode)
}

func TestCreateSKU_rejectsSalePriceAboveRegularPrice(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubSKUPriceHistoryRepo{}, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "SKU-1", PriceMinorUnits: 100, SalePriceMinorUnits: int64Ptr(150),
		CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive,
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrSalePriceExceedsPrice))
}

func TestCreateSKU_allowsSalePriceEqualToOrBelowRegularPrice(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubSKUPriceHistoryRepo{}, &stubOutboxRepo{})

	sku, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "SKU-1", PriceMinorUnits: 100, SalePriceMinorUnits: int64Ptr(80),
		CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive,
	})

	require.NoError(t, err)
	require.Equal(t, int64(80), *sku.SalePriceMinorUnits)
}

func TestCreateSKU_rejectsSaleEndDateWithoutSalePrice(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubSKUPriceHistoryRepo{}, &stubOutboxRepo{})

	endDate := time.Now().Add(24 * time.Hour)
	_, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "SKU-1", PriceMinorUnits: 100, SaleEndDate: &endDate,
		CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive,
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrSaleEndDateRequiresSalePrice))
}

func TestCreateSKU_allowsSaleEndDateWithSalePrice(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubSKUPriceHistoryRepo{}, &stubOutboxRepo{})

	endDate := time.Now().Add(24 * time.Hour)
	sku, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "SKU-1", PriceMinorUnits: 100, SalePriceMinorUnits: int64Ptr(80), SaleEndDate: &endDate,
		CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive,
	})

	require.NoError(t, err)
	require.Equal(t, endDate, *sku.SaleEndDate)
}

func TestCreateSKU_persistsCasePackRelationship(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-unit-1"}}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubSKUPriceHistoryRepo{}, &stubOutboxRepo{})

	sku, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "SKU-CASE-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, CaseOfSKUID: strPtr("sku-unit-1"), UnitsPerCase: int64Ptr(36),
	})

	require.NoError(t, err)
	require.Equal(t, "sku-unit-1", *sku.CaseOfSKUID)
	require.Equal(t, int64(36), *sku.UnitsPerCase)
}

func TestCreateSKU_rejectsCaseOfSKUIDWithoutUnitsPerCase(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubSKUPriceHistoryRepo{}, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "SKU-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, CaseOfSKUID: strPtr("sku-unit-1"),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrCaseOfSKURequiresUnitsPerCase))
}

func TestCreateSKU_rejectsUnitsPerCaseWithoutCaseOfSKUID(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubSKUPriceHistoryRepo{}, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "SKU-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, UnitsPerCase: int64Ptr(36),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrCaseOfSKURequiresUnitsPerCase))
}

func TestCreateSKU_rejectsNonPositiveUnitsPerCase(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubSKUPriceHistoryRepo{}, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "SKU-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, CaseOfSKUID: strPtr("sku-unit-1"), UnitsPerCase: int64Ptr(0),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrUnitsPerCaseMustBePositive))
}

func TestCreateSKU_rejectsCaseOfSKUIDThatIsItselfACase(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-already-a-case", CaseOfSKUID: strPtr("sku-deeper"), UnitsPerCase: int64Ptr(12)}}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubSKUPriceHistoryRepo{}, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "SKU-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, CaseOfSKUID: strPtr("sku-already-a-case"), UnitsPerCase: int64Ptr(36),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrCaseOfSKUCannotChain))
}

func TestCreateSKU_rejectsCaseOfSKUIDThatDoesNotExist(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{findByIDErr: pmsDomain.ErrSKUNotFound}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubSKUPriceHistoryRepo{}, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "SKU-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, CaseOfSKUID: strPtr("missing-sku"), UnitsPerCase: int64Ptr(36),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrSKUNotFound))
}

func TestCreateSKU_persistsUnitType(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubSKUPriceHistoryRepo{}, &stubOutboxRepo{})

	sku, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "BULK-1", PriceMinorUnits: 0, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, UnitType: pmsDomain.SKUUnitTypeKilo,
	})

	require.NoError(t, err)
	require.Equal(t, pmsDomain.SKUUnitTypeKilo, sku.UnitType)
}

func TestCreateSKU_defaultsUnitTypeToUnitWhenNotGiven(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubSKUPriceHistoryRepo{}, &stubOutboxRepo{})

	sku, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "SKU-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive,
	})

	require.NoError(t, err)
	require.Equal(t, pmsDomain.SKUUnitTypeUnit, sku.UnitType)
}

func TestCreateSKU_persistsPortionedFromRelationship(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-bulk-1", UnitType: pmsDomain.SKUUnitTypeKilo}}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubSKUPriceHistoryRepo{}, &stubOutboxRepo{})

	sku, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "PORTION-378G", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, PortionedFromSKUID: strPtr("sku-bulk-1"), WeightGrams: int64Ptr(378),
	})

	require.NoError(t, err)
	require.Equal(t, "sku-bulk-1", *sku.PortionedFromSKUID)
}

func TestCreateSKU_rejectsBulkSKUThatIsAlsoPortionedFromAnother(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubSKUPriceHistoryRepo{}, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "SKU-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, UnitType: pmsDomain.SKUUnitTypeKilo, PortionedFromSKUID: strPtr("sku-bulk-1"),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrBulkSKUCannotBePortioned))
}

func TestCreateSKU_rejectsPortionedFromSKUThatDoesNotTrackWeight(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-not-bulk", UnitType: pmsDomain.SKUUnitTypeUnit}}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubSKUPriceHistoryRepo{}, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "SKU-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, PortionedFromSKUID: strPtr("sku-not-bulk"),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrPortionedFromSKUMustTrackWeight))
}

func TestCreateSKU_rejectsPortionedFromSKUThatDoesNotExist(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{findByIDErr: pmsDomain.ErrSKUNotFound}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubSKUPriceHistoryRepo{}, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "SKU-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, PortionedFromSKUID: strPtr("missing-sku"),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrSKUNotFound))
}

func TestCreateSKU_persistsTracksExpiry(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubSKUPriceHistoryRepo{}, &stubOutboxRepo{})

	sku, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "SKU-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, TracksExpiry: true,
	})

	require.NoError(t, err)
	require.True(t, sku.TracksExpiry)
}

func TestCreateSKU_trimsWhitespaceFromBarcode(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubSKUPriceHistoryRepo{}, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "SKU-1", Barcode: "  0123456789012 \n", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive,
	})

	require.NoError(t, err)
	require.Equal(t, "0123456789012", skuRepo.created.Barcode, "a scanner/copy-paste can leave stray whitespace around an otherwise-correct barcode -- must not silently break exact-match lookup")
}

func TestCreateSKU_rejectsInvalidUnitType(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubSKUPriceHistoryRepo{}, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "SKU-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, UnitType: "pallet",
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrInvalidSKUUnitType))
}

func TestCreateSKU_recordsInitialPriceHistory(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{}
	priceHistoryRepo := &stubSKUPriceHistoryRepo{}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, priceHistoryRepo, &stubOutboxRepo{})

	sku, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "SKU-1", PriceMinorUnits: 999, SalePriceMinorUnits: int64Ptr(799),
		CurrencyCode: "EUR", Status: pmsDomain.SKUStatusActive,
	})

	require.NoError(t, err)
	require.Len(t, priceHistoryRepo.created, 1)
	entry := priceHistoryRepo.created[0]
	require.Equal(t, sku.ID, entry.SKUID)
	require.Equal(t, int64(999), entry.PriceMinorUnits)
	require.Equal(t, int64(799), *entry.SalePriceMinorUnits)
	require.Equal(t, "EUR", entry.CurrencyCode)
	require.Equal(t, pmsDomain.PriceChangeReasonCreated, entry.Reason)
	require.Nil(t, entry.ValidUntil)
	require.Empty(t, priceHistoryRepo.closedSKUIDs, "nothing to close on a brand new SKU")
}

// TestCreateSKU_enqueuesProductChanged is a regression test for a brand-new
// variant-mode product never getting enrolled in any channel:
// CreateProductUseCase's own pms.product.changed fires enrollment before
// any variant/SKU exists, which finds zero variants and bails without ever
// creating a ProductChannelListing. Nothing re-triggers enrollment unless
// this create path also raises the event.
func TestCreateSKU_enqueuesProductChanged(t *testing.T) {
	variantRepo := &stubVariantRepo{findResult: &pmsDomain.Variant{ID: "variant-1", ProductID: "p1"}}
	skuRepo := &stubSKURepo{}
	outboxRepo := &stubOutboxRepo{}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubSKUPriceHistoryRepo{}, outboxRepo)

	_, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "SKU-900ML", PriceMinorUnits: 999, CurrencyCode: "EUR",
		Status: pmsDomain.SKUStatusActive,
	})

	require.NoError(t, err)
	require.Len(t, outboxRepo.calls, 1)
	require.Equal(t, "pms.product.changed", outboxRepo.calls[0])
	require.Contains(t, string(outboxRepo.payloads[0]), `"product_id":"p1"`)
}
