package search_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"

	appsearch "github.com/adirkuhn/commerceos/application/search"
)

func TestGlobalSearch_aggregatesAllFourModules(t *testing.T) {
	var gotProductSearch *string
	var gotSupplierQuery, gotCustomerQuery, gotOrderQuery string
	productRepo := &stubProductRepo{listFn: func(_ context.Context, filter pmsDomain.ProductFilter) ([]pmsDomain.Product, int64, error) {
		gotProductSearch = filter.Search
		return []pmsDomain.Product{{ID: "prod-1", Name: "Açaí Bowl"}}, 1, nil
	}}
	supplierRepo := &stubSupplierRepo{searchFn: func(_ context.Context, query string, _ int) ([]suppliersDomain.Supplier, error) {
		gotSupplierQuery = query
		return []suppliersDomain.Supplier{{ID: "sup-1", Name: "Açaí Distribuidora"}}, nil
	}}
	customerRepo := &stubCustomerRepo{searchFn: func(_ context.Context, query string, _ int) ([]customersDomain.Customer, error) {
		gotCustomerQuery = query
		return []customersDomain.Customer{{ID: "cust-1", FirstName: "Açaí"}}, nil
	}}
	salesOrderRepo := &stubSalesOrderRepo{searchFn: func(_ context.Context, query string, _ int) ([]omsDomain.SalesOrder, error) {
		gotOrderQuery = query
		return []omsDomain.SalesOrder{{ID: "so-1", Number: "SO-1"}}, nil
	}}

	uc := appsearch.NewGlobalSearchUseCase(newMockGormDB(t), productRepo, supplierRepo, customerRepo, salesOrderRepo)

	result, err := uc.Execute(context.Background(), "acai")

	require.NoError(t, err)
	require.NotNil(t, gotProductSearch)
	assert.Equal(t, "acai", *gotProductSearch)
	assert.Equal(t, "acai", gotSupplierQuery)
	assert.Equal(t, "acai", gotCustomerQuery)
	assert.Equal(t, "acai", gotOrderQuery)

	require.Len(t, result.Products, 1)
	assert.Equal(t, "Açaí Bowl", result.Products[0].Name)
	require.Len(t, result.Suppliers, 1)
	assert.Equal(t, "Açaí Distribuidora", result.Suppliers[0].Name)
	require.Len(t, result.Customers, 1)
	assert.Equal(t, "Açaí", result.Customers[0].FirstName)
	require.Len(t, result.SalesOrders, 1)
	assert.Equal(t, "SO-1", result.SalesOrders[0].Number)
}

func TestGlobalSearch_emptyQuery_isNoOp(t *testing.T) {
	calledAny := false
	productRepo := &stubProductRepo{listFn: func(context.Context, pmsDomain.ProductFilter) ([]pmsDomain.Product, int64, error) {
		calledAny = true
		return nil, 0, nil
	}}
	supplierRepo := &stubSupplierRepo{searchFn: func(context.Context, string, int) ([]suppliersDomain.Supplier, error) {
		calledAny = true
		return nil, nil
	}}
	customerRepo := &stubCustomerRepo{searchFn: func(context.Context, string, int) ([]customersDomain.Customer, error) {
		calledAny = true
		return nil, nil
	}}
	salesOrderRepo := &stubSalesOrderRepo{searchFn: func(context.Context, string, int) ([]omsDomain.SalesOrder, error) {
		calledAny = true
		return nil, nil
	}}
	// No mock DB expectations set (newMockGormDB not even called) -- an
	// empty query must short-circuit before ever opening a transaction.
	uc := appsearch.NewGlobalSearchUseCase(nil, productRepo, supplierRepo, customerRepo, salesOrderRepo)

	result, err := uc.Execute(context.Background(), "   ")

	require.NoError(t, err)
	assert.False(t, calledAny, "an empty/whitespace-only query must not touch any repo")
	assert.Empty(t, result.Products)
	assert.Empty(t, result.Suppliers)
	assert.Empty(t, result.Customers)
	assert.Empty(t, result.SalesOrders)
}

func TestGlobalSearch_oneModuleFails_wholeSearchFails(t *testing.T) {
	productRepo := &stubProductRepo{listFn: func(context.Context, pmsDomain.ProductFilter) ([]pmsDomain.Product, int64, error) {
		return nil, 0, errors.New("db exploded")
	}}
	supplierRepo := &stubSupplierRepo{searchFn: func(context.Context, string, int) ([]suppliersDomain.Supplier, error) { return nil, nil }}
	customerRepo := &stubCustomerRepo{searchFn: func(context.Context, string, int) ([]customersDomain.Customer, error) { return nil, nil }}
	salesOrderRepo := &stubSalesOrderRepo{searchFn: func(context.Context, string, int) ([]omsDomain.SalesOrder, error) { return nil, nil }}

	uc := appsearch.NewGlobalSearchUseCase(newMockGormDB(t), productRepo, supplierRepo, customerRepo, salesOrderRepo)

	_, err := uc.Execute(context.Background(), "acai")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "db exploded")
}
