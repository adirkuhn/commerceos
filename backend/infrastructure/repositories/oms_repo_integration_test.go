//go:build integration

package repositories_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

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
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID, SKUID: &skuID,
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
	require.NotNil(t, lineItems[0].SKUID)
	require.Equal(t, skuID, *lineItems[0].SKUID)
	require.Equal(t, int64(2), lineItems[0].Quantity)
	require.Equal(t, "Widget A", lineItems[0].Name)
	require.Equal(t, "WID-A", lineItems[0].SKUCode)
	require.Equal(t, "0123456789012", lineItems[0].Barcode)
	require.Equal(t, int64(500), lineItems[0].PriceMinorUnits)
	require.Equal(t, int64(1000), lineItems[0].TotalMinorUnits)
}

// A line item unlinked to any catalog SKU (see ImportWooOrdersUseCase --
// a Woo order's SKU string that never matched anything in CommerceOS) must
// still persist and round-trip with a real, nil sku_id column, not error
// on the now-relaxed NOT NULL constraint (079_sales_order_line_item_sku_optional.sql).
func TestSalesOrderLineItemRepo_Integration_RoundTripsANilSKUID(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)

	var orderID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		order := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourceWooCommerce,
			ExternalOrderID: "woo-order-unlinked-1", WarehouseID: warehouseID, CurrencyCode: "USD",
			TotalMinorUnits: 500, OccurredAt: now, CreatedAt: now,
		}
		if err := salesOrderRepo.Create(ctx, &order); err != nil {
			return err
		}
		orderID = order.ID

		line := omsDomain.SalesOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID, SKUID: nil,
			Name: "Mystery Item", SKUCode: "UNKNOWN-SKU", Quantity: 1,
			PriceMinorUnits: 500, CreatedAt: now,
		}
		return lineItemRepo.Create(ctx, &line)
	})
	require.NoError(t, err)

	var lineItems []omsDomain.SalesOrderLineItem
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		lineItems, err = lineItemRepo.ListBySalesOrder(ctx, orderID)
		return err
	})
	require.NoError(t, err)
	require.Len(t, lineItems, 1)
	require.Nil(t, lineItems[0].SKUID)
	require.Equal(t, "Mystery Item", lineItems[0].Name)
	require.Equal(t, "UNKNOWN-SKU", lineItems[0].SKUCode)
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
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID, SKUID: &skuID,
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
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: order.ID, SKUID: &skuID,
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
			StockMovementID: movementID, BatchID: nil, QuantityDelta: -1, CreatedAt: now,
		})
	})
	require.NoError(t, err)

	var links []omsDomain.SalesOrderLineItemStockMovement
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		links, err = linkRepo.ListByLineItem(ctx, lineItemID)
		return err
	})
	require.NoError(t, err)
	require.Len(t, links, 1)
	require.Equal(t, movementID, links[0].StockMovementID)
	require.Nil(t, links[0].BatchID, "nil batch id = the general ledger, same convention as inventory.Batch")
	require.Equal(t, int64(-1), links[0].QuantityDelta)
}

// TestSalesOrderRepo_Integration_CreateReturnsTypedDuplicateExternalOrderError
// proves Create maps the (tenant_id, source, external_order_id) unique
// violation to the typed omsDomain.ErrDuplicateExternalOrder -- what
// ImportWooOrdersUseCase.createOne relies on (errors.Is) to detect it lost a
// create race against a concurrent import of the same Woo order and fall
// back to routing through the update path, instead of surfacing a raw DB
// error as a run failure.
func TestSalesOrderRepo_Integration_CreateReturnsTypedDuplicateExternalOrderError(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		order := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourceWooCommerce,
			ExternalOrderID: "woo-order-dup-typed", WarehouseID: warehouseID, CurrencyCode: "USD",
			OccurredAt: now, CreatedAt: now,
		}
		return salesOrderRepo.Create(ctx, &order)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		dup := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourceWooCommerce,
			ExternalOrderID: "woo-order-dup-typed", WarehouseID: warehouseID, CurrencyCode: "USD",
			OccurredAt: now, CreatedAt: now,
		}
		return salesOrderRepo.Create(ctx, &dup)
	})
	require.ErrorIs(t, err, omsDomain.ErrDuplicateExternalOrder)
}

