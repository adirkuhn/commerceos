package channels_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

type bolOrdersHarness struct {
	connRepo       *stubConnectionRepo
	bolClient      *fakeBolClient
	salesOrderRepo *stubOrdersSalesOrderRepo
	lineItemRepo   *stubOrdersLineItemRepo
	linkRepo       *stubOrdersLinkRepo
	skuRepo        *stubChannelsSKURepo
	stock          *stubSaleStockAllocator
	updateOrder    *stubUpdateSalesOrderUC
}

func newBolOrdersHarness() *bolOrdersHarness {
	conn := bolConn()
	return &bolOrdersHarness{
		connRepo:       &stubConnectionRepo{conn: &conn},
		bolClient:      &fakeBolClient{},
		salesOrderRepo: &stubOrdersSalesOrderRepo{byExternalID: map[string]omsDomain.SalesOrder{}},
		lineItemRepo:   &stubOrdersLineItemRepo{},
		linkRepo:       &stubOrdersLinkRepo{},
		skuRepo:        &stubChannelsSKURepo{byBarcode: map[string]pmsDomain.SKU{}},
		stock:          &stubSaleStockAllocator{},
		updateOrder:    &stubUpdateSalesOrderUC{},
	}
}

func (h *bolOrdersHarness) newImportUseCase(t *testing.T, n int) *appchannels.ImportBolOrdersUseCase {
	t.Helper()
	return appchannels.NewImportBolOrdersUseCase(
		newMockGormDB(t, n), h.connRepo, &fakeBolClientFactory{client: h.bolClient}, testEncryptionSecret,
		h.salesOrderRepo, h.lineItemRepo, h.linkRepo, h.skuRepo, h.stock, h.updateOrder,
	)
}

func simpleBolOrder() integrationsDomain.BolOrder {
	return integrationsDomain.BolOrder{
		OrderID: "A2K8290LP8",
		OrderItems: []integrationsDomain.BolOrderItem{
			{OrderItemID: "item-1", EAN: "0630509892495", Quantity: 2, QuantityShipped: 0, QuantityCancelled: 0},
		},
	}
}

func simpleBolOrderDetail() *integrationsDomain.BolOrder {
	return &integrationsDomain.BolOrder{
		OrderID: "A2K8290LP8",
		ShipmentDetails: integrationsDomain.BolShipmentDetails{
			FirstName: "Billie", Surname: "Jansen", StreetName: "Dorpstraat", HouseNumber: "1",
			ZipCode: "1111ZZ", City: "Utrecht", CountryCode: "NL", Email: "billie@verkopen.bol.com",
		},
		OrderItems: []integrationsDomain.BolOrderItem{
			{OrderItemID: "item-1", EAN: "0630509892495", Title: "Widget A", Quantity: 2, UnitPriceMinorUnits: 1000},
		},
	}
}

func TestImportBolOrders_createsANewOrderAndDecrementsStock(t *testing.T) {
	h := newBolOrdersHarness()
	h.bolClient.listOrdersFn = func(_ context.Context, page int, status string) ([]integrationsDomain.BolOrder, error) {
		assert.Equal(t, "OPEN", status)
		if page == 1 {
			return []integrationsDomain.BolOrder{simpleBolOrder()}, nil
		}
		return nil, nil
	}
	h.bolClient.getOrderFn = func(_ context.Context, orderID string) (*integrationsDomain.BolOrder, error) {
		assert.Equal(t, "A2K8290LP8", orderID)
		return simpleBolOrderDetail(), nil
	}
	h.skuRepo.byBarcode["0630509892495"] = pmsDomain.SKU{ID: "sku-1", Barcode: "0630509892495"}
	uc := h.newImportUseCase(t, 6)

	result, err := uc.Execute(context.Background(), appchannels.ImportBolOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1",
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Created)
	assert.Empty(t, result.Errors)
	require.Len(t, h.salesOrderRepo.created, 1)
	created := h.salesOrderRepo.created[0]
	assert.Equal(t, omsDomain.SalesOrderSourceBol, created.Source)
	assert.Equal(t, "A2K8290LP8", created.ExternalOrderID)
	assert.Equal(t, omsDomain.SalesOrderStatusProcessing, created.Status)
	// The customer pays bol.com, not the store -- no transaction id of its
	// own, but the order must still read as "paid via bol.com".
	assert.Equal(t, "bol", created.PaymentMethod)
	assert.Equal(t, "bol.com", created.PaymentMethodTitle)
	assert.Equal(t, omsDomain.PaymentStatusPaid, created.PaymentStatus())
	assert.True(t, created.StockDecremented)
	assert.False(t, created.Fulfilled)
	assert.Equal(t, "Billie", created.Shipping.FirstName)
	assert.Equal(t, int64(2000), created.TotalMinorUnits)
	require.Len(t, h.lineItemRepo.created, 1)
	require.NotNil(t, h.lineItemRepo.created[0].SKUID)
	assert.Equal(t, "sku-1", *h.lineItemRepo.created[0].SKUID)
	assert.Equal(t, int64(1000), h.lineItemRepo.created[0].PriceMinorUnits)
	require.Len(t, h.stock.calls, 1)
	assert.Equal(t, int64(2), h.stock.calls[0].Quantity)
	require.Len(t, h.linkRepo.created, 1)
}

