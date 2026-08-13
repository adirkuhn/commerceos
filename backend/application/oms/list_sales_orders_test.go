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

func TestListSalesOrders_propagatesRepoError(t *testing.T) {
	repo := &stubSalesOrderRepo{listErr: assert.AnError}
	uc := appoms.NewListSalesOrdersUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background(), appoms.ListSalesOrdersInput{})

	require.Error(t, err)
}
