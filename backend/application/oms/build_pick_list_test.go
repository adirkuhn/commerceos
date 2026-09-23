package oms_test

import (
	"context"
	"errors"
	"testing"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mapSalesOrderFinder struct {
	orders map[string]omsDomain.SalesOrder
	err    error
}

func (m *mapSalesOrderFinder) FindByID(_ context.Context, id string) (*omsDomain.SalesOrder, error) {
	if m.err != nil {
		return nil, m.err
	}
	o, ok := m.orders[id]
	if !ok {
		return nil, omsDomain.ErrSalesOrderNotFound
	}
	return &o, nil
}

type mapLineItemLister struct {
	byOrder map[string][]omsDomain.SalesOrderLineItem
	err     error
}

func (m *mapLineItemLister) ListBySalesOrder(_ context.Context, id string) ([]omsDomain.SalesOrderLineItem, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.byOrder[id], nil
}

func pickListFixture() (*mapSalesOrderFinder, *mapLineItemLister) {
	sku := "sku-a"
	return &mapSalesOrderFinder{orders: map[string]omsDomain.SalesOrder{
			"o1": {ID: "o1", Number: "1001"},
			"o2": {ID: "o2", Number: "1002"},
		}}, &mapLineItemLister{byOrder: map[string][]omsDomain.SalesOrderLineItem{
			"o1": {{ID: "li-1", SalesOrderID: "o1", SKUID: &sku, SKUCode: "A", Name: "Apple", Quantity: 2}},
			"o2": {{ID: "li-2", SalesOrderID: "o2", SKUID: &sku, SKUCode: "A", Name: "Apple", Quantity: 1}},
		}}
}

func TestBuildPickList_aggregatesSelectedOrders(t *testing.T) {
	orders, lines := pickListFixture()
	uc := appoms.NewBuildPickListUseCase(newMockGormDB(t), orders, lines)

	result, err := uc.Execute(context.Background(), []string{"o1", "o2"})

	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	assert.Equal(t, int64(3), result.Items[0].TotalQuantity)
	assert.Len(t, result.Items[0].Orders, 2)
}

func TestBuildPickList_ignoresDuplicateIDs(t *testing.T) {
	orders, lines := pickListFixture()
	uc := appoms.NewBuildPickListUseCase(newMockGormDB(t), orders, lines)

	result, err := uc.Execute(context.Background(), []string{"o1", "o1"})

	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	assert.Equal(t, int64(2), result.Items[0].TotalQuantity)
}

func TestBuildPickList_rejectsEmptySelection(t *testing.T) {
	orders, lines := pickListFixture()
	uc := appoms.NewBuildPickListUseCase(newMockGormDB(t), orders, lines)

	_, err := uc.Execute(context.Background(), nil)

	assert.ErrorIs(t, err, omsDomain.ErrPickListOrderIDsRequired)
}

func TestBuildPickList_rejectsTooManyOrders(t *testing.T) {
	orders, lines := pickListFixture()
	uc := appoms.NewBuildPickListUseCase(newMockGormDB(t), orders, lines)
	ids := make([]string, omsDomain.MaxPickListOrders+1)
	for i := range ids {
		ids[i] = string(rune('a'+i%26)) + string(rune('0'+i/26))
	}

	_, err := uc.Execute(context.Background(), ids)

	assert.ErrorIs(t, err, omsDomain.ErrPickListTooManyOrders)
}

func TestBuildPickList_propagatesOrderNotFound(t *testing.T) {
	orders, lines := pickListFixture()
	uc := appoms.NewBuildPickListUseCase(newMockGormDB(t), orders, lines)

	_, err := uc.Execute(context.Background(), []string{"o1", "missing"})

	assert.ErrorIs(t, err, omsDomain.ErrSalesOrderNotFound)
}

func TestBuildPickList_propagatesLineItemError(t *testing.T) {
	orders, _ := pickListFixture()
	boom := errors.New("boom")
	uc := appoms.NewBuildPickListUseCase(newMockGormDB(t), orders, &mapLineItemLister{err: boom})

	_, err := uc.Execute(context.Background(), []string{"o1"})

	assert.ErrorIs(t, err, boom)
}
