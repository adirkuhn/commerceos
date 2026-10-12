package promotions_test

import (
	"context"
	"errors"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
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
	uc := apppromotions.NewCreatePromotionUseCase(newMockGormDB(t), promoRepo, lineRepo, skuChecker, &stubSKUBatchLister{}, &eventrecordtest.Outbox{})

	result, err := uc.Execute(context.Background(), apppromotions.CreatePromotionInput{
		Name: "Summer Acai", Type: promotionsDomain.PromotionTypePercentage,
		StartsAt: time.Now(), Status: promotionsDomain.PromotionStatusActive,
		Lines: []apppromotions.PromotionLineInput{
			{SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 1000},
		},
	})

	require.NoError(t, err)
	require.Equal(t, "Summer Acai", result.Promotion.Name)
	require.Empty(t, result.Warnings)
	require.Len(t, lineRepo.created, 1)
	require.Equal(t, result.Promotion.ID, lineRepo.created[0].PromotionID)
	require.Equal(t, "sku-1", lineRepo.created[0].SKUID)
	require.Equal(t, int64(1000), lineRepo.created[0].DiscountValue)
}

func TestCreatePromotion_rejectsEmptyName(t *testing.T) {
	uc := apppromotions.NewCreatePromotionUseCase(newMockGormDB(t), &stubPromotionRepo{}, &stubPromotionSKURepo{}, &stubSKUExistenceChecker{}, &stubSKUBatchLister{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), apppromotions.CreatePromotionInput{
		Name: "   ", Type: promotionsDomain.PromotionTypePercentage, StartsAt: time.Now(),
		Lines: []apppromotions.PromotionLineInput{{SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 1000}},
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, promotionsDomain.ErrPromotionNameRequired))
}

func TestCreatePromotion_rejectsInvalidType(t *testing.T) {
	uc := apppromotions.NewCreatePromotionUseCase(newMockGormDB(t), &stubPromotionRepo{}, &stubPromotionSKURepo{}, &stubSKUExistenceChecker{}, &stubSKUBatchLister{}, &eventrecordtest.Outbox{})

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
	uc := apppromotions.NewCreatePromotionUseCase(newMockGormDB(t), &stubPromotionRepo{}, &stubPromotionSKURepo{}, &stubSKUExistenceChecker{}, &stubSKUBatchLister{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), apppromotions.CreatePromotionInput{
		Name: "Promo", Type: promotionsDomain.PromotionTypePercentage, StartsAt: starts, EndsAt: &ends,
		Lines: []apppromotions.PromotionLineInput{{SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 1000}},
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, promotionsDomain.ErrPromotionEndBeforeStart))
}

func TestCreatePromotion_rejectsNoLines(t *testing.T) {
	uc := apppromotions.NewCreatePromotionUseCase(newMockGormDB(t), &stubPromotionRepo{}, &stubPromotionSKURepo{}, &stubSKUExistenceChecker{}, &stubSKUBatchLister{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), apppromotions.CreatePromotionInput{
		Name: "Promo", Type: promotionsDomain.PromotionTypePercentage, StartsAt: time.Now(),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, promotionsDomain.ErrPromotionRequiresAtLeastOneSKU))
}

func TestCreatePromotion_rejectsDuplicateSKUInLines(t *testing.T) {
	uc := apppromotions.NewCreatePromotionUseCase(newMockGormDB(t), &stubPromotionRepo{}, &stubPromotionSKURepo{}, &stubSKUExistenceChecker{}, &stubSKUBatchLister{}, &eventrecordtest.Outbox{})

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
	uc := apppromotions.NewCreatePromotionUseCase(newMockGormDB(t), &stubPromotionRepo{}, &stubPromotionSKURepo{}, &stubSKUExistenceChecker{}, &stubSKUBatchLister{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), apppromotions.CreatePromotionInput{
		Name: "Promo", Type: promotionsDomain.PromotionTypePercentage, StartsAt: time.Now(),
		Lines: []apppromotions.PromotionLineInput{{SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 10001}},
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, promotionsDomain.ErrInvalidDiscountValue))
}

func TestCreatePromotion_rejectsNonPositiveFixedAmount(t *testing.T) {
	uc := apppromotions.NewCreatePromotionUseCase(newMockGormDB(t), &stubPromotionRepo{}, &stubPromotionSKURepo{}, &stubSKUExistenceChecker{}, &stubSKUBatchLister{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), apppromotions.CreatePromotionInput{
		Name: "Promo", Type: promotionsDomain.PromotionTypeFixedAmount, StartsAt: time.Now(),
		Lines: []apppromotions.PromotionLineInput{{SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypeFixedAmount, DiscountValue: 0}},
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, promotionsDomain.ErrInvalidDiscountValue))
}

