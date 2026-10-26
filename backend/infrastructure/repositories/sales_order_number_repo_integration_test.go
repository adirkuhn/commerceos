//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// Runs as commerceos_app so RLS applies, like production.
func TestSalesOrderNumberRepo_Integration_countsPerTenantFrom1001(t *testing.T) {
	_, dsn := newIntegrationDBAndDSN(t)
	db := newAppRoleDB(t, dsn)
	repo := repositories.NewSalesOrderNumberRepo(db)
	_, ctxA := newTestTenant()
	_, ctxB := newTestTenant()
	next := func(ctx context.Context) string {
		var n string
		require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			var err error
			n, err = repo.Next(ctx)
			return err
		}))
		return n
	}

	assert.Equal(t, "CO-1001", next(ctxA))
	assert.Equal(t, "CO-1002", next(ctxA))
	assert.Equal(t, "CO-1001", next(ctxB))
}

// Migration 235 numbers the orders made in CommerceOS so far, oldest first,
// and continues the counter after them; channel orders are left alone.
func TestMigration235_Integration_numbersOrdersMadeHere(t *testing.T) {
	db, dsn := newIntegrationDBAndDSN(t)
	tenantID, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	sqlDB, err := goose.OpenDBWithDriver("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	require.NoError(t, goose.DownTo(sqlDB, migrationsDir, 234))
	t.Cleanup(func() { _ = goose.Up(sqlDB, migrationsDir) })

	now := time.Now()
	orders := []omsDomain.SalesOrder{
		{Source: omsDomain.SalesOrderSourceManual, ExternalOrderID: "m-2", CreatedAt: now},
		{Source: omsDomain.SalesOrderSourceManual, ExternalOrderID: "m-1", CreatedAt: now.Add(-time.Hour)},
		{Source: omsDomain.SalesOrderSourceWhatsApp, ExternalOrderID: "w-1", CreatedAt: now.Add(time.Hour)},
		{Source: omsDomain.SalesOrderSourceManual, ExternalOrderID: "m-typed", Number: "2026-17", CreatedAt: now},
		{Source: omsDomain.SalesOrderSourceWooCommerce, ExternalOrderID: "26217", Number: "26217", CreatedAt: now},
	}
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		for i := range orders {
			o := &orders[i]
			o.TenantID, o.WarehouseID, o.CurrencyCode, o.OccurredAt, o.UpdatedAt = tenantID, warehouseID, "EUR", now, now
			if err := repositories.NewSalesOrderRepo(db).Create(ctx, o); err != nil {
				return err
			}
		}
		return nil
	}))
	require.NoError(t, goose.UpTo(sqlDB, migrationsDir, 235))

	numbers := map[string]string{}
	var rows []struct{ ExternalOrderID, Number string }
	require.NoError(t, db.Raw(`SELECT external_order_id, number FROM sales_orders WHERE tenant_id = ?`, tenantID).Scan(&rows).Error)
	for _, r := range rows {
		numbers[r.ExternalOrderID] = r.Number
	}
	assert.Equal(t, map[string]string{"m-1": "CO-1001", "m-2": "CO-1002", "w-1": "CO-1003", "m-typed": "2026-17", "26217": "26217"}, numbers)

	var next string
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		next, err = repositories.NewSalesOrderNumberRepo(db).Next(ctx)
		return err
	}))
	assert.Equal(t, "CO-1004", next)
}
