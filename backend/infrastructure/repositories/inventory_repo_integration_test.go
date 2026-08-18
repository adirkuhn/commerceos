//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// seedProductSKUWarehouse creates the minimal chain of rows
// (product -> variant -> sku, plus a warehouse) a batch or stock movement
// needs to satisfy its foreign keys, using the real repos rather than raw
// SQL so a schema change that breaks these repos' own Create methods fails
// this test too.
func seedProductSKUWarehouse(t *testing.T, db *gorm.DB, ctx context.Context) (skuID, warehouseID string) {
	t.Helper()
	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	warehouseRepo := repositories.NewWarehouseRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		product := &pmsDomain.Product{
			TenantID: tenant.IDFromContext(ctx), Name: "Integration Test Product", Slug: "integration-test-product",
			Status: pmsDomain.ProductStatusActive, CreatedAt: now, UpdatedAt: now,
		}
		if err := productRepo.Create(ctx, product); err != nil {
			return err
		}
		variant := &pmsDomain.Variant{
			TenantID: tenant.IDFromContext(ctx), ProductID: product.ID, Name: "Default", CreatedAt: now, UpdatedAt: now,
		}
		if err := variantRepo.Create(ctx, variant); err != nil {
			return err
		}
		sku := &pmsDomain.SKU{
			TenantID: tenant.IDFromContext(ctx), VariantID: variant.ID, SKUCode: "ITEST-SKU-1",
			CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive, CreatedAt: now, UpdatedAt: now,
		}
		if err := skuRepo.Create(ctx, sku); err != nil {
			return err
		}
		skuID = sku.ID

		warehouse := &invDomain.Warehouse{
			TenantID: tenant.IDFromContext(ctx), Name: "Integration Test Warehouse", Active: true,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := warehouseRepo.Create(ctx, warehouse); err != nil {
			return err
		}
		warehouseID = warehouse.ID
		return nil
	})
	require.NoError(t, err)
	return skuID, warehouseID
}

func TestBatchRepo_Integration_CreateFindListUpdateDelete(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	batchRepo := repositories.NewBatchRepo(db)

	var created invDomain.Batch
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		created = invDomain.Batch{
			TenantID: tenant.IDFromContext(ctx), SKUID: skuID, WarehouseID: warehouseID,
			LotNumber: "LOT-1", ReceivedAt: now, CreatedAt: now, UpdatedAt: now,
		}
		return batchRepo.Create(ctx, &created)
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)

	var found *invDomain.Batch
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = batchRepo.FindByID(ctx, created.ID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, "LOT-1", found.LotNumber)

	var listed []invDomain.Batch
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		listed, err = batchRepo.ListBySKU(ctx, skuID)
		return err
	})
	require.NoError(t, err)
	require.Len(t, listed, 1)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		found.LotNumber = "LOT-2"
		found.UpdatedAt = time.Now()
		return batchRepo.Update(ctx, found)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = batchRepo.FindByID(ctx, created.ID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, "LOT-2", found.LotNumber)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return batchRepo.Delete(ctx, created.ID)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		_, err := batchRepo.FindByID(ctx, created.ID)
		return err
	})
	require.ErrorIs(t, err, invDomain.ErrBatchNotFound)
}

func TestBatchRepo_Integration_ListBySKUs(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	batchRepo := repositories.NewBatchRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		b := invDomain.Batch{
			TenantID: tenant.IDFromContext(ctx), SKUID: skuID, WarehouseID: warehouseID,
			LotNumber: "LOT-BULK-1", ReceivedAt: now, CreatedAt: now, UpdatedAt: now,
		}
		return batchRepo.Create(ctx, &b)
	})
	require.NoError(t, err)

	var found []invDomain.Batch
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = batchRepo.ListBySKUs(ctx, []string{skuID, "00000000-0000-0000-0000-000000000000"})
		return err
	})
	require.NoError(t, err)
	require.Len(t, found, 1)
	require.Equal(t, "LOT-BULK-1", found[0].LotNumber)
}

