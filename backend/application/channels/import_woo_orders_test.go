package channels_test

import (
	"context"
	"errors"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/crypto"
)

type ordersHarness struct {
	origins        *stubImportedOrigins
	connRepo       *stubConnectionRepo
	wooFactory     *stubWooClientFactory
	wooClient      *stubWooClient
	salesOrderRepo *stubOrdersSalesOrderRepo
	lineItemRepo   *stubOrdersLineItemRepo
	linkRepo       *stubOrdersLinkRepo
	skuRepo        *stubChannelsSKURepo
	customerRepo   *stubOrdersCustomerRepo
	stock          *stubSaleStockAllocator
	updateOrder    *stubUpdateSalesOrderUC
}

func newOrdersHarness(orders map[int][]integrationsDomain.WooOrder) *ordersHarness {
	wooClient := &stubWooClient{ordersByPage: orders}
	encrypted, err := crypto.Encrypt(testEncryptionSecret, "consumer-secret")
	if err != nil {
		panic(err)
	}
	return &ordersHarness{
		connRepo: &stubConnectionRepo{conn: &integrationsDomain.Connection{
			ID: "conn-1", Provider: integrationsDomain.ProviderWooCommerce,
			StoreURL: "https://mystore.com", ConsumerKey: "ck", ConsumerSecretEncrypted: encrypted,
		}},
		wooFactory:     &stubWooClientFactory{client: wooClient},
		wooClient:      wooClient,
		salesOrderRepo: &stubOrdersSalesOrderRepo{byExternalID: map[string]omsDomain.SalesOrder{}},
		lineItemRepo:   &stubOrdersLineItemRepo{},
		linkRepo:       &stubOrdersLinkRepo{},
		skuRepo:        &stubChannelsSKURepo{byCode: map[string]pmsDomain.SKU{}},
		customerRepo:   &stubOrdersCustomerRepo{byWooID: map[int64]customersDomain.Customer{}, byEmail: map[string]customersDomain.Customer{}},
		stock:          &stubSaleStockAllocator{},
		updateOrder:    &stubUpdateSalesOrderUC{},
	}
}

func (h *ordersHarness) newImportUseCase(t *testing.T, n int) *appchannels.ImportWooOrdersUseCase {
	t.Helper()
	h.origins = &stubImportedOrigins{orders: h.salesOrderRepo}
	return appchannels.NewImportWooOrdersUseCase(
		newMockGormDB(t, n), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.salesOrderRepo, h.origins, h.lineItemRepo, h.linkRepo, h.skuRepo, h.customerRepo, h.stock, h.updateOrder, &eventrecordtest.Outbox{},
	)
}

func simpleWooOrder() integrationsDomain.WooOrder {
	return integrationsDomain.WooOrder{
		ID: 501, Number: "501", OrderKey: "wc_order_abc", Status: "processing",
		Billing:  integrationsDomain.WooAddress{FirstName: "Jane", LastName: "Doe", Email: "jane@example.com"},
		Shipping: integrationsDomain.WooAddress{FirstName: "Jane", LastName: "Doe"},
		Currency: "USD", Total: "20.00", DiscountTotal: "0.00", ShippingTotal: "0.00", TotalTax: "0.00",
		ShippingMethod: "flat_rate", ShippingMethodTitle: "Flat rate",
		DateCreated: "2026-01-01T10:00:00", DateModified: "2026-01-01T10:00:00",
		LineItems: []integrationsDomain.WooOrderLineItem{
			{ProductID: 12, SKU: "WID-A", Name: "Widget A", Quantity: 2, Price: "10", Subtotal: "20.00", Total: "20.00"},
		},
	}
}

func ptrWooOrder(wo integrationsDomain.WooOrder) *integrationsDomain.WooOrder { return &wo }

func TestImportWooOrders_createsANewOrderAndDecrementsStock(t *testing.T) {
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {simpleWooOrder()}})
	taxClassID := "tax-class-standard"
	h.skuRepo.byCode["WID-A"] = pmsDomain.SKU{ID: "sku-1", SKUCode: "WID-A", TaxClassID: &taxClassID}
	uc := h.newImportUseCase(t, 4)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Created)
	assert.Empty(t, result.Errors)
	require.Len(t, h.salesOrderRepo.created, 1)
	created := h.salesOrderRepo.created[0]
	assert.Equal(t, omsDomain.SalesOrderSourceWooCommerce, created.Source)
	assert.Equal(t, "501", created.ExternalOrderID)
	assert.Equal(t, omsDomain.SalesOrderStatusProcessing, created.Status)
	assert.True(t, created.StockDecremented, "processing reduces stock")
	assert.False(t, created.Fulfilled, "processing starts unfulfilled")
	assert.Equal(t, "flat_rate", created.ShippingMethod)
	assert.Equal(t, "Flat rate", created.ShippingMethodTitle)
	require.Len(t, h.lineItemRepo.created, 1)
	require.NotNil(t, h.lineItemRepo.created[0].SKUID)
	assert.Equal(t, "sku-1", *h.lineItemRepo.created[0].SKUID)
	assert.Equal(t, int64(1000), h.lineItemRepo.created[0].PriceMinorUnits)
	require.NotNil(t, h.lineItemRepo.created[0].TaxClassID)
	assert.Equal(t, "tax-class-standard", *h.lineItemRepo.created[0].TaxClassID)
	require.Len(t, h.stock.calls, 1)
	assert.Equal(t, int64(2), h.stock.calls[0].Quantity)
	require.Len(t, h.linkRepo.created, 1)
}

