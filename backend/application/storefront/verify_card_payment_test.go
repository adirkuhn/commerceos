package storefront_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/internal/crypto"
)

// fakeCardChannelRepo/fakeCardConnRepo back the two card-payment use cases'
// "resolve this channel's Stripe connection" step -- FindByID follows the
// real repos' PK-lookup convention (typed not-found error, never nil,nil).
type fakeCardChannelRepo struct {
	channelsDomain.SalesChannelRepository
	byID map[string]channelsDomain.SalesChannel
}

func (f *fakeCardChannelRepo) FindByID(_ context.Context, id string) (*channelsDomain.SalesChannel, error) {
	if c, ok := f.byID[id]; ok {
		return &c, nil
	}
	return nil, channelsDomain.ErrSalesChannelNotFound
}

type fakeCardConnRepo struct {
	integrationsDomain.ConnectionRepository
	byID map[string]integrationsDomain.Connection
}

func (f *fakeCardConnRepo) FindByID(_ context.Context, id string) (*integrationsDomain.Connection, error) {
	if c, ok := f.byID[id]; ok {
		return &c, nil
	}
	return nil, integrationsDomain.ErrConnectionNotFound
}

type fakeCardPaymentClient struct {
	createOut   integrationsDomain.CardPaymentIntent
	createErr   error
	retrieve    integrationsDomain.CardPaymentStatus
	retrieveErr error
	gotAmount   int64
	gotCurrency string
	gotID       string
}

func (f *fakeCardPaymentClient) CreatePaymentIntent(_ context.Context, amountMinorUnits int64, currencyCode string) (integrationsDomain.CardPaymentIntent, error) {
	f.gotAmount, f.gotCurrency = amountMinorUnits, currencyCode
	if f.createErr != nil {
		return integrationsDomain.CardPaymentIntent{}, f.createErr
	}
	return f.createOut, nil
}

func (f *fakeCardPaymentClient) RetrievePaymentIntent(_ context.Context, id string) (integrationsDomain.CardPaymentStatus, error) {
	f.gotID = id
	if f.retrieveErr != nil {
		return integrationsDomain.CardPaymentStatus{}, f.retrieveErr
	}
	return f.retrieve, nil
}

type fakeCardPaymentClientFactory struct {
	client *fakeCardPaymentClient
	err    error
	gotKey string
}

func (f *fakeCardPaymentClientFactory) NewCardPaymentClient(apiKey string) (integrationsDomain.CardPaymentClient, error) {
	f.gotKey = apiKey
	if f.err != nil {
		return nil, f.err
	}
	return f.client, nil
}

const testEncryptionSecret = "test-encryption-secret-32-bytes"

func encryptedSecret(t *testing.T, plaintext string) string {
	t.Helper()
	enc, err := crypto.Encrypt(testEncryptionSecret, plaintext)
	require.NoError(t, err)
	return enc
}

func stripeConnection(id string, extra ...func(*integrationsDomain.Connection)) integrationsDomain.Connection {
	c := integrationsDomain.Connection{ID: id, Provider: integrationsDomain.ProviderStripe, ConsumerKey: "pk_test_123"}
	for _, f := range extra {
		f(&c)
	}
	return c
}

func channelWithPaymentConn(id, connID string) channelsDomain.SalesChannel {
	c := internalChannel(id)
	c.PaymentConnectionID = &connID
	return c
}

func TestVerifyCardPaymentUseCase_Execute(t *testing.T) {
	t.Run("channel has no payment connection wired", func(t *testing.T) {
		channelRepo := &fakeCardChannelRepo{byID: map[string]channelsDomain.SalesChannel{"chan-1": internalChannel("chan-1")}}
		connRepo := &fakeCardConnRepo{}
		factory := &fakeCardPaymentClientFactory{}
		uc := appstorefront.NewVerifyCardPaymentUseCase(newMockGormDB(t, 1), channelRepo, connRepo, factory, testEncryptionSecret)

		_, err := uc.Execute(context.Background(), "chan-1", "pi_123")
		require.ErrorIs(t, err, storefrontDomain.ErrCardPaymentNotSupported)
	})

	t.Run("connection is not Stripe", func(t *testing.T) {
		conn := integrationsDomain.Connection{ID: "conn-1", Provider: integrationsDomain.ProviderRevolut}
		channelRepo := &fakeCardChannelRepo{byID: map[string]channelsDomain.SalesChannel{"chan-1": channelWithPaymentConn("chan-1", "conn-1")}}
		connRepo := &fakeCardConnRepo{byID: map[string]integrationsDomain.Connection{"conn-1": conn}}
		factory := &fakeCardPaymentClientFactory{}
		uc := appstorefront.NewVerifyCardPaymentUseCase(newMockGormDB(t, 1), channelRepo, connRepo, factory, testEncryptionSecret)

		_, err := uc.Execute(context.Background(), "chan-1", "pi_123")
		require.ErrorIs(t, err, storefrontDomain.ErrCardPaymentNotSupported)
	})

	t.Run("retrieves and returns the PaymentIntent status from Stripe", func(t *testing.T) {
		conn := stripeConnection("conn-1", func(c *integrationsDomain.Connection) {
			c.ConsumerSecretEncrypted = encryptedSecret(t, "sk_live_abc")
		})
		channelRepo := &fakeCardChannelRepo{byID: map[string]channelsDomain.SalesChannel{"chan-1": channelWithPaymentConn("chan-1", "conn-1")}}
		connRepo := &fakeCardConnRepo{byID: map[string]integrationsDomain.Connection{"conn-1": conn}}
		client := &fakeCardPaymentClient{retrieve: integrationsDomain.CardPaymentStatus{ID: "pi_123", Status: "succeeded", AmountMinorUnits: 1000, CurrencyCode: "EUR"}}
		factory := &fakeCardPaymentClientFactory{client: client}
		uc := appstorefront.NewVerifyCardPaymentUseCase(newMockGormDB(t, 1), channelRepo, connRepo, factory, testEncryptionSecret)

		status, err := uc.Execute(context.Background(), "chan-1", "pi_123")
		require.NoError(t, err)
		require.Equal(t, "succeeded", status.Status)
		require.Equal(t, int64(1000), status.AmountMinorUnits)
		require.Equal(t, "EUR", status.CurrencyCode)
		require.Equal(t, "sk_live_abc", factory.gotKey)
		require.Equal(t, "pi_123", client.gotID)
	})
}
