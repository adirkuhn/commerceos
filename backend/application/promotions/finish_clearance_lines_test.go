package promotions_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apppromotions "github.com/adirkuhn/commerceos/application/promotions"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
)

type stubSKUBatchLister struct {
	batches []invDomain.Batch
	err     error
}

func (s *stubSKUBatchLister) Execute(_ context.Context, _ string) ([]invDomain.Batch, error) {
	return s.batches, s.err
}

type stubClearanceLineRepo struct {
	openBySKU  []promotionsDomain.PromotionSKU
	finished   []string
	finishErr  error
	unfinished int64
}

func (s *stubClearanceLineRepo) ListOpen(context.Context) ([]promotionsDomain.PromotionSKU, error) {
	return s.openBySKU, nil
}
func (s *stubClearanceLineRepo) ListOpenBySKU(context.Context, string) ([]promotionsDomain.PromotionSKU, error) {
	return s.openBySKU, nil
}
func (s *stubClearanceLineRepo) MarkFinished(_ context.Context, lineID string, _ time.Time) error {
	if s.finishErr != nil {
		return s.finishErr
	}
	s.finished = append(s.finished, lineID)
	return nil
}
func (s *stubClearanceLineRepo) CountUnfinished(context.Context, string) (int64, error) {
	return s.unfinished, nil
}

type stubLineReverter struct {
	reverted []string
	err      error
}

func (s *stubLineReverter) Execute(_ context.Context, line promotionsDomain.PromotionSKU) error {
	s.reverted = append(s.reverted, line.ID)
	return s.err
}

func clearanceLine(applied bool) promotionsDomain.PromotionSKU {
	batchID := "batch-1"
	return promotionsDomain.PromotionSKU{ID: "line-1", PromotionID: "promo-1", SKUID: "sku-1", BatchID: &batchID, Applied: applied}
}

func batchWith(qty int64, expiry time.Time) invDomain.Batch {
	return invDomain.Batch{ID: "batch-1", SKUID: "sku-1", Quantity: qty, ExpiryDate: &expiry, Status: invDomain.BatchStatusActive}
}

func TestFinishClearanceLine_BatchStillSellable_DoesNothing(t *testing.T) {
	lister := &stubSKUBatchLister{batches: []invDomain.Batch{batchWith(4, time.Now().Add(48*time.Hour))}}
	lines := &stubClearanceLineRepo{}
	reverter := &stubLineReverter{}
	uc := apppromotions.NewFinishClearanceLinesUseCase(newMockGormDBN(t, 0), lister, lines, &stubPromotionRepo{}, reverter)

	finished, err := uc.ExecuteForLine(context.Background(), clearanceLine(true))

	require.NoError(t, err)
	require.False(t, finished)
	require.Empty(t, lines.finished)
	require.Empty(t, reverter.reverted)
}

func TestFinishClearanceLine_FinishesAndRevertsWhenBatchIsDone(t *testing.T) {
	cases := map[string][]invDomain.Batch{
		"sold out or fully discarded": {batchWith(0, time.Now().Add(48*time.Hour))},
		"expired":                     {batchWith(4, time.Now().Add(-time.Minute))},
		"marked expired":              {{ID: "batch-1", Quantity: 4, Status: invDomain.BatchStatusExpired}},
		"removed":                     {{ID: "other-batch", Quantity: 9}},
	}
	for name, batches := range cases {
		t.Run(name, func(t *testing.T) {
			lines := &stubClearanceLineRepo{unfinished: 1}
			reverter := &stubLineReverter{}
			promoRepo := &stubPromotionRepo{}
			uc := apppromotions.NewFinishClearanceLinesUseCase(newMockGormDB(t), &stubSKUBatchLister{batches: batches}, lines, promoRepo, reverter)

			finished, err := uc.ExecuteForLine(context.Background(), clearanceLine(true))

			require.NoError(t, err)
			require.True(t, finished)
			require.Equal(t, []string{"line-1"}, lines.finished)
			require.Equal(t, []string{"line-1"}, reverter.reverted, "SKU goes back to its pre-promotion price")
			require.Empty(t, promoRepo.updated, "promotion still has another unfinished line")
		})
	}
}

