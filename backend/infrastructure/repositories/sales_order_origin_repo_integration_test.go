//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// insertLegacyOrder writes a sales order the way the previous release does,
// old columns included -- raw SQL so it keeps working once the domain drops
// those fields.
func insertLegacyOrder(t *testing.T, db *gorm.DB, ctx context.Context, warehouseID, source, externalID string, connectionID *string, orderKey string) string {
	t.Helper()
	var id string
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		tx, _ := txctx.FromContext(ctx)
		return tx.Raw(`INSERT INTO sales_orders (tenant_id, source, external_order_id, connection_id, order_key, warehouse_id,
				currency_code, occurred_at, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, 'EUR', now(), now(), now()) RETURNING id`,
			tenant.IDFromContext(ctx), source, externalID, connectionID, orderKey, warehouseID).Scan(&id).Error
	}))
	return id
}

func withOrigins(t *testing.T, db *gorm.DB, ctx context.Context, fn func(ctx context.Context, repo *repositories.SalesOrderOriginRepo)) {
	t.Helper()
	repo := repositories.NewSalesOrderOriginRepo(db)
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		fn(ctx, repo)
		return nil
	}))
}

// Runs as commerceos_app so RLS applies, like production.
func TestSalesOrderOriginRepo_Integration_savesAndFindsAnOrdersOrigin(t *testing.T) {
	_, dsn := newIntegrationDBAndDSN(t)
	db := newAppRoleDB(t, dsn)
	tenantID, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	shopA := seedWooConnection(t, db, ctx, "https://a.example")
	shopB := seedWooConnection(t, db, ctx, "https://b.example")
	orderA := insertLegacyOrder(t, db, ctx, warehouseID, "manual", "x-1", nil, "")
	orderB := insertLegacyOrder(t, db, ctx, warehouseID, "manual", "x-2", nil, "")
	pos := insertLegacyOrder(t, db, ctx, warehouseID, "manual", "x-3", nil, "")

	withOrigins(t, db, ctx, func(ctx context.Context, repo *repositories.SalesOrderOriginRepo) {
		save := func(order string, origin integrationsDomain.Origin, conn *string, ext, key string) {
			o, err := integrationsDomain.NewSalesOrderOrigin(integrationsDomain.SalesOrderOriginInput{
				TenantID: tenantID, SalesOrderID: order, Origin: origin, ConnectionID: conn, ExternalOrderID: ext, ExternalKey: key, CreatedAt: time.Now(),
			})
			require.NoError(t, err)
			require.NoError(t, repo.Save(ctx, &o))
			assert.NotEmpty(t, o.ID)
		}
		save(orderA, integrationsDomain.OriginWooCommerce, &shopA, "1001", "wc_order_a")
		save(orderB, integrationsDomain.OriginWooCommerce, &shopB, "1001", "")
		save(pos, integrationsDomain.OriginPOS, nil, "pos-42", "")

		got, err := repo.FindBySalesOrderID(ctx, orderA)
		require.NoError(t, err)
		assert.Equal(t, integrationsDomain.OriginWooCommerce, got.Origin)
		assert.Equal(t, shopA, *got.ConnectionID)
		assert.Equal(t, "1001", got.ExternalOrderID)
		assert.Equal(t, "wc_order_a", got.ExternalKey)

		got, err = repo.FindByConnectionExternalID(ctx, shopB, "1001")
		require.NoError(t, err, "the same order number in a second shop is its own order")
		assert.Equal(t, orderB, got.SalesOrderID)

		got, err = repo.FindByOriginExternalID(ctx, integrationsDomain.OriginPOS, "pos-42")
		require.NoError(t, err)
		assert.Equal(t, pos, got.SalesOrderID)
		assert.Nil(t, got.ConnectionID)
		assert.Empty(t, got.ExternalKey)

		_, err = repo.FindBySalesOrderID(ctx, "00000000-0000-0000-0000-000000000000")
		assert.ErrorIs(t, err, integrationsDomain.ErrSalesOrderOriginNotFound)
		_, err = repo.FindByConnectionExternalID(ctx, shopA, "9999")
		assert.ErrorIs(t, err, integrationsDomain.ErrSalesOrderOriginNotFound)
		_, err = repo.FindByOriginExternalID(ctx, integrationsDomain.OriginPOS, "1001")
		assert.ErrorIs(t, err, integrationsDomain.ErrSalesOrderOriginNotFound)
	})

	_, otherCtx := newTestTenant()
	withOrigins(t, db, otherCtx, func(ctx context.Context, repo *repositories.SalesOrderOriginRepo) {
		_, err := repo.FindBySalesOrderID(ctx, orderA)
		assert.ErrorIs(t, err, integrationsDomain.ErrSalesOrderOriginNotFound, "another tenant's order")
	})
}

