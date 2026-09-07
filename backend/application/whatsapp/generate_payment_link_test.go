package whatsapp_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appwhatsapp "github.com/adirkuhn/commerceos/application/whatsapp"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

type fakeSalesChannelRepoForPayment struct {
	channelsDomain.SalesChannelRepository
	channels []channelsDomain.SalesChannel
}

func (f *fakeSalesChannelRepoForPayment) ListByTenant(_ context.Context) ([]channelsDomain.SalesChannel, error) {
	return f.channels, nil
}

func newGeneratePaymentLinkUseCase(t *testing.T, n int, channels []channelsDomain.SalesChannel, connRepo *fakeConnectionRepo, stripeFactory, revolutFactory *fakeEvolutionAPIClientFactoryForPayment) *appwhatsapp.GeneratePaymentLinkUseCase {
	t.Helper()
	if connRepo == nil {
		connRepo = &fakeConnectionRepo{}
	}
	if stripeFactory == nil {
		stripeFactory = &fakeEvolutionAPIClientFactoryForPayment{}
	}
	if revolutFactory == nil {
		revolutFactory = &fakeEvolutionAPIClientFactoryForPayment{}
	}
	return appwhatsapp.NewGeneratePaymentLinkUseCase(
		newMockGormDB(t, n), &fakeSalesChannelRepoForPayment{channels: channels}, connRepo,
		stripeFactory, revolutFactory, testEncryptionSecret, "https://app.example.com",
	)
}

type fakePaymentLinkClient struct {
	link integrationsDomain.PaymentLink
	err  error
}

func (f *fakePaymentLinkClient) CreatePaymentLink(_ context.Context, _ integrationsDomain.PaymentLinkRequest) (integrationsDomain.PaymentLink, error) {
	return f.link, f.err
}

func (f *fakePaymentLinkClient) Refund(_ context.Context, _ string, _ int64, _ string) (string, error) {
	return "", nil
}

type fakeEvolutionAPIClientFactoryForPayment struct {
	client *fakePaymentLinkClient
	err    error
}

func (f *fakeEvolutionAPIClientFactoryForPayment) NewClient(_ string) (integrationsDomain.PaymentLinkClient, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.client == nil {
		return &fakePaymentLinkClient{}, nil
	}
	return f.client, nil
}

func whatsappChannelWithPayment(connID *string) channelsDomain.SalesChannel {
	return channelsDomain.SalesChannel{ID: "channel-1", Type: channelsDomain.ChannelTypeWhatsApp, Active: true, PaymentConnectionID: connID}
}

func TestGeneratePaymentLink_stripeConfigured_usesStripeFactory(t *testing.T) {
	connID := "conn-stripe-1"
	connRepo := &fakeConnectionRepo{conns: []integrationsDomain.Connection{
		{ID: "conn-stripe-1", Provider: integrationsDomain.ProviderStripe, ConsumerSecretEncrypted: encryptedSecret(t, "sk_test")},
	}}
	stripeFactory := &fakeEvolutionAPIClientFactoryForPayment{client: &fakePaymentLinkClient{link: integrationsDomain.PaymentLink{URL: "https://checkout.stripe.com/pay/1", ExternalID: "cs_1"}}}
	uc := newGeneratePaymentLinkUseCase(t, 1, []channelsDomain.SalesChannel{whatsappChannelWithPayment(&connID)}, connRepo, stripeFactory, nil)

	link, ok, err := uc.Execute(context.Background(), 1999, "EUR", "Order #1", "1")

	require.NoError(t, err)
	assert.True(t, ok)
	require.NotNil(t, link)
	assert.Equal(t, "https://checkout.stripe.com/pay/1", link.URL)
}

func TestGeneratePaymentLink_revolutConfigured_usesRevolutFactory(t *testing.T) {
	connID := "conn-revolut-1"
	connRepo := &fakeConnectionRepo{conns: []integrationsDomain.Connection{
		{ID: "conn-revolut-1", Provider: integrationsDomain.ProviderRevolut, ConsumerSecretEncrypted: encryptedSecret(t, "revolut-key")},
	}}
	revolutFactory := &fakeEvolutionAPIClientFactoryForPayment{client: &fakePaymentLinkClient{link: integrationsDomain.PaymentLink{URL: "https://checkout.revolut.com/pay/1"}}}
	uc := newGeneratePaymentLinkUseCase(t, 1, []channelsDomain.SalesChannel{whatsappChannelWithPayment(&connID)}, connRepo, nil, revolutFactory)

	link, ok, err := uc.Execute(context.Background(), 1999, "EUR", "Order #1", "1")

	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "https://checkout.revolut.com/pay/1", link.URL)
}

func TestGeneratePaymentLink_noWhatsAppChannel_returnsNotOKNotError(t *testing.T) {
	uc := newGeneratePaymentLinkUseCase(t, 1, nil, nil, nil, nil)

	link, ok, err := uc.Execute(context.Background(), 1999, "EUR", "Order #1", "1")

	require.NoError(t, err)
	assert.False(t, ok)
	assert.Nil(t, link)
}

func TestGeneratePaymentLink_noPaymentMethodConfigured_returnsNotOKNotError(t *testing.T) {
	uc := newGeneratePaymentLinkUseCase(t, 1, []channelsDomain.SalesChannel{whatsappChannelWithPayment(nil)}, nil, nil, nil)

	link, ok, err := uc.Execute(context.Background(), 1999, "EUR", "Order #1", "1")

	require.NoError(t, err)
	assert.False(t, ok)
	assert.Nil(t, link)
}

func TestGeneratePaymentLink_providerCreatePaymentLinkFails_propagatesError(t *testing.T) {
	connID := "conn-stripe-1"
	connRepo := &fakeConnectionRepo{conns: []integrationsDomain.Connection{
		{ID: "conn-stripe-1", Provider: integrationsDomain.ProviderStripe, ConsumerSecretEncrypted: encryptedSecret(t, "sk_test")},
	}}
	stripeFactory := &fakeEvolutionAPIClientFactoryForPayment{client: &fakePaymentLinkClient{err: errors.New("stripe down")}}
	uc := newGeneratePaymentLinkUseCase(t, 1, []channelsDomain.SalesChannel{whatsappChannelWithPayment(&connID)}, connRepo, stripeFactory, nil)

	_, ok, err := uc.Execute(context.Background(), 1999, "EUR", "Order #1", "1")

	require.Error(t, err)
	assert.False(t, ok)
}
