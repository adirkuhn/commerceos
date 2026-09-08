package storefront_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
)

func TestRequestMagicLink_existingCustomer_createsTokenAndEmailsTheLink(t *testing.T) {
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": internalChannel("channel-1")}}
	customerRepo := &fakeCustomerRepo{byEmail: map[string]customersDomain.Customer{"shopper@example.com": {ID: "customer-1"}}}
	tokenRepo := &fakeMagicLinkTokenRepo{}
	emailClient := &fakeEmailClient{}
	uc := appstorefront.NewRequestMagicLinkUseCase(newMockGormDB(t, 2), channelRepo, customerRepo, &fakeCustomerCreator{}, tokenRepo, emailClient)

	err := uc.Execute(context.Background(), appstorefront.RequestMagicLinkInput{
		SalesChannelID: "channel-1", Email: "Shopper@Example.com", RedirectURL: "https://shop.example.com/auth/verify",
	})

	require.NoError(t, err)
	require.Equal(t, "customer-1", tokenRepo.created.CustomerID)
	require.Equal(t, "channel-1", tokenRepo.created.SalesChannelID)
	require.NotEmpty(t, tokenRepo.created.Token)
	require.Len(t, emailClient.sent, 1)
	require.Equal(t, "shopper@example.com", emailClient.sent[0].To)
	require.Contains(t, emailClient.sent[0].HTMLBody, "https://shop.example.com/auth/verify?token="+tokenRepo.created.Token)
}

func TestRequestMagicLink_unknownEmail_createsCustomerFirst(t *testing.T) {
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": internalChannel("channel-1")}}
	customerRepo := &fakeCustomerRepo{}
	createCustomer := &fakeCustomerCreator{out: &customersDomain.Customer{ID: "new-customer-1", Email: "new@example.com"}}
	tokenRepo := &fakeMagicLinkTokenRepo{}
	emailClient := &fakeEmailClient{}
	uc := appstorefront.NewRequestMagicLinkUseCase(newMockGormDB(t, 2), channelRepo, customerRepo, createCustomer, tokenRepo, emailClient)

	err := uc.Execute(context.Background(), appstorefront.RequestMagicLinkInput{
		SalesChannelID: "channel-1", Email: "new@example.com", RedirectURL: "https://shop.example.com/auth/verify",
	})

	require.NoError(t, err)
	require.Equal(t, "new-customer-1", tokenRepo.created.CustomerID)
}

func TestRequestMagicLink_rejectsUnknownChannel(t *testing.T) {
	uc := appstorefront.NewRequestMagicLinkUseCase(newMockGormDB(t, 1), &fakeChannelRepo{}, &fakeCustomerRepo{}, &fakeCustomerCreator{}, &fakeMagicLinkTokenRepo{}, &fakeEmailClient{})

	err := uc.Execute(context.Background(), appstorefront.RequestMagicLinkInput{SalesChannelID: "missing-channel", Email: "a@b.com", RedirectURL: "https://x"})

	require.ErrorIs(t, err, storefrontDomain.ErrNoInternalChannel)
}

func TestRequestMagicLink_propagatesEmailSendFailure(t *testing.T) {
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": internalChannel("channel-1")}}
	customerRepo := &fakeCustomerRepo{byEmail: map[string]customersDomain.Customer{"shopper@example.com": {ID: "customer-1"}}}
	emailClient := &fakeEmailClient{err: errors.New("smtp down")}
	uc := appstorefront.NewRequestMagicLinkUseCase(newMockGormDB(t, 2), channelRepo, customerRepo, &fakeCustomerCreator{}, &fakeMagicLinkTokenRepo{}, emailClient)

	err := uc.Execute(context.Background(), appstorefront.RequestMagicLinkInput{SalesChannelID: "channel-1", Email: "shopper@example.com", RedirectURL: "https://x"})

	require.Error(t, err)
}
