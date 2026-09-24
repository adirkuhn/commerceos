package promotions_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apppromotions "github.com/adirkuhn/commerceos/application/promotions"
	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
)

func TestUpdatePromotion_replacesLinesWholesale(t *testing.T) {
	promoRepo := &stubPromotionRepo{findResult: &promotionsDomain.Promotion{ID: "promo-1", Name: "Old"}}
	lineRepo := &stubPromotionSKURepo{}
	uc := apppromotions.NewUpdatePromotionUseCase(newMockGormDBN(t, 2), promoRepo, lineRepo, &stubSKUExistenceChecker{}, &stubSKUBatchLister{}, &stubLineReverter{})

	promo, err := uc.Execute(context.Background(), apppromotions.UpdatePromotionInput{
		ID: "promo-1", Name: "New", Type: promotionsDomain.PromotionTypeFixedAmount, StartsAt: time.Now(),
		Status: promotionsDomain.PromotionStatusActive,
		Lines: []apppromotions.PromotionLineInput{
			{SKUID: "sku-2", DiscountType: promotionsDomain.PromotionTypeFixedAmount, DiscountValue: 200},
		},
	})

	require.NoError(t, err)
	require.Equal(t, "New", promo.Name)
	require.Equal(t, []string{"promo-1"}, lineRepo.deletedByPromotion)
	require.Len(t, lineRepo.created, 1)
	require.Equal(t, "sku-2", lineRepo.created[0].SKUID)
}

func TestUpdatePromotion_propagatesNotFound(t *testing.T) {
	promoRepo := &stubPromotionRepo{}
	uc := apppromotions.NewUpdatePromotionUseCase(newMockGormDBN(t, 2), promoRepo, &stubPromotionSKURepo{}, &stubSKUExistenceChecker{}, &stubSKUBatchLister{}, &stubLineReverter{})

	_, err := uc.Execute(context.Background(), apppromotions.UpdatePromotionInput{
		ID: "missing", Name: "New", Type: promotionsDomain.PromotionTypePercentage, StartsAt: time.Now(),
		Status: promotionsDomain.PromotionStatusActive,
		Lines:  []apppromotions.PromotionLineInput{{SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 500}},
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, promotionsDomain.ErrPromotionNotFound))
}

func TestUpdatePromotion_rejectsInvalidStatus(t *testing.T) {
	promoRepo := &stubPromotionRepo{findResult: &promotionsDomain.Promotion{ID: "promo-1"}}
	uc := apppromotions.NewUpdatePromotionUseCase(newMockGormDBN(t, 2), promoRepo, &stubPromotionSKURepo{}, &stubSKUExistenceChecker{}, &stubSKUBatchLister{}, &stubLineReverter{})

	_, err := uc.Execute(context.Background(), apppromotions.UpdatePromotionInput{
		ID: "promo-1", Name: "New", Type: promotionsDomain.PromotionTypePercentage, StartsAt: time.Now(),
		Status: "paused",
		Lines:  []apppromotions.PromotionLineInput{{SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 500}},
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, promotionsDomain.ErrInvalidPromotionStatus))
}
