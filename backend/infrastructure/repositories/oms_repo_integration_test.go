//go:build integration

package repositories_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
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
			Name: "Widget A", SKUCode: "WID-A", Barcode: "0123456789012", Quantity: 2,
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
	require.Equal(t, "0123456789012", lineItems[0].Barcode)
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
			TotalMinorUnits: 1480, PaymentMethod: "cash", PaymentMethodTitle: "Cash",
			ShippingMethod: "flat_rate", ShippingMethodTitle: "Flat rate", DatePaid: &datePaid,
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
	require.Equal(t, "flat_rate", found.ShippingMethod)
	require.Equal(t, "Flat rate", found.ShippingMethodTitle)
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

// TestSalesOrderRepo_Integration_ListByConnection proves ListByConnection
// only returns orders stamped with that connection's ID -- a POS order
// (no connection at all) and a different connection's order must not
// leak in, since "delete all imported products" trusts this to scope
// exactly which orders belong to the connection being cleared.
func TestSalesOrderRepo_Integration_ListByConnection(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	connRepo := repositories.NewConnectionRepo(db)

	var conn1ID, conn2ID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		for i, id := range []*string{&conn1ID, &conn2ID} {
			conn := &integrationsDomain.Connection{
				TenantID: tenant.IDFromContext(ctx), Provider: integrationsDomain.ProviderWooCommerce,
				StoreURL:                integrationsDomain.StoreURL(fmt.Sprintf("https://itest-list-by-conn-%d.example.com", i)),
				ConsumerKey:             "ck",
				ConsumerSecretEncrypted: "enc", Status: integrationsDomain.StatusConnected,
				CreatedAt: now, UpdatedAt: now,
			}
			if err := connRepo.Create(ctx, conn); err != nil {
				return err
			}
			*id = conn.ID
		}

		orders := []omsDomain.SalesOrder{
			{TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourceWooCommerce, ExternalOrderID: "woo-conn1-a", ConnectionID: &conn1ID, WarehouseID: warehouseID, CurrencyCode: "USD", OccurredAt: now, CreatedAt: now},
			{TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourceWooCommerce, ExternalOrderID: "woo-conn1-b", ConnectionID: &conn1ID, WarehouseID: warehouseID, CurrencyCode: "USD", OccurredAt: now, CreatedAt: now},
			{TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourceWooCommerce, ExternalOrderID: "woo-conn2-a", ConnectionID: &conn2ID, WarehouseID: warehouseID, CurrencyCode: "USD", OccurredAt: now, CreatedAt: now},
			{TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS, ExternalOrderID: "pos-no-conn", WarehouseID: warehouseID, CurrencyCode: "USD", OccurredAt: now, CreatedAt: now},
		}
		for _, o := range orders {
			o := o
			if err := salesOrderRepo.Create(ctx, &o); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err)

	var found []omsDomain.SalesOrder
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = salesOrderRepo.ListByConnection(ctx, conn1ID)
		return err
	})
	require.NoError(t, err)
	require.Len(t, found, 2)
	for _, o := range found {
		require.NotNil(t, o.ConnectionID)
		require.Equal(t, conn1ID, *o.ConnectionID)
	}
}

// TestSalesOrderRepo_Integration_ListSortsByTotalAndDefaultsToOccurredAtDesc
// proves salesOrderOrderClause actually drives a real ORDER BY against
// Postgres for both an explicit sort field (total) and the default
// (occurred_at desc, newest first) when SortBy/SortDir are left unset.
func TestSalesOrderRepo_Integration_ListSortsByTotalAndDefaultsToOccurredAtDesc(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)

	var lowID, midID, highID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		// OccurredAt is staggered (not "mid" latest by coincidence) so the
		// default-sort assertion below has an unambiguous "most recent".
		specs := []struct {
			extID      string
			total      int64
			occurredAt time.Time
			id         *string
		}{
			{"sort-low", 500, now, &lowID},
			{"sort-high", 2500, now.Add(1 * time.Second), &highID},
			{"sort-mid", 1500, now.Add(2 * time.Second), &midID},
		}
		for _, s := range specs {
			order := omsDomain.SalesOrder{
				TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS,
				ExternalOrderID: s.extID, WarehouseID: warehouseID, CurrencyCode: "USD",
				TotalMinorUnits: s.total, OccurredAt: s.occurredAt, CreatedAt: now,
			}
			if err := salesOrderRepo.Create(ctx, &order); err != nil {
				return err
			}
			*s.id = order.ID
		}
		return nil
	})
	require.NoError(t, err)

	var byTotalAsc []omsDomain.SalesOrder
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		byTotalAsc, _, err = salesOrderRepo.List(ctx, omsDomain.ListSalesOrdersFilter{
			Limit: 3, SortBy: omsDomain.SalesOrderSortTotal, SortDir: omsDomain.SortAsc,
		})
		return err
	})
	require.NoError(t, err)
	require.Len(t, byTotalAsc, 3)
	require.Equal(t, []string{lowID, midID, highID}, []string{byTotalAsc[0].ID, byTotalAsc[1].ID, byTotalAsc[2].ID})

	var byDefault []omsDomain.SalesOrder
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		byDefault, _, err = salesOrderRepo.List(ctx, omsDomain.ListSalesOrdersFilter{Limit: 3})
		return err
	})
	require.NoError(t, err)
	require.Len(t, byDefault, 3)
	require.Equal(t, midID, byDefault[0].ID, "no SortBy/SortDir set -- must default to occurred_at desc, the most-recently-created of these three")
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

