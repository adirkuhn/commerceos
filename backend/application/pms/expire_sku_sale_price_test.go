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

func TestExpireSKUSalePrice_clearsSalePriceAndEndDate(t *testing.T) {
	pastEnd := time.Now().Add(-24 * time.Hour)
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{
		ID: "sku-1", VariantID: "variant-1", PriceMinorUnits: 100,
		SalePriceMinorUnits: int64Ptr(80), SaleEndDate: &pastEnd,
	}}
	variantRepo := &stubVariantRepo{findResult: &pmsDomain.Variant{ID: "variant-1", ProductID: "product-1"}}
	outboxRepo := &stubOutboxRepo{}
	uc := apppms.NewExpireSKUSalePriceUseCase(newMockGormDB(t), variantRepo, skuRepo, outboxRepo)

	sku, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.Nil(t, sku.SalePriceMinorUnits)
	require.Nil(t, sku.SaleEndDate)
	require.Equal(t, int64(100), sku.PriceMinorUnits)
}

func TestExpireSKUSalePrice_raisesProductChangedEvent(t *testing.T) {
	pastEnd := time.Now().Add(-24 * time.Hour)
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{
		ID: "sku-1", VariantID: "variant-1", SalePriceMinorUnits: int64Ptr(80), SaleEndDate: &pastEnd,
	}}
	variantRepo := &stubVariantRepo{findResult: &pmsDomain.Variant{ID: "variant-1", ProductID: "product-1"}}
	outboxRepo := &stubOutboxRepo{}
	uc := apppms.NewExpireSKUSalePriceUseCase(newMockGormDB(t), variantRepo, skuRepo, outboxRepo)

	_, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.Equal(t, []string{"pms.product.changed"}, outboxRepo.calls)
}

func TestExpireSKUSalePrice_propagatesNotFound(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDErr: pmsDomain.ErrSKUNotFound}
	variantRepo := &stubVariantRepo{}
	uc := apppms.NewExpireSKUSalePriceUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), "missing")

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrSKUNotFound))
}
