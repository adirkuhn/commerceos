//go:build integration

package repositories_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// TestSalesOrderRepo_Integration_SearchByNumberAndBillingName proves the
// same unaccent()/pg_trgm search Products already has
// (127_search_trigram_indexes.sql) actually works against real Postgres
// for sales orders too -- powers global search (Cmd+K).
func TestSalesOrderRepo_Integration_SearchByNumberAndBillingName(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		orders := []omsDomain.SalesOrder{
			{
				TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS,
				ExternalOrderID: "search-order-1", Number: "SO-10042", WarehouseID: warehouseID, CurrencyCode: "USD",
				TotalMinorUnits: 1000, OccurredAt: now, CreatedAt: now,
				Billing: omsDomain.Address{FirstName: "José", LastName: "Conceição"},
			},
			{
				TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS,
				ExternalOrderID: "search-order-2", Number: "SO-20099", WarehouseID: warehouseID, CurrencyCode: "USD",
				TotalMinorUnits: 1000, OccurredAt: now, CreatedAt: now,
				Billing: omsDomain.Address{FirstName: "Renée", LastName: "Müller"},
			},
		}
		for i := range orders {
			if err := salesOrderRepo.Create(ctx, &orders[i]); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err)

	for _, tc := range []struct {
		query      string
		wantNumber string
	}{
		{"SO-10042", "SO-10042"},
		{"SO-1004", "SO-10042"}, // partial number
		{"jose conceicao", "SO-10042"},
		{"renee muller", "SO-20099"},
	} {
		var results []omsDomain.SalesOrder
		err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			var err error
			results, err = salesOrderRepo.Search(ctx, tc.query, 10)
			return err
		})
		require.NoError(t, err)
		require.Len(t, results, 1, "query %q should match exactly one order", tc.query)
		assert.Equal(t, tc.wantNumber, results[0].Number, "query %q", tc.query)
	}
}

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

// TestSalesOrderRepo_Integration_RefundFieldsRoundTrip covers the new
// payment_provider/payment_external_id/needs_manual_refund_review columns
// (142_sales_orders_refund_fields.sql) and the partially_refunded status
// value -- real Postgres, so it also catches the CHECK constraint actually
// accepting the new status.
func TestSalesOrderRepo_Integration_RefundFieldsRoundTrip(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)

	var orderID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		order := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourceWhatsApp,
			ExternalOrderID: "wa-order-refund-1", WarehouseID: warehouseID,
			Status: omsDomain.SalesOrderStatusProcessing, CurrencyCode: "USD", TotalMinorUnits: 1000,
			PaymentProvider: "stripe", PaymentExternalID: "cs_test_123",
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
	require.Equal(t, "stripe", found.PaymentProvider)
	require.Equal(t, "cs_test_123", found.PaymentExternalID)
	require.False(t, found.NeedsManualRefundReview, "defaults false")

	// A real refund replay writes needs_manual_refund_review directly and,
	// separately, the new partially_refunded status -- both via the
	// generic Update(fields map) path, same as CompletePickingUseCase does.
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return salesOrderRepo.Update(ctx, orderID, map[string]any{"needs_manual_refund_review": true})
	})
	require.NoError(t, err)
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return salesOrderRepo.Update(ctx, orderID, map[string]any{"status": string(omsDomain.SalesOrderStatusPartiallyRefunded)})
	})
	require.NoError(t, err, "the sales_orders_status_check CHECK constraint must accept partially_refunded")

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = salesOrderRepo.FindByID(ctx, orderID)
		return err
	})
	require.NoError(t, err)
	require.True(t, found.NeedsManualRefundReview)
	require.Equal(t, omsDomain.SalesOrderStatusPartiallyRefunded, found.Status)
}

// TestSalesOrderLineItemRepo_Integration_RefundedFieldsRoundTrip covers the
// new refunded_quantity/refunded_amount_minor_units columns
// (143_sales_order_line_items_refund_fields.sql).
func TestSalesOrderLineItemRepo_Integration_RefundedFieldsRoundTrip(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)

	var orderID, lineItemID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		order := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS,
			ExternalOrderID: "pos-order-refund-1", WarehouseID: warehouseID,
			Status: omsDomain.SalesOrderStatusProcessing, CurrencyCode: "USD", TotalMinorUnits: 1000,
			OccurredAt: now, CreatedAt: now, UpdatedAt: now,
		}
		if err := salesOrderRepo.Create(ctx, &order); err != nil {
			return err
		}
		orderID = order.ID

		li := omsDomain.SalesOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID, SKUID: &skuID, Name: "Widget",
			Quantity: 3, PriceMinorUnits: 500, CreatedAt: now,
		}
		if err := lineItemRepo.Create(ctx, &li); err != nil {
			return err
		}
		lineItemID = li.ID
		return nil
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return lineItemRepo.Update(ctx, lineItemID, map[string]any{"refunded_quantity": int64(2), "refunded_amount_minor_units": int64(1000)})
	})
	require.NoError(t, err)

	var found *omsDomain.SalesOrderLineItem
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = lineItemRepo.FindByID(ctx, lineItemID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, int64(2), found.RefundedQuantity)
	require.Equal(t, int64(1000), found.RefundedAmountMinorUnits)
}

func TestSalesOrderLineItemRepo_Integration_SoldExpiryDateRoundTrips(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)

	expiry := time.Date(2027, 6, 12, 0, 0, 0, 0, time.UTC)
	var lineItemID, noExpiryLineItemID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		order := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourceWooCommerce,
			ExternalOrderID: "woo-order-expiry-1", WarehouseID: warehouseID,
			Status: omsDomain.SalesOrderStatusProcessing, CurrencyCode: "USD", TotalMinorUnits: 1000,
			OccurredAt: now, CreatedAt: now, UpdatedAt: now,
		}
		if err := salesOrderRepo.Create(ctx, &order); err != nil {
			return err
		}

		li := omsDomain.SalesOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: order.ID, SKUID: &skuID, Name: "Widget",
			Quantity: 1, PriceMinorUnits: 500, CreatedAt: now, SoldExpiryDate: &expiry,
		}
		if err := lineItemRepo.Create(ctx, &li); err != nil {
			return err
		}
		lineItemID = li.ID

		// Most lines have no Woo expiry meta at all -- must round-trip nil,
		// not a zero-value date.
		noExpiry := omsDomain.SalesOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: order.ID, SKUID: &skuID, Name: "Gadget",
			Quantity: 1, PriceMinorUnits: 500, CreatedAt: now,
		}
		if err := lineItemRepo.Create(ctx, &noExpiry); err != nil {
			return err
		}
		noExpiryLineItemID = noExpiry.ID
		return nil
	})
	require.NoError(t, err)

	var found, foundNoExpiry *omsDomain.SalesOrderLineItem
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		if found, err = lineItemRepo.FindByID(ctx, lineItemID); err != nil {
			return err
		}
		foundNoExpiry, err = lineItemRepo.FindByID(ctx, noExpiryLineItemID)
		return err
	})
	require.NoError(t, err)
	require.NotNil(t, found.SoldExpiryDate)
	require.Equal(t, "2027-06-12", found.SoldExpiryDate.Format("2006-01-02"))
	require.Nil(t, foundNoExpiry.SoldExpiryDate)
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