func TestBatchRepo_Integration_FindByIDNotFound(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	batchRepo := repositories.NewBatchRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		_, err := batchRepo.FindByID(ctx, "00000000-0000-0000-0000-000000000000")
		return err
	})
	require.ErrorIs(t, err, invDomain.ErrBatchNotFound)
}

func TestBatchRepo_Integration_RLSHidesOtherTenantsBatches(t *testing.T) {
	// Deliberately routed through commerceos_app, not the migration-owning
	// superuser: RLS policies are never enforced against a superuser (or
	// any BYPASSRLS role) regardless of what they say, so exercising this
	// through the superuser connection would pass even if RLS were
	// completely broken. commerceos_app is what backend/relay/scheduler
	// actually connect as in production (see migration
	// 020_app_role_and_force_rls.sql) -- this test only means something if
	// it goes through the same role.
	superDB, superDSN := newIntegrationDBAndDSN(t)
	db := newAppRoleDB(t, superDSN)
	_, ctxA := newTestTenant()
	skuID, warehouseID := seedProductSKUWarehouse(t, superDB, ctxA)
	batchRepo := repositories.NewBatchRepo(db)

	err := tenant.WithTenant(ctxA, db, func(ctx context.Context) error {
		now := time.Now()
		b := invDomain.Batch{
			TenantID: tenant.IDFromContext(ctx), SKUID: skuID, WarehouseID: warehouseID,
			ReceivedAt: now, CreatedAt: now, UpdatedAt: now,
		}
		return batchRepo.Create(ctx, &b)
	})
	require.NoError(t, err)

	// A second tenant querying the same SKU id (RLS is on tenant_id, not a
	// join condition) must see nothing — this is the entire point of RLS,
	// and the one thing sqlmock-based tests structurally cannot verify.
	_, ctxB := newTestTenant()
	var listedForB []invDomain.Batch
	err = tenant.WithTenant(ctxB, db, func(ctx context.Context) error {
		var err error
		listedForB, err = batchRepo.ListBySKU(ctx, skuID)
		return err
	})
	require.NoError(t, err)
	require.Empty(t, listedForB, "RLS must prevent tenant B from seeing tenant A's batch")
}

func TestStockMovementRepo_Integration_CreateAndList(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	batchRepo := repositories.NewBatchRepo(db)
	movementRepo := repositories.NewStockMovementRepo(db)

	var batchID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		b := invDomain.Batch{
			TenantID: tenant.IDFromContext(ctx), SKUID: skuID, WarehouseID: warehouseID,
			ReceivedAt: now, CreatedAt: now, UpdatedAt: now,
		}
		if err := batchRepo.Create(ctx, &b); err != nil {
			return err
		}
		batchID = b.ID
		return nil
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		lotted := invDomain.StockMovement{
			TenantID: tenant.IDFromContext(ctx), SKUID: skuID, WarehouseID: warehouseID, BatchID: &batchID,
			Type: invDomain.MovementTypeReceipt, QuantityDelta: 10, OccurredAt: time.Now(), CreatedAt: time.Now(),
		}
		if err := movementRepo.Create(ctx, &lotted); err != nil {
			return err
		}
		unlotted := invDomain.StockMovement{
			TenantID: tenant.IDFromContext(ctx), SKUID: skuID, WarehouseID: warehouseID,
			Type: invDomain.MovementTypeAdjustment, QuantityDelta: 5, OccurredAt: time.Now(), CreatedAt: time.Now(),
		}
		return movementRepo.Create(ctx, &unlotted)
	})
	require.NoError(t, err)

	var bySKU, byBatch []invDomain.StockMovement
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		bySKU, err = movementRepo.ListBySKU(ctx, skuID)
		if err != nil {
			return err
		}
		byBatch, err = movementRepo.ListByBatch(ctx, batchID)
		return err
	})
	require.NoError(t, err)
	require.Len(t, bySKU, 2, "both the lotted and un-lotted movement")
	require.Len(t, byBatch, 1, "only the movement referencing this batch")
	require.Equal(t, int64(10), byBatch[0].QuantityDelta)
}

