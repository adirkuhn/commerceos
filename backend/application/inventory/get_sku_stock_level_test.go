package inventory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appinv "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
)

func strPtr(s string) *string { return &s }

func TestGetSKUStockLevel_sumsUnlottedMovementsIntoTotalAndSyncable(t *testing.T) {
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-1", QuantityDelta: 10, Type: invDomain.MovementTypeReceipt},
		{SKUID: "sku-1", QuantityDelta: -4, Type: invDomain.MovementTypeSale},
	}}
	batchRepo := &stubBatchRepo{}
	uc := appinv.NewGetSKUStockLevelUseCase(newMockGormDB(t), movementRepo, batchRepo)

	level, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.Equal(t, int64(6), level.Total)
	require.Equal(t, int64(6), level.Syncable)
}

func TestGetSKUStockLevel_excludesExpiredBatchFromSyncableButNotTotal(t *testing.T) {
	past := time.Now().Add(-24 * time.Hour)
	future := time.Now().Add(24 * time.Hour)
	batchRepo := &stubBatchRepo{batches: []invDomain.Batch{
		{ID: "expired-batch", SKUID: "sku-1", ExpiryDate: &past},
		{ID: "fresh-batch", SKUID: "sku-1", ExpiryDate: &future},
	}}
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-1", BatchID: strPtr("expired-batch"), QuantityDelta: 5, Type: invDomain.MovementTypeReceipt},
		{SKUID: "sku-1", BatchID: strPtr("fresh-batch"), QuantityDelta: 7, Type: invDomain.MovementTypeReceipt},
	}}
	uc := appinv.NewGetSKUStockLevelUseCase(newMockGormDB(t), movementRepo, batchRepo)

	level, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.Equal(t, int64(12), level.Total, "total counts every movement regardless of expiry")
	require.Equal(t, int64(7), level.Syncable, "syncable excludes the expired batch's stock")
}

func TestGetSKUStockLevel_batchWithNoExpiryCountsAsSyncable(t *testing.T) {
	batchRepo := &stubBatchRepo{batches: []invDomain.Batch{{ID: "no-expiry-batch", SKUID: "sku-1", ExpiryDate: nil}}}
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-1", BatchID: strPtr("no-expiry-batch"), QuantityDelta: 9, Type: invDomain.MovementTypeReceipt},
	}}
	uc := appinv.NewGetSKUStockLevelUseCase(newMockGormDB(t), movementRepo, batchRepo)

	level, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.Equal(t, int64(9), level.Syncable)
}