func TestSalesOrderRepo_Integration_FindByNumberReturnsNilNotErrorWhenMissing(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	salesOrderRepo := repositories.NewSalesOrderRepo(db)

	var found *omsDomain.SalesOrder
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = salesOrderRepo.FindByNumber(ctx, "does-not-exist")
		return err
	})
	require.NoError(t, err)
	require.Nil(t, found)
}

func TestSalesOrderRepo_Integration_FindByNumberFindsTheOrder(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		order := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourceWhatsApp,
			Number: "WA-1A2B3C4D", ExternalOrderID: "wa-order-1", WarehouseID: warehouseID, CurrencyCode: "USD",
			OccurredAt: now, CreatedAt: now,
		}
		return salesOrderRepo.Create(ctx, &order)
	})
	require.NoError(t, err)

	var found *omsDomain.SalesOrder
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = salesOrderRepo.FindByNumber(ctx, "WA-1A2B3C4D")
		return err
	})
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "wa-order-1", found.ExternalOrderID)
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

// TestSalesOrderRepo_Integration_ListFiltersByAwaitingPacking proves the
// packing queue's filter against real Postgres: not packed, still
// pending/processing/on-hold (picked or not -- orders being picked right now
// included, the queue shows them locked), and not a POS sale. A completed or
// shipped order is out even though it's fulfilled and was never packed.
func TestSalesOrderRepo_Integration_ListFiltersByAwaitingPacking(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	pickingRepo := repositories.NewPickingSessionRepo(db)

	var pickedID, unpickedID, finishedPickingID, inProgressID, pausedID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		specs := []struct {
			extID     string
			status    omsDomain.SalesOrderStatus
			fulfilled bool
			packed    bool
			picking   omsDomain.PickingSessionStatus // "" means no session
			id        *string
		}{
			{"rtp-picked", omsDomain.SalesOrderStatusProcessing, true, false, "", &pickedID},
			{"rtp-unpicked", omsDomain.SalesOrderStatusPending, false, false, "", &unpickedID},
			{"rtp-picking-completed", omsDomain.SalesOrderStatusProcessing, false, false, omsDomain.PickingSessionStatusCompleted, &finishedPickingID},
			{"rtp-packed", omsDomain.SalesOrderStatusProcessing, true, true, "", nil},
			{"rtp-cancelled-unpicked", omsDomain.SalesOrderStatusCancelled, false, false, "", nil},
			{"rtp-picking-in-progress", omsDomain.SalesOrderStatusProcessing, false, false, omsDomain.PickingSessionStatusInProgress, &inProgressID},
			{"rtp-picking-paused", omsDomain.SalesOrderStatusOnHold, false, false, omsDomain.PickingSessionStatusPaused, &pausedID},
			{"rtp-completed-fulfilled", omsDomain.SalesOrderStatusCompleted, true, false, "", nil},
			{"rtp-shipped-fulfilled", omsDomain.SalesOrderStatusShipped, true, false, "", nil},
			{"rtp-pos", omsDomain.SalesOrderStatusProcessing, false, false, "", nil},
		}
		for _, s := range specs {
			source := omsDomain.SalesOrderSourceWooCommerce
			if s.extID == "rtp-pos" {
				source = omsDomain.SalesOrderSourcePOS
			}
			order := omsDomain.SalesOrder{
				TenantID: tenant.IDFromContext(ctx), Source: source,
				ExternalOrderID: s.extID, WarehouseID: warehouseID, CurrencyCode: "USD",
				Status: s.status, Fulfilled: s.fulfilled, Packed: s.packed, OccurredAt: now, CreatedAt: now,
			}
			if err := salesOrderRepo.Create(ctx, &order); err != nil {
				return err
			}
			if s.id != nil {
				*s.id = order.ID
			}
			if s.picking != "" {
				session := omsDomain.PickingSession{
					TenantID: tenant.IDFromContext(ctx), SalesOrderID: order.ID, Status: s.picking, StartedAt: now,
				}
				if err := pickingRepo.Create(ctx, &session); err != nil {
					return err
				}
			}
		}
		return nil
	})
	require.NoError(t, err)

	var ready []omsDomain.SalesOrder
	var total int64
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		ready, total, err = salesOrderRepo.List(ctx, omsDomain.ListSalesOrdersFilter{Limit: 50, AwaitingPacking: true})
		return err
	})
	require.NoError(t, err)
	ids := make([]string, len(ready))
	for i, o := range ready {
		ids[i] = o.ID
	}
	assert.ElementsMatch(t, []string{pickedID, unpickedID, finishedPickingID, inProgressID, pausedID}, ids)
	assert.Equal(t, int64(5), total)
}

func TestSalesOrderRepo_Integration_ListFiltersBySource(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)

	var posID, wooID, bolID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		specs := []struct {
			extID  string
			source omsDomain.SalesOrderSource
			id     *string
		}{
			{"source-filter-pos", omsDomain.SalesOrderSourcePOS, &posID},
			{"source-filter-woo", omsDomain.SalesOrderSourceWooCommerce, &wooID},
			{"source-filter-bol", omsDomain.SalesOrderSourceBol, &bolID},
		}
		for _, s := range specs {
			order := omsDomain.SalesOrder{
				TenantID: tenant.IDFromContext(ctx), Source: s.source,
				ExternalOrderID: s.extID, WarehouseID: warehouseID, CurrencyCode: "USD",
				Status: omsDomain.SalesOrderStatusProcessing, OccurredAt: now, CreatedAt: now,
			}
			if err := salesOrderRepo.Create(ctx, &order); err != nil {
				return err
			}
			*s.id = order.ID
		}
		return nil
	})
	require.NoError(t, err)

	var matched []omsDomain.SalesOrder
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		matched, _, err = salesOrderRepo.List(ctx, omsDomain.ListSalesOrdersFilter{
			Limit:  50,
			Source: []omsDomain.SalesOrderSource{omsDomain.SalesOrderSourcePOS, omsDomain.SalesOrderSourceWooCommerce},
		})
		return err
	})
	require.NoError(t, err)
	require.Len(t, matched, 2)
	gotIDs := []string{matched[0].ID, matched[1].ID}
	assert.ElementsMatch(t, []string{posID, wooID}, gotIDs)
}

func TestSalesOrderRepo_Integration_ListFiltersByCustomerID(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	customerRepo := repositories.NewCustomerRepo(db)

	var customerAID, customerBID string
	var matchingIDs []string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		customerA := customersDomain.Customer{TenantID: tenant.IDFromContext(ctx), Email: "a@example.com", DateCreated: now, DateModified: now}
		if err := customerRepo.Create(ctx, &customerA); err != nil {
			return err
		}
		customerB := customersDomain.Customer{TenantID: tenant.IDFromContext(ctx), Email: "b@example.com", DateCreated: now, DateModified: now}
		if err := customerRepo.Create(ctx, &customerB); err != nil {
			return err
		}
		customerAID, customerBID = customerA.ID, customerB.ID

		for _, spec := range []struct {
			extID      string
			customerID *string
		}{
			{"customer-filter-a-1", &customerAID},
			{"customer-filter-a-2", &customerAID},
			{"customer-filter-b-1", &customerBID},
			{"customer-filter-guest-1", nil},
		} {
			order := omsDomain.SalesOrder{
				TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS,
				ExternalOrderID: spec.extID, WarehouseID: warehouseID, CustomerID: spec.customerID, CurrencyCode: "USD",
				Status: omsDomain.SalesOrderStatusProcessing, OccurredAt: now, CreatedAt: now,
			}
			if err := salesOrderRepo.Create(ctx, &order); err != nil {
				return err
			}
			if spec.customerID != nil && *spec.customerID == customerAID {
				matchingIDs = append(matchingIDs, order.ID)
			}
		}
		return nil
	})
	require.NoError(t, err)

	var matched []omsDomain.SalesOrder
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		matched, _, err = salesOrderRepo.List(ctx, omsDomain.ListSalesOrdersFilter{Limit: 50, CustomerID: &customerAID})
		return err
	})
	require.NoError(t, err)
	require.Len(t, matched, 2)
	gotIDs := []string{matched[0].ID, matched[1].ID}
	assert.ElementsMatch(t, matchingIDs, gotIDs)
	for _, o := range matched {
		require.NotNil(t, o.CustomerID)
		assert.Equal(t, customerAID, *o.CustomerID)
	}
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