// TestImportWooOrders_importsSoldExpiryDateFromWooLineItemMeta pins the
// commerceos-woo plugin's commerceos_expiry_date order-item meta ("2006-01-02",
// see class-commerceos-orders.php) round-tripping into
// SalesOrderLineItem.SoldExpiryDate, the exact format toWooMetaData already
// writes when CommerceOS pushes a batch's expiry to Woo.
func TestImportWooOrders_importsSoldExpiryDateFromWooLineItemMeta(t *testing.T) {
	wo := simpleWooOrder()
	wo.LineItems[0].MetaData = []integrationsDomain.WooMetaData{
		{Key: "some_other_plugin_field", Value: "irrelevant"},
		{Key: "commerceos_expiry_date", Value: "2027-06-12"},
	}
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {wo}})
	h.skuRepo.byCode["WID-A"] = pmsDomain.SKU{ID: "sku-1", SKUCode: "WID-A"}
	uc := h.newImportUseCase(t, 4)

	_, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err)
	require.Len(t, h.lineItemRepo.created, 1)
	require.NotNil(t, h.lineItemRepo.created[0].SoldExpiryDate)
	assert.Equal(t, "2027-06-12", h.lineItemRepo.created[0].SoldExpiryDate.Format("2006-01-02"))
}

// TestImportWooOrders_noExpiryMeta_leavesSoldExpiryDateNil is the normal
// case -- most Woo line items carry no commerceos_expiry_date meta at all
// (the plugin only ever adds the key when the product had an expiry set).
func TestImportWooOrders_noExpiryMeta_leavesSoldExpiryDateNil(t *testing.T) {
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {simpleWooOrder()}})
	h.skuRepo.byCode["WID-A"] = pmsDomain.SKU{ID: "sku-1", SKUCode: "WID-A"}
	uc := h.newImportUseCase(t, 4)

	_, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err)
	require.Len(t, h.lineItemRepo.created, 1)
	assert.Nil(t, h.lineItemRepo.created[0].SoldExpiryDate)
}

// TestImportWooOrders_completedOrderArrivesAlreadyFulfilled proves an order
// imported well after the fact (already completed on Woo, never seen while
// processing) starts fulfilled too -- see SalesOrder.Fulfilled's doc.
func TestImportWooOrders_completedOrderArrivesAlreadyFulfilled(t *testing.T) {
	wo := simpleWooOrder()
	wo.Status = "completed"
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {wo}})
	h.skuRepo.byCode["WID-A"] = pmsDomain.SKU{ID: "sku-1", SKUCode: "WID-A"}
	uc := h.newImportUseCase(t, 4)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Created)
	require.Len(t, h.salesOrderRepo.created, 1)
	assert.True(t, h.salesOrderRepo.created[0].Fulfilled)
}

// TestImportWooOrders_skipStockMovementsCreatesOrderWithoutTouchingInventory
// proves SkipStockMovements still creates the order/line items normally but
// never decrements stock, even for a status that would otherwise reduce it
// -- used by the one-time bootstrap that runs orders import right after a
// connection's first products import, since that import already set each
// SKU's stock to Woo's current (already-net-of-these-orders) quantity.
func TestImportWooOrders_skipStockMovementsCreatesOrderWithoutTouchingInventory(t *testing.T) {
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {simpleWooOrder()}})
	h.skuRepo.byCode["WID-A"] = pmsDomain.SKU{ID: "sku-1", SKUCode: "WID-A"}
	uc := h.newImportUseCase(t, 3)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1", SkipStockMovements: true,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Created)
	assert.Empty(t, result.Errors)
	require.Len(t, h.salesOrderRepo.created, 1)
	assert.Equal(t, omsDomain.SalesOrderStatusProcessing, h.salesOrderRepo.created[0].Status)
	assert.False(t, h.salesOrderRepo.created[0].StockDecremented, "skipped even though processing would normally reduce stock")
	require.Len(t, h.lineItemRepo.created, 1, "line items are still created")
	assert.Empty(t, h.stock.calls)
	assert.Empty(t, h.linkRepo.created)
}

func TestImportWooOrders_pendingOrderDoesNotDecrementStock(t *testing.T) {
	order := simpleWooOrder()
	order.Status = "pending"
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {order}})
	h.skuRepo.byCode["WID-A"] = pmsDomain.SKU{ID: "sku-1", SKUCode: "WID-A"}
	uc := h.newImportUseCase(t, 3)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Created)
	assert.Empty(t, h.stock.calls)
	assert.False(t, h.salesOrderRepo.created[0].StockDecremented)
}

// matchingBilling/matchingShipping mirror simpleWooOrder()'s own
// Billing/Shipping exactly (same shape wooAddressToOMS would produce) --
// used to seed an "existing" fixture that already reflects a prior import
// of that same order, so a test can isolate the one field it's actually
// changing instead of every re-poll also (correctly, since this is the fix)
// re-syncing address/note drift it didn't intend to exercise.
var matchingBilling = omsDomain.Address{FirstName: "Jane", LastName: "Doe", Email: "jane@example.com"}
var matchingShipping = omsDomain.Address{FirstName: "Jane", LastName: "Doe"}

func TestImportWooOrders_isIdempotentWhenStatusUnchanged(t *testing.T) {
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {simpleWooOrder()}})
	h.salesOrderRepo.byExternalID["501"] = omsDomain.SalesOrder{
		ID: "so-1", Status: omsDomain.SalesOrderStatusProcessing,
		Billing: matchingBilling, Shipping: matchingShipping,
		// Already linked -- otherwise this order's real billing email would
		// trigger the customer-link backfill below and this wouldn't be a
		// true no-op any more (see the dedicated backfill test for that).
		CustomerID: strPtr("customer-1"),
	}
	uc := h.newImportUseCase(t, 2)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Unchanged)
	assert.Empty(t, h.salesOrderRepo.created, "must not create a duplicate order")
	assert.Empty(t, h.updateOrder.calls, "status unchanged -- no update call")
}

