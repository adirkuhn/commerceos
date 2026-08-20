package pms_test

import (
	"context"
	"testing"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListCostPriceHistory_returnsRepoResult(t *testing.T) {
	historyRepo := &stubSKUCostPriceHistoryRepo{listResult: []pmsDomain.SKUCostPriceHistory{
		{SKUID: "sku-1", CostPriceMinorUnits: 700},
	}}
	uc := apppms.NewListCostPriceHistoryUseCase(newMockGormDB(t), historyRepo)

	result, err := uc.Execute(context.Background(), "sku-1", 30)

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, int64(700), result[0].CostPriceMinorUnits)
}
