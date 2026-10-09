package channels_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

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

func halfSaved(wo integrationsDomain.WooOrder) integrationsDomain.WooOrder {
	wo.LineItems = wo.LineItems[:1]
	return wo
}

func importOrders(t *testing.T, h *ordersHarness) *appchannels.ImportOrdersResult {
	t.Helper()
	result, err := h.newImportUseCase(t, 8).Execute(context.Background(), appchannels.ImportWooOrdersInput{ConnectionID: "conn-1", DefaultWarehouseID: "warehouse-1"})
	require.NoError(t, err)
	return result
}

func TestImportWooOrders_orderThatAddsUp_isNotFetchedAgain(t *testing.T) {
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {pinducaWooOrder()}})

	importOrders(t, h)

	assert.Empty(t, h.wooClient.getOrderCalled)
	assert.Len(t, h.lineItemRepo.created, 2)
}

func TestImportWooOrders_halfSavedOrder_isReadAgainAndImportedWithEveryLine(t *testing.T) {
	complete := pinducaWooOrder()
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {halfSaved(complete)}})
	h.wooClient.getOrderResult = &complete

	importOrders(t, h)

	assert.Equal(t, []string{"501"}, h.wooClient.getOrderCalled)
	require.Len(t, h.lineItemRepo.created, 2)
	assert.Equal(t, "7896015976606", h.lineItemRepo.created[1].SKUCode, "the line the first read missed")
}

func TestImportWooOrders_stillDoesntAddUp_isImportedExactlyAsWooReturnsIt(t *testing.T) {
	partial := halfSaved(pinducaWooOrder())
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {partial}})
	h.wooClient.getOrderResult = &partial

	result := importOrders(t, h)

	assert.Equal(t, 1, result.Created)
	assert.Len(t, h.lineItemRepo.created, 1, "never skipped, never invented")
}

func TestImportWooOrders_reReadFails_importsWhatWasListed(t *testing.T) {
	h := newOrdersHarness(map[int][]integrationsDomain.WooOrder{1: {halfSaved(pinducaWooOrder())}})
	h.wooClient.getOrderErr = errors.New("woo: timeout")

	result := importOrders(t, h)

	assert.Equal(t, 1, result.Created)
	assert.Len(t, h.lineItemRepo.created, 1)
}