func TestImportWooOrders_updatesStatusOnAnExistingOrder(t *testing.T) {
	order := simpleWooOrder()
	order.Status = "completed"
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {order}})
	h.salesOrderRepo.byExternalID["501"] = omsDomain.SalesOrder{
		ID: "so-1", Status: omsDomain.SalesOrderStatusProcessing,
		Billing: matchingBilling, Shipping: matchingShipping,
	}
	uc := h.newImportUseCase(t, 2)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Updated)
	require.Len(t, h.updateOrder.calls, 1)
	assert.Equal(t, "so-1", h.updateOrder.ids[0])
	require.NotNil(t, h.updateOrder.calls[0].Status)
	assert.Equal(t, omsDomain.SalesOrderStatusCompleted, *h.updateOrder.calls[0].Status)
	assert.Nil(t, h.updateOrder.calls[0].Billing, "address unchanged from prior import -- no needless write")
	assert.Nil(t, h.updateOrder.calls[0].Shipping, "address unchanged from prior import -- no needless write")
}

// Regression (prod 2026-11-03, order 27300): after a Woo API outage the
// import read Woo's stale "processing" and moved a shipped order back to
// processing. A shipped order must keep its status while other fields still
// re-sync.
func TestImportWooOrders_staleWooStatusNeverRegressesAShippedOrder(t *testing.T) {
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {simpleWooOrder()}}) // Woo still says "processing"
	h.salesOrderRepo.byExternalID["501"] = omsDomain.SalesOrder{
		ID: "so-1", Status: omsDomain.SalesOrderStatusShipped,
		Billing:  omsDomain.Address{FirstName: "Stale", LastName: "Name", Email: "jane@example.com"},
		Shipping: matchingShipping, CustomerID: strPtr("customer-1"),
	}
	uc := h.newImportUseCase(t, 2)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err)
	require.Len(t, h.updateOrder.calls, 1)
	assert.Equal(t, 1, result.Updated)
	call := h.updateOrder.calls[0]
	assert.Nil(t, call.Status, "shipped must not regress to Woo's stale processing")
	require.NotNil(t, call.Billing, "non-status fields still re-sync")
	assert.Equal(t, matchingBilling, *call.Billing)
}

// Shipped is pushed to Woo as "on-hold"; reading that echo back must not
// turn the order into on-hold (prod had 0 shipped Woo orders because of it).
func TestImportWooOrders_onHoldEchoOfOurOwnShippedPushIsNoChange(t *testing.T) {
	order := simpleWooOrder()
	order.Status = "on-hold"
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {order}})
	h.salesOrderRepo.byExternalID["501"] = omsDomain.SalesOrder{
		ID: "so-1", Status: omsDomain.SalesOrderStatusShipped,
		Billing: matchingBilling, Shipping: matchingShipping, CustomerID: strPtr("customer-1"),
	}
	uc := h.newImportUseCase(t, 2)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Unchanged)
	assert.Empty(t, h.updateOrder.calls)
}

// Order 27270 went completed -> on-hold in the same import.
func TestImportWooOrders_staleWooStatusNeverRegressesACompletedOrder(t *testing.T) {
	order := simpleWooOrder()
	order.Status = "on-hold"
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {order}})
	h.salesOrderRepo.byExternalID["501"] = omsDomain.SalesOrder{
		ID: "so-1", Status: omsDomain.SalesOrderStatusCompleted,
		Billing: matchingBilling, Shipping: matchingShipping, CustomerID: strPtr("customer-1"),
	}
	uc := h.newImportUseCase(t, 2)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Unchanged)
	assert.Empty(t, h.updateOrder.calls)
}

// A cancellation/refund decided in Woo still reaches a shipped order.
func TestImportWooOrders_wooRefundStillAppliesToAShippedOrder(t *testing.T) {
	order := simpleWooOrder()
	order.Status = "refunded"
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {order}})
	h.salesOrderRepo.byExternalID["501"] = omsDomain.SalesOrder{
		ID: "so-1", Status: omsDomain.SalesOrderStatusShipped,
		Billing: matchingBilling, Shipping: matchingShipping, CustomerID: strPtr("customer-1"),
	}
	uc := h.newImportUseCase(t, 2)

	_, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err)
	require.Len(t, h.updateOrder.calls, 1)
	require.NotNil(t, h.updateOrder.calls[0].Status)
	assert.Equal(t, omsDomain.SalesOrderStatusRefunded, *h.updateOrder.calls[0].Status)
}

// TestImportWooOrders_reconciliationResyncsBillingShippingWhenTheyDriftFromWoo
// is the direct fix for the reported bug: a customer's name/phone (billing +
// shipping) got corrected in Woo (e.g. Woo matched the order to an existing
// customer account by email, but the order itself carries a different
// name/phone than that customer's stored default) after CommerceOS's first
// import already snapshotted the order under the old data. Before this fix,
// routeToExisting only ever compared/updated Status -- a re-poll with an
// unchanged status silently discarded the corrected address forever.
func TestImportWooOrders_reconciliationResyncsBillingShippingWhenTheyDriftFromWoo(t *testing.T) {
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {simpleWooOrder()}}) // status stays "processing"
	h.salesOrderRepo.byExternalID["501"] = omsDomain.SalesOrder{
		ID: "so-1", Status: omsDomain.SalesOrderStatusProcessing,
		Billing:  omsDomain.Address{FirstName: "Stale", LastName: "Name", Email: "jane@example.com", Phone: "0000000000"},
		Shipping: omsDomain.Address{FirstName: "Stale", LastName: "Name"},
	}
	uc := h.newImportUseCase(t, 2)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Updated, "address drift counts as a real update, not Unchanged")
	assert.Equal(t, 0, result.Unchanged)
	require.Len(t, h.updateOrder.calls, 1)
	assert.Equal(t, "so-1", h.updateOrder.ids[0])
	call := h.updateOrder.calls[0]
	assert.Nil(t, call.Status, "status itself didn't change -- must not re-trigger stock transition logic")
	require.NotNil(t, call.Billing)
	assert.Equal(t, matchingBilling, *call.Billing)
	require.NotNil(t, call.Shipping)
	assert.Equal(t, matchingShipping, *call.Shipping)
}