// TestStockMovementRepo_Integration_ListDistinctSKUIDsAndBulkFetch proves
// the bulk queries application/inventory.ListStockLevelsUseCase relies on
// (incremental-sync support for external integrations like a POS desktop)
// against real Postgres: changed_since actually filters on created_at, and
// the bulk ListBySKUs/BatchRepo.ListBySKUs fetch the right rows.
func TestStockMovementRepo_Integration_ListDistinctSKUIDsAndBulkFetch(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	movementRepo := repositories.NewStockMovementRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		m := invDomain.StockMovement{
			TenantID: tenant.IDFromContext(ctx), SKUID: skuID, WarehouseID: warehouseID,
			Type: invDomain.MovementTypeReceipt, QuantityDelta: 10, OccurredAt: time.Now(), CreatedAt: time.Now(),
		}
		return movementRepo.Create(ctx, &m)
	})
	require.NoError(t, err)

	cutoffAfter := time.Now()

	var idsNoFilter, idsChangedAfter, idsNothingSince []string
	var totalNoFilter, totalChangedAfter, totalNothingSince int64
	var bulkMovements []invDomain.StockMovement
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		idsNoFilter, totalNoFilter, err = movementRepo.ListDistinctSKUIDs(ctx, nil, 50, 0)
		if err != nil {
			return err
		}
		before := time.Now().Add(-time.Hour)
		idsChangedAfter, totalChangedAfter, err = movementRepo.ListDistinctSKUIDs(ctx, &before, 50, 0)
		if err != nil {
			return err
		}
		idsNothingSince, totalNothingSince, err = movementRepo.ListDistinctSKUIDs(ctx, &cutoffAfter, 50, 0)
		if err != nil {
			return err
		}
		bulkMovements, err = movementRepo.ListBySKUs(ctx, []string{skuID})
		return err
	})
	require.NoError(t, err)

	require.Contains(t, idsNoFilter, skuID)
	require.Equal(t, int64(len(idsNoFilter)) >= 1, totalNoFilter >= 1)
	require.Contains(t, idsChangedAfter, skuID, "the movement happened after 'before', so this sku counts as changed")
	require.GreaterOrEqual(t, totalChangedAfter, int64(1))
	require.NotContains(t, idsNothingSince, skuID, "nothing changed after cutoffAfter -- the movement is older than that")
	require.Equal(t, int64(0), totalNothingSince)
	require.Len(t, bulkMovements, 1)
	require.Equal(t, skuID, bulkMovements[0].SKUID)
}

func TestStockMovementRepo_Integration_DBRejectsZeroDeltaAndBadType(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	movementRepo := repositories.NewStockMovementRepo(db)

	// The app layer's validateSign rejects zero deltas before this ever
	// reaches the repo -- this test proves the DB-level CHECK constraint
	// backs that up independently, in case a future caller bypasses the
	// use case.
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		m := invDomain.StockMovement{
			TenantID: tenant.IDFromContext(ctx), SKUID: skuID, WarehouseID: warehouseID,
			Type: invDomain.MovementTypeAdjustment, QuantityDelta: 0, OccurredAt: time.Now(), CreatedAt: time.Now(),
		}
		return movementRepo.Create(ctx, &m)
	})
	require.Error(t, err)
}

// TestStockMovementRepo_Integration_SplitTypesPassDBCheckConstraint is
// migration 045: the stock_movements_type_check constraint must accept
// split_out/split_in (SplitCaseIntoUnitsUseCase's pair), same as it already
// accepts every other MovementType.
func TestStockMovementRepo_Integration_SplitTypesPassDBCheckConstraint(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	caseSKUID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	movementRepo := repositories.NewStockMovementRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		splitOut := invDomain.StockMovement{
			TenantID: tenant.IDFromContext(ctx), SKUID: caseSKUID, WarehouseID: warehouseID,
			Type: invDomain.MovementTypeSplitOut, QuantityDelta: -2, OccurredAt: time.Now(), CreatedAt: time.Now(),
		}
		if err := movementRepo.Create(ctx, &splitOut); err != nil {
			return err
		}
		splitIn := invDomain.StockMovement{
			TenantID: tenant.IDFromContext(ctx), SKUID: caseSKUID, WarehouseID: warehouseID,
			Type: invDomain.MovementTypeSplitIn, QuantityDelta: 72, OccurredAt: time.Now(), CreatedAt: time.Now(),
		}
		return movementRepo.Create(ctx, &splitIn)
	})
	require.NoError(t, err)
}