func TestSalesOrderRepo_Integration_Update(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)

	var orderID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		order := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourceManual,
			ExternalOrderID: "manual-order-update-1", WarehouseID: warehouseID, CurrencyCode: "USD",
			Status: omsDomain.SalesOrderStatusPending, OccurredAt: now, CreatedAt: now,
		}
		if err := salesOrderRepo.Create(ctx, &order); err != nil {
			return err
		}
		orderID = order.ID
		return nil
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return salesOrderRepo.Update(ctx, orderID, map[string]any{
			"status": string(omsDomain.SalesOrderStatusCompleted), "stock_decremented": true,
			"customer_note": "call before delivery",
		})
	})
	require.NoError(t, err)

	var updated *omsDomain.SalesOrder
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		updated, err = salesOrderRepo.FindByID(ctx, orderID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, omsDomain.SalesOrderStatusCompleted, updated.Status)
	require.True(t, updated.StockDecremented)
	require.Equal(t, "call before delivery", updated.CustomerNote)
}

func TestSalesOrderRepo_Integration_UpdateReturnsNotFoundForAMissingOrder(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	salesOrderRepo := repositories.NewSalesOrderRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return salesOrderRepo.Update(ctx, "00000000-0000-0000-0000-000000000000", map[string]any{"customer_note": "x"})
	})
	require.ErrorIs(t, err, omsDomain.ErrSalesOrderNotFound)
}

func TestSalesOrderRepo_Integration_DeleteCascadesLineItems(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)

	var orderID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		order := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourceManual,
			ExternalOrderID: "manual-order-delete-1", WarehouseID: warehouseID, CurrencyCode: "USD",
			OccurredAt: now, CreatedAt: now,
		}
		if err := salesOrderRepo.Create(ctx, &order); err != nil {
			return err
		}
		orderID = order.ID
		line := omsDomain.SalesOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID, SKUID: skuID,
			Quantity: 1, PriceMinorUnits: 500, CreatedAt: now,
		}
		return lineItemRepo.Create(ctx, &line)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		if err := lineItemRepo.DeleteBySalesOrder(ctx, orderID); err != nil {
			return err
		}
		return salesOrderRepo.Delete(ctx, orderID)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		_, err := salesOrderRepo.FindByID(ctx, orderID)
		return err
	})
	require.ErrorIs(t, err, omsDomain.ErrSalesOrderNotFound)
}

func TestSalesOrderLineItemStockMovementRepo_Integration_Create(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)
	movementRepo := repositories.NewStockMovementRepo(db)
	linkRepo := repositories.NewSalesOrderLineItemStockMovementRepo(db)

	var lineItemID, movementID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		order := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourceManual,
			ExternalOrderID: "manual-order-link-1", WarehouseID: warehouseID, CurrencyCode: "USD",
			OccurredAt: now, CreatedAt: now,
		}
		if err := salesOrderRepo.Create(ctx, &order); err != nil {
			return err
		}
		line := omsDomain.SalesOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: order.ID, SKUID: skuID,
			Quantity: 1, PriceMinorUnits: 500, CreatedAt: now,
		}
		if err := lineItemRepo.Create(ctx, &line); err != nil {
			return err
		}
		lineItemID = line.ID

		movement := invDomain.StockMovement{
			TenantID: tenant.IDFromContext(ctx), SKUID: skuID, WarehouseID: warehouseID,
			Type: invDomain.MovementTypeSale, QuantityDelta: -1, OccurredAt: now, CreatedAt: now,
		}
		if err := movementRepo.Create(ctx, &movement); err != nil {
			return err
		}
		movementID = movement.ID

		return linkRepo.Create(ctx, &omsDomain.SalesOrderLineItemStockMovement{
			TenantID: tenant.IDFromContext(ctx), SalesOrderLineItemID: lineItemID,
			StockMovementID: movementID, CreatedAt: now,
		})
	})
	require.NoError(t, err)
}
