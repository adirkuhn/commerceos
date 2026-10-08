package channels_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

type fakeDifferenceRecorder struct {
	calls []appoms.RecordChannelOrderDifferencesInput
	err   error
}

func (f *fakeDifferenceRecorder) Execute(_ context.Context, in appoms.RecordChannelOrderDifferencesInput) error {
	f.calls = append(f.calls, in)
	return f.err
}

func (h *ordersHarness) newImportUseCaseWithDifferences(t *testing.T, n int, recorder *fakeDifferenceRecorder) *appchannels.ImportWooOrdersUseCase {
	t.Helper()
	return appchannels.NewImportWooOrdersUseCase(
		newMockGormDB(t, n), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.salesOrderRepo, h.lineItemRepo, h.linkRepo, h.skuRepo, h.customerRepo, h.stock, h.updateOrder, &eventrecordtest.Outbox{},
		recorder,
	)
}

// pinducaWooOrder: 2 lines, shipping with tax, and a fee -- adds up exactly.
func pinducaWooOrder() integrationsDomain.WooOrder {
	wo := simpleWooOrder()
	wo.Total, wo.ShippingTotal, wo.ShippingTax, wo.TotalTax = "16.00", "4.00", "0.36", "1.00"
	wo.FeeLines = []integrationsDomain.WooFeeLine{{Name: "Ice pack", Total: "1.00", TotalTax: "0.09"}}
	wo.LineItems = []integrationsDomain.WooOrderLineItem{
		{SKU: "7891095031139", Name: "Batata palha", Quantity: 2, Price: "2.47", Total: "4.94", TotalTax: "0.44"},
		{SKU: "7896015976606", Name: "Mistura de Pão de Queijo - Pinduca - 500g", Quantity: 1, Price: "3.48", Total: "4.86", TotalTax: "0.31"},
	}
	return wo
}

func TestImportWooOrders_newOrder_isComparedWithWoo(t *testing.T) {
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {pinducaWooOrder()}})
	recorder := &fakeDifferenceRecorder{}

	_, err := h.newImportUseCaseWithDifferences(t, 6, recorder).Execute(context.Background(), appchannels.ImportWooOrdersInput{ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1"})

	require.NoError(t, err)
	require.Len(t, recorder.calls, 1)
	in := recorder.calls[0]
	assert.Equal(t, h.salesOrderRepo.created[0].ID, in.SalesOrderID)
	assert.Equal(t, []omsDomain.ChannelLine{
		{SKUCode: "7891095031139", Name: "Batata palha", Quantity: 2},
		{SKUCode: "7896015976606", Name: "Mistura de Pão de Queijo - Pinduca - 500g", Quantity: 1},
	}, in.ChannelLines)
	assert.Equal(t, int64(1600), in.ChannelTotalMinorUnits)
	assert.Equal(t, int64(1600), in.ChannelLinesTotalMinorUnits, "lines + tax + shipping + shipping tax + fees")
	assert.Empty(t, h.wooClient.getOrderCalled, "a complete order is not fetched again")
}

func TestImportWooOrders_existingOrder_isComparedEveryTime(t *testing.T) {
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {pinducaWooOrder()}})
	h.salesOrderRepo.byExternalID["501"] = omsDomain.SalesOrder{
		ID: "so-existing", ExternalOrderID: "501", Status: omsDomain.SalesOrderStatusProcessing,
	}
	recorder := &fakeDifferenceRecorder{}

	_, err := h.newImportUseCaseWithDifferences(t, 6, recorder).Execute(context.Background(), appchannels.ImportWooOrdersInput{ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1"})

	require.NoError(t, err)
	require.Len(t, recorder.calls, 1)
	assert.Equal(t, "so-existing", recorder.calls[0].SalesOrderID)
	assert.Empty(t, h.lineItemRepo.created, "an existing order's lines are never changed")
}

func TestImportWooOrders_halfSavedOrder_isFetchedAgainAndImportedComplete(t *testing.T) {
	complete := pinducaWooOrder()
	partial := complete
	partial.LineItems = complete.LineItems[:1]
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {partial}})
	h.wooClient.getOrderResult = &complete
	recorder := &fakeDifferenceRecorder{}

	_, err := h.newImportUseCaseWithDifferences(t, 8, recorder).Execute(context.Background(), appchannels.ImportWooOrdersInput{ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1"})

	require.NoError(t, err)
	assert.Equal(t, []string{"501"}, h.wooClient.getOrderCalled)
	assert.Len(t, h.lineItemRepo.created, 2, "imported 1:1 from the complete order")
	require.Len(t, recorder.calls, 1)
	assert.Len(t, recorder.calls[0].ChannelLines, 2)
}

func TestImportWooOrders_stillDoesntAddUp_importsAsIsAndReportsIt(t *testing.T) {
	partial := pinducaWooOrder()
	partial.LineItems = partial.LineItems[:1]
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {partial}})
	h.wooClient.getOrderResult = &partial
	recorder := &fakeDifferenceRecorder{}

	_, err := h.newImportUseCaseWithDifferences(t, 8, recorder).Execute(context.Background(), appchannels.ImportWooOrdersInput{ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1"})

	require.NoError(t, err)
	assert.Len(t, h.lineItemRepo.created, 1, "1:1 with what Woo returned -- nothing invented")
	require.Len(t, recorder.calls, 1)
	assert.Equal(t, int64(1600), recorder.calls[0].ChannelTotalMinorUnits)
	assert.Equal(t, int64(1083), recorder.calls[0].ChannelLinesTotalMinorUnits)
}

func TestImportWooOrders_refetchFails_importsWhatWasListed(t *testing.T) {
	partial := pinducaWooOrder()
	partial.LineItems = partial.LineItems[:1]
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {partial}})
	h.wooClient.getOrderErr = errors.New("woo: timeout")
	recorder := &fakeDifferenceRecorder{}

	result, err := h.newImportUseCaseWithDifferences(t, 8, recorder).Execute(context.Background(), appchannels.ImportWooOrdersInput{ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1"})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Created)
	assert.Len(t, recorder.calls, 1)
}

func TestImportWooOrders_comparisonFails_isReportedButImportStands(t *testing.T) {
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {pinducaWooOrder()}})
	recorder := &fakeDifferenceRecorder{err: errors.New("db down")}

	result, err := h.newImportUseCaseWithDifferences(t, 6, recorder).Execute(context.Background(), appchannels.ImportWooOrdersInput{ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1"})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Created)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "compare with woo")
}