func TestCreatePromotion_propagatesSKUNotFound(t *testing.T) {
	skuChecker := &stubSKUExistenceChecker{findErr: pmsDomain.ErrSKUNotFound}
	uc := apppromotions.NewCreatePromotionUseCase(newMockGormDB(t), &stubPromotionRepo{}, &stubPromotionSKURepo{}, skuChecker, &stubSKUBatchLister{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), apppromotions.CreatePromotionInput{
		Name: "Promo", Type: promotionsDomain.PromotionTypePercentage, StartsAt: time.Now(),
		Lines: []apppromotions.PromotionLineInput{{SKUID: "missing-sku", DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 1000}},
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrSKUNotFound))
}

func TestCreatePromotion_defaultsStatusToDraft(t *testing.T) {
	promoRepo := &stubPromotionRepo{}
	uc := apppromotions.NewCreatePromotionUseCase(newMockGormDB(t), promoRepo, &stubPromotionSKURepo{}, &stubSKUExistenceChecker{}, &stubSKUBatchLister{}, &eventrecordtest.Outbox{})

	result, err := uc.Execute(context.Background(), apppromotions.CreatePromotionInput{
		Name: "Promo", Type: promotionsDomain.PromotionTypePercentage, StartsAt: time.Now(),
		Lines: []apppromotions.PromotionLineInput{{SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 1000}},
	})

	require.NoError(t, err)
	require.Equal(t, promotionsDomain.PromotionStatusDraft, result.Promotion.Status)
}

func TestCreatePromotion_warnsWhenSKUAlreadyHasAManualSalePrice(t *testing.T) {
	existingSalePrice := int64(799)
	skuChecker := &stubSKUExistenceChecker{findResult: &pmsDomain.SKU{ID: "sku-1", SKUCode: "ACAI-1L", SalePriceMinorUnits: &existingSalePrice}}
	uc := apppromotions.NewCreatePromotionUseCase(newMockGormDB(t), &stubPromotionRepo{}, &stubPromotionSKURepo{}, skuChecker, &stubSKUBatchLister{}, &eventrecordtest.Outbox{})

	result, err := uc.Execute(context.Background(), apppromotions.CreatePromotionInput{
		Name: "Promo", Type: promotionsDomain.PromotionTypePercentage, StartsAt: time.Now(),
		Lines: []apppromotions.PromotionLineInput{{SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 1000}},
	})

	require.NoError(t, err)
	require.Len(t, result.Warnings, 1)
	require.Contains(t, result.Warnings[0], "ACAI-1L")
}

func TestCreatePromotion_noWarningWhenSKUHasNoManualSalePrice(t *testing.T) {
	skuChecker := &stubSKUExistenceChecker{findResult: &pmsDomain.SKU{ID: "sku-1", SKUCode: "ACAI-1L"}}
	uc := apppromotions.NewCreatePromotionUseCase(newMockGormDB(t), &stubPromotionRepo{}, &stubPromotionSKURepo{}, skuChecker, &stubSKUBatchLister{}, &eventrecordtest.Outbox{})

	result, err := uc.Execute(context.Background(), apppromotions.CreatePromotionInput{
		Name: "Promo", Type: promotionsDomain.PromotionTypePercentage, StartsAt: time.Now(),
		Lines: []apppromotions.PromotionLineInput{{SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 1000}},
	})

	require.NoError(t, err)
	require.Empty(t, result.Warnings)
}

func TestCreatePromotion_acceptsFixedPrice(t *testing.T) {
	lineRepo := &stubPromotionSKURepo{}
	uc := apppromotions.NewCreatePromotionUseCase(newMockGormDB(t), &stubPromotionRepo{}, lineRepo, &stubSKUExistenceChecker{}, &stubSKUBatchLister{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), apppromotions.CreatePromotionInput{
		Name: "Clearance", Type: promotionsDomain.PromotionTypeFixedPrice, StartsAt: time.Now(),
		Lines: []apppromotions.PromotionLineInput{{SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypeFixedPrice, DiscountValue: 499}},
	})

	require.NoError(t, err)
	require.Len(t, lineRepo.created, 1)
	require.Equal(t, promotionsDomain.PromotionTypeFixedPrice, lineRepo.created[0].DiscountType)
}

func TestCreatePromotion_rejectsNonPositiveFixedPrice(t *testing.T) {
	uc := apppromotions.NewCreatePromotionUseCase(newMockGormDB(t), &stubPromotionRepo{}, &stubPromotionSKURepo{}, &stubSKUExistenceChecker{}, &stubSKUBatchLister{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), apppromotions.CreatePromotionInput{
		Name: "Clearance", Type: promotionsDomain.PromotionTypeFixedPrice, StartsAt: time.Now(),
		Lines: []apppromotions.PromotionLineInput{{SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypeFixedPrice, DiscountValue: 0}},
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, promotionsDomain.ErrInvalidDiscountValue))
}
