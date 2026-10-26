//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// Editing a processing manual order puts its stock back and takes it again
// for the new lines, and writes the new totals and addresses.
func TestEditSalesOrder_Integration_replacesLinesAndRetakesStock(t *testing.T) {
	db := newIntegrationDB(t)
	tenantID, ctx := newTestTenant()
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		return repositories.NewStockMovementRepo(db).Create(ctx, &invDomain.StockMovement{TenantID: tenantID, SKUID: skuID, WarehouseID: warehouseID,
			Type: invDomain.MovementTypeReceipt, QuantityDelta: 20, OccurredAt: now, CreatedAt: now})
	}))

	orders, lines, links := repositories.NewSalesOrderRepo(db), repositories.NewSalesOrderLineItemRepo(db), repositories.NewSalesOrderLineItemStockMovementRepo(db)
	outbox := repositories.NewOutboxRepository(db)
	movements := repositories.NewStockMovementRepo(db)
	allocate := appinventory.NewAllocateSaleStockUseCase(db, movements, repositories.NewBatchRepo(db), outbox)
	created, err := appoms.NewCreateSalesOrderUseCase(db, orders, lines, links, repositories.NewSKURepo(db), allocate, outbox).
		WithNumbers(repositories.NewSalesOrderNumberRepo(db)).
		Execute(ctx, appoms.CreateSalesOrderInput{Status: omsDomain.SalesOrderStatusProcessing, WarehouseID: warehouseID, CurrencyCode: "EUR", OccurredAt: time.Now(),
			LineItems: []appoms.CreateSalesOrderLineItemInput{{SKUID: skuID, Name: "Mate", Quantity: 10, PriceMinorUnits: 299, TotalMinorUnits: 2990}}})
	require.NoError(t, err)
	assert.Equal(t, "CO-1001", created.SalesOrder.Number)

	stock := func() int64 {
		var n int64
		require.NoError(t, db.Raw(`SELECT COALESCE(SUM(quantity_delta), 0) FROM stock_movements WHERE sku_id = ?`, skuID).Scan(&n).Error)
		return n
	}
	require.Equal(t, int64(10), stock())

	edit := appoms.NewEditSalesOrderUseCase(db, orders, lines, links, repositories.NewSKURepo(db), allocate,
		appinventory.NewRecordStockMovementUseCase(db, movements, outbox, nil),
		repositories.NewPickingSessionRepo(db), repositories.NewPackingSessionRepo(db), outbox)
	_, err = edit.Execute(ctx, appoms.EditSalesOrderInput{
		ID: created.SalesOrder.ID, WarehouseID: warehouseID, OccurredAt: time.Now(), CustomerNote: "Back door",
		Billing:            omsDomain.Address{Company: "Claus Hoofddorp B.V."},
		TaxTotalMinorUnits: 99, TotalMinorUnits: 1196,
		LineItems: []appoms.CreateSalesOrderLineItemInput{{SKUID: skuID, Name: "Mate", Quantity: 4, PriceMinorUnits: 299,
			TotalMinorUnits: 1196, TotalTaxMinorUnits: 99, TotalIncludesTax: true}},
	})
	require.NoError(t, err)

	assert.Equal(t, int64(16), stock(), "10 given back, 4 taken")
	var order *omsDomain.SalesOrder
	var got []omsDomain.SalesOrderLineItem
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		if order, err = orders.FindByID(ctx, created.SalesOrder.ID); err != nil {
			return err
		}
		got, err = lines.ListBySalesOrder(ctx, created.SalesOrder.ID)
		return err
	}))
	assert.Equal(t, "CO-1001", order.Number)
	assert.Equal(t, int64(1196), order.TotalMinorUnits)
	assert.Equal(t, int64(99), order.TaxTotalMinorUnits)
	assert.Equal(t, "Back door", order.CustomerNote)
	assert.Equal(t, "Claus Hoofddorp B.V.", order.Billing.Company)
	assert.False(t, order.NeedsStockReview)
	require.Len(t, got, 1)
	assert.Equal(t, int64(4), got[0].Quantity)
	assert.Equal(t, int64(99), got[0].TotalTaxMinorUnits)
	assert.Equal(t, omsDomain.StockStatusDecremented, got[0].StockStatus)
}
