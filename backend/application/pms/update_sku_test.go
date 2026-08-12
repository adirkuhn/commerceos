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

func TestUpdateSKU_rejectsDimensionsOnVirtualSKU(t *testing.T) {
	skuRepo := &stubSKURepo{}
	variantRepo := &stubVariantRepo{}
	uc := apppms.NewUpdateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), apppms.UpdateSKUInput{
		ID: "sku-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, IsVirtual: true, LengthMM: int64Ptr(10),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrVirtualSKUCannotHaveDimensions))
}

func TestUpdateSKU_allowsVirtualSKUWithoutDimensions(t *testing.T) {
	skuRepo := &stubSKURepo{}
	variantRepo := &stubVariantRepo{}
	uc := apppms.NewUpdateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubOutboxRepo{})

	sku, err := uc.Execute(context.Background(), apppms.UpdateSKUInput{
		ID: "sku-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, IsVirtual: true,
	})

	require.NoError(t, err)
	require.True(t, sku.IsVirtual)
}

func TestUpdateSKU_raisesProductChangedEventForTheSKUsProduct(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", VariantID: "variant-1"}}
	variantRepo := &stubVariantRepo{findResult: &pmsDomain.Variant{ID: "variant-1", ProductID: "product-1"}}
	outboxRepo := &stubOutboxRepo{}
	uc := apppms.NewUpdateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, outboxRepo)

	_, err := uc.Execute(context.Background(), apppms.UpdateSKUInput{
		ID: "sku-1", PriceMinorUnits: 100, CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive,
	})

	require.NoError(t, err)
	require.Equal(t, []string{"pms.product.changed"}, outboxRepo.calls)
}

func TestUpdateSKU_rejectsSalePriceAboveRegularPrice(t *testing.T) {
	skuRepo := &stubSKURepo{}
	variantRepo := &stubVariantRepo{}
	uc := apppms.NewUpdateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), apppms.UpdateSKUInput{
		ID: "sku-1", PriceMinorUnits: 100, SalePriceMinorUnits: int64Ptr(150),
		CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive,
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrSalePriceExceedsPrice))
}

func TestUpdateSKU_canClearSalePriceBackToNil(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", VariantID: "variant-1", SalePriceMinorUnits: int64Ptr(80)}}
	variantRepo := &stubVariantRepo{findResult: &pmsDomain.Variant{ID: "variant-1", ProductID: "product-1"}}
	uc := apppms.NewUpdateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubOutboxRepo{})

	sku, err := uc.Execute(context.Background(), apppms.UpdateSKUInput{
		ID: "sku-1", PriceMinorUnits: 100, SalePriceMinorUnits: nil,
		CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive,
	})

	require.NoError(t, err)
	require.Nil(t, sku.SalePriceMinorUnits)
}

func TestUpdateSKU_rejectsSaleEndDateWithoutSalePrice(t *testing.T) {
	skuRepo := &stubSKURepo{}
	variantRepo := &stubVariantRepo{}
	uc := apppms.NewUpdateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubOutboxRepo{})

	endDate := time.Now().Add(24 * time.Hour)
	_, err := uc.Execute(context.Background(), apppms.UpdateSKUInput{
		ID: "sku-1", PriceMinorUnits: 100, SaleEndDate: &endDate,
		CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive,
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrSaleEndDateRequiresSalePrice))
}

func TestUpdateSKU_canClearSaleEndDateBackToNil(t *testing.T) {
	oldEnd := time.Now().Add(24 * time.Hour)
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", VariantID: "variant-1", SalePriceMinorUnits: int64Ptr(80), SaleEndDate: &oldEnd}}
	variantRepo := &stubVariantRepo{findResult: &pmsDomain.Variant{ID: "variant-1", ProductID: "product-1"}}
	uc := apppms.NewUpdateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubOutboxRepo{})

	sku, err := uc.Execute(context.Background(), apppms.UpdateSKUInput{
		ID: "sku-1", PriceMinorUnits: 100, SalePriceMinorUnits: int64Ptr(80), SaleEndDate: nil,
		CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive,
	})

	require.NoError(t, err)
	require.Nil(t, sku.SaleEndDate)
}

func TestUpdateSKU_persistsCasePackRelationship(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-unit-1"}}
	variantRepo := &stubVariantRepo{}
	uc := apppms.NewUpdateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubOutboxRepo{})

	sku, err := uc.Execute(context.Background(), apppms.UpdateSKUInput{
		ID: "sku-case-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, CaseOfSKUID: strPtr("sku-unit-1"), UnitsPerCase: int64Ptr(36),
	})

	require.NoError(t, err)
	require.Equal(t, "sku-unit-1", *sku.CaseOfSKUID)
	require.Equal(t, int64(36), *sku.UnitsPerCase)
}

func TestUpdateSKU_rejectsCaseOfSKUIDWithoutUnitsPerCase(t *testing.T) {
	skuRepo := &stubSKURepo{}
	variantRepo := &stubVariantRepo{}
	uc := apppms.NewUpdateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), apppms.UpdateSKUInput{
		ID: "sku-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, CaseOfSKUID: strPtr("sku-unit-1"),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrCaseOfSKURequiresUnitsPerCase))
}

