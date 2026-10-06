package channels_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/events"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/crypto"
)

func newProductImportUC(t *testing.T, h *harness, n int, outbox *eventrecordtest.Outbox) *appchannels.ImportWooProductsUseCase {
	t.Helper()
	return appchannels.NewImportWooProductsUseCase(
		newMockGormDB(t, n), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.categoryRepo, h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, h.stock, h.media,
		h.taxClassRepo, h.taxRateRepo, h.taxLinkRepo, outbox,
	)
}

func TestImportWooProducts_RecordsProductCategoryAndListing(t *testing.T) {
	h := newHarness([]integrationsDomain.WooProduct{simpleWooProduct()})
	outbox := &eventrecordtest.Outbox{}

	_, err := newProductImportUC(t, h, 4, outbox).Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	category := eventsBySubject(outbox, "channels.category.imported")
	require.Len(t, category, 1)
	assert.Equal(t, "new-category-1", category[0].Payload["entity_id"])
	assert.Equal(t, "salgadinhos-e-snacks", category[0].Data()["slug"])

	product := eventsBySubject(outbox, "channels.product.imported")
	require.Len(t, product, 1)
	assert.Equal(t, "product-1", product[0].Payload["entity_id"])
	data := product[0].Data()
	assert.Equal(t, "conn-1", data["connection_id"])
	assert.Equal(t, "sku-1", data["sku_id"])
	assert.Equal(t, "new-category-1", data["category_id"])
	assert.Equal(t, "EUR", data["currency_code"])

	listing := eventsBySubject(outbox, "channels.product_listing.created")
	require.Len(t, listing, 1)
	assert.Equal(t, "import", listing[0].Data()["source"])
	assert.Len(t, eventsBySubject(outbox, "channels.product_listing.synced"), 1)
}

func TestImportWooProducts_RecordsTaxClassAndRate(t *testing.T) {
	product := simpleWooProduct()
	product.TaxClass = "reduced-rate"
	h := newHarness([]integrationsDomain.WooProduct{product})
	h.wooClient.taxRates = []integrationsDomain.WooTaxRate{{ID: "1", Country: "NL", Rate: "21.0000", Class: "reduced-rate"}}
	outbox := &eventrecordtest.Outbox{}

	_, err := newProductImportUC(t, h, 5, outbox).Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	class := eventsBySubject(outbox, "channels.tax_class.imported")
	require.Len(t, class, 1)
	assert.Equal(t, map[string]any{"provider": "woocommerce", "external_slug": "reduced-rate", "name": "Reduced Rate"}, class[0].Data())
	rate := eventsBySubject(outbox, "channels.tax_rate.imported")
	require.Len(t, rate, 1)
	assert.Equal(t, "NL", rate[0].Data()["country"])
	assert.Equal(t, float64(2100), rate[0].Data()["rate_basis_points"])
}

func TestImportWooCustomers_RecordsCustomerImported(t *testing.T) {
	h := newCustomersHarness(map[int][]integrationsDomain.WooCustomer{1: {simpleWooCustomer()}})
	outbox := &eventrecordtest.Outbox{}
	uc := appchannels.NewImportWooCustomersUseCase(newMockGormDB(t, 3), h.connRepo, h.wooFactory, testEncryptionSecret, h.customerRepo, outbox)

	_, err := uc.Execute(context.Background(), appchannels.ImportWooCustomersInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "channels.customer.imported", e.Subject)
	assert.Equal(t, "customer-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{
		"provider": "woocommerce", "external_customer_id": float64(42), "email": "jane@example.com", "source": "customer_import",
	}, e.Data())
}