func TestImportBolOrders_fullyShippedOrderArrivesCompleted(t *testing.T) {
	h := newBolOrdersHarness()
	order := simpleBolOrder()
	order.OrderItems[0].QuantityShipped = 2
	h.bolClient.listOrdersFn = func(_ context.Context, page int, _ string) ([]integrationsDomain.BolOrder, error) {
		if page == 1 {
			return []integrationsDomain.BolOrder{order}, nil
		}
		return nil, nil
	}
	detail := simpleBolOrderDetail()
	detail.OrderItems[0].QuantityShipped = 2
	h.bolClient.getOrderFn = func(_ context.Context, _ string) (*integrationsDomain.BolOrder, error) { return detail, nil }
	h.skuRepo.byBarcode["0630509892495"] = pmsDomain.SKU{ID: "sku-1"}
	uc := h.newImportUseCase(t, 6)

	result, err := uc.Execute(context.Background(), appchannels.ImportBolOrdersInput{ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1"})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Created)
	require.Len(t, h.salesOrderRepo.created, 1)
	assert.Equal(t, omsDomain.SalesOrderStatusCompleted, h.salesOrderRepo.created[0].Status)
	assert.True(t, h.salesOrderRepo.created[0].Fulfilled)
}

func TestImportBolOrders_cancelledOrderWithNothingShippedArrivesCancelledAndDoesNotDecrementStock(t *testing.T) {
	h := newBolOrdersHarness()
	order := simpleBolOrder()
	order.OrderItems[0].QuantityCancelled = 2
	order.OrderItems[0].CancellationRequest = true
	h.bolClient.listOrdersFn = func(_ context.Context, page int, _ string) ([]integrationsDomain.BolOrder, error) {
		if page == 1 {
			return []integrationsDomain.BolOrder{order}, nil
		}
		return nil, nil
	}
	detail := simpleBolOrderDetail()
	detail.OrderItems[0].QuantityCancelled = 2
	h.bolClient.getOrderFn = func(_ context.Context, _ string) (*integrationsDomain.BolOrder, error) { return detail, nil }
	uc := h.newImportUseCase(t, 5)

	result, err := uc.Execute(context.Background(), appchannels.ImportBolOrdersInput{ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1"})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Created)
	require.Len(t, h.salesOrderRepo.created, 1)
	assert.Equal(t, omsDomain.SalesOrderStatusCancelled, h.salesOrderRepo.created[0].Status)
	assert.False(t, h.salesOrderRepo.created[0].StockDecremented)
	assert.Empty(t, h.stock.calls)
}

func TestImportBolOrders_isIdempotentWhenStatusUnchanged(t *testing.T) {
	h := newBolOrdersHarness()
	h.bolClient.listOrdersFn = func(_ context.Context, page int, _ string) ([]integrationsDomain.BolOrder, error) {
		if page == 1 {
			return []integrationsDomain.BolOrder{simpleBolOrder()}, nil
		}
		return nil, nil
	}
	h.salesOrderRepo.byExternalID["A2K8290LP8"] = omsDomain.SalesOrder{ID: "so-1", Status: omsDomain.SalesOrderStatusProcessing}
	uc := h.newImportUseCase(t, 4)

	result, err := uc.Execute(context.Background(), appchannels.ImportBolOrdersInput{ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1"})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Unchanged)
	assert.Equal(t, 0, result.Created)
	assert.Empty(t, h.updateOrder.calls)
}

func TestImportBolOrders_updatesStatusOnAnExistingOrder(t *testing.T) {
	h := newBolOrdersHarness()
	order := simpleBolOrder()
	order.OrderItems[0].QuantityShipped = 2
	h.bolClient.listOrdersFn = func(_ context.Context, page int, _ string) ([]integrationsDomain.BolOrder, error) {
		if page == 1 {
			return []integrationsDomain.BolOrder{order}, nil
		}
		return nil, nil
	}
	h.salesOrderRepo.byExternalID["A2K8290LP8"] = omsDomain.SalesOrder{ID: "so-1", Status: omsDomain.SalesOrderStatusProcessing}
	uc := h.newImportUseCase(t, 4)

	result, err := uc.Execute(context.Background(), appchannels.ImportBolOrdersInput{ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1"})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Updated)
	require.Len(t, h.updateOrder.calls, 1)
	require.NotNil(t, h.updateOrder.calls[0].Status)
	assert.Equal(t, omsDomain.SalesOrderStatusCompleted, *h.updateOrder.calls[0].Status)
}

func TestImportBolOrders_unmatchedEANImportsUnlinkedInsteadOfDroppingTheLine(t *testing.T) {
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
	uc := h.newImportUseCase(t, 5)

	result, err := uc.Execute(context.Background(), appchannels.ImportBolOrdersInput{ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1"})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Created)
	require.Len(t, h.salesOrderRepo.created, 1)
	require.NotEmpty(t, h.salesOrderRepo.updateCalls, "order must be flagged for the Unmatched SKUs review page")
	assert.Equal(t, true, h.salesOrderRepo.updateCalls[len(h.salesOrderRepo.updateCalls)-1].Fields["needs_sku_review"])
	require.Len(t, h.lineItemRepo.created, 1)
	assert.Nil(t, h.lineItemRepo.created[0].SKUID)
	assert.Empty(t, h.stock.calls, "unlinked line has nothing to decrement yet")
}

func TestImportBolOrders_skipStockMovementsCreatesOrderWithoutTouchingInventory(t *testing.T) {
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
	h.skuRepo.byBarcode["0630509892495"] = pmsDomain.SKU{ID: "sku-1"}
	uc := h.newImportUseCase(t, 5)

	result, err := uc.Execute(context.Background(), appchannels.ImportBolOrdersInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1", SkipStockMovements: true,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Created)
	assert.False(t, h.salesOrderRepo.created[0].StockDecremented)
	assert.Empty(t, h.stock.calls)
	assert.Empty(t, h.linkRepo.created)
}

func TestImportBolOrders_losingTheCreateRaceRoutesToUpdateInsteadOfFailing(t *testing.T) {
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
	h.salesOrderRepo.createErr = omsDomain.ErrDuplicateExternalOrder
	h.salesOrderRepo.winnerAfterCreateFail = map[string]omsDomain.SalesOrder{
		"A2K8290LP8": {ID: "so-winner", Status: omsDomain.SalesOrderStatusProcessing},
	}
	uc := h.newImportUseCase(t, 6)

	result, err := uc.Execute(context.Background(), appchannels.ImportBolOrdersInput{ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1"})

	require.NoError(t, err)
	assert.Empty(t, result.Errors)
	assert.Equal(t, 1, result.Unchanged)
	assert.Equal(t, 0, result.Created)
}

func TestImportBolOrders_aPageFetchFailureIsReportedNotReturnedAsAHardError(t *testing.T) {
	h := newBolOrdersHarness()
	h.bolClient.listOrdersFn = func(_ context.Context, _ int, _ string) ([]integrationsDomain.BolOrder, error) {
		return nil, assert.AnError
	}
	uc := h.newImportUseCase(t, 2)

	result, err := uc.Execute(context.Background(), appchannels.ImportBolOrdersInput{ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1"})

	require.NoError(t, err)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "page 1")
}
