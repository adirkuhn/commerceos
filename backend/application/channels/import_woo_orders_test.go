package channels_test

import (
	"context"
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
	connRepo       *stubConnectionRepo
	wooFactory     *stubWooClientFactory
	wooClient      *stubWooClient
	salesOrderRepo *stubOrdersSalesOrderRepo
	lineItemRepo   *stubOrdersLineItemRepo
	linkRepo       *stubOrdersLinkRepo
	skuRepo        *stubChannelsSKURepo
	customerRepo   *stubOrdersCustomerRepo
	stock          *stubStockRecorder
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
		customerRepo:   &stubOrdersCustomerRepo{byWooID: map[int64]customersDomain.Customer{}},
		stock:          &stubStockRecorder{},
		updateOrder:    &stubUpdateSalesOrderUC{},
	}
}

func (h *ordersHarness) newImportUseCase(t *testing.T, n int) *appchannels.ImportWooOrdersUseCase {
	t.Helper()
	return appchannels.NewImportWooOrdersUseCase(
		newMockGormDB(t, n), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.salesOrderRepo, h.lineItemRepo, h.linkRepo, h.skuRepo, h.customerRepo, h.stock, h.updateOrder,
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
	assert.Equal(t, "flat_rate", created.ShippingMethod)
	assert.Equal(t, "Flat rate", created.ShippingMethodTitle)
	require.Len(t, h.lineItemRepo.created, 1)
	require.NotNil(t, h.lineItemRepo.created[0].SKUID)
	assert.Equal(t, "sku-1", *h.lineItemRepo.created[0].SKUID)
	assert.Equal(t, int64(1000), h.lineItemRepo.created[0].PriceMinorUnits)
	require.NotNil(t, h.lineItemRepo.created[0].TaxClassID)
	assert.Equal(t, "tax-class-standard", *h.lineItemRepo.created[0].TaxClassID)
	require.Len(t, h.stock.calls, 1)
	assert.Equal(t, int64(-2), h.stock.calls[0].QuantityDelta)
	require.Len(t, h.linkRepo.created, 1)
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

func TestImportWooOrders_isIdempotentWhenStatusUnchanged(t *testing.T) {
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {simpleWooOrder()}})
	h.salesOrderRepo.byExternalID["501"] = omsDomain.SalesOrder{ID: "so-1", Status: omsDomain.SalesOrderStatusProcessing}
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
	h.salesOrderRepo.byExternalID["501"] = omsDomain.SalesOrder{ID: "so-1", Status: omsDomain.SalesOrderStatusProcessing}
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
}

// A line item whose Woo SKU string never matched a CommerceOS SKU (never
// imported, or Woo sent none at all) is still imported -- name/quantity/
// price/tax are already present on the order regardless of a catalog
// match, so there's no reason to lose that data. It's just left unlinked
// (SKUID nil, same "not linked to a catalog product yet" pattern
// procurement.PurchaseOrderLineItem already uses) and reported as an
// informational note, not an error, since nothing failed.
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
	assert.Nil(t, created.TaxClassID, "no SKU to source a tax class from")
	assert.Empty(t, h.stock.calls, "no SKU means nothing to decrement")
	assert.NotEmpty(t, result.Errors, "unlinked line should be surfaced for later manual linking")
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

func TestImportWooOrders_propagatesAWooRequestFailure(t *testing.T) {
	h := newOrdersHarness(nil)
	h.wooClient.ordersErr = assert.AnError
	uc := h.newImportUseCase(t, 1)

	_, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1",
	})

	assert.ErrorIs(t, err, integrationsDomain.ErrWooRequestFailed)
}

func TestImportWooOrders_propagatesAStockMovementErrorAsAPerOrderError(t *testing.T) {
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {simpleWooOrder()}})
	h.skuRepo.byCode["WID-A"] = pmsDomain.SKU{ID: "sku-1", SKUCode: "WID-A"}
	h.stock.err = invDomain.ErrInsufficientStock
	uc := h.newImportUseCase(t, 3)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err, "one order's movement failure must not abort the whole import run")
	assert.Equal(t, 1, result.Created, "order+line items still recorded even though the movement failed")
	assert.NotEmpty(t, result.Errors)
}