func TestSalesOrderRepo_Integration_RevenueByChannelBetween_GroupsBySourceWithinBoundedWindow(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)

	now := time.Now()
	outside := now.AddDate(0, 0, -30)
	seedDashboardOrder(t, db, ctx, salesOrderRepo, lineItemRepo, warehouseID, "rbc-1", omsDomain.SalesOrderSourcePOS, omsDomain.SalesOrderStatusCompleted, "EUR", now, "SKU-A", "A", 1, 1000)
	seedDashboardOrder(t, db, ctx, salesOrderRepo, lineItemRepo, warehouseID, "rbc-2", omsDomain.SalesOrderSourceWooCommerce, omsDomain.SalesOrderStatusCompleted, "EUR", now, "SKU-B", "B", 1, 700)
	// Outside [from, to) -- must not be counted, proving this is a bounded window and not RevenueByChannelSince's open-ended one.
	seedDashboardOrder(t, db, ctx, salesOrderRepo, lineItemRepo, warehouseID, "rbc-3", omsDomain.SalesOrderSourcePOS, omsDomain.SalesOrderStatusCompleted, "EUR", outside, "SKU-C", "C", 1, 9999)

	var rows []omsDomain.ChannelRevenue
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		rows, err = salesOrderRepo.RevenueByChannelBetween(ctx, now.AddDate(0, 0, -1), now.AddDate(0, 0, 1), "EUR")
		return err
	})
	require.NoError(t, err)

	bySource := make(map[omsDomain.SalesOrderSource]int64, len(rows))
	for _, r := range rows {
		bySource[r.Source] = r.TotalMinorUnits
	}
	require.Equal(t, int64(1000), bySource[omsDomain.SalesOrderSourcePOS])
	require.Equal(t, int64(700), bySource[omsDomain.SalesOrderSourceWooCommerce])
}

func TestSalesOrderRepo_Integration_RevenueByChannelTrendBetween_BucketsByDayAndSource(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)

	newer := time.Now()
	older := newer.AddDate(0, 0, -3) // far enough apart to land in different UTC day buckets regardless of local timezone
	seedDashboardOrder(t, db, ctx, salesOrderRepo, lineItemRepo, warehouseID, "rbct-1", omsDomain.SalesOrderSourcePOS, omsDomain.SalesOrderStatusCompleted, "EUR", newer, "SKU-A", "A", 1, 1000)
	seedDashboardOrder(t, db, ctx, salesOrderRepo, lineItemRepo, warehouseID, "rbct-2", omsDomain.SalesOrderSourceWooCommerce, omsDomain.SalesOrderStatusCompleted, "EUR", older, "SKU-B", "B", 1, 500)

	var points []omsDomain.ChannelRevenuePoint
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		points, err = salesOrderRepo.RevenueByChannelTrendBetween(ctx, older.AddDate(0, 0, -1), newer.AddDate(0, 0, 1), "EUR", omsDomain.SalesTrendGranularityDay)
		return err
	})
	require.NoError(t, err)
	require.Len(t, points, 2)

	require.True(t, points[0].PeriodStart.Before(points[1].PeriodStart), "points must be ordered oldest first")
	require.Equal(t, omsDomain.SalesOrderSourceWooCommerce, points[0].Source)
	require.Equal(t, int64(500), points[0].RevenueMinorUnits)
	require.Equal(t, omsDomain.SalesOrderSourcePOS, points[1].Source)
	require.Equal(t, int64(1000), points[1].RevenueMinorUnits)
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

func TestSalesOrderRepo_Integration_SalesSummaryBetween_AggregatesRevenueOrdersUnitsAndAOV(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)

	now := time.Now()
	before := now.AddDate(0, -1, 0)
	seedDashboardOrder(t, db, ctx, salesOrderRepo, lineItemRepo, warehouseID, "sum-1", omsDomain.SalesOrderSourcePOS, omsDomain.SalesOrderStatusCompleted, "EUR", now, "SKU-A", "A", 2, 1000)
	seedDashboardOrder(t, db, ctx, salesOrderRepo, lineItemRepo, warehouseID, "sum-2", omsDomain.SalesOrderSourcePOS, omsDomain.SalesOrderStatusProcessing, "EUR", now, "SKU-B", "B", 3, 500)
	// Excluded: cancelled (never reduced stock).
	seedDashboardOrder(t, db, ctx, salesOrderRepo, lineItemRepo, warehouseID, "sum-3", omsDomain.SalesOrderSourcePOS, omsDomain.SalesOrderStatusCancelled, "EUR", now, "SKU-C", "C", 1, 9999)
	// Excluded: different currency.
	seedDashboardOrder(t, db, ctx, salesOrderRepo, lineItemRepo, warehouseID, "sum-4", omsDomain.SalesOrderSourcePOS, omsDomain.SalesOrderStatusCompleted, "USD", now, "SKU-D", "D", 1, 9999)
	// Excluded: before the window.
	seedDashboardOrder(t, db, ctx, salesOrderRepo, lineItemRepo, warehouseID, "sum-5", omsDomain.SalesOrderSourcePOS, omsDomain.SalesOrderStatusCompleted, "EUR", before, "SKU-E", "E", 1, 9999)

	var summary omsDomain.SalesSummary
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		summary, err = salesOrderRepo.SalesSummaryBetween(ctx, now.AddDate(0, 0, -1), now.AddDate(0, 0, 1), "EUR")
		return err
	})
	require.NoError(t, err)
	require.Equal(t, int64(1500), summary.RevenueMinorUnits)
	require.Equal(t, int64(2), summary.OrderCount)
	require.Equal(t, int64(5), summary.UnitsSold)
	require.Equal(t, int64(750), summary.AverageOrderValueMinorUnits)
}

func TestSalesOrderRepo_Integration_SalesSummaryBetween_ZeroOrdersDoesNotDivideByZero(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	salesOrderRepo := repositories.NewSalesOrderRepo(db)

	var summary omsDomain.SalesSummary
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		summary, err = salesOrderRepo.SalesSummaryBetween(ctx, time.Now().AddDate(0, 0, -1), time.Now(), "EUR")
		return err
	})
	require.NoError(t, err)
	require.Equal(t, omsDomain.SalesSummary{}, summary)
}

