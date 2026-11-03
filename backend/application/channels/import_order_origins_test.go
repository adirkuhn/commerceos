package channels_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

// COM-15: an imported order's link to its shop is its origin row --
// connection + the shop's own order id (+ Woo's order key).
func TestImportWooOrders_recordsTheOrdersOriginWithIt(t *testing.T) {
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {simpleWooOrder()}})
	uc := h.newImportUseCase(t, 4)

	_, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1"})

	require.NoError(t, err)
	require.Len(t, h.origins.recorded, 1)
	got := h.origins.recorded[0]
	assert.Equal(t, h.salesOrderRepo.created[0].ID, got.SalesOrderID)
	assert.Equal(t, integrationsDomain.OriginWooCommerce, got.Origin)
	assert.Equal(t, "conn-1", *got.ConnectionID)
	assert.Equal(t, "501", got.ExternalOrderID)
	assert.Equal(t, "wc_order_abc", got.ExternalKey)
}

// Re-import goes through the importing connection: order #501 of another
// shop is not this one.
func TestImportWooOrders_looksAnExistingOrderUpThroughTheImportingConnection(t *testing.T) {
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {simpleWooOrder()}})
	h.salesOrderRepo.byExternalID["501"] = omsDomain.SalesOrder{ID: "sales-order-501", ExternalOrderID: "501", Status: omsDomain.SalesOrderStatusProcessing}
	uc := h.newImportUseCase(t, 2)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1"})

	require.NoError(t, err)
	assert.Empty(t, h.salesOrderRepo.created)
	assert.Empty(t, h.origins.recorded)
	assert.Equal(t, []string{"conn-1"}, h.origins.gotConnections)
	assert.Equal(t, 1, result.Unchanged+result.Updated)
}

func TestImportWooOrders_anOriginThatCannotBeRecordedFailsThatOrder(t *testing.T) {
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {simpleWooOrder()}})
	uc := h.newImportUseCase(t, 3)
	h.origins.recordErr = errors.New("origin down")

	result, err := uc.Execute(context.Background(), appchannels.ImportWooOrdersInput{ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1"})

	require.NoError(t, err)
	assert.Equal(t, 0, result.Created)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "origin down")
}

func TestImportBolOrders_recordsTheOrdersOriginWithIt(t *testing.T) {
	h := newBolOrdersHarness()
	h.bolClient.listOrdersFn = func(_ context.Context, page int, _ string) ([]integrationsDomain.BolOrder, error) {
		if page == 1 {
			return []integrationsDomain.BolOrder{simpleBolOrder()}, nil
		}
		return nil, nil
	}
	h.bolClient.getOrderFn = func(_ context.Context, _ string) (*integrationsDomain.BolOrder, error) { return simpleBolOrderDetail(), nil }
	uc := h.newImportUseCase(t, 6)

	_, err := uc.Execute(context.Background(), appchannels.ImportBolOrdersInput{ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1"})

	require.NoError(t, err)
	require.Len(t, h.origins.recorded, 1)
	got := h.origins.recorded[0]
	assert.Equal(t, h.salesOrderRepo.created[0].ID, got.SalesOrderID)
	assert.Equal(t, integrationsDomain.OriginBol, got.Origin)
	assert.Equal(t, "conn-1", *got.ConnectionID)
	assert.Equal(t, "A2K8290LP8", got.ExternalOrderID)
	assert.Empty(t, got.ExternalKey)
}

// Losing the create race on the origin's unique key (a second replica
// imported it first) routes to the existing order, not an error.
func TestImportBolOrders_aLostRaceOnTheOriginGoesToTheWinnersOrder(t *testing.T) {
	h := newBolOrdersHarness()
	h.bolClient.listOrdersFn = func(_ context.Context, page int, _ string) ([]integrationsDomain.BolOrder, error) {
		if page == 1 {
			return []integrationsDomain.BolOrder{simpleBolOrder()}, nil
		}
		return nil, nil
	}
	h.bolClient.getOrderFn = func(_ context.Context, _ string) (*integrationsDomain.BolOrder, error) { return simpleBolOrderDetail(), nil }
	h.salesOrderRepo.winnerAfterCreateFail = map[string]omsDomain.SalesOrder{
		"A2K8290LP8": {ID: "winner", ExternalOrderID: "A2K8290LP8", Status: omsDomain.SalesOrderStatusProcessing},
	}
	uc := h.newImportUseCase(t, 4)
	h.origins.recordErr = integrationsDomain.ErrExternalOrderAlreadyImported

	result, err := uc.Execute(context.Background(), appchannels.ImportBolOrdersInput{ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1"})

	require.NoError(t, err)
	assert.Empty(t, result.Errors)
	assert.Equal(t, 1, result.Unchanged)
	assert.Equal(t, 0, result.Created)
}
