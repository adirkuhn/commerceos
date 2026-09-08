package storefront_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
)

func activeSession() storefrontDomain.CustomerSession {
	return storefrontDomain.CustomerSession{
		ID: "session-1", TenantID: "tenant-1", SalesChannelID: "channel-1", CustomerID: "customer-1",
		Token: "SESSION123", ExpiresAt: time.Now().Add(time.Hour),
	}
}

func TestGetAccount_returnsTheSignedInCustomer(t *testing.T) {
	sessionRepo := &fakeCustomerSessionRepo{byToken: map[string]storefrontDomain.CustomerSession{"SESSION123": activeSession()}}
	customerRepo := &fakeCustomerRepo{byID: map[string]customersDomain.Customer{"customer-1": {ID: "customer-1", Email: "shopper@example.com", FirstName: "Ada"}}}
	uc := appstorefront.NewGetAccountUseCase(newMockGormDB(t, 1), sessionRepo, customerRepo)

	account, err := uc.Execute(context.Background(), "channel-1", "SESSION123")

	require.NoError(t, err)
	require.Equal(t, "shopper@example.com", account.Email)
	require.Equal(t, "Ada", account.FirstName)
}

func TestGetAccount_rejectsUnknownSession(t *testing.T) {
	uc := appstorefront.NewGetAccountUseCase(newMockGormDB(t, 1), &fakeCustomerSessionRepo{}, &fakeCustomerRepo{})

	_, err := uc.Execute(context.Background(), "channel-1", "missing-session")

	require.ErrorIs(t, err, storefrontDomain.ErrSessionNotFound)
}

func TestGetAccount_rejectsExpiredSession(t *testing.T) {
	expired := activeSession()
	expired.ExpiresAt = time.Now().Add(-time.Hour)
	sessionRepo := &fakeCustomerSessionRepo{byToken: map[string]storefrontDomain.CustomerSession{"SESSION123": expired}}
	uc := appstorefront.NewGetAccountUseCase(newMockGormDB(t, 1), sessionRepo, &fakeCustomerRepo{})

	_, err := uc.Execute(context.Background(), "channel-1", "SESSION123")

	require.ErrorIs(t, err, storefrontDomain.ErrSessionNotFound)
}

func TestGetAccount_rejectsWrongChannelSession(t *testing.T) {
	sessionRepo := &fakeCustomerSessionRepo{byToken: map[string]storefrontDomain.CustomerSession{"SESSION123": activeSession()}}
	uc := appstorefront.NewGetAccountUseCase(newMockGormDB(t, 1), sessionRepo, &fakeCustomerRepo{})

	_, err := uc.Execute(context.Background(), "channel-2", "SESSION123")

	require.ErrorIs(t, err, storefrontDomain.ErrSessionNotFound)
}