func TestUpdateSKU_rejectsSelfReferentialCaseOfSKUID(t *testing.T) {
	skuRepo := &stubSKURepo{}
	variantRepo := &stubVariantRepo{}
	uc := apppms.NewUpdateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), apppms.UpdateSKUInput{
		ID: "sku-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, CaseOfSKUID: strPtr("sku-1"), UnitsPerCase: int64Ptr(36),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrCaseOfSKUCannotBeSelf))
}

func TestUpdateSKU_rejectsCaseOfSKUIDThatIsItselfACase(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-already-a-case", CaseOfSKUID: strPtr("sku-deeper"), UnitsPerCase: int64Ptr(12)}}
	variantRepo := &stubVariantRepo{}
	uc := apppms.NewUpdateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), apppms.UpdateSKUInput{
		ID: "sku-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, CaseOfSKUID: strPtr("sku-already-a-case"), UnitsPerCase: int64Ptr(36),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrCaseOfSKUCannotChain))
}

func TestUpdateSKU_canClearCasePackRelationshipBackToNil(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", CaseOfSKUID: strPtr("sku-unit-1"), UnitsPerCase: int64Ptr(36)}}
	variantRepo := &stubVariantRepo{}
	uc := apppms.NewUpdateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubOutboxRepo{})

	sku, err := uc.Execute(context.Background(), apppms.UpdateSKUInput{
		ID: "sku-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive,
	})

	require.NoError(t, err)
	require.Nil(t, sku.CaseOfSKUID)
	require.Nil(t, sku.UnitsPerCase)
}

func TestUpdateSKU_persistsUnitType(t *testing.T) {
	skuRepo := &stubSKURepo{}
	variantRepo := &stubVariantRepo{}
	uc := apppms.NewUpdateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubOutboxRepo{})

	sku, err := uc.Execute(context.Background(), apppms.UpdateSKUInput{
		ID: "sku-bulk-1", PriceMinorUnits: 0, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, UnitType: pmsDomain.SKUUnitTypeKilo,
	})

	require.NoError(t, err)
	require.Equal(t, pmsDomain.SKUUnitTypeKilo, sku.UnitType)
}

func TestUpdateSKU_defaultsUnitTypeToUnitWhenNotGiven(t *testing.T) {
	skuRepo := &stubSKURepo{}
	variantRepo := &stubVariantRepo{}
	uc := apppms.NewUpdateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubOutboxRepo{})

	sku, err := uc.Execute(context.Background(), apppms.UpdateSKUInput{
		ID: "sku-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive,
	})

	require.NoError(t, err)
	require.Equal(t, pmsDomain.SKUUnitTypeUnit, sku.UnitType)
}

func TestUpdateSKU_persistsPortionedFromRelationship(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-bulk-1", UnitType: pmsDomain.SKUUnitTypeKilo}}
	variantRepo := &stubVariantRepo{}
	uc := apppms.NewUpdateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubOutboxRepo{})

	sku, err := uc.Execute(context.Background(), apppms.UpdateSKUInput{
		ID: "sku-378g", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, PortionedFromSKUID: strPtr("sku-bulk-1"),
	})

	require.NoError(t, err)
	require.Equal(t, "sku-bulk-1", *sku.PortionedFromSKUID)
}

func TestUpdateSKU_rejectsSelfReferentialPortionedFromSKUID(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", UnitType: pmsDomain.SKUUnitTypeKilo}}
	variantRepo := &stubVariantRepo{}
	uc := apppms.NewUpdateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), apppms.UpdateSKUInput{
		ID: "sku-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, PortionedFromSKUID: strPtr("sku-1"),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrPortionedFromSKUCannotBeSelf))
}

func TestUpdateSKU_rejectsBulkSKUThatIsAlsoPortionedFromAnother(t *testing.T) {
	skuRepo := &stubSKURepo{}
	variantRepo := &stubVariantRepo{}
	uc := apppms.NewUpdateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), apppms.UpdateSKUInput{
		ID: "sku-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, UnitType: pmsDomain.SKUUnitTypeKilo, PortionedFromSKUID: strPtr("sku-bulk-1"),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrBulkSKUCannotBePortioned))
}

func TestUpdateSKU_rejectsPortionedFromSKUThatDoesNotTrackWeight(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-not-bulk", UnitType: pmsDomain.SKUUnitTypeUnit}}
	variantRepo := &stubVariantRepo{}
	uc := apppms.NewUpdateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), apppms.UpdateSKUInput{
		ID: "sku-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, PortionedFromSKUID: strPtr("sku-not-bulk"),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrPortionedFromSKUMustTrackWeight))
}

func TestUpdateSKU_canClearPortionedFromRelationshipBackToNil(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", PortionedFromSKUID: strPtr("sku-bulk-1")}}
	variantRepo := &stubVariantRepo{}
	uc := apppms.NewUpdateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubOutboxRepo{})

	sku, err := uc.Execute(context.Background(), apppms.UpdateSKUInput{
		ID: "sku-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive,
	})

	require.NoError(t, err)
	require.Nil(t, sku.PortionedFromSKUID)
}

func TestUpdateSKU_persistsTracksExpiry(t *testing.T) {
	skuRepo := &stubSKURepo{}
	variantRepo := &stubVariantRepo{}
	uc := apppms.NewUpdateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubOutboxRepo{})

	sku, err := uc.Execute(context.Background(), apppms.UpdateSKUInput{
		ID: "sku-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, TracksExpiry: true,
	})

	require.NoError(t, err)
	require.True(t, sku.TracksExpiry)
}
