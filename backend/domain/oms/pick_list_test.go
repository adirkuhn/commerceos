package oms_test

import (
	"testing"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func strPtr(s string) *string { return &s }

func TestBuildPickList_sumsSameSKUAcrossOrders(t *testing.T) {
	orders := []omsDomain.SalesOrder{{ID: "o1", Number: "1001"}, {ID: "o2", Number: "1002"}}
	lineItems := []omsDomain.SalesOrderLineItem{
		{ID: "li-1", SalesOrderID: "o1", SKUID: strPtr("sku-a"), SKUCode: "A-1", Barcode: "111", Name: "Apple", Quantity: 2},
		{ID: "li-2", SalesOrderID: "o2", SKUID: strPtr("sku-a"), SKUCode: "A-1", Barcode: "111", Name: "Apple", Quantity: 3},
	}

	list := omsDomain.BuildPickList(orders, lineItems)

	require.Len(t, list.Items, 1)
	item := list.Items[0]
	assert.Equal(t, "sku-a", *item.SKUID)
	assert.Equal(t, "A-1", item.SKUCode)
	assert.Equal(t, "111", item.Barcode)
	assert.Equal(t, int64(5), item.TotalQuantity)
	assert.Equal(t, []omsDomain.PickListOrderLine{
		{SalesOrderID: "o1", OrderNumber: "1001", LineItemID: "li-1", Quantity: 2},
		{SalesOrderID: "o2", OrderNumber: "1002", LineItemID: "li-2", Quantity: 3},
	}, item.Orders)
}

func TestBuildPickList_subtractsRefundedAndSkipsFullyRefunded(t *testing.T) {
	orders := []omsDomain.SalesOrder{{ID: "o1", Number: "1001"}}
	lineItems := []omsDomain.SalesOrderLineItem{
		{ID: "li-1", SalesOrderID: "o1", SKUID: strPtr("sku-a"), SKUCode: "A", Name: "Apple", Quantity: 3, RefundedQuantity: 1},
		{ID: "li-2", SalesOrderID: "o1", SKUID: strPtr("sku-b"), SKUCode: "B", Name: "Banana", Quantity: 2, RefundedQuantity: 2},
	}

	list := omsDomain.BuildPickList(orders, lineItems)

	require.Len(t, list.Items, 1)
	assert.Equal(t, int64(2), list.Items[0].TotalQuantity)
}

func TestBuildPickList_groupsUnlinkedLinesByName(t *testing.T) {
	orders := []omsDomain.SalesOrder{{ID: "o1", Number: "1001"}, {ID: "o2", Number: "1002"}}
	lineItems := []omsDomain.SalesOrderLineItem{
		{ID: "li-1", SalesOrderID: "o1", Name: "Gift wrap", Quantity: 1},
		{ID: "li-2", SalesOrderID: "o2", Name: "Gift wrap", Quantity: 1},
	}

	list := omsDomain.BuildPickList(orders, lineItems)

	require.Len(t, list.Items, 1)
	assert.Nil(t, list.Items[0].SKUID)
	assert.Equal(t, int64(2), list.Items[0].TotalQuantity)
}

func TestBuildPickList_sortsBySKUCodeThenNameWithUncodedLast(t *testing.T) {
	orders := []omsDomain.SalesOrder{{ID: "o1", Number: "1001"}}
	lineItems := []omsDomain.SalesOrderLineItem{
		{ID: "li-1", SalesOrderID: "o1", SKUID: strPtr("sku-b"), SKUCode: "B", Name: "Banana", Quantity: 1},
		{ID: "li-2", SalesOrderID: "o1", Name: "Zucchini (unlinked)", Quantity: 1},
		{ID: "li-3", SalesOrderID: "o1", SKUID: strPtr("sku-a"), SKUCode: "A", Name: "Apple", Quantity: 1},
		{ID: "li-4", SalesOrderID: "o1", Name: "Avocado (unlinked)", Quantity: 1},
	}

	list := omsDomain.BuildPickList(orders, lineItems)

	names := make([]string, len(list.Items))
	for i, it := range list.Items {
		names[i] = it.Name
	}
	assert.Equal(t, []string{"Apple", "Banana", "Avocado (unlinked)", "Zucchini (unlinked)"}, names)
}

func TestBuildPickList_fallsBackToExternalOrderIDForNumber(t *testing.T) {
	orders := []omsDomain.SalesOrder{{ID: "o1", ExternalOrderID: "ext-9"}}
	lineItems := []omsDomain.SalesOrderLineItem{{ID: "li-1", SalesOrderID: "o1", SKUID: strPtr("sku-a"), Name: "Apple", Quantity: 1}}

	list := omsDomain.BuildPickList(orders, lineItems)

	require.Len(t, list.Items, 1)
	assert.Equal(t, "ext-9", list.Items[0].Orders[0].OrderNumber)
}

func TestBuildPickList_emptyInput(t *testing.T) {
	list := omsDomain.BuildPickList(nil, nil)
	assert.Empty(t, list.Items)
}
