package promotions_test

import (
	"context"
	"errors"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/require"

	apppromotions "github.com/adirkuhn/commerceos/application/promotions"
	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
)

func TestDeletePromotion_deletesByID(t *testing.T) {
	promoRepo := &stubPromotionRepo{findResult: &promotionsDomain.Promotion{ID: "promo-1"}}
	uc := apppromotions.NewDeletePromotionUseCase(newMockGormDBN(t, 2), promoRepo, &stubPromotionSKURepo{}, &stubLineReverter{}, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "promo-1")

	require.NoError(t, err)
	require.Equal(t, []string{"promo-1"}, promoRepo.deleted)
}

func TestDeletePromotion_propagatesNotFound(t *testing.T) {
	promoRepo := &stubPromotionRepo{}
	uc := apppromotions.NewDeletePromotionUseCase(newMockGormDBN(t, 2), promoRepo, &stubPromotionSKURepo{}, &stubLineReverter{}, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "missing")

	require.Error(t, err)
	require.True(t, errors.Is(err, promotionsDomain.ErrPromotionNotFound))
	require.Empty(t, promoRepo.deleted)
}