// TestImportWooOrders_reconciliationResyncsCustomerNoteWhenItDrifts covers
// the same "first-import snapshot never refreshes" bug for the order's
// customer note (a staff/customer edit made in Woo after the original
// import).
func TestImportWooOrders_reconciliationResyncsCustomerNoteWhenItDrifts(t *testing.T) {
	order := simpleWooOrder()
	order.CustomerNote = "Please leave at the door"
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {order}})
	h.salesOrderRepo.byExternalID["501"] = omsDomain.SalesOrder{
		ID: "so-1", Status: omsDomain.SalesOrderStatusProcessing,
		Billing: matchingBilling, Shipping: matchingShipping, CustomerNote: "",
	}
	uc := h.newImportUseCase(t, 2)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Updated)
	require.Len(t, h.updateOrder.calls, 1)
	require.NotNil(t, h.updateOrder.calls[0].CustomerNote)
	assert.Equal(t, "Please leave at the door", *h.updateOrder.calls[0].CustomerNote)
}

// A line item whose Woo SKU string never matched a CommerceOS SKU (never
// imported, or Woo sent none at all) is still imported -- name/quantity/
// price/tax are already present on the order regardless of a catalog
// match, so there's no reason to lose that data. It's just left unlinked
// (SKUID nil, same "not linked to a catalog product yet" pattern
// procurement.PurchaseOrderLineItem already uses), the order flagged
// NeedsSKUReview, and the line marked StockStatusPending (its order is
// stock-reducing, so a decrement is owed once it's linked) -- surfaced on
// the "Unmatched SKUs" admin page (LinkLineItemSKUUseCase), not as a run
// failure: nothing here actually failed, so this must never land in
// result.Errors (cmd/scheduler/dispatch.go treats any non-empty Errors as
// last_run_status=error).
func TestImportWooOrders_importsAnUnmatchedLineItemUnlinkedInsteadOfDroppingIt(t *testing.T) {
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {simpleWooOrder()}})
	// h.skuRepo.byCode intentionally has no "WID-A" entry.
	uc := h.newImportUseCase(t, 3)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Created)
	require.Len(t, h.lineItemRepo.created, 1, "the line item is kept, just unlinked -- not dropped")
	created := h.lineItemRepo.created[0]
	assert.Nil(t, created.SKUID)
	assert.Equal(t, "Widget A", created.Name)
	assert.Equal(t, "WID-A", created.SKUCode)
	assert.Equal(t, int64(2), created.Quantity)
	assert.Equal(t, int64(1000), created.PriceMinorUnits)
	assert.Equal(t, omsDomain.StockStatusPending, created.StockStatus, "processing is stock-reducing -- a decrement is owed once this line is linked")
	assert.Nil(t, created.TaxClassID, "no SKU to source a tax class from")
	assert.Empty(t, h.stock.calls, "no SKU means nothing to decrement")
	assert.Empty(t, result.Errors, "unlinked SKU is not a run failure -- it's surfaced via needs_sku_review, not result.Errors")
	require.NotEmpty(t, h.salesOrderRepo.updateCalls, "order must be flagged for the Unmatched SKUs review page")
	assert.Equal(t, true, h.salesOrderRepo.updateCalls[len(h.salesOrderRepo.updateCalls)-1].Fields["needs_sku_review"])
}

// A blank Woo SKU string (a real WooCommerce quirk -- an order line
// snapshot can omit it even when the product itself has one) falls back to
// matching the line item's own name against the catalog product's name,
// instead of leaving a real product's sale unlinked.
func TestImportWooOrders_blankWooSKU_fallsBackToMatchingByLineItemName(t *testing.T) {
	order := simpleWooOrder()
	order.LineItems[0].SKU = ""
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {order}})
	taxClassID := "tax-class-standard"
	h.skuRepo.byProductName = map[string]pmsDomain.SKU{"Widget A": {ID: "sku-1", SKUCode: "WID-A", TaxClassID: &taxClassID}}
	uc := h.newImportUseCase(t, 4)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Created)
	require.Len(t, h.lineItemRepo.created, 1)
	created := h.lineItemRepo.created[0]
	require.NotNil(t, created.SKUID)
	assert.Equal(t, "sku-1", *created.SKUID)
	assert.Equal(t, omsDomain.StockStatusDecremented, created.StockStatus)
	require.Empty(t, h.salesOrderRepo.updateCalls, "resolved via name fallback -- never flagged needs_sku_review")
}

// A Woo SKU string that IS present but doesn't match any catalog SKU code
// (stale/renamed SKU on Woo's side) still gets the same name fallback --
// not just the blank-string case.
func TestImportWooOrders_wooSKUCodeDoesNotMatch_fallsBackToMatchingByLineItemName(t *testing.T) {
	order := simpleWooOrder()
	order.LineItems[0].SKU = "STALE-CODE"
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {order}})
	h.skuRepo.byProductName = map[string]pmsDomain.SKU{"Widget A": {ID: "sku-1", SKUCode: "WID-A"}}
	uc := h.newImportUseCase(t, 4)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Created)
	require.Len(t, h.lineItemRepo.created, 1)
	require.NotNil(t, h.lineItemRepo.created[0].SKUID)
	assert.Equal(t, "sku-1", *h.lineItemRepo.created[0].SKUID)
}

