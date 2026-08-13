//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
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
			Name: "Widget A", SKUCode: "WID-A", Quantity: 2,
			PriceMinorUnits: 500, SubtotalMinorUnits: 1000, TotalMinorUnits: 1000, CreatedAt: now,
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
	require.Equal(t, "Widget A", lineItems[0].Name)
	require.Equal(t, "WID-A", lineItems[0].SKUCode)
	require.Equal(t, int64(500), lineItems[0].PriceMinorUnits)
	require.Equal(t, int64(1000), lineItems[0].TotalMinorUnits)
}

// TestSalesOrderRepo_Integration_WooFieldsRoundTrip proves the new
// Woo-mirrored columns (billing/shipping address, status, customer link,
// payment info) actually round-trip through real Postgres -- in particular
// that the explicit billing_address_1/shipping_address_1 column tags
// (GORM's default snake-casing of "Address1" is ambiguous) map correctly.
func TestSalesOrderRepo_Integration_WooFieldsRoundTrip(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	customerRepo := repositories.NewCustomerRepo(db)

	var orderID, customerID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		customer := customersDomain.Customer{
			TenantID: tenant.IDFromContext(ctx), Email: "jane@example.com", FirstName: "Jane", LastName: "Doe",
			DateCreated: now, DateModified: now,
		}
		if err := customerRepo.Create(ctx, &customer); err != nil {
			return err
		}
		customerID = customer.ID

		datePaid := now
		order := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS,
			ExternalOrderID: "pos-order-woo-1", WarehouseID: warehouseID, CustomerID: &customerID,
			Status: omsDomain.SalesOrderStatusCompleted, Number: "1001", OrderKey: "wc_order_abc123",
			Billing: omsDomain.Address{
				FirstName: "Jane", LastName: "Doe", Address1: "123 Main St", Address2: "Apt 4",
				City: "Springfield", State: "IL", Postcode: "62701", Country: "US", Email: "jane@example.com", Phone: "555-0100",
			},
			Shipping: omsDomain.Address{
				FirstName: "Jane", LastName: "Doe", Address1: "456 Oak Ave", City: "Springfield", Country: "US",
			},
			CurrencyCode: "USD", DiscountTotalMinorUnits: 100, ShippingTotalMinorUnits: 500, TaxTotalMinorUnits: 80,
			TotalMinorUnits: 1480, PaymentMethod: "cash", PaymentMethodTitle: "Cash", DatePaid: &datePaid,
			OccurredAt: now, CreatedAt: now, UpdatedAt: now,
		}
		if err := salesOrderRepo.Create(ctx, &order); err != nil {
			return err
		}
		orderID = order.ID
		return nil
	})
	require.NoError(t, err)

	var found *omsDomain.SalesOrder
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = salesOrderRepo.FindByID(ctx, orderID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, omsDomain.SalesOrderStatusCompleted, found.Status)
	require.Equal(t, "1001", found.Number)
	require.NotNil(t, found.CustomerID)
	require.Equal(t, customerID, *found.CustomerID)
	require.Equal(t, "123 Main St", found.Billing.Address1)
	require.Equal(t, "Apt 4", found.Billing.Address2)
	require.Equal(t, "jane@example.com", found.Billing.Email)
	require.Equal(t, "456 Oak Ave", found.Shipping.Address1)
	require.Empty(t, found.Shipping.Email, "shipping has no email field, even though the struct is shared with billing")
	require.Equal(t, int64(1480), found.TotalMinorUnits)
	require.NotNil(t, found.DatePaid)
}

func TestSalesOrderRepo_Integration_List(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		for _, extID := range []string{"pos-list-1", "pos-list-2"} {
			order := omsDomain.SalesOrder{
				TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS,
				ExternalOrderID: extID, WarehouseID: warehouseID, CurrencyCode: "USD",
				OccurredAt: now, CreatedAt: now,
			}
			if err := salesOrderRepo.Create(ctx, &order); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err)

	var orders []omsDomain.SalesOrder
	var total int64
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		orders, total, err = salesOrderRepo.List(ctx, omsDomain.ListSalesOrdersFilter{Limit: 50})
		return err
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, total, int64(2))
	require.GreaterOrEqual(t, len(orders), 2)
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
