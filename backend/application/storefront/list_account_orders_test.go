package storefront_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
)

func TestListAccountOrders_scopesToTheSessionsCustomerAndChannel(t *testing.T) {
	sessionRepo := &fakeCustomerSessionRepo{byToken: map[string]storefrontDomain.CustomerSession{"SESSION123": activeSession()}}
	salesOrderRepo := &fakeSalesOrderRepo{listOut: []omsDomain.SalesOrder{
		{Number: "WEB-1", Status: omsDomain.SalesOrderStatusCompleted, TotalMinorUnits: 1000, CurrencyCode: "EUR"},
	}}
	uc := appstorefront.NewListAccountOrdersUseCase(newMockGormDB(t, 1), sessionRepo, salesOrderRepo)

	orders, err := uc.Execute(context.Background(), "channel-1", "SESSION123")

	require.NoError(t, err)
	require.Len(t, orders, 1)
	require.Equal(t, "WEB-1", orders[0].OrderNumber)
	require.NotNil(t, salesOrderRepo.gotFilter.CustomerID)
	require.Equal(t, "customer-1", *salesOrderRepo.gotFilter.CustomerID)
	require.NotNil(t, salesOrderRepo.gotFilter.SalesChannelID)
	require.Equal(t, "channel-1", *salesOrderRepo.gotFilter.SalesChannelID)
}

func TestListAccountOrders_rejectsUnknownSession(t *testing.T) {
	uc := appstorefront.NewListAccountOrdersUseCase(newMockGormDB(t, 1), &fakeCustomerSessionRepo{}, &fakeSalesOrderRepo{})

	_, err := uc.Execute(context.Background(), "channel-1", "missing-session")

	require.ErrorIs(t, err, storefrontDomain.ErrSessionNotFound)
}