// Neither the SKU code nor the line item's name resolves to anything --
// still left unlinked, same as before this fallback existed. No guessing.
func TestImportWooOrders_nameFallbackAlsoMisses_staysUnlinked(t *testing.T) {
	order := simpleWooOrder()
	order.LineItems[0].SKU = ""
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {order}})
	// h.skuRepo.byProductName intentionally has no "Widget A" entry either.
	uc := h.newImportUseCase(t, 3)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Created)
	require.Len(t, h.lineItemRepo.created, 1)
	assert.Nil(t, h.lineItemRepo.created[0].SKUID)
	require.NotEmpty(t, h.salesOrderRepo.updateCalls)
	assert.Equal(t, true, h.salesOrderRepo.updateCalls[len(h.salesOrderRepo.updateCalls)-1].Fields["needs_sku_review"])
}

// A refund/adjustment on the Woo side can surface as a negative quantity
// on an order's line_items -- domain/oms.SalesOrderLineItem has no
// representation for that (refunds are tracked at the whole-order Status
// level, see SalesOrderStatusRefunded), and the DB rejects it outright
// (sales_order_line_items_quantity_check requires quantity > 0). Before
// this fix, that INSERT failed and aborted the entire order (including
// every OTHER, valid line item) with a raw constraint-violation error.
func TestImportWooOrders_skipsNonPositiveQuantityLineItemsButStillCreatesTheOrder(t *testing.T) {
	order := simpleWooOrder()
	order.LineItems = append(order.LineItems, integrationsDomain.WooOrderLineItem{
		ProductID: 13, SKU: "WID-B", Name: "Widget B (refund adjustment)", Quantity: -1, Price: "10", Subtotal: "-10.00", Total: "-10.00",
	})
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {order}})
	taxClassID := "tax-class-standard"
	h.skuRepo.byCode["WID-A"] = pmsDomain.SKU{ID: "sku-1", SKUCode: "WID-A", TaxClassID: &taxClassID}
	h.skuRepo.byCode["WID-B"] = pmsDomain.SKU{ID: "sku-2", SKUCode: "WID-B", TaxClassID: &taxClassID}
	uc := h.newImportUseCase(t, 4)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Created)
	require.Len(t, h.lineItemRepo.created, 1, "the valid line item is still created despite the bad one")
	assert.Equal(t, "WID-A", h.lineItemRepo.created[0].SKUCode)
	assert.NotEmpty(t, result.Errors, "the skipped non-positive-quantity line should be surfaced, not silently dropped")
}

// TestImportWooOrders_insufficientStockMarksLineItemPendingInsteadOfFailingTheOrder
// is the direct fix for the reported bug ("woo order 24002: stock movement:
// recordStockMovement: insufficient stock for this batch/ledger" etc.): a
// matched-SKU line item whose decrement hits ErrInsufficientStock no longer
// aborts the whole order import (nor gets counted as a run failure) -- it's
// created normally, marked StockStatusPending, and the order flagged
// NeedsStockReview for the "Stock Issues" admin page.
func TestImportWooOrders_insufficientStockMarksLineItemPendingInsteadOfFailingTheOrder(t *testing.T) {
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {simpleWooOrder()}})
	h.skuRepo.byCode["WID-A"] = pmsDomain.SKU{ID: "sku-1", SKUCode: "WID-A"}
	h.stock.err = invDomain.ErrInsufficientStock
	uc := h.newImportUseCase(t, 5) // connect + findByExternalID + header + setLineItemStockStatus + setOrderNeedsStockReview

	result, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Created, "order is still created despite the stock shortfall")
	assert.Empty(t, result.Errors, "insufficient stock is not a run failure -- it's surfaced via needs_stock_review")
	require.Len(t, h.lineItemRepo.updateCalls, 1)
	assert.Equal(t, string(omsDomain.StockStatusPending), h.lineItemRepo.updateCalls[0].Fields["stock_status"])
	require.NotEmpty(t, h.salesOrderRepo.updateCalls)
	last := h.salesOrderRepo.updateCalls[len(h.salesOrderRepo.updateCalls)-1]
	assert.Equal(t, true, last.Fields["needs_stock_review"])
	assert.Empty(t, h.linkRepo.created, "the failed movement was never recorded, so nothing to link")
}

// TestImportWooOrders_losingTheCreateRaceRoutesToUpdateInsteadOfFailing is
// the mid-task-added hardening: a second concurrent import of the same Woo
// order (two scheduler replicas, or the manual fetch-order endpoint racing
// the scheduler) can lose the DB's (tenant_id, source, external_order_id)
// unique-constraint race on Create. That must be treated as "the order
// already exists now", routed through the same update path importOne uses
// for a known order, never as a double-created order or a run failure.
func TestImportWooOrders_losingTheCreateRaceRoutesToUpdateInsteadOfFailing(t *testing.T) {
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {simpleWooOrder()}})
	h.skuRepo.byCode["WID-A"] = pmsDomain.SKU{ID: "sku-1", SKUCode: "WID-A"}
	h.salesOrderRepo.createErr = omsDomain.ErrDuplicateExternalOrder
	// Simulate the winner of the race having already committed the order
	// under this external ID by the time this call's Create fails and falls
	// back to looking it up -- the *first* FindByExternalID call (before
	// Create is attempted) must still see nothing, same as a genuine race.
	h.salesOrderRepo.winnerAfterCreateFail = map[string]omsDomain.SalesOrder{
		// Already linked -- otherwise this order's real billing email would
		// trigger the customer-link backfill and this wouldn't be a true
		// no-op any more (see the dedicated backfill test for that).
		"501": {ID: "so-winner", Status: omsDomain.SalesOrderStatusProcessing, Billing: matchingBilling, Shipping: matchingShipping, CustomerID: strPtr("customer-1")},
	}
	uc := h.newImportUseCase(t, 4) // connect + importOne's findByExternalID + failed-create header tx + createOne's findExisting fallback lookup

	result, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err)
	assert.Empty(t, result.Errors, "losing the create race is not a failure")
	assert.Equal(t, 0, result.Created, "must not double-create")
	assert.Equal(t, 1, result.Unchanged, "winner's order already has this status -- a no-op, same as importOne's normal existing-order path")
	assert.Empty(t, h.stock.calls, "must never double-book the winner's stock decrement")
}

