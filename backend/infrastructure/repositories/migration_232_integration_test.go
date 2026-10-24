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

// Migration 232 marks which line totals contain their VAT and computes the
// VAT that items added while packing never had.
func TestMigration232_Integration_marksTaxInclusiveLinesAndTaxesAddedItems(t *testing.T) {
	db, dsn := newIntegrationDBAndDSN(t)
	tenantID, ctx := newTestTenant()
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	_, posLineID := seedSalesOrderWithLineItemForPackingDB(t, db, ctx, skuID, warehouseID)

	require.NoError(t, db.Exec(`INSERT INTO tenant_settings (tenant_id, prices_include_tax, default_tax_country) VALUES (?, true, 'NL')`, tenantID).Error)
	require.NoError(t, db.Exec(`INSERT INTO tax_classes (tenant_id, name, is_default) VALUES (?, 'Reduced', true)`, tenantID).Error)
	require.NoError(t, db.Exec(`INSERT INTO tax_rates (tenant_id, tax_class_id, country, rate_basis_points)
		SELECT tenant_id, id, 'nl', 900 FROM tax_classes WHERE tenant_id = ?`, tenantID).Error)

	var wooOrderID, wooLineID, addedLineID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		order := omsDomain.SalesOrder{TenantID: tenantID, Source: omsDomain.SalesOrderSourceWooCommerce, ExternalOrderID: "woo-26217",
			WarehouseID: warehouseID, CurrencyCode: "EUR", Status: omsDomain.SalesOrderStatusProcessing,
			TaxTotalMinorUnits: 124, TotalMinorUnits: 1895, OccurredAt: now, CreatedAt: now, UpdatedAt: now}
		if err := repositories.NewSalesOrderRepo(db).Create(ctx, &order); err != nil {
			return err
		}
		wooOrderID = order.ID
		lines := []*omsDomain.SalesOrderLineItem{
			{TenantID: tenantID, SalesOrderID: order.ID, SKUID: &skuID, Name: "Caldo de Cana", Quantity: 3,
				PriceMinorUnits: 458, SubtotalMinorUnits: 1373, TotalMinorUnits: 1373, SubtotalTaxMinorUnits: 124, TotalTaxMinorUnits: 124, CreatedAt: now},
			{TenantID: tenantID, SalesOrderID: order.ID, SKUID: &skuID, Name: "Canjiquinha", Quantity: 2,
				PriceMinorUnits: 199, SubtotalMinorUnits: 398, TotalMinorUnits: 398, CreatedAt: now},
		}
		for _, l := range lines {
			if err := repositories.NewSalesOrderLineItemRepo(db).Create(ctx, l); err != nil {
				return err
			}
		}
		wooLineID, addedLineID = lines[0].ID, lines[1].ID
		session := omsDomain.PackingSession{TenantID: tenantID, SalesOrderID: order.ID, Status: omsDomain.PackingSessionStatusCompleted, StartedAt: now}
		if err := repositories.NewPackingSessionRepo(db).Create(ctx, &session); err != nil {
			return err
		}
		scans := repositories.NewPackScanRepo(db)
		scan := omsDomain.PackScan{TenantID: tenantID, SessionID: session.ID, SKUID: skuID, ActionType: omsDomain.PackScanActionNewItem, Quantity: 2, ScannedAt: now}
		if err := scans.Create(ctx, &scan); err != nil {
			return err
		}
		return scans.MarkReplayed(ctx, scan.ID, now)
	})
	require.NoError(t, err)

	sqlDB, err := goose.OpenDBWithDriver("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	require.NoError(t, goose.DownTo(sqlDB, migrationsDir, 231))
	require.NoError(t, goose.UpTo(sqlDB, migrationsDir, 232))

	type line struct {
		TotalIncludesTax   bool
		TotalTaxMinorUnits int64
	}
	read := func(id string) line {
		var l line
		require.NoError(t, db.Raw(`SELECT total_includes_tax, total_tax_minor_units FROM sales_order_line_items WHERE id = ?`, id).Scan(&l).Error)
		return l
	}
	assert.Equal(t, line{false, 124}, read(wooLineID), "a Woo line keeps its VAT on top")
	assert.Equal(t, line{true, 33}, read(addedLineID), "1.99 x 2 incl. 9%% VAT holds 0.33 VAT")
	assert.Equal(t, line{true, 0}, read(posLineID), "POS totals include VAT")

	var order struct{ TaxTotalMinorUnits, TotalMinorUnits int64 }
	require.NoError(t, db.Raw(`SELECT tax_total_minor_units, total_minor_units FROM sales_orders WHERE id = ?`, wooOrderID).Scan(&order).Error)
	assert.Equal(t, int64(157), order.TaxTotalMinorUnits)
	assert.Equal(t, int64(1895), order.TotalMinorUnits, "the added item's price already held its VAT")
}

func TestSalesOrderLineItemRepo_Integration_TotalIncludesTaxRoundTrips(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	_, lineID := seedSalesOrderWithLineItemForPackingDB(t, db, ctx, skuID, warehouseID)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)

	var orderID string
	var found *omsDomain.SalesOrderLineItem
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		existing, err := lineItemRepo.FindByID(ctx, lineID)
		if err != nil {
			return err
		}
		orderID = existing.SalesOrderID
		li := omsDomain.SalesOrderLineItem{TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID, SKUID: &skuID, Name: "Extra",
			Quantity: 2, PriceMinorUnits: 199, TotalMinorUnits: 398, TotalTaxMinorUnits: 33, TotalIncludesTax: true, CreatedAt: time.Now()}
		if err := lineItemRepo.Create(ctx, &li); err != nil {
			return err
		}
		found, err = lineItemRepo.FindByID(ctx, li.ID)
		return err
	})
	require.NoError(t, err)
	assert.True(t, found.TotalIncludesTax)
	assert.Equal(t, int64(365), found.TotalExclTax())
}