func TestSalesOrderRepo_Integration_ProductPerformanceBetween_GroupsBySKUWithOrderCountAndSaleDates(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)

	earlier := time.Now().AddDate(0, 0, -2)
	later := time.Now().AddDate(0, 0, -1)
	before := time.Now().AddDate(0, -1, 0)
	seedDashboardOrder(t, db, ctx, salesOrderRepo, lineItemRepo, warehouseID, "perf-1", omsDomain.SalesOrderSourcePOS, omsDomain.SalesOrderStatusCompleted, "EUR", earlier, "SKU-A", "Acai", 2, 1000)
	seedDashboardOrder(t, db, ctx, salesOrderRepo, lineItemRepo, warehouseID, "perf-2", omsDomain.SalesOrderSourcePOS, omsDomain.SalesOrderStatusProcessing, "EUR", later, "SKU-A", "Acai", 3, 1500)
	// Excluded: cancelled.
	seedDashboardOrder(t, db, ctx, salesOrderRepo, lineItemRepo, warehouseID, "perf-3", omsDomain.SalesOrderSourcePOS, omsDomain.SalesOrderStatusCancelled, "EUR", later, "SKU-A", "Acai", 9, 9999)
	// Excluded: before the window.
	seedDashboardOrder(t, db, ctx, salesOrderRepo, lineItemRepo, warehouseID, "perf-4", omsDomain.SalesOrderSourcePOS, omsDomain.SalesOrderStatusCompleted, "EUR", before, "SKU-A", "Acai", 9, 9999)

	var rows []omsDomain.ProductPerformance
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		rows, err = salesOrderRepo.ProductPerformanceBetween(ctx, earlier.Add(-time.Hour), later.Add(time.Hour), "EUR")
		return err
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	row := rows[0]
	require.Equal(t, "SKU-A", row.SKUCode)
	require.Equal(t, int64(5), row.QuantitySold)
	require.Equal(t, int64(2500), row.RevenueMinorUnits)
	require.Equal(t, int64(2), row.OrderCount)
	require.WithinDuration(t, earlier, row.FirstSaleAt, time.Second)
	require.WithinDuration(t, later, row.LastSaleAt, time.Second)
}

func TestSalesOrderRepo_Integration_ProductPerformanceBetween_OrdersByRevenueDesc(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)

	now := time.Now()
	seedDashboardOrder(t, db, ctx, salesOrderRepo, lineItemRepo, warehouseID, "rank-1", omsDomain.SalesOrderSourcePOS, omsDomain.SalesOrderStatusCompleted, "EUR", now, "SKU-LOW", "Low", 1, 500)
	seedDashboardOrder(t, db, ctx, salesOrderRepo, lineItemRepo, warehouseID, "rank-2", omsDomain.SalesOrderSourcePOS, omsDomain.SalesOrderStatusCompleted, "EUR", now, "SKU-HIGH", "High", 1, 5000)

	var rows []omsDomain.ProductPerformance
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		rows, err = salesOrderRepo.ProductPerformanceBetween(ctx, now.AddDate(0, 0, -1), now.AddDate(0, 0, 1), "EUR")
		return err
	})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, "SKU-HIGH", rows[0].SKUCode)
	require.Equal(t, "SKU-LOW", rows[1].SKUCode)
}

func TestSalesOrderRepo_Integration_CustomerPerformanceBetween_AggregatesPerCustomerExcludingGuests(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	customerRepo := repositories.NewCustomerRepo(db)

	now := time.Now()
	before := now.AddDate(0, -1, 0)

	var customerID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		customer := customersDomain.Customer{
			TenantID: tenant.IDFromContext(ctx), Email: "jane@example.com", FirstName: "Jane", LastName: "Doe",
			DateCreated: now, DateModified: now,
		}
		if err := customerRepo.Create(ctx, &customer); err != nil {
			return err
		}
		customerID = customer.ID

		orders := []struct {
			extID    string
			status   omsDomain.SalesOrderStatus
			currency string
			occurred time.Time
			total    int64
			custID   *string
		}{
			{"cust-perf-1", omsDomain.SalesOrderStatusCompleted, "EUR", now, 1000, &customerID},
			{"cust-perf-2", omsDomain.SalesOrderStatusProcessing, "EUR", now, 500, &customerID},
			// Excluded: guest order (no customer_id).
			{"cust-perf-3", omsDomain.SalesOrderStatusCompleted, "EUR", now, 9999, nil},
			// Excluded: cancelled.
			{"cust-perf-4", omsDomain.SalesOrderStatusCancelled, "EUR", now, 9999, &customerID},
			// Excluded: before the window.
			{"cust-perf-5", omsDomain.SalesOrderStatusCompleted, "EUR", before, 9999, &customerID},
		}
		for _, o := range orders {
			order := omsDomain.SalesOrder{
				TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS, ExternalOrderID: o.extID,
				WarehouseID: warehouseID, CustomerID: o.custID, Status: o.status, CurrencyCode: o.currency,
				TotalMinorUnits: o.total, OccurredAt: o.occurred, CreatedAt: o.occurred,
			}
			if err := salesOrderRepo.Create(ctx, &order); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err)

	var rows []omsDomain.CustomerPerformance
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		rows, err = salesOrderRepo.CustomerPerformanceBetween(ctx, now.AddDate(0, 0, -1), now.AddDate(0, 0, 1), "EUR")
		return err
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	row := rows[0]
	require.Equal(t, customerID, row.CustomerID)
	require.Equal(t, "Jane Doe", row.CustomerName)
	require.Equal(t, int64(2), row.OrderCount)
	require.Equal(t, int64(1500), row.RevenueMinorUnits)
	require.Equal(t, int64(750), row.AverageOrderValueMinorUnits)
}

func TestSalesOrderRepo_Integration_CustomerPerformanceBetween_OrdersByRevenueDesc(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	customerRepo := repositories.NewCustomerRepo(db)

	now := time.Now()
	var lowID, highID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		low := customersDomain.Customer{TenantID: tenant.IDFromContext(ctx), Email: "low@example.com", FirstName: "Low", LastName: "Spender", DateCreated: now, DateModified: now}
		if err := customerRepo.Create(ctx, &low); err != nil {
			return err
		}
		lowID = low.ID
		high := customersDomain.Customer{TenantID: tenant.IDFromContext(ctx), Email: "high@example.com", FirstName: "High", LastName: "Spender", DateCreated: now, DateModified: now}
		if err := customerRepo.Create(ctx, &high); err != nil {
			return err
		}
		highID = high.ID

		if err := salesOrderRepo.Create(ctx, &omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS, ExternalOrderID: "rank-cust-1",
			WarehouseID: warehouseID, CustomerID: &lowID, Status: omsDomain.SalesOrderStatusCompleted, CurrencyCode: "EUR",
			TotalMinorUnits: 500, OccurredAt: now, CreatedAt: now,
		}); err != nil {
			return err
		}
		return salesOrderRepo.Create(ctx, &omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS, ExternalOrderID: "rank-cust-2",
			WarehouseID: warehouseID, CustomerID: &highID, Status: omsDomain.SalesOrderStatusCompleted, CurrencyCode: "EUR",
			TotalMinorUnits: 5000, OccurredAt: now, CreatedAt: now,
		})
	})
	require.NoError(t, err)

	var rows []omsDomain.CustomerPerformance
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		rows, err = salesOrderRepo.CustomerPerformanceBetween(ctx, now.AddDate(0, 0, -1), now.AddDate(0, 0, 1), "EUR")
		return err
	})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, highID, rows[0].CustomerID)
	require.Equal(t, lowID, rows[1].CustomerID)
}

