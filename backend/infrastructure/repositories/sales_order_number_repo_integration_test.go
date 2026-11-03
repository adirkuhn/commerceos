//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

	// Raw SQL: at version 234 the order still carried its source and
	// external id (the domain no longer does, COM-15).
	now := time.Now()
	orders := []struct {
		source, externalID, number string
		createdAt                  time.Time
	}{
		{"manual", "m-2", "", now},
		{"manual", "m-1", "", now.Add(-time.Hour)},
		{"whatsapp", "w-1", "", now.Add(time.Hour)},
		{"manual", "m-typed", "2026-17", now},
		{"woocommerce", "26217", "26217", now},
	}
	for _, o := range orders {
		require.NoError(t, db.Exec(`INSERT INTO sales_orders (tenant_id, source, external_order_id, number, warehouse_id,
				currency_code, occurred_at, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, 'EUR', ?, ?, ?)`, tenantID, o.source, o.externalID, o.number, warehouseID, now, o.createdAt, now).Error)
	}
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