// TestSalesOrderRepo_Integration_ConcurrentCreatesForTheSameExternalOrderIDOnlyOneWins
// is the mid-task-added hardening test: two goroutines racing to create a
// sales order for the same (tenant, source, external_order_id) -- simulating
// two scheduler replicas, or the manual fetch-order endpoint racing the
// scheduler, both importing the same Woo order at once -- must not both
// succeed. Exactly one Create call succeeds; the other gets
// ErrDuplicateExternalOrder from the DB-level unique constraint, never a
// second row.
func TestSalesOrderRepo_Integration_ConcurrentCreatesForTheSameExternalOrderIDOnlyOneWins(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)

	const externalOrderID = "woo-order-race-1"
	attempt := func() error {
		return tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			now := time.Now()
			order := omsDomain.SalesOrder{
				TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourceWooCommerce,
				ExternalOrderID: externalOrderID, WarehouseID: warehouseID, CurrencyCode: "USD",
				OccurredAt: now, CreatedAt: now,
			}
			return salesOrderRepo.Create(ctx, &order)
		})
	}

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = attempt()
		}(i)
	}
	wg.Wait()

	successes, duplicates := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, omsDomain.ErrDuplicateExternalOrder):
			duplicates++
		default:
			require.NoError(t, err, "unexpected error type -- must be nil or ErrDuplicateExternalOrder")
		}
	}
	require.Equal(t, 1, successes, "exactly one concurrent create must win")
	require.Equal(t, 1, duplicates, "the loser must see the typed duplicate error, not silently succeed")

	var total int64
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var listErr error
		_, total, listErr = salesOrderRepo.List(ctx, omsDomain.ListSalesOrdersFilter{Limit: 50})
		return listErr
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), total, "the database must hold exactly one order for this external id, never two")
}

// TestSalesOrderRepo_Integration_TrySetStockDecrementedOnlyOneOfTwoConcurrentCallersWins
// is the direct integration-level proof for the fix to the reported bug
// ("insufficient stock for this batch/ledger" cascading from an earlier
// double-decrement): two goroutines racing to flip the same order's
// stock_decremented from false to true (simulating UpdateSalesOrderUseCase
// being invoked twice concurrently for the same order, e.g. by two
// scheduler replicas or the manual fetch-order endpoint) must not both
// report success -- exactly one gets swapped=true and is responsible for
// actually decrementing stock; the other gets swapped=false and must treat
// it as a no-op.
func TestSalesOrderRepo_Integration_TrySetStockDecrementedOnlyOneOfTwoConcurrentCallersWins(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)

	var orderID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		order := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourceWooCommerce,
			ExternalOrderID: "woo-order-decrement-race-1", WarehouseID: warehouseID, CurrencyCode: "USD",
			StockDecremented: false, OccurredAt: now, CreatedAt: now,
		}
		return salesOrderRepo.Create(ctx, &order)
	})
	require.NoError(t, err)
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var findErr error
		var so *omsDomain.SalesOrder
		so, findErr = salesOrderRepo.FindByExternalID(ctx, omsDomain.SalesOrderSourceWooCommerce, "woo-order-decrement-race-1")
		if findErr != nil {
			return findErr
		}
		orderID = so.ID
		return nil
	})
	require.NoError(t, err)

	attempt := func() (bool, error) {
		var swapped bool
		err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			var err error
			swapped, err = salesOrderRepo.TrySetStockDecremented(ctx, orderID, false, true)
			return err
		})
		return swapped, err
	}

	var wg sync.WaitGroup
	results := make([]bool, 2)
	errs := make([]error, 2)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = attempt()
		}(i)
	}
	wg.Wait()

	swappedCount := 0
	for i, err := range errs {
		require.NoError(t, err)
		if results[i] {
			swappedCount++
		}
	}
	require.Equal(t, 1, swappedCount, "exactly one concurrent caller must win the swap -- the other must see swapped=false and skip its movement loop")

	var final *omsDomain.SalesOrder
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		final, err = salesOrderRepo.FindByID(ctx, orderID)
		return err
	})
	require.NoError(t, err)
	require.True(t, final.StockDecremented)
}

