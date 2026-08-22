package oms_test

import (
	"context"
	"testing"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListSalesOrders_defaultsPageAndPerPage(t *testing.T) {
	repo := &stubSalesOrderRepo{listResult: []omsDomain.SalesOrder{{ID: "so-1"}}, listTotal: 1}
	uc := appoms.NewListSalesOrdersUseCase(newMockGormDB(t), repo)

	result, err := uc.Execute(context.Background(), appoms.ListSalesOrdersInput{})

	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
	assert.Equal(t, 20, repo.gotListFilter.Limit)
	assert.Equal(t, 0, repo.gotListFilter.Offset)
}

func TestListSalesOrders_computesOffsetFromPage(t *testing.T) {
	repo := &stubSalesOrderRepo{}
	uc := appoms.NewListSalesOrdersUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background(), appoms.ListSalesOrdersInput{Page: 3, PerPage: 20})

	require.NoError(t, err)
	assert.Equal(t, 40, repo.gotListFilter.Offset)
}

func TestListSalesOrders_capsPerPageAtMax(t *testing.T) {
	repo := &stubSalesOrderRepo{}
	uc := appoms.NewListSalesOrdersUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background(), appoms.ListSalesOrdersInput{PerPage: 10000})

	require.NoError(t, err)
	assert.Equal(t, 200, repo.gotListFilter.Limit)
}

func TestListSalesOrders_passesSortByAndSortDirThrough(t *testing.T) {
	repo := &stubSalesOrderRepo{}
	uc := appoms.NewListSalesOrdersUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background(), appoms.ListSalesOrdersInput{
		SortBy: omsDomain.SalesOrderSortTotal, SortDir: omsDomain.SortAsc,
	})

	require.NoError(t, err)
	assert.Equal(t, omsDomain.SalesOrderSortTotal, repo.gotListFilter.SortBy)
	assert.Equal(t, omsDomain.SortAsc, repo.gotListFilter.SortDir)
}

func TestListSalesOrders_passesFulfilledFilterThrough(t *testing.T) {
	repo := &stubSalesOrderRepo{}
	uc := appoms.NewListSalesOrdersUseCase(newMockGormDB(t), repo)
	unfulfilled := false

	_, err := uc.Execute(context.Background(), appoms.ListSalesOrdersInput{Fulfilled: &unfulfilled})

	require.NoError(t, err)
	require.NotNil(t, repo.gotListFilter.Fulfilled)
	assert.False(t, *repo.gotListFilter.Fulfilled)
}

func TestListSalesOrders_propagatesRepoError(t *testing.T) {
	repo := &stubSalesOrderRepo{listErr: assert.AnError}
	uc := appoms.NewListSalesOrdersUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background(), appoms.ListSalesOrdersInput{})

	require.Error(t, err)
}
