package storefront_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
)

func orderForLookup(channelID, email string) omsDomain.SalesOrder {
	return omsDomain.SalesOrder{
		Number: "WEB-ABCD1234", Status: omsDomain.SalesOrderStatusProcessing, Fulfilled: false,
		TotalMinorUnits: 1000, CurrencyCode: "EUR", SalesChannelID: &channelID,
		Billing: omsDomain.Address{Email: email},
	}
}

func TestLookupOrder_returnsStatusWhenNumberAndEmailAndChannelMatch(t *testing.T) {
	repo := &fakeSalesOrderRepo{byNumber: map[string]omsDomain.SalesOrder{
		"WEB-ABCD1234": orderForLookup("channel-1", "shopper@example.com"),
	}}
	uc := appstorefront.NewLookupOrderUseCase(newMockGormDB(t, 1), repo)

	status, err := uc.Execute(context.Background(), "channel-1", "WEB-ABCD1234", "Shopper@Example.com")

	require.NoError(t, err)
	require.Equal(t, "WEB-ABCD1234", status.OrderNumber)
	require.Equal(t, omsDomain.SalesOrderStatusProcessing, status.Status)
}

func TestLookupOrder_rejectsMismatchedEmail(t *testing.T) {
	repo := &fakeSalesOrderRepo{byNumber: map[string]omsDomain.SalesOrder{
		"WEB-ABCD1234": orderForLookup("channel-1", "shopper@example.com"),
	}}
	uc := appstorefront.NewLookupOrderUseCase(newMockGormDB(t, 1), repo)

	_, err := uc.Execute(context.Background(), "channel-1", "WEB-ABCD1234", "someone-else@example.com")

	require.ErrorIs(t, err, storefrontDomain.ErrOrderNotFound)
}

func TestLookupOrder_rejectsOrderFromADifferentChannel(t *testing.T) {
	repo := &fakeSalesOrderRepo{byNumber: map[string]omsDomain.SalesOrder{
		"WEB-ABCD1234": orderForLookup("channel-B2B", "shopper@example.com"),
	}}
	uc := appstorefront.NewLookupOrderUseCase(newMockGormDB(t, 1), repo)

	_, err := uc.Execute(context.Background(), "channel-B2C", "WEB-ABCD1234", "shopper@example.com")

	require.ErrorIs(t, err, storefrontDomain.ErrOrderNotFound)
}

func TestLookupOrder_rejectsUnknownOrderNumber(t *testing.T) {
	uc := appstorefront.NewLookupOrderUseCase(newMockGormDB(t, 1), &fakeSalesOrderRepo{})

	_, err := uc.Execute(context.Background(), "channel-1", "WEB-NOPE0000", "shopper@example.com")

	require.ErrorIs(t, err, storefrontDomain.ErrOrderNotFound)
}
