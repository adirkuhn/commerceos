package promotions_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	apppromotions "github.com/adirkuhn/commerceos/application/promotions"
	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
)

func TestGetPromotion_returnsPromotionAndLines(t *testing.T) {
	promoRepo := &stubPromotionRepo{findResult: &promotionsDomain.Promotion{ID: "promo-1", Name: "Summer"}}
	lineRepo := &stubPromotionSKURepo{listResult: []promotionsDomain.PromotionSKU{
		{ID: "line-1", PromotionID: "promo-1", SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 1000},
	}}
	uc := apppromotions.NewGetPromotionUseCase(newMockGormDB(t), promoRepo, lineRepo)

	result, err := uc.Execute(context.Background(), "promo-1")

	require.NoError(t, err)
	require.Equal(t, "Summer", result.Promotion.Name)
	require.Len(t, result.Lines, 1)
	require.Equal(t, "sku-1", result.Lines[0].SKUID)
}

func TestGetPromotion_propagatesNotFound(t *testing.T) {
	promoRepo := &stubPromotionRepo{}
	uc := apppromotions.NewGetPromotionUseCase(newMockGormDB(t), promoRepo, &stubPromotionSKURepo{})

	_, err := uc.Execute(context.Background(), "missing")

	require.Error(t, err)
	require.True(t, errors.Is(err, promotionsDomain.ErrPromotionNotFound))
}