func TestImportWooOrders_linksToAnExistingCustomerByWooCustomerID(t *testing.T) {
	order := simpleWooOrder()
	order.CustomerID = 77
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {order}})
	h.skuRepo.byCode["WID-A"] = pmsDomain.SKU{ID: "sku-1", SKUCode: "WID-A"}
	h.customerRepo.byWooID[77] = customersDomain.Customer{ID: "customer-1"}
	uc := h.newImportUseCase(t, 4)

	_, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err)
	require.Len(t, h.salesOrderRepo.created, 1)
	require.NotNil(t, h.salesOrderRepo.created[0].CustomerID)
	assert.Equal(t, "customer-1", *h.salesOrderRepo.created[0].CustomerID)
}

// Reported bug: a guest checkout (no Woo account, wo.CustomerID == 0) with
// a real name+email never got linked to any customers.Customer -- the order
// showed a name but the frontend had no customer_id to link it, and no
// Customer record existed to show that buyer's order history/amount spent
// at all. Woo requires a billing email even for guest checkout, so
// find-or-create by that email fixes it.
func TestImportWooOrders_guestCheckoutCreatesACustomerFromBillingWhenNoneMatchesByEmail(t *testing.T) {
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {simpleWooOrder()}})
	h.skuRepo.byCode["WID-A"] = pmsDomain.SKU{ID: "sku-1", SKUCode: "WID-A"}
	uc := h.newImportUseCase(t, 4)

	_, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err)
	require.Len(t, h.customerRepo.created, 1)
	created := h.customerRepo.created[0]
	assert.Equal(t, "jane@example.com", created.Email)
	assert.Equal(t, "Jane", created.FirstName)
	assert.Equal(t, "Doe", created.LastName)
	assert.Nil(t, created.WooCustomerID, "guest checkout has no Woo account to link")

	require.Len(t, h.salesOrderRepo.created, 1)
	require.NotNil(t, h.salesOrderRepo.created[0].CustomerID)
	assert.Equal(t, created.ID, *h.salesOrderRepo.created[0].CustomerID)
}

func TestImportWooOrders_guestCheckoutLinksToAnExistingCustomerByEmailInsteadOfDuplicating(t *testing.T) {
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {simpleWooOrder()}})
	h.skuRepo.byCode["WID-A"] = pmsDomain.SKU{ID: "sku-1", SKUCode: "WID-A"}
	h.customerRepo.byEmail["jane@example.com"] = customersDomain.Customer{ID: "customer-9", Email: "jane@example.com"}
	uc := h.newImportUseCase(t, 4)

	_, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err)
	assert.Empty(t, h.customerRepo.created, "must reuse the existing customer, not duplicate it")
	require.Len(t, h.salesOrderRepo.created, 1)
	require.NotNil(t, h.salesOrderRepo.created[0].CustomerID)
	assert.Equal(t, "customer-9", *h.salesOrderRepo.created[0].CustomerID)
}

func TestImportWooOrders_guestCheckoutWithNoBillingEmailStaysUnlinked(t *testing.T) {
	order := simpleWooOrder()
	order.Billing.Email = ""
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {order}})
	h.skuRepo.byCode["WID-A"] = pmsDomain.SKU{ID: "sku-1", SKUCode: "WID-A"}
	uc := h.newImportUseCase(t, 4)

	_, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err)
	assert.Empty(t, h.customerRepo.created, "nothing to key a customer off of")
	require.Len(t, h.salesOrderRepo.created, 1)
	assert.Nil(t, h.salesOrderRepo.created[0].CustomerID)
}

// A registered Woo account not yet imported into CommerceOS (e.g. the
// one-time customer import hasn't run, or the account was created after
// it) previously left the order permanently unlinked. Now it's created
// from the order's own billing details instead, same as a guest.
func TestImportWooOrders_registeredWooCustomerNotYetImported_createsOneFromBillingDetails(t *testing.T) {
	order := simpleWooOrder()
	order.CustomerID = 77
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {order}})
	h.skuRepo.byCode["WID-A"] = pmsDomain.SKU{ID: "sku-1", SKUCode: "WID-A"}
	uc := h.newImportUseCase(t, 4)

	_, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err)
	require.Len(t, h.customerRepo.created, 1)
	created := h.customerRepo.created[0]
	require.NotNil(t, created.WooCustomerID)
	assert.Equal(t, int64(77), *created.WooCustomerID)
	assert.Equal(t, "jane@example.com", created.Email)
	require.Len(t, h.salesOrderRepo.created, 1)
	require.NotNil(t, h.salesOrderRepo.created[0].CustomerID)
	assert.Equal(t, created.ID, *h.salesOrderRepo.created[0].CustomerID)
}

// The production bug this fix targets: an order imported before this fix
// existed has CustomerID nil forever, with no code path that would ever
// revisit it -- until now. Every re-poll of an unlinked existing order is a
// free chance to backfill it, so a store doesn't need a one-off migration
// to fix orders that already went through the old, gap-having import.
func TestImportWooOrders_backfillsCustomerLinkOnResyncOfAPreviouslyUnlinkedOrder(t *testing.T) {
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {simpleWooOrder()}})
	h.salesOrderRepo.byExternalID["501"] = omsDomain.SalesOrder{
		ID: "so-1", Status: omsDomain.SalesOrderStatusProcessing,
		Billing: matchingBilling, Shipping: matchingShipping,
		CustomerID: nil, // never linked on first import
	}
	uc := h.newImportUseCase(t, 2)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Updated)
	require.Len(t, h.updateOrder.calls, 1)
	require.NotNil(t, h.updateOrder.calls[0].CustomerID)
	require.Len(t, h.customerRepo.created, 1, "no existing customer matched this email, so one is created")
	assert.Equal(t, h.customerRepo.created[0].ID, *h.updateOrder.calls[0].CustomerID)
}

