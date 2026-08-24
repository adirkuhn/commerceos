package pms_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func TestSetPromotionSalePrice_ApplySetsSalePriceAndReturnsPreviousValues(t *testing.T) {
	existingSaleEnd := time.Now().AddDate(0, 0, 3)
	existingSalePrice := int64(899)
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{
		ID: "sku-1", VariantID: "v1", PriceMinorUnits: 1000, SalePriceMinorUnits: &existingSalePrice, SaleEndDate: &existingSaleEnd,
	}}
	variantRepo := &stubVariantRepo{findResult: &pmsDomain.Variant{ID: "v1", ProductID: "p1"}}
	historyRepo := &stubSKUPriceHistoryRepo{}
	uc := apppms.NewSetPromotionSalePriceUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubOutboxRepo{}, historyRepo)

	prevPrice, prevEnd, err := uc.Apply(context.Background(), "sku-1", 700)

	require.NoError(t, err)
	require.NotNil(t, prevPrice)
	require.Equal(t, existingSalePrice, *prevPrice)
	require.NotNil(t, prevEnd)
	require.True(t, prevEnd.Equal(existingSaleEnd))
	require.NotNil(t, skuRepo.updated)
	require.NotNil(t, skuRepo.updated.SalePriceMinorUnits)
	require.Equal(t, int64(700), *skuRepo.updated.SalePriceMinorUnits)
	require.Nil(t, skuRepo.updated.SaleEndDate, "promotion controls duration via its own EndsAt, not the SKU's own field")
	require.Len(t, historyRepo.created, 1)
	require.Equal(t, pmsDomain.PriceChangeReasonPromotionApplied, historyRepo.created[0].Reason)
}

func TestSetPromotionSalePrice_ApplyReturnsNilPreviousValuesWhenNoExistingSale(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", VariantID: "v1", PriceMinorUnits: 1000}}
	variantRepo := &stubVariantRepo{findResult: &pmsDomain.Variant{ID: "v1", ProductID: "p1"}}
	uc := apppms.NewSetPromotionSalePriceUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubOutboxRepo{}, &stubSKUPriceHistoryRepo{})

	prevPrice, prevEnd, err := uc.Apply(context.Background(), "sku-1", 700)

	require.NoError(t, err)
	require.Nil(t, prevPrice)
	require.Nil(t, prevEnd)
}

func TestSetPromotionSalePrice_RevertRestoresPreviousValues(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", VariantID: "v1", PriceMinorUnits: 1000, SalePriceMinorUnits: int64Ptr(700)}}
	variantRepo := &stubVariantRepo{findResult: &pmsDomain.Variant{ID: "v1", ProductID: "p1"}}
	historyRepo := &stubSKUPriceHistoryRepo{}
	uc := apppms.NewSetPromotionSalePriceUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubOutboxRepo{}, historyRepo)

	prevPrice := int64(899)
	prevEnd := time.Now().AddDate(0, 0, 3)
	err := uc.Revert(context.Background(), "sku-1", &prevPrice, &prevEnd)

	require.NoError(t, err)
	require.NotNil(t, skuRepo.updated.SalePriceMinorUnits)
	require.Equal(t, prevPrice, *skuRepo.updated.SalePriceMinorUnits)
	require.NotNil(t, skuRepo.updated.SaleEndDate)
	require.True(t, skuRepo.updated.SaleEndDate.Equal(prevEnd))
	require.Len(t, historyRepo.created, 1)
	require.Equal(t, pmsDomain.PriceChangeReasonPromotionEnded, historyRepo.created[0].Reason)
}

func TestSetPromotionSalePrice_RevertToNilClearsSalePrice(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", VariantID: "v1", PriceMinorUnits: 1000, SalePriceMinorUnits: int64Ptr(700)}}
	variantRepo := &stubVariantRepo{findResult: &pmsDomain.Variant{ID: "v1", ProductID: "p1"}}
	uc := apppms.NewSetPromotionSalePriceUseCase(newMockGormDB(t), variantRepo, skuRepo, &stubOutboxRepo{}, &stubSKUPriceHistoryRepo{})

	err := uc.Revert(context.Background(), "sku-1", nil, nil)

	require.NoError(t, err)
	require.Nil(t, skuRepo.updated.SalePriceMinorUnits)
	require.Nil(t, skuRepo.updated.SaleEndDate)
}

func TestSetPromotionSalePrice_ApplyPropagatesSKUNotFound(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDErr: pmsDomain.ErrSKUNotFound}
	uc := apppms.NewSetPromotionSalePriceUseCase(newMockGormDB(t), &stubVariantRepo{}, skuRepo, &stubOutboxRepo{}, &stubSKUPriceHistoryRepo{})

	_, _, err := uc.Apply(context.Background(), "missing", 700)

	require.Error(t, err)
}
