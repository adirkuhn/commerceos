package inventory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appinv "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
)

func TestListBatchMovements_returnsMovementsForBatch(t *testing.T) {
	movementRepo := &stubStockMovementRepo{byBatch: []invDomain.StockMovement{
		{ID: "movement-1", SKUID: "sku-1", BatchID: strPtr("batch-1"), Type: invDomain.MovementTypeReceipt, QuantityDelta: 10},
	}}
	uc := appinv.NewListBatchMovementsUseCase(newMockGormDB(t), movementRepo)

	movements, err := uc.Execute(context.Background(), "batch-1")

	require.NoError(t, err)
	require.Len(t, movements, 1)
	require.Equal(t, "movement-1", movements[0].ID)
}