// A second order claiming the same shop order is a lost import race.
func TestSalesOrderOriginRepo_Integration_theSameShopOrderTwiceIsAlreadyImported(t *testing.T) {
	db := newIntegrationDB(t)
	tenantID, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	shop := seedWooConnection(t, db, ctx, "https://shop.example")
	first := insertLegacyOrder(t, db, ctx, warehouseID, "manual", "x-1", nil, "")
	second := insertLegacyOrder(t, db, ctx, warehouseID, "manual", "x-2", nil, "")
	save := func(ctx context.Context, order string) error {
		o, err := integrationsDomain.NewSalesOrderOrigin(integrationsDomain.SalesOrderOriginInput{
			TenantID: tenantID, SalesOrderID: order, Origin: integrationsDomain.OriginWooCommerce, ConnectionID: &shop, ExternalOrderID: "1001",
		})
		require.NoError(t, err)
		return repositories.NewSalesOrderOriginRepo(db).Save(ctx, &o)
	}
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error { return save(ctx, first) }))

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error { return save(ctx, second) })

	assert.ErrorIs(t, err, integrationsDomain.ErrExternalOrderAlreadyImported)
}

func TestSalesOrderOriginRepo_Integration_savingAgainReplacesTheOrdersOneRow(t *testing.T) {
	db := newIntegrationDB(t)
	tenantID, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	order := insertLegacyOrder(t, db, ctx, warehouseID, "manual", "x-1", nil, "")

	withOrigins(t, db, ctx, func(ctx context.Context, repo *repositories.SalesOrderOriginRepo) {
		o, err := integrationsDomain.NewSalesOrderOrigin(integrationsDomain.SalesOrderOriginInput{
			TenantID: tenantID, SalesOrderID: order, Origin: integrationsDomain.OriginWhatsApp, CreatedAt: time.Now(),
		})
		require.NoError(t, err)
		require.NoError(t, repo.Save(ctx, &o))

		got, err := repo.FindBySalesOrderID(ctx, order)
		require.NoError(t, err)
		assert.Equal(t, integrationsDomain.OriginWhatsApp, got.Origin)
		assert.Empty(t, got.ExternalOrderID)

		var rows int64
		tx, _ := txctx.FromContext(ctx)
		require.NoError(t, tx.Raw(`SELECT count(*) FROM sales_order_origins WHERE sales_order_id = ?`, order).Scan(&rows).Error)
		assert.Equal(t, int64(1), rows)
	})
}

// The previous release still writes sales_orders.source & co: the trigger
// keeps the origin row in step, so a deploy window or a rollback loses
// nothing.
func TestSalesOrderOrigins_Integration_anOrderWrittenTheOldWayGetsItsOrigin(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	shop := seedWooConnection(t, db, ctx, "https://shop.example")

	woo := insertLegacyOrder(t, db, ctx, warehouseID, "woocommerce", "26217", &shop, "wc_order_k")
	manual := insertLegacyOrder(t, db, ctx, warehouseID, "manual", "4b3f0c1e-random-uuid", nil, "")

	withOrigins(t, db, ctx, func(ctx context.Context, repo *repositories.SalesOrderOriginRepo) {
		got, err := repo.FindBySalesOrderID(ctx, woo)
		require.NoError(t, err)
		assert.Equal(t, integrationsDomain.OriginWooCommerce, got.Origin)
		assert.Equal(t, shop, *got.ConnectionID)
		assert.Equal(t, "26217", got.ExternalOrderID)
		assert.Equal(t, "wc_order_k", got.ExternalKey)

		got, err = repo.FindBySalesOrderID(ctx, manual)
		require.NoError(t, err)
		assert.Equal(t, integrationsDomain.OriginManual, got.Origin)
		assert.Empty(t, got.ExternalOrderID, "an order made here has no external id")
	})
}

