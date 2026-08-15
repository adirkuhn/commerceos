package inventory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appinv "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
)

func strPtr(s string) *string { return &s }

func TestGetSKUStockLevel_sumsUnlottedMovementsIntoTotalAndSyncable(t *testing.T) {
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-1", QuantityDelta: 10, Type: invDomain.MovementTypeReceipt},
		{SKUID: "sku-1", QuantityDelta: -4, Type: invDomain.MovementTypeSale},
	}}
	batchRepo := &stubBatchRepo{}
	uc := appinv.NewGetSKUStockLevelUseCase(newMockGormDB(t), movementRepo, batchRepo, &stubSettingsRepo{})

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
	uc := appinv.NewGetSKUStockLevelUseCase(newMockGormDB(t), movementRepo, batchRepo, &stubSettingsRepo{})

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
	uc := appinv.NewGetSKUStockLevelUseCase(newMockGormDB(t), movementRepo, batchRepo, &stubSettingsRepo{})

	level, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.Equal(t, int64(9), level.Syncable)
}

// ── Inventory Rules enabled: no-batches / no-expiry / nearest-expiry tiers ──

func rulesEnabledSettings() *stubSettingsRepo {
	return &stubSettingsRepo{result: &settingsDomain.TenantSettings{InventorySyncRulesEnabled: true}}
}

func TestGetSKUStockLevel_rulesEnabled_noBatches_syncsLedgerTotal(t *testing.T) {
	batchRepo := &stubBatchRepo{}
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-1", QuantityDelta: 20, Type: invDomain.MovementTypeReceipt},
		{SKUID: "sku-1", QuantityDelta: -3, Type: invDomain.MovementTypeSale},
	}}
	uc := appinv.NewGetSKUStockLevelUseCase(newMockGormDB(t), movementRepo, batchRepo, rulesEnabledSettings())

	level, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.Equal(t, int64(17), level.Total)
	require.Equal(t, int64(17), level.Syncable)
}

func TestGetSKUStockLevel_rulesEnabled_batchesWithNoExpiry_syncsSum(t *testing.T) {
	batchRepo := &stubBatchRepo{batches: []invDomain.Batch{
		{ID: "batch-a", SKUID: "sku-1", ExpiryDate: nil},
		{ID: "batch-b", SKUID: "sku-1", ExpiryDate: nil},
	}}
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-1", BatchID: strPtr("batch-a"), QuantityDelta: 5, Type: invDomain.MovementTypeReceipt},
		{SKUID: "sku-1", BatchID: strPtr("batch-b"), QuantityDelta: 8, Type: invDomain.MovementTypeReceipt},
	}}
	uc := appinv.NewGetSKUStockLevelUseCase(newMockGormDB(t), movementRepo, batchRepo, rulesEnabledSettings())

	level, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.Equal(t, int64(13), level.Syncable, "no batch has an expiry -- sums like today's default rule")
	require.Nil(t, level.SourceBatch, "sum-of-batches isn't attributable to one lot")
}

func TestGetSKUStockLevel_rulesEnabled_batchesWithExpiry_syncsOnlyNearestExpiryBatchQty(t *testing.T) {
	soon := time.Now().Add(2 * 24 * time.Hour)
	later := time.Now().Add(30 * 24 * time.Hour)
	batchRepo := &stubBatchRepo{batches: []invDomain.Batch{
		{ID: "batch-soon", SKUID: "sku-1", LotNumber: "LOT-SOON", ExpiryDate: &soon},
		{ID: "batch-later", SKUID: "sku-1", LotNumber: "LOT-LATER", ExpiryDate: &later},
	}}
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-1", BatchID: strPtr("batch-soon"), QuantityDelta: 4, Type: invDomain.MovementTypeReceipt},
		{SKUID: "sku-1", BatchID: strPtr("batch-later"), QuantityDelta: 100, Type: invDomain.MovementTypeReceipt},
	}}
	uc := appinv.NewGetSKUStockLevelUseCase(newMockGormDB(t), movementRepo, batchRepo, rulesEnabledSettings())

	level, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.Equal(t, int64(104), level.Total)
	require.Equal(t, int64(4), level.Syncable, "only the nearest-expiry batch's own quantity syncs, deliberately under-reporting the rest")
	require.NotNil(t, level.SourceBatch, "channel pushes need to know which lot the synced qty came from")
	require.Equal(t, "LOT-SOON", level.SourceBatch.LotNumber)
	require.True(t, soon.Equal(level.SourceBatch.ExpiryDate))
}

func TestGetSKUStockLevel_rulesEnabled_mixedBatchesWithAndWithoutExpiry_ignoresNoExpiryBatch(t *testing.T) {
	soon := time.Now().Add(5 * 24 * time.Hour)
	batchRepo := &stubBatchRepo{batches: []invDomain.Batch{
		{ID: "batch-expiry", SKUID: "sku-1", ExpiryDate: &soon},
		{ID: "batch-no-expiry", SKUID: "sku-1", ExpiryDate: nil},
	}}
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-1", BatchID: strPtr("batch-expiry"), QuantityDelta: 6, Type: invDomain.MovementTypeReceipt},
		{SKUID: "sku-1", BatchID: strPtr("batch-no-expiry"), QuantityDelta: 50, Type: invDomain.MovementTypeReceipt},
	}}
	uc := appinv.NewGetSKUStockLevelUseCase(newMockGormDB(t), movementRepo, batchRepo, rulesEnabledSettings())

	level, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.Equal(t, int64(6), level.Syncable, "any expiry batch present switches the whole SKU to nearest-expiry-only mode")
}

func TestGetSKUStockLevel_rulesEnabled_allExpiryBatchesAlreadyExpired_syncsZero(t *testing.T) {
	past := time.Now().Add(-24 * time.Hour)
	batchRepo := &stubBatchRepo{batches: []invDomain.Batch{{ID: "expired-batch", SKUID: "sku-1", ExpiryDate: &past}}}
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-1", BatchID: strPtr("expired-batch"), QuantityDelta: 10, Type: invDomain.MovementTypeReceipt},
	}}
	uc := appinv.NewGetSKUStockLevelUseCase(newMockGormDB(t), movementRepo, batchRepo, rulesEnabledSettings())

	level, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.Equal(t, int64(10), level.Total)
	require.Equal(t, int64(0), level.Syncable, "no not-yet-expired batch to report -- nothing safe to promise on channels")
}

func TestGetSKUStockLevel_rulesDisabled_keepsDefaultSumRuleEvenWithExpiryBatches(t *testing.T) {
	soon := time.Now().Add(2 * 24 * time.Hour)
	later := time.Now().Add(30 * 24 * time.Hour)
	batchRepo := &stubBatchRepo{batches: []invDomain.Batch{
		{ID: "batch-soon", SKUID: "sku-1", ExpiryDate: &soon},
		{ID: "batch-later", SKUID: "sku-1", ExpiryDate: &later},
	}}
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-1", BatchID: strPtr("batch-soon"), QuantityDelta: 4, Type: invDomain.MovementTypeReceipt},
		{SKUID: "sku-1", BatchID: strPtr("batch-later"), QuantityDelta: 100, Type: invDomain.MovementTypeReceipt},
	}}
	uc := appinv.NewGetSKUStockLevelUseCase(newMockGormDB(t), movementRepo, batchRepo, &stubSettingsRepo{})

	level, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.Equal(t, int64(104), level.Syncable, "flag off (default) -- unchanged legacy sum-of-non-expired-batches behavior")
}
