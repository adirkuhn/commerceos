package promotions_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	apppromotions "github.com/adirkuhn/commerceos/application/promotions"
	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
)

func TestListPromotions_returnsRepoResult(t *testing.T) {
	promoRepo := &stubPromotionRepo{list: []promotionsDomain.Promotion{{ID: "promo-1", Name: "Summer"}}}
	uc := apppromotions.NewListPromotionsUseCase(newMockGormDB(t), promoRepo)

	result, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, "Summer", result[0].Name)
}
