package promotions_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apppromotions "github.com/adirkuhn/commerceos/application/promotions"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
)

func TestCreatePromotion_persistsPromotionAndLines(t *testing.T) {
	promoRepo := &stubPromotionRepo{}
	lineRepo := &stubPromotionSKURepo{}
	skuChecker := &stubSKUExistenceChecker{}
	uc := apppromotions.NewCreatePromotionUseCase(newMockGormDB(t), promoRepo, lineRepo, skuChecker)

	promo, err := uc.Execute(context.Background(), apppromotions.CreatePromotionInput{
		Name: "Summer Acai", Type: promotionsDomain.PromotionTypePercentage,
		StartsAt: time.Now(), Status: promotionsDomain.PromotionStatusActive,
		Lines: []apppromotions.PromotionLineInput{
			{SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 1000},
		},
	})

	require.NoError(t, err)
	require.Equal(t, "Summer Acai", promo.Name)
	require.Len(t, lineRepo.created, 1)
	require.Equal(t, promo.ID, lineRepo.created[0].PromotionID)
	require.Equal(t, "sku-1", lineRepo.created[0].SKUID)
	require.Equal(t, int64(1000), lineRepo.created[0].DiscountValue)
}

func TestCreatePromotion_rejectsEmptyName(t *testing.T) {
	uc := apppromotions.NewCreatePromotionUseCase(newMockGormDB(t), &stubPromotionRepo{}, &stubPromotionSKURepo{}, &stubSKUExistenceChecker{})

	_, err := uc.Execute(context.Background(), apppromotions.CreatePromotionInput{
		Name: "   ", Type: promotionsDomain.PromotionTypePercentage, StartsAt: time.Now(),
		Lines: []apppromotions.PromotionLineInput{{SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 1000}},
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, promotionsDomain.ErrPromotionNameRequired))
}

func TestCreatePromotion_rejectsInvalidType(t *testing.T) {
	uc := apppromotions.NewCreatePromotionUseCase(newMockGormDB(t), &stubPromotionRepo{}, &stubPromotionSKURepo{}, &stubSKUExistenceChecker{})

	_, err := uc.Execute(context.Background(), apppromotions.CreatePromotionInput{
		Name: "Promo", Type: "bogo", StartsAt: time.Now(),
		Lines: []apppromotions.PromotionLineInput{{SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 1000}},
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, promotionsDomain.ErrInvalidPromotionType))
}

func TestCreatePromotion_rejectsEndBeforeStart(t *testing.T) {
	starts := time.Now()
	ends := starts.Add(-time.Hour)
	uc := apppromotions.NewCreatePromotionUseCase(newMockGormDB(t), &stubPromotionRepo{}, &stubPromotionSKURepo{}, &stubSKUExistenceChecker{})

	_, err := uc.Execute(context.Background(), apppromotions.CreatePromotionInput{
		Name: "Promo", Type: promotionsDomain.PromotionTypePercentage, StartsAt: starts, EndsAt: &ends,
		Lines: []apppromotions.PromotionLineInput{{SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 1000}},
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, promotionsDomain.ErrPromotionEndBeforeStart))
}

func TestCreatePromotion_rejectsNoLines(t *testing.T) {
	uc := apppromotions.NewCreatePromotionUseCase(newMockGormDB(t), &stubPromotionRepo{}, &stubPromotionSKURepo{}, &stubSKUExistenceChecker{})

	_, err := uc.Execute(context.Background(), apppromotions.CreatePromotionInput{
		Name: "Promo", Type: promotionsDomain.PromotionTypePercentage, StartsAt: time.Now(),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, promotionsDomain.ErrPromotionRequiresAtLeastOneSKU))
}

func TestCreatePromotion_rejectsDuplicateSKUInLines(t *testing.T) {
	uc := apppromotions.NewCreatePromotionUseCase(newMockGormDB(t), &stubPromotionRepo{}, &stubPromotionSKURepo{}, &stubSKUExistenceChecker{})

	_, err := uc.Execute(context.Background(), apppromotions.CreatePromotionInput{
		Name: "Promo", Type: promotionsDomain.PromotionTypePercentage, StartsAt: time.Now(),
		Lines: []apppromotions.PromotionLineInput{
			{SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 1000},
			{SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 500},
		},
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, promotionsDomain.ErrDuplicatePromotionSKU))
}

func TestCreatePromotion_rejectsPercentageOutOfRange(t *testing.T) {
	uc := apppromotions.NewCreatePromotionUseCase(newMockGormDB(t), &stubPromotionRepo{}, &stubPromotionSKURepo{}, &stubSKUExistenceChecker{})

	_, err := uc.Execute(context.Background(), apppromotions.CreatePromotionInput{
		Name: "Promo", Type: promotionsDomain.PromotionTypePercentage, StartsAt: time.Now(),
		Lines: []apppromotions.PromotionLineInput{{SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 10001}},
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, promotionsDomain.ErrInvalidDiscountValue))
}

func TestCreatePromotion_rejectsNonPositiveFixedAmount(t *testing.T) {
	uc := apppromotions.NewCreatePromotionUseCase(newMockGormDB(t), &stubPromotionRepo{}, &stubPromotionSKURepo{}, &stubSKUExistenceChecker{})

	_, err := uc.Execute(context.Background(), apppromotions.CreatePromotionInput{
		Name: "Promo", Type: promotionsDomain.PromotionTypeFixedAmount, StartsAt: time.Now(),
		Lines: []apppromotions.PromotionLineInput{{SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypeFixedAmount, DiscountValue: 0}},
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, promotionsDomain.ErrInvalidDiscountValue))
}

func TestCreatePromotion_propagatesSKUNotFound(t *testing.T) {
	skuChecker := &stubSKUExistenceChecker{findErr: pmsDomain.ErrSKUNotFound}
	uc := apppromotions.NewCreatePromotionUseCase(newMockGormDB(t), &stubPromotionRepo{}, &stubPromotionSKURepo{}, skuChecker)

	_, err := uc.Execute(context.Background(), apppromotions.CreatePromotionInput{
		Name: "Promo", Type: promotionsDomain.PromotionTypePercentage, StartsAt: time.Now(),
		Lines: []apppromotions.PromotionLineInput{{SKUID: "missing-sku", DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 1000}},
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrSKUNotFound))
}

func TestCreatePromotion_defaultsStatusToDraft(t *testing.T) {
	promoRepo := &stubPromotionRepo{}
	uc := apppromotions.NewCreatePromotionUseCase(newMockGormDB(t), promoRepo, &stubPromotionSKURepo{}, &stubSKUExistenceChecker{})

	promo, err := uc.Execute(context.Background(), apppromotions.CreatePromotionInput{
		Name: "Promo", Type: promotionsDomain.PromotionTypePercentage, StartsAt: time.Now(),
		Lines: []apppromotions.PromotionLineInput{{SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 1000}},
	})

	require.NoError(t, err)
	require.Equal(t, promotionsDomain.PromotionStatusDraft, promo.Status)
}