// TestSalesOrderLineItemRepo_Integration_StockStatusRoundTripsAndUpdate
// proves the new stock_status column (StockStatusPending/Decremented) and
// SalesOrderLineItemRepository.Update/FindByID work against real Postgres --
// LinkLineItemSKUUseCase and RetryStockDecrementUseCase both depend on
// Update to move a line item's stock_status/sku_id.
func TestSalesOrderLineItemRepo_Integration_StockStatusRoundTripsAndUpdate(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)

	var orderID, lineItemID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		order := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourceWooCommerce,
			ExternalOrderID: "woo-order-stock-status-1", WarehouseID: warehouseID, CurrencyCode: "USD",
			OccurredAt: now, CreatedAt: now,
		}
		if err := salesOrderRepo.Create(ctx, &order); err != nil {
			return err
		}
		orderID = order.ID

		line := omsDomain.SalesOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID, SKUID: nil,
			Name: "Mystery Item", SKUCode: "UNKNOWN-SKU", Quantity: 1, StockStatus: omsDomain.StockStatusPending,
			PriceMinorUnits: 500, CreatedAt: now,
		}
		if err := lineItemRepo.Create(ctx, &line); err != nil {
			return err
		}
		lineItemID = line.ID
		return nil
	})
	require.NoError(t, err)

	var afterCreate *omsDomain.SalesOrderLineItem
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		afterCreate, err = lineItemRepo.FindByID(ctx, lineItemID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, omsDomain.StockStatusPending, afterCreate.StockStatus)
	require.Nil(t, afterCreate.SKUID)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return lineItemRepo.Update(ctx, lineItemID, map[string]any{
			"sku_id": skuID, "stock_status": string(omsDomain.StockStatusDecremented),
		})
	})
	require.NoError(t, err)

	var afterUpdate *omsDomain.SalesOrderLineItem
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		afterUpdate, err = lineItemRepo.FindByID(ctx, lineItemID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, omsDomain.StockStatusDecremented, afterUpdate.StockStatus)
	require.NotNil(t, afterUpdate.SKUID)
	require.Equal(t, skuID, *afterUpdate.SKUID)
}

// TestSalesOrderRepo_Integration_ListFiltersByNeedsStockReviewAndNeedsSKUReview
// proves the two new list filters actually narrow the query against real
// Postgres -- the "Stock Issues"/"Unmatched SKUs" admin pages depend on
// these to only show flagged orders.
func TestSalesOrderRepo_Integration_ListFiltersByNeedsStockReviewAndNeedsSKUReview(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)

	var stockReviewID, skuReviewID, cleanID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		specs := []struct {
			extID            string
			needsStockReview bool
			needsSKUReview   bool
			id               *string
		}{
			{"list-filter-stock-review", true, false, &stockReviewID},
			{"list-filter-sku-review", false, true, &skuReviewID},
			{"list-filter-clean", false, false, &cleanID},
		}
		for _, s := range specs {
			order := omsDomain.SalesOrder{
				TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourceWooCommerce,
				ExternalOrderID: s.extID, WarehouseID: warehouseID, CurrencyCode: "USD",
				OccurredAt: now, CreatedAt: now,
			}
			if err := salesOrderRepo.Create(ctx, &order); err != nil {
				return err
			}
			*s.id = order.ID
			if err := salesOrderRepo.Update(ctx, order.ID, map[string]any{
				"needs_stock_review": s.needsStockReview, "needs_sku_review": s.needsSKUReview,
			}); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err)

	trueVal := true
	var stockReviewOrders, skuReviewOrders []omsDomain.SalesOrder
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		stockReviewOrders, _, err = salesOrderRepo.List(ctx, omsDomain.ListSalesOrdersFilter{Limit: 50, NeedsStockReview: &trueVal})
		if err != nil {
			return err
		}
		skuReviewOrders, _, err = salesOrderRepo.List(ctx, omsDomain.ListSalesOrdersFilter{Limit: 50, NeedsSKUReview: &trueVal})
		return err
	})
	require.NoError(t, err)

	require.Len(t, stockReviewOrders, 1)
	require.Equal(t, stockReviewID, stockReviewOrders[0].ID)
	require.Len(t, skuReviewOrders, 1)
	require.Equal(t, skuReviewID, skuReviewOrders[0].ID)
}