func TestImportWooOrders_RecordsOrderAndItsNewCustomer(t *testing.T) {
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {simpleWooOrder()}})
	h.skuRepo.byCode["WID-A"] = pmsDomain.SKU{ID: "sku-1", SKUCode: "WID-A"}
	outbox := &eventrecordtest.Outbox{}
	uc := appchannels.NewImportWooOrdersUseCase(newMockGormDB(t, 4), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.salesOrderRepo, h.lineItemRepo, h.linkRepo, h.skuRepo, h.customerRepo, h.stock, h.updateOrder, outbox)

	_, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1"})

	require.NoError(t, err)
	customer := eventsBySubject(outbox, "channels.customer.imported")
	require.Len(t, customer, 1)
	assert.Equal(t, "order_import", customer[0].Data()["source"])
	order := eventsBySubject(outbox, "channels.order.imported")
	require.Len(t, order, 1)
	assert.Equal(t, "sales-order-1", order[0].Payload["entity_id"])
	assert.Equal(t, map[string]any{
		"provider": "woocommerce", "connection_id": "conn-1", "external_order_id": "501", "number": "501",
		"status": "processing", "currency_code": "USD", "total_minor_units": float64(2000),
		"line_count": float64(1), "unlinked_line_count": float64(0), "needs_sku_review": false,
	}, order[0].Data())
}

func TestImportBolOrders_RecordsOrderImported(t *testing.T) {
	h := newBolOrdersHarness()
	h.bolClient.listOrdersFn = func(_ context.Context, page int, _ string) ([]integrationsDomain.BolOrder, error) {
		if page == 1 {
			return []integrationsDomain.BolOrder{simpleBolOrder()}, nil
		}
		return nil, nil
	}
	h.bolClient.getOrderFn = func(_ context.Context, _ string) (*integrationsDomain.BolOrder, error) {
		return simpleBolOrderDetail(), nil
	}
	h.skuRepo.byBarcode["0630509892495"] = pmsDomain.SKU{ID: "sku-1", Barcode: "0630509892495"}
	outbox := &eventrecordtest.Outbox{}
	uc := appchannels.NewImportBolOrdersUseCase(newMockGormDB(t, 6), h.connRepo, &fakeBolClientFactory{client: h.bolClient}, testEncryptionSecret,
		h.salesOrderRepo, h.lineItemRepo, h.linkRepo, h.skuRepo, h.stock, h.updateOrder, outbox)

	_, err := uc.Execute(context.Background(), appchannels.ImportBolOrdersInput{ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1"})

	require.NoError(t, err)
	order := eventsBySubject(outbox, "channels.order.imported")
	require.Len(t, order, 1)
	assert.Equal(t, "bol", order[0].Data()["provider"])
	assert.Equal(t, "A2K8290LP8", order[0].Data()["external_order_id"])
}

// Not enough stock for an imported order's line: the shortage is recorded.
func TestImportWooOrders_RecordsStockShortage(t *testing.T) {
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {simpleWooOrder()}})
	h.skuRepo.byCode["WID-A"] = pmsDomain.SKU{ID: "sku-1", SKUCode: "WID-A"}
	h.stock.err = invDomain.ErrInsufficientStock
	outbox := &eventrecordtest.Outbox{}
	uc := appchannels.NewImportWooOrdersUseCase(newMockGormDB(t, 6), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.salesOrderRepo, h.lineItemRepo, h.linkRepo, h.skuRepo, h.customerRepo, h.stock, h.updateOrder, outbox)

	_, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1"})

	require.NoError(t, err)
	shortage := eventsBySubject(outbox, "channels.order.stock_shortage_detected")
	require.Len(t, shortage, 1)
	assert.Equal(t, "sales-order-1", shortage[0].Payload["entity_id"])
	assert.Equal(t, []any{"line-item-1"}, shortage[0].Data()["pending_line_item_ids"])
}