func TestSalesOrderRepo_Integration_CategoryPerformanceBetween_GroupsByCategoryAndRollsUpUncategorized(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)
	categoryRepo := repositories.NewCategoryRepo(db)
	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)

	now := time.Now()
	var beveragesSKUID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		category := pmsDomain.Category{TenantID: tenant.IDFromContext(ctx), Name: "Beverages", Slug: "beverages-cat-perf", CreatedAt: now, UpdatedAt: now}
		if err := categoryRepo.Create(ctx, &category); err != nil {
			return err
		}
		product := &pmsDomain.Product{TenantID: tenant.IDFromContext(ctx), CategoryID: &category.ID, Name: "Acai", Slug: "acai-cat-perf", Status: pmsDomain.ProductStatusActive, CreatedAt: now, UpdatedAt: now}
		if err := productRepo.Create(ctx, product); err != nil {
			return err
		}
		variant := &pmsDomain.Variant{TenantID: tenant.IDFromContext(ctx), ProductID: product.ID, Name: "Default", CreatedAt: now, UpdatedAt: now}
		if err := variantRepo.Create(ctx, variant); err != nil {
			return err
		}
		sku := &pmsDomain.SKU{TenantID: tenant.IDFromContext(ctx), VariantID: variant.ID, SKUCode: "ACAI-CAT-PERF-1", PriceMinorUnits: 999, CurrencyCode: "EUR", Status: pmsDomain.SKUStatusActive, CreatedAt: now, UpdatedAt: now}
		if err := skuRepo.Create(ctx, sku); err != nil {
			return err
		}
		beveragesSKUID = sku.ID
		return nil
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		order1 := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS, ExternalOrderID: "cat-perf-1", WarehouseID: warehouseID,
			Status: omsDomain.SalesOrderStatusCompleted, CurrencyCode: "EUR", TotalMinorUnits: 1000, OccurredAt: now, CreatedAt: now,
		}
		if err := salesOrderRepo.Create(ctx, &order1); err != nil {
			return err
		}
		if err := lineItemRepo.Create(ctx, &omsDomain.SalesOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: order1.ID, SKUID: &beveragesSKUID, Name: "Acai", SKUCode: "ACAI-CAT-PERF-1",
			Quantity: 2, TotalMinorUnits: 1000, CreatedAt: now,
		}); err != nil {
			return err
		}

		// Adhoc line item (no linked SKU) -- must roll into "Uncategorized".
		order2 := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS, ExternalOrderID: "cat-perf-2", WarehouseID: warehouseID,
			Status: omsDomain.SalesOrderStatusCompleted, CurrencyCode: "EUR", TotalMinorUnits: 500, OccurredAt: now, CreatedAt: now,
		}
		if err := salesOrderRepo.Create(ctx, &order2); err != nil {
			return err
		}
		return lineItemRepo.Create(ctx, &omsDomain.SalesOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: order2.ID, Name: "Mystery Item", SKUCode: "UNKNOWN",
			Quantity: 1, TotalMinorUnits: 500, CreatedAt: now,
		})
	})
	require.NoError(t, err)

	var rows []omsDomain.CategoryPerformance
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		rows, err = salesOrderRepo.CategoryPerformanceBetween(ctx, now.AddDate(0, 0, -1), now.AddDate(0, 0, 1), "EUR")
		return err
	})
	require.NoError(t, err)
	require.Len(t, rows, 2)

	byName := make(map[string]omsDomain.CategoryPerformance, len(rows))
	for _, r := range rows {
		byName[r.CategoryName] = r
	}
	beverages := byName["Beverages"]
	require.NotNil(t, beverages.CategoryID)
	require.Equal(t, int64(2), beverages.QuantitySold)
	require.Equal(t, int64(1000), beverages.RevenueMinorUnits)
	require.Equal(t, int64(1), beverages.OrderCount)

	uncategorized := byName["Uncategorized"]
	require.Nil(t, uncategorized.CategoryID)
	require.Equal(t, int64(1), uncategorized.QuantitySold)
	require.Equal(t, int64(500), uncategorized.RevenueMinorUnits)
}

func TestSalesOrderRepo_Integration_ProfitabilityBetween_ComputesProfitOnlyOverKnownCostRevenue(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)
	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)

	now := time.Now()
	costKnown := int64(300)
	var costedSKUID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		product := &pmsDomain.Product{TenantID: tenant.IDFromContext(ctx), Name: "Acai", Slug: "acai-profitability", Status: pmsDomain.ProductStatusActive, CreatedAt: now, UpdatedAt: now}
		if err := productRepo.Create(ctx, product); err != nil {
			return err
		}
		variant := &pmsDomain.Variant{TenantID: tenant.IDFromContext(ctx), ProductID: product.ID, Name: "Default", CreatedAt: now, UpdatedAt: now}
		if err := variantRepo.Create(ctx, variant); err != nil {
			return err
		}
		sku := &pmsDomain.SKU{
			TenantID: tenant.IDFromContext(ctx), VariantID: variant.ID, SKUCode: "ACAI-PROFIT-1", PriceMinorUnits: 999,
			CurrencyCode: "EUR", Status: pmsDomain.SKUStatusActive, CreatedAt: now, UpdatedAt: now,
		}
		if err := skuRepo.Create(ctx, sku); err != nil {
			return err
		}
		costedSKUID = sku.ID
		// SKURepo.Create deliberately never writes cost_price_minor_units
		// (only the cron/manual-override paths may) -- set it via the same
		// dedicated method they use.
		return skuRepo.UpdateCostPrice(ctx, sku.ID, costKnown)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		// Known-cost line: revenue 1000, qty 2, cost 300*2=600 -> profit 400.
		order1 := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS, ExternalOrderID: "profit-1", WarehouseID: warehouseID,
			Status: omsDomain.SalesOrderStatusCompleted, CurrencyCode: "EUR", TotalMinorUnits: 1000, OccurredAt: now, CreatedAt: now,
		}
		if err := salesOrderRepo.Create(ctx, &order1); err != nil {
			return err
		}
		if err := lineItemRepo.Create(ctx, &omsDomain.SalesOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: order1.ID, SKUID: &costedSKUID, Name: "Acai", SKUCode: "ACAI-PROFIT-1",
			Quantity: 2, TotalMinorUnits: 1000, CreatedAt: now,
		}); err != nil {
			return err
		}

		// Unknown-cost line (adhoc, no SKU): revenue 500, must count toward
		// RevenueMinorUnits but NOT toward CostKnownRevenueMinorUnits/profit.
		order2 := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS, ExternalOrderID: "profit-2", WarehouseID: warehouseID,
			Status: omsDomain.SalesOrderStatusCompleted, CurrencyCode: "EUR", TotalMinorUnits: 500, OccurredAt: now, CreatedAt: now,
		}
		if err := salesOrderRepo.Create(ctx, &order2); err != nil {
			return err
		}
		return lineItemRepo.Create(ctx, &omsDomain.SalesOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: order2.ID, Name: "Mystery Item", SKUCode: "UNKNOWN",
			Quantity: 1, TotalMinorUnits: 500, CreatedAt: now,
		})
	})
	require.NoError(t, err)

	var summary omsDomain.ProfitabilitySummary
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		summary, err = salesOrderRepo.ProfitabilityBetween(ctx, now.AddDate(0, 0, -1), now.AddDate(0, 0, 1), "EUR")
		return err
	})
	require.NoError(t, err)
	require.Equal(t, int64(1500), summary.RevenueMinorUnits)
	require.Equal(t, int64(1000), summary.CostKnownRevenueMinorUnits)
	require.Equal(t, int64(600), summary.CostMinorUnits)
	require.Equal(t, int64(400), summary.ProfitMinorUnits)
	require.Equal(t, int64(4000), summary.MarginBasisPoints) // 400/1000 = 40.00%
}

