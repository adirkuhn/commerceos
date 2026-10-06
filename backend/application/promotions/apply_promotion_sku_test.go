package promotions_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apppromotions "github.com/adirkuhn/commerceos/application/promotions"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
)

type stubSetPromotionSalePrice struct {
	gotSKUID       string
	gotSalePrice   int64
	applyPrevPrice *int64
	applyPrevEnd   *time.Time
	applyErr       error

	revertGotSKUID string
	revertGotPrice *int64
	revertGotEnd   *time.Time
	revertErr      error
}

func (s *stubSetPromotionSalePrice) Apply(_ context.Context, skuID string, salePriceMinorUnits int64) (*int64, *time.Time, error) {
	s.gotSKUID, s.gotSalePrice = skuID, salePriceMinorUnits
	if s.applyErr != nil {
		return nil, nil, s.applyErr
	}
	return s.applyPrevPrice, s.applyPrevEnd, nil
}

func (s *stubSetPromotionSalePrice) Revert(_ context.Context, skuID string, previousSalePriceMinorUnits *int64, previousSaleEndDate *time.Time) error {
	s.revertGotSKUID, s.revertGotPrice, s.revertGotEnd = skuID, previousSalePriceMinorUnits, previousSaleEndDate
	return s.revertErr
}

func TestApplyPromotionSKU_ComputesPercentageDiscountAndMarksApplied(t *testing.T) {
	skuLookup := &stubSKUExistenceChecker{findResult: &pmsDomain.SKU{ID: "sku-1", PriceMinorUnits: 1000}}
	setSalePrice := &stubSetPromotionSalePrice{applyPrevPrice: nil, applyPrevEnd: nil}
	lineRepo := &stubPromotionSKURepo{}
	uc := apppromotions.NewApplyPromotionSKUUseCase(newMockGormDBN(t, 2), skuLookup, setSalePrice, lineRepo, &eventrecordtest.Outbox{})

	line := promotionsDomain.PromotionSKU{ID: "line-1", SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 1000}
	err := uc.Execute(context.Background(), line)

	require.NoError(t, err)
	require.Equal(t, "sku-1", setSalePrice.gotSKUID)
	require.Equal(t, int64(900), setSalePrice.gotSalePrice) // 1000 - 10% = 900
	require.Len(t, lineRepo.markedApplied, 1)
	require.Equal(t, "line-1", lineRepo.markedApplied[0].lineID)
}

func TestApplyPromotionSKU_ComputesFixedAmountDiscount(t *testing.T) {
	skuLookup := &stubSKUExistenceChecker{findResult: &pmsDomain.SKU{ID: "sku-1", PriceMinorUnits: 1000}}
	setSalePrice := &stubSetPromotionSalePrice{}
	uc := apppromotions.NewApplyPromotionSKUUseCase(newMockGormDBN(t, 2), skuLookup, setSalePrice, &stubPromotionSKURepo{}, &eventrecordtest.Outbox{})

	line := promotionsDomain.PromotionSKU{ID: "line-1", SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypeFixedAmount, DiscountValue: 300}
	err := uc.Execute(context.Background(), line)

	require.NoError(t, err)
	require.Equal(t, int64(700), setSalePrice.gotSalePrice)
}

func TestApplyPromotionSKU_ClampsDiscountToAtLeastOneMinorUnit(t *testing.T) {
	skuLookup := &stubSKUExistenceChecker{findResult: &pmsDomain.SKU{ID: "sku-1", PriceMinorUnits: 100}}
	setSalePrice := &stubSetPromotionSalePrice{}
	uc := apppromotions.NewApplyPromotionSKUUseCase(newMockGormDBN(t, 2), skuLookup, setSalePrice, &stubPromotionSKURepo{}, &eventrecordtest.Outbox{})

	// Fixed amount discount larger than the price -- must never go to 0 or negative.
	line := promotionsDomain.PromotionSKU{ID: "line-1", SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypeFixedAmount, DiscountValue: 500}
	err := uc.Execute(context.Background(), line)

	require.NoError(t, err)
	require.Equal(t, int64(1), setSalePrice.gotSalePrice)
}

func TestApplyPromotionSKU_PropagatesSKULookupError(t *testing.T) {
	skuLookup := &stubSKUExistenceChecker{findErr: pmsDomain.ErrSKUNotFound}
	uc := apppromotions.NewApplyPromotionSKUUseCase(newMockGormDB(t), skuLookup, &stubSetPromotionSalePrice{}, &stubPromotionSKURepo{}, &eventrecordtest.Outbox{})

	line := promotionsDomain.PromotionSKU{ID: "line-1", SKUID: "missing", DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 1000}
	err := uc.Execute(context.Background(), line)

	require.Error(t, err)
}

func TestApplyPromotionSKU_PropagatesSetSalePriceError(t *testing.T) {
	skuLookup := &stubSKUExistenceChecker{findResult: &pmsDomain.SKU{ID: "sku-1", PriceMinorUnits: 1000}}
	setSalePrice := &stubSetPromotionSalePrice{applyErr: context.DeadlineExceeded}
	uc := apppromotions.NewApplyPromotionSKUUseCase(newMockGormDBN(t, 2), skuLookup, setSalePrice, &stubPromotionSKURepo{}, &eventrecordtest.Outbox{})

	line := promotionsDomain.PromotionSKU{ID: "line-1", SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 1000}
	err := uc.Execute(context.Background(), line)

	require.Error(t, err)
}

func TestRevertPromotionSKU_RestoresPreviousValuesAndMarksReverted(t *testing.T) {
	setSalePrice := &stubSetPromotionSalePrice{}
	lineRepo := &stubPromotionSKURepo{}
	uc := apppromotions.NewRevertPromotionSKUUseCase(newMockGormDB(t), setSalePrice, lineRepo, &eventrecordtest.Outbox{})

	prevPrice := int64(899)
	prevEnd := time.Now()
	line := promotionsDomain.PromotionSKU{ID: "line-1", SKUID: "sku-1", PreviousSalePriceMinorUnits: &prevPrice, PreviousSaleEndDate: &prevEnd}
	err := uc.Execute(context.Background(), line)

	require.NoError(t, err)
	require.Equal(t, "sku-1", setSalePrice.revertGotSKUID)
	require.Equal(t, prevPrice, *setSalePrice.revertGotPrice)
	require.Len(t, lineRepo.markedReverted, 1)
	require.Equal(t, "line-1", lineRepo.markedReverted[0])
}

func TestRevertPromotionSKU_PropagatesSetSalePriceError(t *testing.T) {
	setSalePrice := &stubSetPromotionSalePrice{revertErr: context.DeadlineExceeded}
	uc := apppromotions.NewRevertPromotionSKUUseCase(newMockGormDB(t), setSalePrice, &stubPromotionSKURepo{}, &eventrecordtest.Outbox{})

	line := promotionsDomain.PromotionSKU{ID: "line-1", SKUID: "sku-1"}
	err := uc.Execute(context.Background(), line)

	require.Error(t, err)
}
