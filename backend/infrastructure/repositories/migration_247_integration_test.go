//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type legacyColumns struct {
	Source          string
	ExternalOrderID string
	ConnectionID    *string
	OrderKey        string
}

func readLegacyColumns(t *testing.T, ctx context.Context, salesOrderID string) legacyColumns {
	t.Helper()
	tx, _ := txctx.FromContext(ctx)
	var c legacyColumns
	require.NoError(t, tx.Raw(`SELECT source, external_order_id, connection_id, order_key FROM sales_orders WHERE id = ?`, salesOrderID).Scan(&c).Error)
	return c
}

// The previous release still reads sales_orders.source & co: an order this
// release writes (no such columns in its model) gets them from its origin.
func TestMigration247_Integration_anOrderWrittenNowStillFillsTheOldColumns(t *testing.T) {
	db := newIntegrationDB(t)
	tenantID, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	shop := seedWooConnection(t, db, ctx, "https://shop.example")
	origins := appintegrations.NewSalesOrderOrigins(repositories.NewSalesOrderOriginRepo(db))

	create := func(ctx context.Context, origin integrationsDomain.SalesOrderOriginInput) string {
		now := time.Now()
		o := omsDomain.SalesOrder{TenantID: tenantID, WarehouseID: warehouseID, CurrencyCode: "EUR", OccurredAt: now, CreatedAt: now, UpdatedAt: now}
		require.NoError(t, repositories.NewSalesOrderRepo(db).Create(ctx, &o))
		origin.TenantID, origin.SalesOrderID = tenantID, o.ID
		require.NoError(t, origins.Record(ctx, origin))
		return o.ID
	}
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		woo := create(ctx, integrationsDomain.SalesOrderOriginInput{
			Origin: integrationsDomain.OriginWooCommerce, ConnectionID: &shop, ExternalOrderID: "26217", ExternalKey: "wc_order_k",
		})
		manual := create(ctx, integrationsDomain.SalesOrderOriginInput{Origin: integrationsDomain.OriginManual, ExternalKey: "CO-1001"})

		got := readLegacyColumns(t, ctx, woo)
		assert.Equal(t, "woocommerce", got.Source)
		assert.Equal(t, "26217", got.ExternalOrderID)
		require.NotNil(t, got.ConnectionID)
		assert.Equal(t, shop, *got.ConnectionID)
		assert.Equal(t, "wc_order_k", got.OrderKey)

		got = readLegacyColumns(t, ctx, manual)
		assert.Equal(t, "manual", got.Source)
		assert.NotEmpty(t, got.ExternalOrderID, "a unique placeholder, as before")
		assert.Nil(t, got.ConnectionID)
		assert.Equal(t, "CO-1001", got.OrderKey)
		return nil
	}))
}

// Order #1001 from two Woo shops: both import (the old per-source unique
// key would have refused the second).
func TestMigration247_Integration_twoShopsMayHaveTheSameOrderNumber(t *testing.T) {
	db := newIntegrationDB(t)
	tenantID, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	shopA := seedWooConnection(t, db, ctx, "https://a.example")
	shopB := seedWooConnection(t, db, ctx, "https://b.example")
	origins := appintegrations.NewSalesOrderOrigins(repositories.NewSalesOrderOriginRepo(db))

	for _, shop := range []string{shopA, shopB} {
		require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			now := time.Now()
			o := omsDomain.SalesOrder{TenantID: tenantID, WarehouseID: warehouseID, CurrencyCode: "EUR", OccurredAt: now, CreatedAt: now, UpdatedAt: now}
			if err := repositories.NewSalesOrderRepo(db).Create(ctx, &o); err != nil {
				return err
			}
			return origins.Record(ctx, integrationsDomain.SalesOrderOriginInput{
				TenantID: tenantID, SalesOrderID: o.ID, Origin: integrationsDomain.OriginWooCommerce, ConnectionID: &shop, ExternalOrderID: "1001",
			})
		}), "shop %s", shop)
	}
}

// An order without a number showed its external id instead; the number
// takes it, so DisplayNumber reads the same.
func TestMigration247_Integration_anOrderWithoutANumberTakesItsExternalID(t *testing.T) {
	db, dsn := newIntegrationDBAndDSN(t)
	tenantID, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	sqlDB, err := goose.OpenDBWithDriver("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	require.NoError(t, goose.DownTo(sqlDB, migrationsDir, 246))
	t.Cleanup(func() { _ = goose.Up(sqlDB, migrationsDir) })

	insert := func(source, ext, number string) string {
		var id string
		require.NoError(t, db.Raw(`INSERT INTO sales_orders (tenant_id, source, external_order_id, number, warehouse_id, currency_code,
				occurred_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, 'EUR', now(), now(), now()) RETURNING id`,
			tenantID, source, ext, number, warehouseID).Scan(&id).Error)
		return id
	}
	pos := insert("pos", "pos-77", "")
	numbered := insert("bol", "B-1", "B-1-own")
	manual := insert("manual", "random-uuid", "")

	require.NoError(t, goose.UpTo(sqlDB, migrationsDir, 247))

	number := func(id string) string {
		var n string
		require.NoError(t, db.Raw(`SELECT number FROM sales_orders WHERE id = ?`, id).Scan(&n).Error)
		return n
	}
	assert.Equal(t, "pos-77", number(pos))
	assert.Equal(t, "B-1-own", number(numbered))
	assert.Equal(t, "", number(manual), "an order made here never showed its placeholder id")
}