func TestSalesOrderRepo_Integration_ProfitabilityBetween_ZeroKnownCostRevenueGivesZeroMargin(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	salesOrderRepo := repositories.NewSalesOrderRepo(db)

	var summary omsDomain.ProfitabilitySummary
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		summary, err = salesOrderRepo.ProfitabilityBetween(ctx, time.Now().AddDate(0, 0, -1), time.Now(), "EUR")
		return err
	})
	require.NoError(t, err)
	require.Equal(t, omsDomain.ProfitabilitySummary{}, summary)
}

func TestSalesOrderRepo_Integration_ProfitabilityTrendBetween_BucketsByDayComputingProfitOnlyOverKnownCostRevenue(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)
	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)

	newer := time.Now()
	older := newer.AddDate(0, 0, -3) // far enough apart to land in different UTC day buckets regardless of local timezone
	costKnown := int64(300)
	var costedSKUID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		product := &pmsDomain.Product{TenantID: tenant.IDFromContext(ctx), Name: "Acai", Slug: "acai-profitability-trend", Status: pmsDomain.ProductStatusActive, CreatedAt: newer, UpdatedAt: newer}
		if err := productRepo.Create(ctx, product); err != nil {
			return err
		}
		variant := &pmsDomain.Variant{TenantID: tenant.IDFromContext(ctx), ProductID: product.ID, Name: "Default", CreatedAt: newer, UpdatedAt: newer}
		if err := variantRepo.Create(ctx, variant); err != nil {
			return err
		}
		sku := &pmsDomain.SKU{
			TenantID: tenant.IDFromContext(ctx), VariantID: variant.ID, SKUCode: "ACAI-PROFIT-TREND-1", PriceMinorUnits: 999,
			CurrencyCode: "EUR", Status: pmsDomain.SKUStatusActive, CreatedAt: newer, UpdatedAt: newer,
		}
		if err := skuRepo.Create(ctx, sku); err != nil {
			return err
		}
		costedSKUID = sku.ID
		return skuRepo.UpdateCostPrice(ctx, sku.ID, costKnown)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		// Newer day, known cost: revenue 1000, qty 2, cost 300*2=600 -> profit 400.
		orderNew := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS, ExternalOrderID: "profit-trend-new", WarehouseID: warehouseID,
			Status: omsDomain.SalesOrderStatusCompleted, CurrencyCode: "EUR", TotalMinorUnits: 1000, OccurredAt: newer, CreatedAt: newer,
		}
		if err := salesOrderRepo.Create(ctx, &orderNew); err != nil {
			return err
		}
		if err := lineItemRepo.Create(ctx, &omsDomain.SalesOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderNew.ID, SKUID: &costedSKUID, Name: "Acai", SKUCode: "ACAI-PROFIT-TREND-1",
			Quantity: 2, TotalMinorUnits: 1000, CreatedAt: newer,
		}); err != nil {
			return err
		}

		// Older day, unknown cost (adhoc): revenue 500, must count toward
		// revenue but not toward cost-known revenue/profit for that bucket.
		orderOld := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS, ExternalOrderID: "profit-trend-old", WarehouseID: warehouseID,
			Status: omsDomain.SalesOrderStatusCompleted, CurrencyCode: "EUR", TotalMinorUnits: 500, OccurredAt: older, CreatedAt: older,
		}
		if err := salesOrderRepo.Create(ctx, &orderOld); err != nil {
			return err
		}
		return lineItemRepo.Create(ctx, &omsDomain.SalesOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderOld.ID, Name: "Mystery Item", SKUCode: "UNKNOWN",
			Quantity: 1, TotalMinorUnits: 500, CreatedAt: older,
		})
	})
	require.NoError(t, err)

	var points []omsDomain.ProfitabilityPoint
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		points, err = salesOrderRepo.ProfitabilityTrendBetween(ctx, older.AddDate(0, 0, -1), newer.AddDate(0, 0, 1), "EUR", omsDomain.SalesTrendGranularityDay)
		return err
	})
	require.NoError(t, err)
	require.Len(t, points, 2)

	require.True(t, points[0].PeriodStart.Before(points[1].PeriodStart), "points must be ordered oldest first")
	require.Equal(t, int64(500), points[0].RevenueMinorUnits)
	require.Equal(t, int64(0), points[0].CostKnownRevenueMinorUnits)
	require.Equal(t, int64(0), points[0].ProfitMinorUnits)
	require.Equal(t, int64(0), points[0].MarginBasisPoints)
	require.Equal(t, int64(1000), points[1].RevenueMinorUnits)
	require.Equal(t, int64(1000), points[1].CostKnownRevenueMinorUnits)
	require.Equal(t, int64(600), points[1].CostMinorUnits)
	require.Equal(t, int64(400), points[1].ProfitMinorUnits)
	require.Equal(t, int64(4000), points[1].MarginBasisPoints) // 400/1000 = 40.00%
}

func TestSalesOrderRepo_Integration_SalesTrendBetween_BucketsByDay(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)

	newer := time.Now()
	older := newer.AddDate(0, 0, -3) // far enough apart to land in different UTC day buckets regardless of local timezone
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		orderNew := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS, ExternalOrderID: "trend-new", WarehouseID: warehouseID,
			Status: omsDomain.SalesOrderStatusCompleted, CurrencyCode: "EUR", TotalMinorUnits: 1000, OccurredAt: newer, CreatedAt: newer,
		}
		if err := salesOrderRepo.Create(ctx, &orderNew); err != nil {
			return err
		}
		if err := lineItemRepo.Create(ctx, &omsDomain.SalesOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderNew.ID, Name: "Item", SKUCode: "ITEM-1", Quantity: 2, TotalMinorUnits: 1000, CreatedAt: newer,
		}); err != nil {
			return err
		}

		orderOld := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS, ExternalOrderID: "trend-old", WarehouseID: warehouseID,
			Status: omsDomain.SalesOrderStatusCompleted, CurrencyCode: "EUR", TotalMinorUnits: 500, OccurredAt: older, CreatedAt: older,
		}
		if err := salesOrderRepo.Create(ctx, &orderOld); err != nil {
			return err
		}
		return lineItemRepo.Create(ctx, &omsDomain.SalesOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderOld.ID, Name: "Item", SKUCode: "ITEM-1", Quantity: 3, TotalMinorUnits: 500, CreatedAt: older,
		})
	})
	require.NoError(t, err)

	var points []omsDomain.SalesTrendPoint
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		points, err = salesOrderRepo.SalesTrendBetween(ctx, older.AddDate(0, 0, -1), newer.AddDate(0, 0, 1), "EUR", omsDomain.SalesTrendGranularityDay)
		return err
	})
	require.NoError(t, err)
	require.Len(t, points, 2)

	require.True(t, points[0].PeriodStart.Before(points[1].PeriodStart), "points must be ordered oldest first")
	require.Equal(t, int64(500), points[0].RevenueMinorUnits)
	require.Equal(t, int64(1), points[0].OrderCount)
	require.Equal(t, int64(3), points[0].UnitsSold)
	require.Equal(t, int64(1000), points[1].RevenueMinorUnits)
	require.Equal(t, int64(1), points[1].OrderCount)
	require.Equal(t, int64(2), points[1].UnitsSold)
}