func TestFinishClearanceLine_EndsPromotionWhenLastLineFinishes(t *testing.T) {
	lines := &stubClearanceLineRepo{unfinished: 0}
	promoRepo := &stubPromotionRepo{findResult: &promotionsDomain.Promotion{ID: "promo-1", Status: promotionsDomain.PromotionStatusActive}}
	uc := apppromotions.NewFinishClearanceLinesUseCase(newMockGormDB(t), &stubSKUBatchLister{}, lines, promoRepo, &stubLineReverter{})

	_, err := uc.ExecuteForLine(context.Background(), clearanceLine(true))

	require.NoError(t, err)
	require.Len(t, promoRepo.updated, 1)
	require.Equal(t, promotionsDomain.PromotionStatusEnded, promoRepo.updated[0].Status)
}

func TestFinishClearanceLine_NotYetAppliedLineIsNotReverted(t *testing.T) {
	reverter := &stubLineReverter{}
	uc := apppromotions.NewFinishClearanceLinesUseCase(newMockGormDB(t), &stubSKUBatchLister{}, &stubClearanceLineRepo{unfinished: 1}, &stubPromotionRepo{}, reverter)

	finished, err := uc.ExecuteForLine(context.Background(), clearanceLine(false))

	require.NoError(t, err)
	require.True(t, finished)
	require.Empty(t, reverter.reverted)
}

func TestFinishClearanceLine_PropagatesErrors(t *testing.T) {
	boom := errors.New("boom")

	t.Run("batch lookup", func(t *testing.T) {
		uc := apppromotions.NewFinishClearanceLinesUseCase(newMockGormDBN(t, 0), &stubSKUBatchLister{err: boom}, &stubClearanceLineRepo{}, &stubPromotionRepo{}, &stubLineReverter{})
		_, err := uc.ExecuteForLine(context.Background(), clearanceLine(true))
		require.ErrorIs(t, err, boom)
	})
	t.Run("mark finished", func(t *testing.T) {
		reverter := &stubLineReverter{}
		uc := apppromotions.NewFinishClearanceLinesUseCase(newMockGormDB(t), &stubSKUBatchLister{}, &stubClearanceLineRepo{finishErr: boom}, &stubPromotionRepo{}, reverter)
		_, err := uc.ExecuteForLine(context.Background(), clearanceLine(true))
		require.ErrorIs(t, err, boom)
		require.Empty(t, reverter.reverted, "nothing reverted when the finish wasn't recorded")
	})
	t.Run("revert", func(t *testing.T) {
		uc := apppromotions.NewFinishClearanceLinesUseCase(newMockGormDB(t), &stubSKUBatchLister{}, &stubClearanceLineRepo{unfinished: 1}, &stubPromotionRepo{}, &stubLineReverter{err: boom})
		_, err := uc.ExecuteForLine(context.Background(), clearanceLine(true))
		require.ErrorIs(t, err, boom)
	})
}

func TestFinishClearanceLinesForSKU_ChecksEveryOpenLine(t *testing.T) {
	lines := &stubClearanceLineRepo{openBySKU: []promotionsDomain.PromotionSKU{clearanceLine(true)}, unfinished: 1}
	reverter := &stubLineReverter{}
	uc := apppromotions.NewFinishClearanceLinesUseCase(newMockGormDBN(t, 2), &stubSKUBatchLister{}, lines, &stubPromotionRepo{}, reverter)

	err := uc.ExecuteForSKU(context.Background(), "sku-1")

	require.NoError(t, err)
	require.Equal(t, []string{"line-1"}, lines.finished)
	require.Equal(t, []string{"line-1"}, reverter.reverted)
}
