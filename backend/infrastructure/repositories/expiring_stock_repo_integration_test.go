//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

func TestExpiringStockRepo_Integration_ListExpiringStock(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	batchRepo := repositories.NewBatchRepo(db)
	movementRepo := repositories.NewStockMovementRepo(db)

	today := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	inRange := today.AddDate(0, 0, 20)
	outOfRange := today.AddDate(0, 0, 45)

	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		tx, _ := txctx.FromContext(ctx)
		if err := tx.Exec(`UPDATE skus SET price_minor_units = 499, sale_price_minor_units = 399, currency_code = 'EUR' WHERE id = ?`, skuID).Error; err != nil {
			return err
		}
		if err := tx.Exec(`INSERT INTO media (tenant_id, product_id, url, type, position)
			SELECT ?, v.product_id, url, 'image', pos FROM skus s JOIN variants v ON v.id = s.variant_id,
			(VALUES ('https://cdn.example/second.jpg', 1), ('https://cdn.example/first.jpg', 0)) AS m(url, pos)
			WHERE s.id = ?`, tenant.IDFromContext(ctx), skuID).Error; err != nil {
			return err
		}

		now := time.Now()
		batch := func(lot string, expiry time.Time, status string, qty int64) error {
			b := invDomain.Batch{TenantID: tenant.IDFromContext(ctx), SKUID: skuID, WarehouseID: warehouseID,
				LotNumber: lot, ExpiryDate: &expiry, Status: status, ReceivedAt: now, CreatedAt: now, UpdatedAt: now}
			if err := batchRepo.Create(ctx, &b); err != nil {
				return err
			}
			if qty == 0 {
				return nil
			}
			return movementRepo.Create(ctx, &invDomain.StockMovement{TenantID: tenant.IDFromContext(ctx), SKUID: skuID, WarehouseID: warehouseID,
				BatchID: &b.ID, Type: invDomain.MovementTypeReceipt, QuantityDelta: qty, OccurredAt: now, CreatedAt: now})
		}
		for _, err := range []error{
			batch("A", inRange, invDomain.BatchStatusActive, 3),
			batch("B", inRange, invDomain.BatchStatusActive, 4), // same SKU + date: summed with A
			batch("FAR", outOfRange, invDomain.BatchStatusActive, 9),
			batch("FLAGGED", inRange, invDomain.BatchStatusExpired, 5),
			batch("EMPTY", inRange, invDomain.BatchStatusActive, 0),
		} {
			if err != nil {
				return err
			}
		}
		return nil
	}))

	var lines []invDomain.ExpiringStockLine
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		lines, err = repositories.NewExpiringStockRepo(db).ListExpiringStock(ctx, today, today.AddDate(0, 0, 30))
		return err
	}))

	require.Len(t, lines, 1)
	l := lines[0]
	assert.Equal(t, skuID, l.SKUID)
	assert.Equal(t, "ITEST-SKU-1", l.SKUCode)
	assert.Equal(t, "Integration Test Product", l.ProductName)
	assert.Equal(t, "https://cdn.example/first.jpg", l.ImageURL)
	assert.Equal(t, inRange.Format("2006-01-02"), l.ExpiryDate.Format("2006-01-02"))
	assert.Equal(t, int64(7), l.Quantity)
	assert.Equal(t, int64(499), l.PriceMinorUnits)
	require.NotNil(t, l.SalePriceMinorUnits)
	assert.Equal(t, int64(399), *l.SalePriceMinorUnits)
	assert.Equal(t, "EUR", l.CurrencyCode)
}

func TestExpiringStockRepo_Integration_RequiresTransaction(t *testing.T) {
	db := newIntegrationDB(t)
	_, err := repositories.NewExpiringStockRepo(db).ListExpiringStock(context.Background(), time.Now(), time.Now())
	assert.Error(t, err)
}