func TestSalesOrderRepo_Integration_SalesTrendBetween_RejectsEmptyPeriod(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	salesOrderRepo := repositories.NewSalesOrderRepo(db)

	var points []omsDomain.SalesTrendPoint
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		points, err = salesOrderRepo.SalesTrendBetween(ctx, time.Now().AddDate(0, 0, -1), time.Now(), "EUR", omsDomain.SalesTrendGranularityDay)
		return err
	})
	require.NoError(t, err)
	require.Empty(t, points)
}

func TestSalesOrderRepo_Integration_SalesByWeekdayBetween_SumsAcrossOccurrencesOfSameWeekday(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)

	week1 := time.Now()
	week2 := week1.AddDate(0, 0, 7) // same weekday, one week later
	otherWeekday := week1.AddDate(0, 0, 1)
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		for i, order := range []omsDomain.SalesOrder{
			{TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS, ExternalOrderID: "wd-1", WarehouseID: warehouseID, Status: omsDomain.SalesOrderStatusCompleted, CurrencyCode: "EUR", TotalMinorUnits: 700, OccurredAt: week1, CreatedAt: week1},
			{TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS, ExternalOrderID: "wd-2", WarehouseID: warehouseID, Status: omsDomain.SalesOrderStatusCompleted, CurrencyCode: "EUR", TotalMinorUnits: 300, OccurredAt: week2, CreatedAt: week2},
			{TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS, ExternalOrderID: "wd-3", WarehouseID: warehouseID, Status: omsDomain.SalesOrderStatusCompleted, CurrencyCode: "EUR", TotalMinorUnits: 200, OccurredAt: otherWeekday, CreatedAt: otherWeekday},
		} {
			order := order
			if err := salesOrderRepo.Create(ctx, &order); err != nil {
				return fmt.Errorf("order %d: %w", i, err)
			}
		}
		return nil
	})
	require.NoError(t, err)

	var weekdays []omsDomain.WeekdaySales
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		weekdays, err = salesOrderRepo.SalesByWeekdayBetween(ctx, week1.AddDate(0, 0, -1), week2.AddDate(0, 0, 1), "EUR")
		return err
	})
	require.NoError(t, err)
	require.Len(t, weekdays, 2)

	byWeekday := make(map[time.Weekday]omsDomain.WeekdaySales, len(weekdays))
	for _, w := range weekdays {
		byWeekday[w.Weekday] = w
	}
	sharedWeekday := byWeekday[week1.Weekday()]
	require.Equal(t, int64(1000), sharedWeekday.RevenueMinorUnits)
	require.Equal(t, int64(2), sharedWeekday.OrderCount)

	other := byWeekday[otherWeekday.Weekday()]
	require.Equal(t, int64(200), other.RevenueMinorUnits)
	require.Equal(t, int64(1), other.OrderCount)
}

func firstOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func TestSalesOrderRepo_Integration_CohortSizesBetween_CountsCustomersByFirstOrderMonth(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	customerRepo := repositories.NewCustomerRepo(db)

	cohortMonth := firstOfMonth(time.Now())
	inWindowA := cohortMonth.AddDate(0, 0, 2)
	inWindowB := cohortMonth.AddDate(0, 0, 5)
	priorMonthFirstOrder := cohortMonth.AddDate(0, -1, 10)
	alsoInWindow := cohortMonth.AddDate(0, 0, 3)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		custA := customersDomain.Customer{TenantID: tenant.IDFromContext(ctx), Email: "cohort-a@example.com", FirstName: "A", DateCreated: inWindowA, DateModified: inWindowA}
		if err := customerRepo.Create(ctx, &custA); err != nil {
			return err
		}
		custB := customersDomain.Customer{TenantID: tenant.IDFromContext(ctx), Email: "cohort-b@example.com", FirstName: "B", DateCreated: inWindowB, DateModified: inWindowB}
		if err := customerRepo.Create(ctx, &custB); err != nil {
			return err
		}
		custC := customersDomain.Customer{TenantID: tenant.IDFromContext(ctx), Email: "cohort-c@example.com", FirstName: "C", DateCreated: priorMonthFirstOrder, DateModified: priorMonthFirstOrder}
		if err := customerRepo.Create(ctx, &custC); err != nil {
			return err
		}

		for _, o := range []struct {
			extID    string
			custID   string
			occurred time.Time
		}{
			{"cohort-a-1", custA.ID, inWindowA},
			{"cohort-b-1", custB.ID, inWindowB},
			// C's first-ever order predates the window -- must NOT count
			// toward cohortMonth's cohort, even though C also orders
			// again inside the window.
			{"cohort-c-1", custC.ID, priorMonthFirstOrder},
			{"cohort-c-2", custC.ID, alsoInWindow},
		} {
			custID := o.custID
			if err := salesOrderRepo.Create(ctx, &omsDomain.SalesOrder{
				TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS, ExternalOrderID: o.extID,
				WarehouseID: warehouseID, CustomerID: &custID, Status: omsDomain.SalesOrderStatusCompleted, CurrencyCode: "EUR",
				TotalMinorUnits: 100, OccurredAt: o.occurred, CreatedAt: o.occurred,
			}); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err)

	var sizes []omsDomain.CohortSize
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		sizes, err = salesOrderRepo.CohortSizesBetween(ctx, cohortMonth, cohortMonth.AddDate(0, 1, 0), "EUR")
		return err
	})
	require.NoError(t, err)
	require.Len(t, sizes, 1)
	require.True(t, sizes[0].CohortMonth.Equal(cohortMonth))
	require.Equal(t, int64(2), sizes[0].CustomerCount)
}

func TestSalesOrderRepo_Integration_CohortRetentionBetween_TracksRepeatOrdersAcrossMonths(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	customerRepo := repositories.NewCustomerRepo(db)

	cohortMonth := firstOfMonth(time.Now())
	firstOrderAt := cohortMonth.AddDate(0, 0, 1)
	repeatOrderAt := cohortMonth.AddDate(0, 1, 3)

	var custID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		customer := customersDomain.Customer{TenantID: tenant.IDFromContext(ctx), Email: "retained@example.com", FirstName: "R", DateCreated: firstOrderAt, DateModified: firstOrderAt}
		if err := customerRepo.Create(ctx, &customer); err != nil {
			return err
		}
		custID = customer.ID

		if err := salesOrderRepo.Create(ctx, &omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS, ExternalOrderID: "retain-1",
			WarehouseID: warehouseID, CustomerID: &custID, Status: omsDomain.SalesOrderStatusCompleted, CurrencyCode: "EUR",
			TotalMinorUnits: 1000, OccurredAt: firstOrderAt, CreatedAt: firstOrderAt,
		}); err != nil {
			return err
		}
		return salesOrderRepo.Create(ctx, &omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS, ExternalOrderID: "retain-2",
			WarehouseID: warehouseID, CustomerID: &custID, Status: omsDomain.SalesOrderStatusCompleted, CurrencyCode: "EUR",
			TotalMinorUnits: 400, OccurredAt: repeatOrderAt, CreatedAt: repeatOrderAt,
		})
	})
	require.NoError(t, err)

	var points []omsDomain.CohortRetentionPoint
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		points, err = salesOrderRepo.CohortRetentionBetween(ctx, cohortMonth, cohortMonth.AddDate(0, 2, 0), "EUR")
		return err
	})
	require.NoError(t, err)
	require.Len(t, points, 2)

	byMonth := make(map[int64]omsDomain.CohortRetentionPoint, len(points))
	for _, p := range points {
		require.True(t, p.CohortMonth.Equal(cohortMonth))
		byMonth[p.MonthsSinceCohort] = p
	}
	require.Equal(t, int64(1), byMonth[0].RetainedCustomers)
	require.Equal(t, int64(1000), byMonth[0].RevenueMinorUnits)
	require.Equal(t, int64(1), byMonth[1].RetainedCustomers)
	require.Equal(t, int64(400), byMonth[1].RevenueMinorUnits)
}