func TestBatchRepo_Integration_ListExpiringSoon(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	batchRepo := repositories.NewBatchRepo(db)
	movementRepo := repositories.NewStockMovementRepo(db)

	var expiringBatchID, farFutureBatchID, emptyBatchID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		soon := now.Add(5 * 24 * time.Hour)
		farFuture := now.Add(365 * 24 * time.Hour)

		expiring := invDomain.Batch{
			TenantID: tenant.IDFromContext(ctx), SKUID: skuID, WarehouseID: warehouseID,
			LotNumber: "LOT-SOON", ExpiryDate: &soon, ReceivedAt: now, CreatedAt: now, UpdatedAt: now,
		}
		if err := batchRepo.Create(ctx, &expiring); err != nil {
			return err
		}
		expiringBatchID = expiring.ID

		notExpiring := invDomain.Batch{
			TenantID: tenant.IDFromContext(ctx), SKUID: skuID, WarehouseID: warehouseID,
			LotNumber: "LOT-FAR", ExpiryDate: &farFuture, ReceivedAt: now, CreatedAt: now, UpdatedAt: now,
		}
		if err := batchRepo.Create(ctx, &notExpiring); err != nil {
			return err
		}
		farFutureBatchID = notExpiring.ID

		// Expiring soon but with zero remaining stock -- must be excluded.
		soldOut := invDomain.Batch{
			TenantID: tenant.IDFromContext(ctx), SKUID: skuID, WarehouseID: warehouseID,
			LotNumber: "LOT-SOLD-OUT", ExpiryDate: &soon, ReceivedAt: now, CreatedAt: now, UpdatedAt: now,
		}
		if err := batchRepo.Create(ctx, &soldOut); err != nil {
			return err
		}
		emptyBatchID = soldOut.ID

		for _, m := range []invDomain.StockMovement{
			{TenantID: tenant.IDFromContext(ctx), SKUID: skuID, WarehouseID: warehouseID, BatchID: &expiringBatchID, Type: invDomain.MovementTypeReceipt, QuantityDelta: 10, OccurredAt: now, CreatedAt: now},
			{TenantID: tenant.IDFromContext(ctx), SKUID: skuID, WarehouseID: warehouseID, BatchID: &farFutureBatchID, Type: invDomain.MovementTypeReceipt, QuantityDelta: 10, OccurredAt: now, CreatedAt: now},
			{TenantID: tenant.IDFromContext(ctx), SKUID: skuID, WarehouseID: warehouseID, BatchID: &emptyBatchID, Type: invDomain.MovementTypeReceipt, QuantityDelta: 5, OccurredAt: now, CreatedAt: now},
			{TenantID: tenant.IDFromContext(ctx), SKUID: skuID, WarehouseID: warehouseID, BatchID: &emptyBatchID, Type: invDomain.MovementTypeSale, QuantityDelta: -5, OccurredAt: now, CreatedAt: now},
		} {
			if err := movementRepo.Create(ctx, &m); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err)

	var expiring []invDomain.ExpiringBatch
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		expiring, err = batchRepo.ListExpiringSoon(ctx, 30)
		return err
	})
	require.NoError(t, err)

	require.Len(t, expiring, 1, "only the expiring, still-in-stock batch")
	row := expiring[0]
	require.Equal(t, expiringBatchID, row.BatchID)
	require.Equal(t, skuID, row.SKUID, "SKUID must actually be populated -- GORM's naming strategy doesn't auto-map this column for raw-query scans")
	require.NotEmpty(t, row.SKUCode)
	require.NotEmpty(t, row.ProductName)
	require.NotEmpty(t, row.WarehouseName)
	require.Equal(t, "LOT-SOON", row.LotNumber)
	require.Equal(t, int64(10), row.Quantity)
}

