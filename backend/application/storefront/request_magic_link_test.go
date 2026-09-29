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

const hostedStorefronts = "https://shops.example.com"

// headlessChannel is an internal channel whose merchant configured the
// page their own frontend completes sign-in on.
func headlessChannel(id, signInURL string) channelsDomain.SalesChannel {
	c := internalChannel(id)
	c.StorefrontMode = channelsDomain.StorefrontModeHeadless
	c.SignInURL = &signInURL
	return c
}

func newMagicLinkUC(t *testing.T, txs int, channels map[string]channelsDomain.SalesChannel, customers *fakeCustomerRepo, creator *fakeCustomerCreator, tokens *fakeMagicLinkTokenRepo, email *fakeEmailClient) *appstorefront.RequestMagicLinkUseCase {
	return appstorefront.NewRequestMagicLinkUseCase(newMockGormDB(t, txs), &fakeChannelRepo{byID: channels}, customers, creator, tokens, email, hostedStorefronts)
}

func TestRequestMagicLink_existingCustomer_createsTokenAndEmailsTheLink(t *testing.T) {
	customerRepo := &fakeCustomerRepo{byEmail: map[string]customersDomain.Customer{"shopper@example.com": {ID: "customer-1"}}}
	tokenRepo := &fakeMagicLinkTokenRepo{}
	emailClient := &fakeEmailClient{}
	uc := newMagicLinkUC(t, 2, map[string]channelsDomain.SalesChannel{"channel-1": headlessChannel("channel-1", "https://shop.example.com/auth/verify")},
		customerRepo, &fakeCustomerCreator{}, tokenRepo, emailClient)

	err := uc.Execute(context.Background(), appstorefront.RequestMagicLinkInput{SalesChannelID: "channel-1", Email: "Shopper@Example.com"})

	require.NoError(t, err)
	require.Equal(t, "customer-1", tokenRepo.created.CustomerID)
	require.Equal(t, "channel-1", tokenRepo.created.SalesChannelID)
	require.NotEmpty(t, tokenRepo.created.Token)
	require.Len(t, emailClient.sent, 1)
	require.Equal(t, "shopper@example.com", emailClient.sent[0].To)
	require.Contains(t, emailClient.sent[0].HTMLBody, "https://shop.example.com/auth/verify?token="+tokenRepo.created.Token)
}

// The link always goes to the channel's own sign-in page. Nothing in the
// shopper's request can point it anywhere else -- otherwise anyone could
// request a link for a victim's email aimed at their own site and take the
// token (account takeover).
func TestRequestMagicLink_themedStoreLinksToItsHostedLoginPage(t *testing.T) {
	channel := internalChannel("channel-1")
	channel.StorefrontMode = channelsDomain.StorefrontModeTheme
	tokenRepo := &fakeMagicLinkTokenRepo{}
	emailClient := &fakeEmailClient{}
	uc := newMagicLinkUC(t, 2, map[string]channelsDomain.SalesChannel{"channel-1": channel},
		&fakeCustomerRepo{byEmail: map[string]customersDomain.Customer{"shopper@example.com": {ID: "customer-1"}}}, &fakeCustomerCreator{}, tokenRepo, emailClient)

	require.NoError(t, uc.Execute(context.Background(), appstorefront.RequestMagicLinkInput{SalesChannelID: "channel-1", Email: "shopper@example.com"}))

	require.Contains(t, emailClient.sent[0].HTMLBody, `href="https://shops.example.com/store/channel-1/login?token=`+tokenRepo.created.Token+`"`)
}

func TestRequestMagicLink_headlessStoreWithoutSignInPageSendsNothing(t *testing.T) {
	customerRepo := &fakeCustomerRepo{}
	creator := &fakeCustomerCreator{out: &customersDomain.Customer{ID: "new-customer-1"}}
	tokenRepo := &fakeMagicLinkTokenRepo{}
	emailClient := &fakeEmailClient{}
	uc := newMagicLinkUC(t, 1, map[string]channelsDomain.SalesChannel{"channel-1": internalChannel("channel-1")}, customerRepo, creator, tokenRepo, emailClient)

	err := uc.Execute(context.Background(), appstorefront.RequestMagicLinkInput{SalesChannelID: "channel-1", Email: "new@example.com"})

	require.ErrorIs(t, err, channelsDomain.ErrSignInNotConfigured)
	require.Nil(t, tokenRepo.created)
	require.Empty(t, emailClient.sent)
}

func TestRequestMagicLink_unknownEmail_createsCustomerFirst(t *testing.T) {
	createCustomer := &fakeCustomerCreator{out: &customersDomain.Customer{ID: "new-customer-1", Email: "new@example.com"}}
	tokenRepo := &fakeMagicLinkTokenRepo{}
	uc := newMagicLinkUC(t, 2, map[string]channelsDomain.SalesChannel{"channel-1": headlessChannel("channel-1", "https://shop.example.com/auth/verify")},
		&fakeCustomerRepo{}, createCustomer, tokenRepo, &fakeEmailClient{})

	err := uc.Execute(context.Background(), appstorefront.RequestMagicLinkInput{SalesChannelID: "channel-1", Email: "new@example.com"})

	require.NoError(t, err)
	require.Equal(t, "new-customer-1", tokenRepo.created.CustomerID)
}

func TestRequestMagicLink_rejectsUnknownChannel(t *testing.T) {
	uc := newMagicLinkUC(t, 1, nil, &fakeCustomerRepo{}, &fakeCustomerCreator{}, &fakeMagicLinkTokenRepo{}, &fakeEmailClient{})

	err := uc.Execute(context.Background(), appstorefront.RequestMagicLinkInput{SalesChannelID: "missing-channel", Email: "a@b.com"})

	require.ErrorIs(t, err, storefrontDomain.ErrNoInternalChannel)
}

func TestRequestMagicLink_propagatesEmailSendFailure(t *testing.T) {
	customerRepo := &fakeCustomerRepo{byEmail: map[string]customersDomain.Customer{"shopper@example.com": {ID: "customer-1"}}}
	uc := newMagicLinkUC(t, 2, map[string]channelsDomain.SalesChannel{"channel-1": headlessChannel("channel-1", "https://shop.example.com/v")},
		customerRepo, &fakeCustomerCreator{}, &fakeMagicLinkTokenRepo{}, &fakeEmailClient{err: errors.New("smtp down")})

	err := uc.Execute(context.Background(), appstorefront.RequestMagicLinkInput{SalesChannelID: "channel-1", Email: "shopper@example.com"})

	require.Error(t, err)
}

func TestRequestMagicLink_emailIsInTheStoreLanguage(t *testing.T) {
	channel := headlessChannel("channel-1", "https://loja.example/entrar")
	channel.StorefrontLocale = channelsDomain.StorefrontLocalePortuguese
	emailClient := &fakeEmailClient{}
	uc := newMagicLinkUC(t, 2, map[string]channelsDomain.SalesChannel{"channel-1": channel},
		&fakeCustomerRepo{byEmail: map[string]customersDomain.Customer{"ana@example.com": {ID: "c1"}}}, &fakeCustomerCreator{}, &fakeMagicLinkTokenRepo{}, emailClient)

	require.NoError(t, uc.Execute(context.Background(), appstorefront.RequestMagicLinkInput{SalesChannelID: "channel-1", Email: "ana@example.com"}))

	require.Equal(t, "Seu link de acesso", emailClient.sent[0].Subject)
	require.Contains(t, emailClient.sent[0].HTMLBody, ">Entrar</a>")
}
