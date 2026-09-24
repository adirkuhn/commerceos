package promotions_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apppromotions "github.com/adirkuhn/commerceos/application/promotions"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
)

func clearanceInput(batchID string) []apppromotions.PromotionLineInput {
	return []apppromotions.PromotionLineInput{
		{SKUID: "sku-1", BatchID: &batchID, DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 3000},
	}
}

func createInput(lines []apppromotions.PromotionLineInput) apppromotions.CreatePromotionInput {
	return apppromotions.CreatePromotionInput{
		Name: "Clearance", Type: promotionsDomain.PromotionTypePercentage,
		StartsAt: time.Now(), Status: promotionsDomain.PromotionStatusActive, Lines: lines,
	}
}

func TestCreatePromotion_persistsClearanceLineBatch(t *testing.T) {
	lineRepo := &stubPromotionSKURepo{}
	lister := &stubSKUBatchLister{batches: []invDomain.Batch{batchWith(5, time.Now().Add(72*time.Hour))}}
	uc := apppromotions.NewCreatePromotionUseCase(newMockGormDB(t), &stubPromotionRepo{}, lineRepo, &stubSKUExistenceChecker{}, lister)

	_, err := uc.Execute(context.Background(), createInput(clearanceInput("batch-1")))

	require.NoError(t, err)
	require.Len(t, lineRepo.created, 1)
	require.NotNil(t, lineRepo.created[0].BatchID)
	require.Equal(t, "batch-1", *lineRepo.created[0].BatchID)
}

func TestCreatePromotion_rejectsClearanceBatchNotOnSKU(t *testing.T) {
	lister := &stubSKUBatchLister{batches: []invDomain.Batch{batchWith(5, time.Now().Add(72*time.Hour))}}
	uc := apppromotions.NewCreatePromotionUseCase(newMockGormDBN(t, 0), &stubPromotionRepo{}, &stubPromotionSKURepo{}, &stubSKUExistenceChecker{}, lister)

	_, err := uc.Execute(context.Background(), createInput(clearanceInput("some-other-batch")))

	require.ErrorIs(t, err, promotionsDomain.ErrClearanceBatchNotFound)
}

func TestCreatePromotion_rejectsFinishedClearanceBatch(t *testing.T) {
	lister := &stubSKUBatchLister{batches: []invDomain.Batch{batchWith(0, time.Now().Add(72*time.Hour))}}
	uc := apppromotions.NewCreatePromotionUseCase(newMockGormDBN(t, 0), &stubPromotionRepo{}, &stubPromotionSKURepo{}, &stubSKUExistenceChecker{}, lister)

	_, err := uc.Execute(context.Background(), createInput(clearanceInput("batch-1")))

	require.ErrorIs(t, err, promotionsDomain.ErrClearanceBatchFinished)
}

// Editing an already-running clearance promotion (e.g. renaming it after its
// batch sold out) must not be blocked by re-validating a batch it already had.
func TestUpdatePromotion_keepsExistingClearanceBatchWithoutRevalidating(t *testing.T) {
	batchID := "batch-1"
	promoRepo := &stubPromotionRepo{findResult: &promotionsDomain.Promotion{ID: "promo-1"}}
	lineRepo := &stubPromotionSKURepo{listResult: []promotionsDomain.PromotionSKU{{ID: "line-1", SKUID: "sku-1", BatchID: &batchID}}}
	lister := &stubSKUBatchLister{} // batch is gone -- would fail validation if re-checked
	uc := apppromotions.NewUpdatePromotionUseCase(newMockGormDBN(t, 2), promoRepo, lineRepo, &stubSKUExistenceChecker{}, lister, &stubLineReverter{})

	_, err := uc.Execute(context.Background(), apppromotions.UpdatePromotionInput{
		ID: "promo-1", Name: "Renamed", Type: promotionsDomain.PromotionTypePercentage, StartsAt: time.Now(),
		Status: promotionsDomain.PromotionStatusActive, Lines: clearanceInput(batchID),
	})

	require.NoError(t, err)
	require.Len(t, lineRepo.created, 1)
	require.Equal(t, batchID, *lineRepo.created[0].BatchID)
}

func TestUpdatePromotion_validatesNewlyAddedClearanceBatch(t *testing.T) {
	promoRepo := &stubPromotionRepo{findResult: &promotionsDomain.Promotion{ID: "promo-1"}}
	uc := apppromotions.NewUpdatePromotionUseCase(newMockGormDBN(t, 1), promoRepo, &stubPromotionSKURepo{}, &stubSKUExistenceChecker{}, &stubSKUBatchLister{}, &stubLineReverter{})

	_, err := uc.Execute(context.Background(), apppromotions.UpdatePromotionInput{
		ID: "promo-1", Name: "Clearance", Type: promotionsDomain.PromotionTypePercentage, StartsAt: time.Now(),
		Status: promotionsDomain.PromotionStatusActive, Lines: clearanceInput("batch-1"),
	})

	require.ErrorIs(t, err, promotionsDomain.ErrClearanceBatchNotFound)
}
