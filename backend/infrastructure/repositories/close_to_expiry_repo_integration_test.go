//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// seedExpiryBatch creates a batch of skuID expiring on expiry, with qty
// units received (none when qty is 0), and returns its id.
func seedExpiryBatch(t *testing.T, db *gorm.DB, ctx context.Context, skuID, warehouseID, lot string, expiry time.Time, status string, qty int64) string {
	t.Helper()
	var id string
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		b := invDomain.Batch{TenantID: tenant.IDFromContext(ctx), SKUID: skuID, WarehouseID: warehouseID,
			LotNumber: lot, ExpiryDate: &expiry, Status: status, ReceivedAt: now, CreatedAt: now, UpdatedAt: now}
		if err := repositories.NewBatchRepo(db).Create(ctx, &b); err != nil {
			return err
		}
		id = b.ID
		if qty == 0 {
			return nil
		}
		return repositories.NewStockMovementRepo(db).Create(ctx, &invDomain.StockMovement{TenantID: tenant.IDFromContext(ctx), SKUID: skuID, WarehouseID: warehouseID,
			BatchID: &b.ID, Type: invDomain.MovementTypeReceipt, QuantityDelta: qty, OccurredAt: now, CreatedAt: now})
	}))
	return id
}

func acknowledge(t *testing.T, db *gorm.DB, ctx context.Context, batchID string, windowDays int) {
	t.Helper()
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		created, err := repositories.NewExpiryAcknowledgementRepo(db).Create(ctx, invDomain.ExpiryAcknowledgement{
			TenantID: tenant.IDFromContext(ctx), BatchID: batchID, WindowDays: windowDays, AcknowledgedBy: "u-1", AcknowledgedAt: time.Now(),
		})
		if err == nil && !created {
			t.Fatalf("acknowledgement for %s/%d not created", batchID, windowDays)
		}
		return err
	}))
}

func TestExpiryAcknowledgementRepo_Integration_CreateIsIdempotentAndDeleteReports(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	batchID := seedExpiryBatch(t, db, ctx, skuID, warehouseID, "A", time.Now().AddDate(0, 0, 40), invDomain.BatchStatusActive, 3)
	repo := repositories.NewExpiryAcknowledgementRepo(db)

	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		ack := invDomain.ExpiryAcknowledgement{TenantID: tenant.IDFromContext(ctx), BatchID: batchID, WindowDays: 90, AcknowledgedBy: "u-1", AcknowledgedAt: time.Now()}
		created, err := repo.Create(ctx, ack)
		require.NoError(t, err)
		assert.True(t, created)

		created, err = repo.Create(ctx, ack)
		require.NoError(t, err)
		assert.False(t, created, "second acknowledgement of the same window changes nothing")

		deleted, err := repo.Delete(ctx, batchID, 30)
		require.NoError(t, err)
		assert.False(t, deleted)

		deleted, err = repo.Delete(ctx, batchID, 90)
		require.NoError(t, err)
		assert.True(t, deleted)
		return nil
	}))
}

func TestExpiryAcknowledgementRepo_Integration_IsTenantIsolated(t *testing.T) {
	_, dsn := newIntegrationDBAndDSN(t)
	db := newAppRoleDB(t, dsn) // superusers bypass RLS
	_, ctx := newTestTenant()
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	batchID := seedExpiryBatch(t, db, ctx, skuID, warehouseID, "A", time.Now().AddDate(0, 0, 40), invDomain.BatchStatusActive, 3)
	acknowledge(t, db, ctx, batchID, 90)

	_, otherCtx := newTestTenant()
	require.NoError(t, tenant.WithTenant(otherCtx, db, func(ctx context.Context) error {
		deleted, err := repositories.NewExpiryAcknowledgementRepo(db).Delete(ctx, batchID, 90)
		require.NoError(t, err)
		assert.False(t, deleted, "another tenant cannot see the acknowledgement")
		return nil
	}))
}