// Migration 246 gives every existing order its origin; shop orders whose
// connection was nulled take the tenant's only connection of that provider,
// the rest stay (origin set, no connection) and are reported.
func TestMigration246_Integration_backfillsAnOriginForEveryOrder(t *testing.T) {
	db, dsn := newIntegrationDBAndDSN(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	shop := seedWooConnection(t, db, ctx, "https://shop.example")
	seedBolConnection := func(account string) string {
		var id string
		require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			tx, _ := txctx.FromContext(ctx)
			return tx.Raw(`INSERT INTO integration_connections (tenant_id, provider, store_url, consumer_key, consumer_secret_encrypted,
					webhook_secret_encrypted, status, default_currency, created_at, updated_at)
				VALUES (?, 'bol', ?, 'ck', 'enc', '', 'connected', 'EUR', now(), now()) RETURNING id`, tenant.IDFromContext(ctx), account).Scan(&id).Error
		}))
		return id
	}
	seedBolConnection("bol-account-1")
	seedBolConnection("bol-account-2")

	sqlDB, err := goose.OpenDBWithDriver("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	require.NoError(t, goose.DownTo(sqlDB, migrationsDir, 245))
	t.Cleanup(func() { _ = goose.Up(sqlDB, migrationsDir) })

	wooLinked := insertLegacyOrder(t, db, ctx, warehouseID, "woocommerce", "1", &shop, "wc_1")
	wooNulled := insertLegacyOrder(t, db, ctx, warehouseID, "woocommerce", "2", nil, "")
	bolAmbiguous := insertLegacyOrder(t, db, ctx, warehouseID, "bol", "B-1", nil, "")
	pos := insertLegacyOrder(t, db, ctx, warehouseID, "pos", "pos-1", nil, "")
	manual := insertLegacyOrder(t, db, ctx, warehouseID, "manual", "random-uuid", nil, "")

	require.NoError(t, goose.UpTo(sqlDB, migrationsDir, 246))

	type row struct {
		SalesOrderID    string
		Origin          string
		ConnectionID    *string
		ExternalOrderID *string
		ExternalKey     *string
	}
	origins := map[string]row{}
	var rows []row
	require.NoError(t, db.Raw(`SELECT sales_order_id, origin, connection_id, external_order_id, external_key FROM sales_order_origins
		WHERE sales_order_id IN (?)`, []string{wooLinked, wooNulled, bolAmbiguous, pos, manual}).Scan(&rows).Error)
	for _, r := range rows {
		origins[r.SalesOrderID] = r
	}
	require.Len(t, origins, 5, "every order")

	assert.Equal(t, "woocommerce", origins[wooLinked].Origin)
	assert.Equal(t, shop, *origins[wooLinked].ConnectionID)
	assert.Equal(t, "1", *origins[wooLinked].ExternalOrderID)
	assert.Equal(t, "wc_1", *origins[wooLinked].ExternalKey)

	require.NotNil(t, origins[wooNulled].ConnectionID, "the tenant's only Woo connection")
	assert.Equal(t, shop, *origins[wooNulled].ConnectionID)
	assert.Nil(t, origins[wooNulled].ExternalKey)

	assert.Equal(t, "bol", origins[bolAmbiguous].Origin)
	assert.Nil(t, origins[bolAmbiguous].ConnectionID, "two Bol connections: not guessed")
	assert.Equal(t, "B-1", *origins[bolAmbiguous].ExternalOrderID)

	assert.Equal(t, "pos", origins[pos].Origin)
	assert.Nil(t, origins[pos].ConnectionID)
	assert.Equal(t, "pos-1", *origins[pos].ExternalOrderID)

	assert.Equal(t, "manual", origins[manual].Origin)
	assert.Nil(t, origins[manual].ExternalOrderID)
}
