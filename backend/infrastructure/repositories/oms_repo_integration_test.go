//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestSalesOrderRepo_Integration_CreateFindByIDAndByExternalID(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)

	var orderID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		order := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS,
			ExternalOrderID: "pos-order-int-1", WarehouseID: warehouseID, CurrencyCode: "USD",
			TotalMinorUnits: 1000, OccurredAt: now, CreatedAt: now,
		}
		if err := salesOrderRepo.Create(ctx, &order); err != nil {
			return err
		}
		orderID = order.ID

		line := omsDomain.SalesOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID, SKUID: skuID,
			Quantity: 2, UnitPriceMinorUnits: 500, CreatedAt: now,
		}
		return lineItemRepo.Create(ctx, &line)
	})
	require.NoError(t, err)

	var byID, byExternal *omsDomain.SalesOrder
	var lineItems []omsDomain.SalesOrderLineItem
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		byID, err = salesOrderRepo.FindByID(ctx, orderID)
		if err != nil {
			return err
		}
		byExternal, err = salesOrderRepo.FindByExternalID(ctx, omsDomain.SalesOrderSourcePOS, "pos-order-int-1")
		if err != nil {
			return err
		}
		lineItems, err = lineItemRepo.ListBySalesOrder(ctx, orderID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, orderID, byID.ID)
	require.Equal(t, int64(1000), byID.TotalMinorUnits)
	require.NotNil(t, byExternal)
	require.Equal(t, orderID, byExternal.ID)
	require.Len(t, lineItems, 1)
	require.Equal(t, skuID, lineItems[0].SKUID)
	require.Equal(t, int64(2), lineItems[0].Quantity)
}

func TestSalesOrderRepo_Integration_FindByExternalIDReturnsNilNotErrorWhenMissing(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	salesOrderRepo := repositories.NewSalesOrderRepo(db)

	var found *omsDomain.SalesOrder
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = salesOrderRepo.FindByExternalID(ctx, omsDomain.SalesOrderSourcePOS, "does-not-exist")
		return err
	})
	require.NoError(t, err)
	require.Nil(t, found)
}

func TestSalesOrderRepo_Integration_ExternalOrderIDIsUniquePerTenantAndSource(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		order := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS,
			ExternalOrderID: "pos-order-dup", WarehouseID: warehouseID, CurrencyCode: "USD",
			OccurredAt: now, CreatedAt: now,
		}
		return salesOrderRepo.Create(ctx, &order)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		dup := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS,
			ExternalOrderID: "pos-order-dup", WarehouseID: warehouseID, CurrencyCode: "USD",
			OccurredAt: now, CreatedAt: now,
		}
		return salesOrderRepo.Create(ctx, &dup)
	})
	require.Error(t, err, "DB unique constraint (tenant_id, source, external_order_id) must reject this")
}