func TestCloseToExpiryRepo_Integration_ListsBatchesSplitByAcknowledgement(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	today := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	inBand := today.AddDate(0, 0, 40)

	open := seedExpiryBatch(t, db, ctx, skuID, warehouseID, "OPEN", inBand, invDomain.BatchStatusActive, 3)
	seen := seedExpiryBatch(t, db, ctx, skuID, warehouseID, "SEEN", inBand.AddDate(0, 0, 1), invDomain.BatchStatusActive, 4)
	seenOtherWindow := seedExpiryBatch(t, db, ctx, skuID, warehouseID, "SEEN-30", inBand.AddDate(0, 0, 2), invDomain.BatchStatusActive, 5)
	seedExpiryBatch(t, db, ctx, skuID, warehouseID, "FAR", today.AddDate(0, 0, 120), invDomain.BatchStatusActive, 9)
	seedExpiryBatch(t, db, ctx, skuID, warehouseID, "FLAGGED", inBand, invDomain.BatchStatusExpired, 5)
	seedExpiryBatch(t, db, ctx, skuID, warehouseID, "EMPTY", inBand, invDomain.BatchStatusActive, 0)
	acknowledge(t, db, ctx, seen, 90)
	acknowledge(t, db, ctx, seenOtherWindow, 30)

	var clearanceID string
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		tx, _ := txctx.FromContext(ctx)
		if err := tx.Exec(`UPDATE skus SET price_minor_units = 499, sale_price_minor_units = 399, currency_code = 'EUR' WHERE id = ?`, skuID).Error; err != nil {
			return err
		}
		if err := tx.Raw(`INSERT INTO promotions (tenant_id, name, type, starts_at, status) VALUES (?, 'Clearance OPEN', 'fixed_price', now(), 'active') RETURNING id`,
			tenant.IDFromContext(ctx)).Scan(&clearanceID).Error; err != nil {
			return err
		}
		return tx.Exec(`INSERT INTO promotion_skus (tenant_id, promotion_id, sku_id, discount_type, discount_value, batch_id) VALUES (?, ?, ?, 'fixed_price', 299, ?)`,
			tenant.IDFromContext(ctx), clearanceID, skuID, open).Error
	}))

	reader := repositories.NewCloseToExpiryRepo(db)
	from, to := today.AddDate(0, 0, 31), today.AddDate(0, 0, 90)
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		openRows, err := reader.ListCloseToExpiryBatches(ctx, invDomain.CloseToExpiryQuery{From: from, To: to, WindowDays: 90, Limit: 50})
		require.NoError(t, err)
		require.Len(t, openRows, 2)
		assert.Equal(t, []string{open, seenOtherWindow}, []string{openRows[0].BatchID, openRows[1].BatchID})
		first := openRows[0]
		assert.Equal(t, "OPEN", first.LotNumber)
		assert.Equal(t, int64(3), first.Quantity)
		assert.Equal(t, warehouseID, first.WarehouseID)
		assert.NotEmpty(t, first.WarehouseName)
		assert.Equal(t, skuID, first.SKUID)
		assert.Equal(t, "Integration Test Product", first.ProductName)
		assert.NotEmpty(t, first.ProductID)
		assert.Equal(t, int64(499), first.PriceMinorUnits)
		require.NotNil(t, first.SalePriceMinorUnits)
		assert.Equal(t, int64(399), *first.SalePriceMinorUnits)
		assert.Nil(t, first.AcknowledgedAt)
		require.NotNil(t, first.Clearance)
		assert.Equal(t, clearanceID, first.Clearance.PromotionID)
		assert.Equal(t, "Clearance OPEN", first.Clearance.PromotionName)
		assert.Nil(t, openRows[1].Clearance)

		seenRows, err := reader.ListCloseToExpiryBatches(ctx, invDomain.CloseToExpiryQuery{From: from, To: to, WindowDays: 90, Acknowledged: true, Limit: 50})
		require.NoError(t, err)
		require.Len(t, seenRows, 1)
		assert.Equal(t, seen, seenRows[0].BatchID)
		assert.NotNil(t, seenRows[0].AcknowledgedAt)

		paged, err := reader.ListCloseToExpiryBatches(ctx, invDomain.CloseToExpiryQuery{From: from, To: to, WindowDays: 90, Limit: 1, Offset: 1})
		require.NoError(t, err)
		require.Len(t, paged, 1)
		assert.Equal(t, seenOtherWindow, paged[0].BatchID)

		counts, err := reader.CountCloseToExpiryBatches(ctx, from, to, 90)
		require.NoError(t, err)
		assert.Equal(t, invDomain.CloseToExpiryCounts{Open: 2, Acknowledged: 1}, counts)
		return nil
	}))
}