// An existing order that's already linked must never be re-resolved or
// re-created on every routine re-poll -- both a pointless customer lookup
// on every single already-correct order, and (worse) a risk of ever
// clobbering a real link with a different match.
func TestImportWooOrders_neverTouchesCustomerIDOnAnAlreadyLinkedExistingOrder(t *testing.T) {
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {simpleWooOrder()}})
	h.salesOrderRepo.byExternalID["501"] = omsDomain.SalesOrder{
		ID: "so-1", Status: omsDomain.SalesOrderStatusProcessing,
		Billing: matchingBilling, Shipping: matchingShipping,
		CustomerID: strPtr("customer-1"),
	}
	uc := h.newImportUseCase(t, 2)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Unchanged)
	assert.Empty(t, h.customerRepo.created)
	assert.Empty(t, h.updateOrder.calls)
}

func TestImportWooOrders_incrementalPassesModifiedAfterThrough(t *testing.T) {
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{})
	uc := h.newImportUseCase(t, 1)
	cutoff := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	_, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1", ModifiedAfter: &cutoff,
	})

	require.NoError(t, err)
	require.NotNil(t, h.wooClient.ordersModifiedAfter)
	assert.True(t, h.wooClient.ordersModifiedAfter.Equal(cutoff))
}

// A page-fetch failure (a transient Woo/network error) must NOT be a hard
// Execute error -- cmd/scheduler's runOne only advances a schedule's
// incremental cursor (LastRunAt) when the run's status is "success" (see
// that fix's own commit), specifically so a run that couldn't reach every
// page gets retried from the same window next time, instead of the cursor
// silently skipping past whatever was never fetched. A hard error here
// would also have discarded every order already imported earlier in this
// same run's result.
func TestImportWooOrders_aPageFetchFailureIsReportedNotReturnedAsAHardError(t *testing.T) {
	h := newOrdersHarness(nil)
	h.wooClient.ordersErr = assert.AnError
	uc := h.newImportUseCase(t, 1)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err)
	require.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors[0], "woocommerce request failed")
}

func TestImportWooOrders_aPageFetchFailureStillKeepsOrdersImportedEarlierInTheSameRun(t *testing.T) {
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {simpleWooOrder()}})
	h.skuRepo.byCode["WID-A"] = pmsDomain.SKU{ID: "sku-1", SKUCode: "WID-A"}
	h.wooClient.ordersErrOnPage = 2
	h.wooClient.ordersErr = assert.AnError
	uc := h.newImportUseCase(t, 4)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Created, "the order from page 1 must still count, not be discarded")
	require.NotEmpty(t, result.Errors)
}

// ── ExecuteOne (manual "fetch this specific order" admin action) ───────────

func TestImportWooOrders_ExecuteOne_fetchesAndCreatesTheOrder(t *testing.T) {
	h := newOrdersHarness(nil)
	h.wooClient.getOrderResult = ptrWooOrder(simpleWooOrder())
	h.skuRepo.byCode["WID-A"] = pmsDomain.SKU{ID: "sku-1", SKUCode: "WID-A"}
	uc := h.newImportUseCase(t, 5)

	result, err := uc.ExecuteOne(context.Background(), appchannels.FetchWooOrderInput{
		ConnectionID: "conn-1", ExternalOrderID: "501", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"501"}, h.wooClient.getOrderCalled)
	assert.True(t, result.Created)
	assert.False(t, result.Updated)
	assert.False(t, result.Unchanged)
	assert.Empty(t, result.Errors)
	assert.Equal(t, "sales-order-1", result.SalesOrderID)
	require.Len(t, h.salesOrderRepo.created, 1)
	assert.Equal(t, "501", h.salesOrderRepo.created[0].ExternalOrderID)
}