func TestDeleteAllImportedProducts_RecordsDeletedProductsAndCategories(t *testing.T) {
	h := newHarness(nil)
	catID := "cat-1"
	h.productRepo.byID = map[string]pmsDomain.Product{"p1": {ID: "p1", CategoryID: &catID}}
	h.productRepo.countByCategory = map[string]int64{"cat-1": 0}
	h.listingRepo.byConnection = []channelsDomain.ProductChannelListing{{ID: "l1", ProductID: "p1", ConnectionID: "conn-1", ExternalID: "77"}}
	outbox := &eventrecordtest.Outbox{}
	uc := appchannels.NewDeleteAllImportedProductsUseCase(newMockGormDB(t, 4), h.listingRepo, h.productRepo, h.categoryRepo,
		&stubOrdersSalesOrderRepo{}, &stubDeleteSalesOrderUC{}, outbox)

	_, err := uc.Execute(context.Background(), appchannels.DeleteAllImportedProductsInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Equal(t, []string{"channels.imported_product.deleted", "channels.imported_category.deleted"}, outbox.Subjects())
	assert.Equal(t, "p1", outbox.Events[0].Payload["entity_id"])
	assert.Equal(t, map[string]any{"connection_id": "conn-1", "external_id": "77"}, outbox.Events[0].Data())
	assert.Equal(t, "cat-1", outbox.Events[1].Payload["entity_id"])
}

func TestBackfillWooSalePrices_RecordsBackfill(t *testing.T) {
	h, _, _, sku := backfillHarness()
	wp := simpleWooProduct()
	wp.SalePrice, wp.OnSale = "0.99", true
	h.wooClient.products = []integrationsDomain.WooProduct{wp}
	outbox := &eventrecordtest.Outbox{}
	uc := appchannels.NewBackfillWooSalePricesUseCase(newMockGormDB(t, 2), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.listingRepo, h.productRepo, h.variantRepo, h.skuRepo, outbox)

	_, err := uc.Execute(context.Background(), appchannels.BackfillWooSalePricesInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "channels.sale_price.backfilled", e.Subject)
	assert.Equal(t, sku.ID, e.Payload["entity_id"])
	assert.Equal(t, [2]any{nil, float64(99)}, e.Changes()["sale_price_minor_units"])
	assert.Equal(t, map[string]any{"connection_id": "conn-1", "external_id": "23609"}, e.Data())
}

func TestDeleteProductFromChannels_RecordsExternalDelete(t *testing.T) {
	h := newHarness(nil)
	outbox := &eventrecordtest.Outbox{}
	uc := appchannels.NewDeleteProductFromChannelsUseCase(newMockGormDB(t, 2), h.connRepo, h.wooFactory, testEncryptionSecret, outbox)

	require.NoError(t, uc.Execute(context.Background(), []events.ProductDeletedListing{
		{ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555"},
	}))

	e := onlyEvent(t, outbox)
	assert.Equal(t, "channels.external_product.deleted", e.Subject)
	assert.Equal(t, "555", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"provider": "woocommerce", "connection_id": "conn-1"}, e.Data())
}

func TestPushSalesOrderStatusToChannels_RecordsStatusPushed(t *testing.T) {
	connID := "conn-1"
	order := &omsDomain.SalesOrder{
		ID: "so-1", Source: omsDomain.SalesOrderSourceWooCommerce, ExternalOrderID: "501",
		ConnectionID: &connID, Status: omsDomain.SalesOrderStatusOnHold,
	}
	encrypted, err := crypto.Encrypt(testEncryptionSecret, "consumer-secret")
	require.NoError(t, err)
	outbox := &eventrecordtest.Outbox{}
	uc := appchannels.NewPushSalesOrderStatusToChannelsUseCase(newMockGormDB(t, 3), &stubOrdersSalesOrderRepo{findByIDResult: order},
		&stubShipmentRepo{}, &stubCarrierOverrideRepo{}, &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: connID, ConsumerSecretEncrypted: encrypted}},
		&stubWooClientFactory{client: &stubWooClient{}}, testEncryptionSecret, outbox)

	require.NoError(t, uc.Execute(context.Background(), "so-1"))

	e := onlyEvent(t, outbox)
	assert.Equal(t, "channels.order.status_pushed", e.Subject)
	assert.Equal(t, "so-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{
		"provider": "woocommerce", "connection_id": "conn-1", "external_order_id": "501", "status": "on-hold", "channel_status": "on-hold",
	}, e.Data())
}

func TestImportUseCases_FailWhenTheEventCantBeRecorded(t *testing.T) {
	boom := errors.New("outbox down")
	h := newCustomersHarness(map[int][]integrationsDomain.WooCustomer{1: {simpleWooCustomer()}})
	uc := appchannels.NewImportWooCustomersUseCase(newMockGormDB(t, 3), h.connRepo, h.wooFactory, testEncryptionSecret, h.customerRepo,
		&eventrecordtest.Outbox{Err: boom})

	result, err := uc.Execute(context.Background(), appchannels.ImportWooCustomersInput{ConnectionID: "conn-1"})

	require.NoError(t, err, "one customer failing never aborts the import")
	assert.Equal(t, 0, result.Created)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "outbox down")
}
