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

func TestCreateSKU_rejectsDimensionsOnVirtualSKU(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo)

	_, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "SKU-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, IsVirtual: true, WeightGrams: int64Ptr(500),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrVirtualSKUCannotHaveDimensions))
}

func TestCreateSKU_allowsVirtualSKUWithoutDimensions(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo)

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
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo)

	sku, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "SKU-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, IsVirtual: false, WeightGrams: int64Ptr(500),
	})

	require.NoError(t, err)
	require.False(t, sku.IsVirtual)
	require.Equal(t, int64(500), *sku.WeightGrams)
}

func TestCreateSKU_rejectsSalePriceAboveRegularPrice(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo)

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
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo)

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
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo)

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
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo)

	endDate := time.Now().Add(24 * time.Hour)
	sku, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "SKU-1", PriceMinorUnits: 100, SalePriceMinorUnits: int64Ptr(80), SaleEndDate: &endDate,
		CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive,
	})

	require.NoError(t, err)
	require.Equal(t, endDate, *sku.SaleEndDate)
}

func TestCreateSKU_persistsTracksExpiry(t *testing.T) {
	variantRepo := &stubVariantRepo{}
	skuRepo := &stubSKURepo{}
	uc := apppms.NewCreateSKUUseCase(newMockGormDB(t), variantRepo, skuRepo)

	sku, err := uc.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: "variant-1", SKUCode: "SKU-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, TracksExpiry: true,
	})

	require.NoError(t, err)
	require.True(t, sku.TracksExpiry)
}