func TestSalesOrderRepo_Integration_FulfilledRoundTripsAndFilters(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)

	var fulfilledID, unfulfilledID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		fulfilled := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourceWooCommerce,
			ExternalOrderID: "fulfilled-int-1", WarehouseID: warehouseID, CurrencyCode: "USD",
			Status: omsDomain.SalesOrderStatusCompleted, Fulfilled: true, OccurredAt: now, CreatedAt: now,
		}
		if err := salesOrderRepo.Create(ctx, &fulfilled); err != nil {
			return err
		}
		fulfilledID = fulfilled.ID

		unfulfilled := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourceWooCommerce,
			ExternalOrderID: "unfulfilled-int-1", WarehouseID: warehouseID, CurrencyCode: "USD",
			Status: omsDomain.SalesOrderStatusProcessing, Fulfilled: false, OccurredAt: now, CreatedAt: now,
		}
		if err := salesOrderRepo.Create(ctx, &unfulfilled); err != nil {
			return err
		}
		unfulfilledID = unfulfilled.ID
		return nil
	})
	require.NoError(t, err)

	var byID *omsDomain.SalesOrder
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		byID, err = salesOrderRepo.FindByID(ctx, fulfilledID)
		return err
	})
	require.NoError(t, err)
	require.True(t, byID.Fulfilled, "fulfilled must round-trip through Create/FindByID")

	falseVal := false
	var unfulfilledOrders []omsDomain.SalesOrder
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		unfulfilledOrders, _, err = salesOrderRepo.List(ctx, omsDomain.ListSalesOrdersFilter{Limit: 50, Fulfilled: &falseVal})
		return err
	})
	require.NoError(t, err)
	require.Len(t, unfulfilledOrders, 1)
	require.Equal(t, unfulfilledID, unfulfilledOrders[0].ID)

	// Update must also flip it (CompletePickingUseCase/UpdateSalesOrderUseCase
	// both write it via the map[string]any Update path, not the Create path).
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return salesOrderRepo.Update(ctx, unfulfilledID, map[string]any{"fulfilled": true})
	})
	require.NoError(t, err)
	var afterUpdate *omsDomain.SalesOrder
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		afterUpdate, err = salesOrderRepo.FindByID(ctx, unfulfilledID)
		return err
	})
	require.NoError(t, err)
	require.True(t, afterUpdate.Fulfilled)
}

// TestSalesOrderRepo_Integration_ListFiltersByStatus proves Status narrows
// the query against real Postgres -- the picking screens depend on this to
// exclude completed/cancelled/refunded/failed orders from the pick list.
func TestSalesOrderRepo_Integration_ListFiltersByStatus(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)

	var processingID, completedID, cancelledID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		specs := []struct {
			extID  string
			status omsDomain.SalesOrderStatus
			id     *string
		}{
			{"status-filter-processing", omsDomain.SalesOrderStatusProcessing, &processingID},
			{"status-filter-completed", omsDomain.SalesOrderStatusCompleted, &completedID},
			{"status-filter-cancelled", omsDomain.SalesOrderStatusCancelled, &cancelledID},
		}
		for _, s := range specs {
			order := omsDomain.SalesOrder{
				TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourceWooCommerce,
				ExternalOrderID: s.extID, WarehouseID: warehouseID, CurrencyCode: "USD",
				Status: s.status, OccurredAt: now, CreatedAt: now,
			}
			if err := salesOrderRepo.Create(ctx, &order); err != nil {
				return err
			}
			*s.id = order.ID
		}
		return nil
	})
	require.NoError(t, err)

	var pickable []omsDomain.SalesOrder
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		pickable, _, err = salesOrderRepo.List(ctx, omsDomain.ListSalesOrdersFilter{
			Limit: 50,
			Status: []omsDomain.SalesOrderStatus{
				omsDomain.SalesOrderStatusPending, omsDomain.SalesOrderStatusProcessing, omsDomain.SalesOrderStatusOnHold,
			},
		})
		return err
	})
	require.NoError(t, err)
	require.Len(t, pickable, 1)
	require.Equal(t, processingID, pickable[0].ID)
}

