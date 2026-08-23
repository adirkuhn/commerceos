package promotions_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	apppromotions "github.com/adirkuhn/commerceos/application/promotions"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
)

func TestGetPromotion_returnsPromotionAndLinesWithSKUCode(t *testing.T) {
	promoRepo := &stubPromotionRepo{findResult: &promotionsDomain.Promotion{ID: "promo-1", Name: "Summer"}}
	lineRepo := &stubPromotionSKURepo{listResult: []promotionsDomain.PromotionSKU{
		{ID: "line-1", PromotionID: "promo-1", SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 1000},
	}}
	skuLookup := &stubSKUExistenceChecker{findResult: &pmsDomain.SKU{ID: "sku-1", SKUCode: "ACAI-1L"}}
	uc := apppromotions.NewGetPromotionUseCase(newMockGormDB(t), promoRepo, lineRepo, skuLookup)

	result, err := uc.Execute(context.Background(), "promo-1")

	require.NoError(t, err)
	require.Equal(t, "Summer", result.Promotion.Name)
	require.Len(t, result.Lines, 1)
	require.Equal(t, "sku-1", result.Lines[0].SKUID)
	require.Equal(t, "ACAI-1L", result.Lines[0].SKUCode)
}

func TestGetPromotion_blankSKUCodeWhenSKUSinceDeleted(t *testing.T) {
	promoRepo := &stubPromotionRepo{findResult: &promotionsDomain.Promotion{ID: "promo-1", Name: "Summer"}}
	lineRepo := &stubPromotionSKURepo{listResult: []promotionsDomain.PromotionSKU{
		{ID: "line-1", PromotionID: "promo-1", SKUID: "sku-deleted", DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 1000},
	}}
	skuLookup := &stubSKUExistenceChecker{findErr: pmsDomain.ErrSKUNotFound}
	uc := apppromotions.NewGetPromotionUseCase(newMockGormDB(t), promoRepo, lineRepo, skuLookup)

	result, err := uc.Execute(context.Background(), "promo-1")

	require.NoError(t, err)
	require.Equal(t, "", result.Lines[0].SKUCode)
}

func TestGetPromotion_propagatesNotFound(t *testing.T) {
	promoRepo := &stubPromotionRepo{}
	uc := apppromotions.NewGetPromotionUseCase(newMockGormDB(t), promoRepo, &stubPromotionSKURepo{}, &stubSKUExistenceChecker{})

	_, err := uc.Execute(context.Background(), "missing")

	require.Error(t, err)
	require.True(t, errors.Is(err, promotionsDomain.ErrPromotionNotFound))
}
