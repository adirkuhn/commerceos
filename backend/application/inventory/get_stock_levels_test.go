package inventory_test

import (
	"context"
	"testing"

	appinv "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetStockLevels_computesEachRequestedSKUInOnePass(t *testing.T) {
	movementRepo := &stubStockMovementRepo{bySKUs: []invDomain.StockMovement{
		{SKUID: "sku-1", QuantityDelta: 10}, {SKUID: "sku-1", QuantityDelta: -3}, {SKUID: "sku-2", QuantityDelta: 5},
	}}
	uc := appinv.NewGetStockLevelsUseCase(newMockGormDB(t), movementRepo, &stubBatchRepo{}, &stubSettingsRepo{})

	levels, err := uc.Execute(context.Background(), []string{"sku-1", "sku-2", "sku-3"})

	require.NoError(t, err)
	assert.Equal(t, appinv.StockLevel{SKUID: "sku-1", Total: 7, Syncable: 7}, levels["sku-1"])
	assert.Equal(t, int64(5), levels["sku-2"].Syncable)
	assert.Equal(t, appinv.StockLevel{SKUID: "sku-3"}, levels["sku-3"], "no history means no stock")
}

func TestGetStockLevels_nothingRequestedTouchesNothing(t *testing.T) {
	levels, err := appinv.NewGetStockLevelsUseCase(nil, &stubStockMovementRepo{}, &stubBatchRepo{}, &stubSettingsRepo{}).Execute(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, levels)
}