func TestImportWooOrders_ExecuteOne_updatesAnExistingOrdersStatus(t *testing.T) {
	h := newOrdersHarness(nil)
	wo := simpleWooOrder()
	wo.Status = "completed"
	h.wooClient.getOrderResult = ptrWooOrder(wo)
	h.salesOrderRepo.byExternalID["501"] = omsDomain.SalesOrder{ID: "sales-order-1", ExternalOrderID: "501", Status: omsDomain.SalesOrderStatusProcessing}
	uc := h.newImportUseCase(t, 3)

	result, err := uc.ExecuteOne(context.Background(), appchannels.FetchWooOrderInput{
		ConnectionID: "conn-1", ExternalOrderID: "501", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err)
	assert.True(t, result.Updated)
	assert.Equal(t, "sales-order-1", result.SalesOrderID)
	require.Len(t, h.updateOrder.calls, 1)
}

func TestImportWooOrders_ExecuteOne_reportsAnUnknownOrderIDAsAnError(t *testing.T) {
	h := newOrdersHarness(nil)
	h.wooClient.getOrderErr = assert.AnError
	uc := h.newImportUseCase(t, 1)

	_, err := uc.ExecuteOne(context.Background(), appchannels.FetchWooOrderInput{
		ConnectionID: "conn-1", ExternalOrderID: "999999", DefaultWarehouseID: "warehouse-1",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, integrationsDomain.ErrWooRequestFailed)
}

func TestImportWooOrders_ExecuteOne_leavesSalesOrderIDEmptyWhenTheOrderFailsToImport(t *testing.T) {
	h := newOrdersHarness(nil)
	h.wooClient.getOrderResult = ptrWooOrder(simpleWooOrder())
	h.salesOrderRepo.findByExternalIDErr = assert.AnError
	uc := h.newImportUseCase(t, 2)

	result, err := uc.ExecuteOne(context.Background(), appchannels.FetchWooOrderInput{
		ConnectionID: "conn-1", ExternalOrderID: "501", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err, "ExecuteOne itself doesn't hard-fail on a per-order error -- it's reported via result.Errors")
	assert.Empty(t, result.SalesOrderID)
	assert.NotEmpty(t, result.Errors)
}

// TestImportWooOrders_propagatesAStockMovementErrorAsAPerOrderError covers a
// stock movement failure that is NOT ErrInsufficientStock (a genuinely
// unexpected error, e.g. a DB error) -- unlike insufficient stock (see
// TestImportWooOrders_insufficientStockMarksLineItemPendingInsteadOfFailingTheOrder),
// this still surfaces as a result.Errors entry and does not touch
// stock_status/needs_stock_review, since it's not the "known, resolvable via
// the review flow" case.
func TestImportWooOrders_propagatesAStockMovementErrorAsAPerOrderError(t *testing.T) {
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {simpleWooOrder()}})
	h.skuRepo.byCode["WID-A"] = pmsDomain.SKU{ID: "sku-1", SKUCode: "WID-A"}
	h.stock.err = errors.New("db unreachable")
	uc := h.newImportUseCase(t, 3) // connect + findByExternalID + header -- no extra tx, since this error never touches stock_status/needs_stock_review

	result, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err, "one order's movement failure must not abort the whole import run")
	assert.Equal(t, 1, result.Created, "order+line items still recorded even though the movement failed")
	assert.NotEmpty(t, result.Errors)
	assert.Empty(t, h.lineItemRepo.updateCalls, "not ErrInsufficientStock -- must not be marked pending")
	assert.Empty(t, h.salesOrderRepo.updateCalls, "not ErrInsufficientStock -- must not flag needs_stock_review")
}

// An order first imported as pending (no payment yet) and paid later must
// pick up Woo's transaction id, paid date and final payment method on the
// next import -- before, only status/addresses/note were ever re-synced,
// so these stayed empty forever.
func TestImportWooOrders_reimportFillsInPaymentInfoOnceWooRecordsIt(t *testing.T) {
	order := simpleWooOrder()
	order.Status = "processing"
	order.PaymentMethod, order.PaymentMethodTitle = "stripe", "Credit card (Stripe)"
	order.TransactionID = "pi_3Q123"
	order.DatePaid = "2026-10-02T14:31:00"
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {order}})
	h.salesOrderRepo.byExternalID["501"] = omsDomain.SalesOrder{
		ID: "so-1", Status: omsDomain.SalesOrderStatusPending, Billing: matchingBilling, Shipping: matchingShipping,
		PaymentMethod: "stripe", PaymentMethodTitle: "Stripe",
	}
	uc := h.newImportUseCase(t, 2)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1"})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Updated)
	require.Len(t, h.updateOrder.calls, 1)
	call := h.updateOrder.calls[0]
	require.NotNil(t, call.TransactionID)
	assert.Equal(t, "pi_3Q123", *call.TransactionID)
	require.NotNil(t, call.DatePaid)
	assert.Equal(t, time.Date(2026, 10, 2, 14, 31, 0, 0, time.UTC), *call.DatePaid)
	require.NotNil(t, call.PaymentMethodTitle)
	assert.Equal(t, "Credit card (Stripe)", *call.PaymentMethodTitle)
	assert.Nil(t, call.PaymentMethod, "unchanged -- no needless write")
}

func TestImportWooOrders_reimportLeavesUnchangedPaymentInfoAlone(t *testing.T) {
	order := simpleWooOrder()
	order.Status = "processing"
	order.PaymentMethod, order.PaymentMethodTitle, order.TransactionID = "stripe", "Stripe", "pi_3Q123"
	order.DatePaid = "2026-10-02T14:31:00"
	paid := time.Date(2026, 10, 2, 14, 31, 0, 0, time.UTC)
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {order}})
	h.salesOrderRepo.byExternalID["501"] = omsDomain.SalesOrder{
		ID: "so-1", Status: omsDomain.SalesOrderStatusProcessing, Billing: matchingBilling, Shipping: matchingShipping,
		PaymentMethod: "stripe", PaymentMethodTitle: "Stripe", TransactionID: "pi_3Q123", DatePaid: &paid, CustomerID: strPtr("cust-1"),
	}
	uc := h.newImportUseCase(t, 2)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1"})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Unchanged)
	assert.Empty(t, h.updateOrder.calls)
}

// Woo never "unpays" an order by blanking these -- an empty value on Woo's
// side (e.g. a gateway that sets no transaction id) must not wipe what
// CommerceOS already has (e.g. stamped by its own payment webhook).
func TestImportWooOrders_reimportNeverBlanksKnownPaymentInfo(t *testing.T) {
	order := simpleWooOrder()
	order.Status = "processing"
	order.TransactionID, order.DatePaid = "", ""
	paid := time.Date(2026, 10, 2, 14, 31, 0, 0, time.UTC)
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {order}})
	h.salesOrderRepo.byExternalID["501"] = omsDomain.SalesOrder{
		ID: "so-1", Status: omsDomain.SalesOrderStatusProcessing, Billing: matchingBilling, Shipping: matchingShipping,
		PaymentMethod: order.PaymentMethod, PaymentMethodTitle: order.PaymentMethodTitle, TransactionID: "pi_3Q123", DatePaid: &paid, CustomerID: strPtr("cust-1"),
	}
	uc := h.newImportUseCase(t, 2)

	_, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1"})

	require.NoError(t, err)
	assert.Empty(t, h.updateOrder.calls)
}