func TestSalesOrderRepo_Integration_SKUSetSalesBetween_SumsOnlyGivenSKUsWithinWindow(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)

	now := time.Now()
	outsideWindow := now.AddDate(0, 0, -3)

	var skuA, skuB, skuC string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		skuA, err = seedPromotionSKU(ctx, productRepo, variantRepo, skuRepo, "SET-A")
		if err != nil {
			return err
		}
		skuB, err = seedPromotionSKU(ctx, productRepo, variantRepo, skuRepo, "SET-B")
		if err != nil {
			return err
		}
		skuC, err = seedPromotionSKU(ctx, productRepo, variantRepo, skuRepo, "SET-C")
		return err
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		for i, o := range []struct {
			extID    string
			skuID    string
			qty      int64
			total    int64
			occurred time.Time
		}{
			{"set-1", skuA, 2, 1000, now},
			{"set-2", skuB, 1, 500, now},
			{"set-3", skuC, 5, 2000, now},          // excluded: not in the given SKU set
			{"set-4", skuA, 9, 999, outsideWindow}, // excluded: outside the window
		} {
			order := omsDomain.SalesOrder{
				TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS, ExternalOrderID: o.extID,
				WarehouseID: warehouseID, Status: omsDomain.SalesOrderStatusCompleted, CurrencyCode: "EUR",
				TotalMinorUnits: o.total, OccurredAt: o.occurred, CreatedAt: o.occurred,
			}
			if err := salesOrderRepo.Create(ctx, &order); err != nil {
				return fmt.Errorf("order %d: %w", i, err)
			}
			skuID := o.skuID
			if err := lineItemRepo.Create(ctx, &omsDomain.SalesOrderLineItem{
				TenantID: tenant.IDFromContext(ctx), SalesOrderID: order.ID, SKUID: &skuID, Name: "Item", SKUCode: "ITEM",
				Quantity: o.qty, TotalMinorUnits: o.total, CreatedAt: o.occurred,
			}); err != nil {
				return fmt.Errorf("line %d: %w", i, err)
			}
		}
		return nil
	})
	require.NoError(t, err)

	var summary omsDomain.SKUSetSalesSummary
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		summary, err = salesOrderRepo.SKUSetSalesBetween(ctx, []string{skuA, skuB}, now.AddDate(0, 0, -1), now.AddDate(0, 0, 1), "EUR")
		return err
	})
	require.NoError(t, err)
	require.Equal(t, int64(1500), summary.RevenueMinorUnits)
	require.Equal(t, int64(3), summary.UnitsSold)
	require.Equal(t, int64(2), summary.OrderCount)
}

func TestSalesOrderRepo_Integration_SKUSetSalesBetween_EmptySKUSetGivesZero(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	salesOrderRepo := repositories.NewSalesOrderRepo(db)

	var summary omsDomain.SKUSetSalesSummary
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		summary, err = salesOrderRepo.SKUSetSalesBetween(ctx, []string{}, time.Now().AddDate(0, 0, -1), time.Now(), "EUR")
		return err
	})
	require.NoError(t, err)
	require.Equal(t, omsDomain.SKUSetSalesSummary{}, summary)
}

func TestSalesOrderRepo_Integration_ListOrderSKUsBetween_ReturnsDistinctSKUsPerQualifyingOrder(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)

	now := time.Now()
	outsideWindow := now.AddDate(0, 0, -3)

	var skuA, skuB, skuC string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		skuA, err = seedPromotionSKU(ctx, productRepo, variantRepo, skuRepo, "FBT-A")
		if err != nil {
			return err
		}
		skuB, err = seedPromotionSKU(ctx, productRepo, variantRepo, skuRepo, "FBT-B")
		if err != nil {
			return err
		}
		skuC, err = seedPromotionSKU(ctx, productRepo, variantRepo, skuRepo, "FBT-C")
		return err
	})
	require.NoError(t, err)

	var order1ID, order2ID, order3ID, order4ID string
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		createOrderWithLines := func(extID string, occurred time.Time, skuIDs ...string) (string, error) {
			order := omsDomain.SalesOrder{
				TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS, ExternalOrderID: extID,
				WarehouseID: warehouseID, Status: omsDomain.SalesOrderStatusCompleted, CurrencyCode: "EUR",
				TotalMinorUnits: 100, OccurredAt: occurred, CreatedAt: occurred,
			}
			if err := salesOrderRepo.Create(ctx, &order); err != nil {
				return "", err
			}
			for _, skuID := range skuIDs {
				skuID := skuID
				if err := lineItemRepo.Create(ctx, &omsDomain.SalesOrderLineItem{
					TenantID: tenant.IDFromContext(ctx), SalesOrderID: order.ID, SKUID: &skuID, Name: "Item", SKUCode: "ITEM",
					Quantity: 1, TotalMinorUnits: 100, CreatedAt: occurred,
				}); err != nil {
					return "", err
				}
			}
			return order.ID, nil
		}

		var err error
		order1ID, err = createOrderWithLines("fbt-1", now, skuA, skuB)
		if err != nil {
			return err
		}
		order2ID, err = createOrderWithLines("fbt-2", now, skuA, skuB, skuC)
		if err != nil {
			return err
		}
		order3ID, err = createOrderWithLines("fbt-3", now, skuC)
		if err != nil {
			return err
		}
		// Same SKU across two line item rows -- must dedupe to one entry.
		order4ID, err = createOrderWithLines("fbt-4", now, skuA, skuA)
		if err != nil {
			return err
		}
		_, err = createOrderWithLines("fbt-5", outsideWindow, skuA, skuB) // excluded: outside window
		return err
	})
	require.NoError(t, err)

	var orders []omsDomain.OrderSKUs
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		orders, err = salesOrderRepo.ListOrderSKUsBetween(ctx, now.AddDate(0, 0, -1), now.AddDate(0, 0, 1), "EUR")
		return err
	})
	require.NoError(t, err)
	require.Len(t, orders, 4)

	byOrderID := map[string]omsDomain.OrderSKUs{}
	for _, o := range orders {
		byOrderID[o.OrderID] = o
	}
	require.ElementsMatch(t, []string{skuA, skuB}, byOrderID[order1ID].SKUIDs)
	require.ElementsMatch(t, []string{skuA, skuB, skuC}, byOrderID[order2ID].SKUIDs)
	require.ElementsMatch(t, []string{skuC}, byOrderID[order3ID].SKUIDs)
	require.ElementsMatch(t, []string{skuA}, byOrderID[order4ID].SKUIDs, "duplicate SKU across two line items must dedupe")
}