// seedDashboardOrder creates one sales order (+ one line item, if qty > 0)
// with the given source/status/currency/occurred_at -- the shared fixture
// for the dashboard report queries below, which all care about exactly
// those dimensions and nothing else about the order.
func seedDashboardOrder(
	t *testing.T, db *gorm.DB, ctx context.Context, salesOrderRepo *repositories.SalesOrderRepo, lineItemRepo *repositories.SalesOrderLineItemRepo,
	warehouseID, externalID string, source omsDomain.SalesOrderSource, status omsDomain.SalesOrderStatus, currency string, occurredAt time.Time,
	skuCode, productName string, qty, totalMinorUnits int64,
) string {
	t.Helper()
	var orderID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		order := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: source, ExternalOrderID: externalID,
			WarehouseID: warehouseID, Status: status, CurrencyCode: currency,
			TotalMinorUnits: totalMinorUnits, OccurredAt: occurredAt, CreatedAt: occurredAt,
		}
		if err := salesOrderRepo.Create(ctx, &order); err != nil {
			return err
		}
		orderID = order.ID
		if qty <= 0 {
			return nil
		}
		line := omsDomain.SalesOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID, Name: productName, SKUCode: skuCode,
			Quantity: qty, TotalMinorUnits: totalMinorUnits, CreatedAt: occurredAt,
		}
		return lineItemRepo.Create(ctx, &line)
	})
	require.NoError(t, err)
	return orderID
}

func TestSalesOrderRepo_Integration_RevenueSince_CountsOnlyStockReducingStatusAndMatchingCurrency(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)

	now := time.Now()
	before := now.AddDate(0, -1, 0)
	seedDashboardOrder(t, db, ctx, salesOrderRepo, lineItemRepo, warehouseID, "ext-1", omsDomain.SalesOrderSourcePOS, omsDomain.SalesOrderStatusCompleted, "EUR", now, "SKU-A", "A", 1, 1000)
	seedDashboardOrder(t, db, ctx, salesOrderRepo, lineItemRepo, warehouseID, "ext-2", omsDomain.SalesOrderSourcePOS, omsDomain.SalesOrderStatusProcessing, "EUR", now, "SKU-B", "B", 1, 500)
	// Excluded: cancelled (never reduced stock).
	seedDashboardOrder(t, db, ctx, salesOrderRepo, lineItemRepo, warehouseID, "ext-3", omsDomain.SalesOrderSourcePOS, omsDomain.SalesOrderStatusCancelled, "EUR", now, "SKU-C", "C", 1, 9999)
	// Excluded: different currency.
	seedDashboardOrder(t, db, ctx, salesOrderRepo, lineItemRepo, warehouseID, "ext-4", omsDomain.SalesOrderSourcePOS, omsDomain.SalesOrderStatusCompleted, "USD", now, "SKU-D", "D", 1, 9999)
	// Excluded: before the window.
	seedDashboardOrder(t, db, ctx, salesOrderRepo, lineItemRepo, warehouseID, "ext-5", omsDomain.SalesOrderSourcePOS, omsDomain.SalesOrderStatusCompleted, "EUR", before, "SKU-E", "E", 1, 9999)

	var total int64
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		total, err = salesOrderRepo.RevenueSince(ctx, now.AddDate(0, 0, -1), "EUR")
		return err
	})
	require.NoError(t, err)
	require.Equal(t, int64(1500), total)
}

