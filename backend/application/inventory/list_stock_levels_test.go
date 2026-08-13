package inventory_test

import (
	"context"
	"testing"
	"time"

	appinv "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListStockLevels_computesTotalAndSyncablePerSKU(t *testing.T) {
	movementRepo := &stubStockMovementRepo{
		distinctSKUIDs: []string{"sku-1", "sku-2"}, distinctSKUIDsTotal: 2,
		bySKUs: []invDomain.StockMovement{
			{SKUID: "sku-1", QuantityDelta: 10},
			{SKUID: "sku-1", QuantityDelta: -3},
			{SKUID: "sku-2", QuantityDelta: 5},
		},
	}
	batchRepo := &stubBatchRepo{}
	uc := appinv.NewListStockLevelsUseCase(newMockGormDB(t), movementRepo, batchRepo)

	result, err := uc.Execute(context.Background(), appinv.ListStockLevelsInput{})

	require.NoError(t, err)
	assert.Equal(t, int64(2), result.Total)
	require.Len(t, result.Levels, 2)
	assert.Equal(t, appinv.StockLevel{SKUID: "sku-1", Total: 7, Syncable: 7}, result.Levels[0])
	assert.Equal(t, appinv.StockLevel{SKUID: "sku-2", Total: 5, Syncable: 5}, result.Levels[1])
}

func TestListStockLevels_excludesExpiredBatchesFromSyncable(t *testing.T) {
	expired := time.Now().Add(-24 * time.Hour)
	movementRepo := &stubStockMovementRepo{
		distinctSKUIDs: []string{"sku-1"}, distinctSKUIDsTotal: 1,
		bySKUs: []invDomain.StockMovement{
			{SKUID: "sku-1", BatchID: strPtr("batch-1"), QuantityDelta: 10},
		},
	}
	batchRepo := &stubBatchRepo{batches: []invDomain.Batch{{ID: "batch-1", SKUID: "sku-1", ExpiryDate: &expired}}}
	uc := appinv.NewListStockLevelsUseCase(newMockGormDB(t), movementRepo, batchRepo)

	result, err := uc.Execute(context.Background(), appinv.ListStockLevelsInput{})

	require.NoError(t, err)
	require.Len(t, result.Levels, 1)
	assert.Equal(t, int64(10), result.Levels[0].Total)
	assert.Equal(t, int64(0), result.Levels[0].Syncable, "expired batch stock isn't syncable")
}

func TestListStockLevels_passesChangedSinceAndPagingThrough(t *testing.T) {
	since := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	movementRepo := &stubStockMovementRepo{}
	uc := appinv.NewListStockLevelsUseCase(newMockGormDB(t), movementRepo, &stubBatchRepo{})

	_, err := uc.Execute(context.Background(), appinv.ListStockLevelsInput{ChangedSince: &since, Page: 2, PerPage: 10})

	require.NoError(t, err)
	require.NotNil(t, movementRepo.gotChangedSince)
	assert.True(t, since.Equal(*movementRepo.gotChangedSince))
	assert.Equal(t, 10, movementRepo.gotLimit)
	assert.Equal(t, 10, movementRepo.gotOffset, "page 2 * perPage 10 -> offset 10")
}

func TestListStockLevels_defaultsPerPage(t *testing.T) {
	movementRepo := &stubStockMovementRepo{}
	uc := appinv.NewListStockLevelsUseCase(newMockGormDB(t), movementRepo, &stubBatchRepo{})

	_, err := uc.Execute(context.Background(), appinv.ListStockLevelsInput{})

	require.NoError(t, err)
	assert.Equal(t, 50, movementRepo.gotLimit)
}

func TestListStockLevels_capsPerPageAtMax(t *testing.T) {
	movementRepo := &stubStockMovementRepo{}
	uc := appinv.NewListStockLevelsUseCase(newMockGormDB(t), movementRepo, &stubBatchRepo{})

	_, err := uc.Execute(context.Background(), appinv.ListStockLevelsInput{PerPage: 10000})

	require.NoError(t, err)
	assert.Equal(t, 500, movementRepo.gotLimit)
}

func TestListStockLevels_noMatchingSKUsReturnsEmptyWithoutExtraQueries(t *testing.T) {
	movementRepo := &stubStockMovementRepo{distinctSKUIDs: nil, distinctSKUIDsTotal: 0}
	uc := appinv.NewListStockLevelsUseCase(newMockGormDB(t), movementRepo, &stubBatchRepo{})

	result, err := uc.Execute(context.Background(), appinv.ListStockLevelsInput{})

	require.NoError(t, err)
	assert.Equal(t, int64(0), result.Total)
	assert.Empty(t, result.Levels)
}

func TestListStockLevels_propagatesARepoError(t *testing.T) {
	movementRepo := &stubStockMovementRepo{distinctSKUIDsErr: invDomain.ErrInsufficientStock}
	uc := appinv.NewListStockLevelsUseCase(newMockGormDB(t), movementRepo, &stubBatchRepo{})

	_, err := uc.Execute(context.Background(), appinv.ListStockLevelsInput{})

	require.Error(t, err)
}