// The clearance dialog shows how much a sale still makes: each row carries
// the lot's own cost (else the SKU's average), the SKU's tax class (for the
// excl./incl. VAT pair) and whether the SKU is a weight pool (grams, never
// sold itself).
func TestCloseToExpiryRepo_Integration_ListsCostTaxClassAndWeightPool(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	today := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	inBand := today.AddDate(0, 0, 40)

	ownCost := seedExpiryBatch(t, db, ctx, skuID, warehouseID, "OWN-COST", inBand, invDomain.BatchStatusActive, 3)
	noCost := seedExpiryBatch(t, db, ctx, skuID, warehouseID, "NO-COST", inBand.AddDate(0, 0, 1), invDomain.BatchStatusActive, 4)
	var taxClassID string
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		tx, _ := txctx.FromContext(ctx)
		if err := tx.Raw(`INSERT INTO tax_classes (tenant_id, name) VALUES (?, 'Food') RETURNING id`, tenant.IDFromContext(ctx)).Scan(&taxClassID).Error; err != nil {
			return err
		}
		if err := tx.Exec(`UPDATE skus SET cost_price_minor_units = 250, tax_class_id = ? WHERE id = ?`, taxClassID, skuID).Error; err != nil {
			return err
		}
		return tx.Exec(`UPDATE batches SET landed_cost_minor_units = 210 WHERE id = ?`, ownCost).Error
	}))

	reader := repositories.NewCloseToExpiryRepo(db)
	from, to := today.AddDate(0, 0, 31), today.AddDate(0, 0, 90)
	var rows []invDomain.CloseToExpiryBatch
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		rows, err = reader.ListCloseToExpiryBatches(ctx, invDomain.CloseToExpiryQuery{From: from, To: to, WindowDays: 90, Limit: 50})
		return err
	}))
	require.Len(t, rows, 2)
	byID := map[string]invDomain.CloseToExpiryBatch{rows[0].BatchID: rows[0], rows[1].BatchID: rows[1]}

	require.NotNil(t, byID[ownCost].UnitCostMinorUnits)
	assert.Equal(t, int64(210), *byID[ownCost].UnitCostMinorUnits)
	assert.Equal(t, invDomain.UnitCostSourceBatch, byID[ownCost].UnitCostSource)
	require.NotNil(t, byID[noCost].UnitCostMinorUnits)
	assert.Equal(t, int64(250), *byID[noCost].UnitCostMinorUnits)
	assert.Equal(t, invDomain.UnitCostSourceSKU, byID[noCost].UnitCostSource)
	require.NotNil(t, byID[ownCost].TaxClassID)
	assert.Equal(t, taxClassID, *byID[ownCost].TaxClassID)
	assert.False(t, byID[ownCost].WeightPool)

	// No cost anywhere, and a kilo SKU.
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		tx, _ := txctx.FromContext(ctx)
		return tx.Exec(`UPDATE skus SET cost_price_minor_units = NULL, tax_class_id = NULL, unit_type = 'kilo' WHERE id = ?`, skuID).Error
	}))
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		rows, err = reader.ListCloseToExpiryBatches(ctx, invDomain.CloseToExpiryQuery{From: from, To: to, WindowDays: 90, Limit: 50})
		return err
	}))
	byID = map[string]invDomain.CloseToExpiryBatch{rows[0].BatchID: rows[0], rows[1].BatchID: rows[1]}
	assert.Nil(t, byID[noCost].UnitCostMinorUnits)
	assert.Equal(t, invDomain.UnitCostSourceNone, byID[noCost].UnitCostSource)
	assert.Nil(t, byID[noCost].TaxClassID)
	assert.True(t, byID[noCost].WeightPool)
	assert.Equal(t, invDomain.UnitCostSourceBatch, byID[ownCost].UnitCostSource, "the lot's own cost needs no SKU average")
}

func TestCloseToExpiryRepo_Integration_RequiresTransaction(t *testing.T) {
	db := newIntegrationDB(t)
	_, err := repositories.NewCloseToExpiryRepo(db).ListCloseToExpiryBatches(context.Background(), invDomain.CloseToExpiryQuery{})
	assert.Error(t, err)
}