func TestSalesOrderRepo_Integration_RevenueByChannelSince_GroupsBySource(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)

	now := time.Now()
	seedDashboardOrder(t, db, ctx, salesOrderRepo, lineItemRepo, warehouseID, "ch-1", omsDomain.SalesOrderSourcePOS, omsDomain.SalesOrderStatusCompleted, "EUR", now, "SKU-A", "A", 1, 1000)
	seedDashboardOrder(t, db, ctx, salesOrderRepo, lineItemRepo, warehouseID, "ch-2", omsDomain.SalesOrderSourcePOS, omsDomain.SalesOrderStatusCompleted, "EUR", now, "SKU-B", "B", 1, 250)
	seedDashboardOrder(t, db, ctx, salesOrderRepo, lineItemRepo, warehouseID, "ch-3", omsDomain.SalesOrderSourceWooCommerce, omsDomain.SalesOrderStatusCompleted, "EUR", now, "SKU-C", "C", 1, 700)

	var rows []omsDomain.ChannelRevenue
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		rows, err = salesOrderRepo.RevenueByChannelSince(ctx, now.AddDate(0, 0, -1), "EUR")
		return err
	})
	require.NoError(t, err)

	bySource := make(map[omsDomain.SalesOrderSource]int64, len(rows))
	for _, r := range rows {
		bySource[r.Source] = r.TotalMinorUnits
	}
	require.Equal(t, int64(1250), bySource[omsDomain.SalesOrderSourcePOS])
	require.Equal(t, int64(700), bySource[omsDomain.SalesOrderSourceWooCommerce])
}

func TestSalesOrderRepo_Integration_TopProductsSince_RanksByQuantityDescLimited(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)

	now := time.Now()
	seedDashboardOrder(t, db, ctx, salesOrderRepo, lineItemRepo, warehouseID, "tp-1", omsDomain.SalesOrderSourcePOS, omsDomain.SalesOrderStatusCompleted, "EUR", now, "SKU-A", "Widget A", 5, 5000)
	seedDashboardOrder(t, db, ctx, salesOrderRepo, lineItemRepo, warehouseID, "tp-2", omsDomain.SalesOrderSourcePOS, omsDomain.SalesOrderStatusCompleted, "EUR", now, "SKU-A", "Widget A", 3, 3000)
	seedDashboardOrder(t, db, ctx, salesOrderRepo, lineItemRepo, warehouseID, "tp-3", omsDomain.SalesOrderSourcePOS, omsDomain.SalesOrderStatusCompleted, "EUR", now, "SKU-B", "Widget B", 20, 2000)
	seedDashboardOrder(t, db, ctx, salesOrderRepo, lineItemRepo, warehouseID, "tp-4", omsDomain.SalesOrderSourcePOS, omsDomain.SalesOrderStatusCompleted, "EUR", now, "SKU-C", "Widget C", 1, 100)
	// Excluded: cancelled.
	seedDashboardOrder(t, db, ctx, salesOrderRepo, lineItemRepo, warehouseID, "tp-5", omsDomain.SalesOrderSourcePOS, omsDomain.SalesOrderStatusCancelled, "EUR", now, "SKU-Z", "Widget Z", 999, 99900)

	var rows []omsDomain.ProductSales
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		rows, err = salesOrderRepo.TopProductsSince(ctx, now.AddDate(0, 0, -1), "EUR", 2)
		return err
	})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, "SKU-B", rows[0].SKUCode)
	require.Equal(t, int64(20), rows[0].QuantitySold)
	require.Equal(t, int64(2000), rows[0].RevenueMinorUnits)
	require.Equal(t, "SKU-A", rows[1].SKUCode)
	require.Equal(t, int64(8), rows[1].QuantitySold)
	require.Equal(t, int64(8000), rows[1].RevenueMinorUnits)
}
