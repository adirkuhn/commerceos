package oms_test

import (
	"context"
	"testing"
	"time"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListSalesOrders_defaultsPageAndPerPage(t *testing.T) {
	repo := &stubSalesOrderRepo{listResult: []omsDomain.SalesOrder{{ID: "so-1"}}, listTotal: 1}
	uc := appoms.NewListSalesOrdersUseCase(newMockGormDB(t), repo, nil)

	result, err := uc.Execute(context.Background(), appoms.ListSalesOrdersInput{})

	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
	assert.Equal(t, 20, repo.gotListFilter.Limit)
	assert.Equal(t, 0, repo.gotListFilter.Offset)
}

func TestListSalesOrders_computesOffsetFromPage(t *testing.T) {
	repo := &stubSalesOrderRepo{}
	uc := appoms.NewListSalesOrdersUseCase(newMockGormDB(t), repo, nil)

	_, err := uc.Execute(context.Background(), appoms.ListSalesOrdersInput{Page: 3, PerPage: 20})

	require.NoError(t, err)
	assert.Equal(t, 40, repo.gotListFilter.Offset)
}

func TestListSalesOrders_capsPerPageAtMax(t *testing.T) {
	repo := &stubSalesOrderRepo{}
	uc := appoms.NewListSalesOrdersUseCase(newMockGormDB(t), repo, nil)

	_, err := uc.Execute(context.Background(), appoms.ListSalesOrdersInput{PerPage: 10000})

	require.NoError(t, err)
	assert.Equal(t, 200, repo.gotListFilter.Limit)
}

func TestListSalesOrders_passesSortByAndSortDirThrough(t *testing.T) {
	repo := &stubSalesOrderRepo{}
	uc := appoms.NewListSalesOrdersUseCase(newMockGormDB(t), repo, nil)

	_, err := uc.Execute(context.Background(), appoms.ListSalesOrdersInput{
		SortBy: omsDomain.SalesOrderSortTotal, SortDir: omsDomain.SortAsc,
	})

	require.NoError(t, err)
	assert.Equal(t, omsDomain.SalesOrderSortTotal, repo.gotListFilter.SortBy)
	assert.Equal(t, omsDomain.SortAsc, repo.gotListFilter.SortDir)
}

func TestListSalesOrders_passesFulfilledFilterThrough(t *testing.T) {
	repo := &stubSalesOrderRepo{}
	uc := appoms.NewListSalesOrdersUseCase(newMockGormDB(t), repo, nil)
	unfulfilled := false

	_, err := uc.Execute(context.Background(), appoms.ListSalesOrdersInput{Fulfilled: &unfulfilled})

	require.NoError(t, err)
	require.NotNil(t, repo.gotListFilter.Fulfilled)
	assert.False(t, *repo.gotListFilter.Fulfilled)
}

func TestListSalesOrders_passesAwaitingPickingThrough(t *testing.T) {
	repo := &stubSalesOrderRepo{}
	uc := appoms.NewListSalesOrdersUseCase(newMockGormDB(t), repo, nil)

	_, err := uc.Execute(context.Background(), appoms.ListSalesOrdersInput{AwaitingPicking: true})

	require.NoError(t, err)
	assert.True(t, repo.gotListFilter.AwaitingPicking)
}

func TestListSalesOrders_passesStatusFilterThrough(t *testing.T) {
	repo := &stubSalesOrderRepo{}
	uc := appoms.NewListSalesOrdersUseCase(newMockGormDB(t), repo, nil)
	statuses := []omsDomain.SalesOrderStatus{omsDomain.SalesOrderStatusPending, omsDomain.SalesOrderStatusProcessing}

	_, err := uc.Execute(context.Background(), appoms.ListSalesOrdersInput{Status: statuses})

	require.NoError(t, err)
	assert.Equal(t, statuses, repo.gotListFilter.Status)
}

func TestListSalesOrders_passesSourceFilterThrough(t *testing.T) {
	repo := &stubSalesOrderRepo{}
	uc := appoms.NewListSalesOrdersUseCase(newMockGormDB(t), repo, nil)
	sources := []omsDomain.SalesOrderSource{omsDomain.SalesOrderSourcePOS, omsDomain.SalesOrderSourceWooCommerce}

	_, err := uc.Execute(context.Background(), appoms.ListSalesOrdersInput{Source: sources})

	require.NoError(t, err)
	assert.Equal(t, sources, repo.gotListFilter.Source)
}

func TestListSalesOrders_passesCustomerIDFilterThrough(t *testing.T) {
	repo := &stubSalesOrderRepo{}
	uc := appoms.NewListSalesOrdersUseCase(newMockGormDB(t), repo, nil)
	customerID := "cust-1"

	_, err := uc.Execute(context.Background(), appoms.ListSalesOrdersInput{CustomerID: &customerID})

	require.NoError(t, err)
	require.NotNil(t, repo.gotListFilter.CustomerID)
	assert.Equal(t, "cust-1", *repo.gotListFilter.CustomerID)
}

func TestListSalesOrders_propagatesRepoError(t *testing.T) {
	repo := &stubSalesOrderRepo{listErr: assert.AnError}
	uc := appoms.NewListSalesOrdersUseCase(newMockGormDB(t), repo, nil)

	_, err := uc.Execute(context.Background(), appoms.ListSalesOrdersInput{})

	require.Error(t, err)
}

type stubListSettings struct {
	settings settingsDomain.TenantSettings
	err      error
}

func (s *stubListSettings) Get(_ context.Context) (*settingsDomain.TenantSettings, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &s.settings, nil
}

func TestListSalesOrders_shipDueFiltersOnTodayInTheTenantTimezone(t *testing.T) {
	repo := &stubSalesOrderRepo{}
	uc := appoms.NewListSalesOrdersUseCase(newMockGormDB(t), repo, &stubListSettings{settings: settingsDomain.TenantSettings{Timezone: "Europe/Amsterdam"}})
	// 22:30 UTC on the 12th is already the 13th in Amsterdam.
	uc.Now = func() time.Time { return time.Date(2026, 10, 12, 22, 30, 0, 0, time.UTC) }

	_, err := uc.Execute(context.Background(), appoms.ListSalesOrdersInput{ShipDue: true})

	require.NoError(t, err)
	require.NotNil(t, repo.gotListFilter.ShipDueOnOrBefore)
	assert.Equal(t, "2026-10-13", repo.gotListFilter.ShipDueOnOrBefore.Format(time.DateOnly))
}

func TestListSalesOrders_withoutShipDueLeavesTheFilterOff(t *testing.T) {
	repo := &stubSalesOrderRepo{}
	uc := appoms.NewListSalesOrdersUseCase(newMockGormDB(t), repo, &stubListSettings{err: assert.AnError})

	_, err := uc.Execute(context.Background(), appoms.ListSalesOrdersInput{})

	require.NoError(t, err, "settings are only read for ship_due")
	assert.Nil(t, repo.gotListFilter.ShipDueOnOrBefore)
}

func TestListSalesOrders_shipDueWrapsSettingsError(t *testing.T) {
	uc := appoms.NewListSalesOrdersUseCase(newMockGormDB(t), &stubSalesOrderRepo{}, &stubListSettings{err: assert.AnError})

	_, err := uc.Execute(context.Background(), appoms.ListSalesOrdersInput{ShipDue: true})

	require.ErrorIs(t, err, assert.AnError)
	assert.Contains(t, err.Error(), "listSalesOrders")
}