// TestDiscardRecordRepo_Integration_CreateAndList proves 078_discard_records.sql's
// schema, FKs, and the report join (skus/variants/products/warehouses/batches)
// all actually work against real Postgres -- a sqlmock test can only assert
// the SQL string sent, not that the migration and join are correct.
func TestDiscardRecordRepo_Integration_CreateAndList(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	batchRepo := repositories.NewBatchRepo(db)
	movementRepo := repositories.NewStockMovementRepo(db)
	discardRepo := repositories.NewDiscardRecordRepo(db)

	var batchID, movementID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		batch := invDomain.Batch{
			TenantID: tenant.IDFromContext(ctx), SKUID: skuID, WarehouseID: warehouseID,
			LotNumber: "LOT-DISCARD", ReceivedAt: now, CreatedAt: now, UpdatedAt: now,
		}
		if err := batchRepo.Create(ctx, &batch); err != nil {
			return err
		}
		batchID = batch.ID

		movement := invDomain.StockMovement{
			TenantID: tenant.IDFromContext(ctx), SKUID: skuID, WarehouseID: warehouseID, BatchID: &batchID,
			Type: invDomain.MovementTypeLost, QuantityDelta: -4, Reason: "expired", OccurredAt: now, CreatedAt: now,
		}
		if err := movementRepo.Create(ctx, &movement); err != nil {
			return err
		}
		movementID = movement.ID
		return nil
	})
	require.NoError(t, err)

	var created invDomain.DiscardRecord
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		created = invDomain.DiscardRecord{
			TenantID: tenant.IDFromContext(ctx), SKUID: skuID, WarehouseID: warehouseID, BatchID: &batchID,
			MovementID: movementID, Type: invDomain.MovementTypeLost, Quantity: 4, Reason: "expired",
			DiscardedBy: "user-1", DiscardedAt: time.Now(), CreatedAt: time.Now(),
		}
		return discardRepo.Create(ctx, &created)
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)

	var report []invDomain.DiscardRecordReport
	var total int64
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		report, total, err = discardRepo.List(ctx, invDomain.DiscardRecordFilter{})
		return err
	})
	require.NoError(t, err)

	require.Equal(t, int64(1), total)
	require.Len(t, report, 1)
	row := report[0]
	require.Equal(t, created.ID, row.ID)
	require.Equal(t, skuID, row.SKUID)
	require.NotEmpty(t, row.SKUCode)
	require.NotEmpty(t, row.ProductName)
	require.NotEmpty(t, row.WarehouseName)
	require.Equal(t, "LOT-DISCARD", row.LotNumber)
	require.Equal(t, invDomain.MovementTypeLost, row.Type)
	require.Equal(t, int64(4), row.Quantity)
	require.Equal(t, "expired", row.Reason)
	require.Equal(t, "user-1", row.DiscardedBy)
	require.Nil(t, row.PhotoURL)

	// Filtering by an unrelated warehouse must exclude it.
	otherWarehouse := "00000000-0000-0000-0000-000000000000"
	var filtered []invDomain.DiscardRecordReport
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		filtered, _, err = discardRepo.List(ctx, invDomain.DiscardRecordFilter{WarehouseID: &otherWarehouse})
		return err
	})
	require.NoError(t, err)
	require.Empty(t, filtered)

	// Filtering by type: lost matches, damaged doesn't.
	lostType := invDomain.MovementTypeLost
	damagedType := invDomain.MovementTypeDamaged
	var byLost, byDamaged []invDomain.DiscardRecordReport
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		byLost, _, err = discardRepo.List(ctx, invDomain.DiscardRecordFilter{Type: &lostType})
		if err != nil {
			return err
		}
		byDamaged, _, err = discardRepo.List(ctx, invDomain.DiscardRecordFilter{Type: &damagedType})
		return err
	})
	require.NoError(t, err)
	require.Len(t, byLost, 1)
	require.Empty(t, byDamaged)
}
