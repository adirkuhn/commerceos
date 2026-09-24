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

// orderedReverter records, per revert, whether the promotion's lines or the
// promotion itself had already been removed -- the revert must come first,
// or the pre-promotion price snapshot on the line is lost for good.
type orderedReverter struct {
	lineRepo  *stubPromotionSKURepo
	promoRepo *stubPromotionRepo
	reverted  []string
	removedBy []bool
	err       error
}

func (r *orderedReverter) Execute(_ context.Context, line promotionsDomain.PromotionSKU) error {
	r.reverted = append(r.reverted, line.ID)
	r.removedBy = append(r.removedBy, len(r.lineRepo.deletedByPromotion) > 0 || len(r.promoRepo.deleted) > 0)
	return r.err
}

func liveAndIdleLines() []promotionsDomain.PromotionSKU {
	return []promotionsDomain.PromotionSKU{
		{ID: "live", SKUID: "sku-1", Applied: true},
		{ID: "idle", SKUID: "sku-2", Applied: false},
	}
}

func updateInput(lines ...apppromotions.PromotionLineInput) apppromotions.UpdatePromotionInput {
	return apppromotions.UpdatePromotionInput{
		ID: "promo-1", Name: "Edited", Type: promotionsDomain.PromotionTypePercentage, StartsAt: time.Now(),
		Status: promotionsDomain.PromotionStatusActive, Lines: lines,
	}
}

func pctLine(skuID string) apppromotions.PromotionLineInput {
	return apppromotions.PromotionLineInput{SKUID: skuID, DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 1500}
}

func TestUpdatePromotion_restoresLivePricesBeforeReplacingLines(t *testing.T) {
	promoRepo := &stubPromotionRepo{findResult: &promotionsDomain.Promotion{ID: "promo-1"}}
	lineRepo := &stubPromotionSKURepo{listResult: liveAndIdleLines()}
	reverter := &orderedReverter{lineRepo: lineRepo, promoRepo: promoRepo}
	uc := apppromotions.NewUpdatePromotionUseCase(newMockGormDBN(t, 2), promoRepo, lineRepo, &stubSKUExistenceChecker{}, &stubSKUBatchLister{}, reverter)

	_, err := uc.Execute(context.Background(), updateInput(pctLine("sku-1")))

	require.NoError(t, err)
	require.Equal(t, []string{"live"}, reverter.reverted, "only lines currently on the SKU price are reverted")
	require.Equal(t, []bool{false}, reverter.removedBy, "revert happens before the lines are deleted")
	require.Equal(t, []string{"promo-1"}, lineRepo.deletedByPromotion)
	require.Len(t, lineRepo.created, 1)
	require.False(t, lineRepo.created[0].Applied, "new line is re-applied by the sweep with a fresh snapshot")
}

func TestUpdatePromotion_revertFailureLeavesLinesUntouched(t *testing.T) {
	promoRepo := &stubPromotionRepo{findResult: &promotionsDomain.Promotion{ID: "promo-1"}}
	lineRepo := &stubPromotionSKURepo{listResult: liveAndIdleLines()}
	boom := errors.New("price update failed")
	reverter := &orderedReverter{lineRepo: lineRepo, promoRepo: promoRepo, err: boom}
	uc := apppromotions.NewUpdatePromotionUseCase(newMockGormDBN(t, 2), promoRepo, lineRepo, &stubSKUExistenceChecker{}, &stubSKUBatchLister{}, reverter)

	_, err := uc.Execute(context.Background(), updateInput(pctLine("sku-1")))

	require.ErrorIs(t, err, boom)
	require.Empty(t, lineRepo.deletedByPromotion)
	require.Empty(t, promoRepo.updated)
}

func TestUpdatePromotion_invalidLinesNeverTouchPrices(t *testing.T) {
	promoRepo := &stubPromotionRepo{findResult: &promotionsDomain.Promotion{ID: "promo-1"}}
	lineRepo := &stubPromotionSKURepo{listResult: liveAndIdleLines()}
	reverter := &orderedReverter{lineRepo: lineRepo, promoRepo: promoRepo}
	uc := apppromotions.NewUpdatePromotionUseCase(newMockGormDBN(t, 2), promoRepo, lineRepo, &stubSKUExistenceChecker{}, &stubSKUBatchLister{}, reverter)

	_, err := uc.Execute(context.Background(), updateInput(pctLine("sku-1"), pctLine("sku-1")))

	require.ErrorIs(t, err, promotionsDomain.ErrDuplicatePromotionSKU)
	require.Empty(t, reverter.reverted)
}

func TestDeletePromotion_restoresLivePricesBeforeDeleting(t *testing.T) {
	promoRepo := &stubPromotionRepo{findResult: &promotionsDomain.Promotion{ID: "promo-1"}}
	lineRepo := &stubPromotionSKURepo{listResult: liveAndIdleLines()}
	reverter := &orderedReverter{lineRepo: lineRepo, promoRepo: promoRepo}
	uc := apppromotions.NewDeletePromotionUseCase(newMockGormDBN(t, 2), promoRepo, lineRepo, reverter)

	err := uc.Execute(context.Background(), "promo-1")

	require.NoError(t, err)
	require.Equal(t, []string{"live"}, reverter.reverted)
	require.Equal(t, []bool{false}, reverter.removedBy)
	require.Equal(t, []string{"promo-1"}, promoRepo.deleted)
}

func TestDeletePromotion_revertFailureKeepsPromotion(t *testing.T) {
	promoRepo := &stubPromotionRepo{findResult: &promotionsDomain.Promotion{ID: "promo-1"}}
	lineRepo := &stubPromotionSKURepo{listResult: liveAndIdleLines()}
	boom := errors.New("price update failed")
	uc := apppromotions.NewDeletePromotionUseCase(newMockGormDBN(t, 2), promoRepo, lineRepo, &orderedReverter{lineRepo: lineRepo, promoRepo: promoRepo, err: boom})

	err := uc.Execute(context.Background(), "promo-1")

	require.ErrorIs(t, err, boom)
	require.Empty(t, promoRepo.deleted)
}
