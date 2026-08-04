package pms_test

import (
	"context"
	"errors"
	"testing"

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
