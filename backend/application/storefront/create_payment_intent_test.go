package storefront_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
)

// fakeCheckoutEstimator stands in for *CheckoutCartUseCase.EstimateTotal --
// CreatePaymentIntentUseCase must never duplicate subtotal/tax/shipping
// math itself, only ask for the same number checkout will (re)compute.
type fakeCheckoutEstimator struct {
	out *appstorefront.CheckoutEstimate
	err error

	gotToken string
}

func (f *fakeCheckoutEstimator) EstimateTotal(_ context.Context, token string, _ appstorefront.CheckoutAddressInput, _ string) (*appstorefront.CheckoutEstimate, error) {
	f.gotToken = token
	if f.err != nil {
		return nil, f.err
	}
	return f.out, nil
}

func TestCreatePaymentIntentUseCase_Execute(t *testing.T) {
	shipping := appstorefront.CheckoutAddressInput{FirstName: "A", LastName: "B", Address1: "Street 1", City: "Amsterdam", Postcode: "1000AA", Country: "NL"}

	t.Run("channel has no publishable key configured", func(t *testing.T) {
		conn := stripeConnection("conn-1")
		conn.ConsumerKey = ""
		channelRepo := &fakeCardChannelRepo{byID: map[string]channelsDomain.SalesChannel{"chan-1": channelWithPaymentConn("chan-1", "conn-1")}}
		connRepo := &fakeCardConnRepo{byID: map[string]integrationsDomain.Connection{"conn-1": conn}}
		factory := &fakeCardPaymentClientFactory{}
		estimator := &fakeCheckoutEstimator{}
		uc := appstorefront.NewCreatePaymentIntentUseCase(newMockGormDB(t, 1), channelRepo, connRepo, factory, testEncryptionSecret, estimator)

		_, err := uc.Execute(context.Background(), "chan-1", "cart-token", shipping, "")
		require.ErrorIs(t, err, storefrontDomain.ErrCardPaymentNotSupported)
	})

	t.Run("no payment connection wired at all", func(t *testing.T) {
		channelRepo := &fakeCardChannelRepo{byID: map[string]channelsDomain.SalesChannel{"chan-1": internalChannel("chan-1")}}
		connRepo := &fakeCardConnRepo{}
		factory := &fakeCardPaymentClientFactory{}
		estimator := &fakeCheckoutEstimator{}
		uc := appstorefront.NewCreatePaymentIntentUseCase(newMockGormDB(t, 1), channelRepo, connRepo, factory, testEncryptionSecret, estimator)

		_, err := uc.Execute(context.Background(), "chan-1", "cart-token", shipping, "")
		require.ErrorIs(t, err, storefrontDomain.ErrCardPaymentNotSupported)
	})

	t.Run("estimator error propagates", func(t *testing.T) {
		conn := stripeConnection("conn-1", func(c *integrationsDomain.Connection) {
			c.ConsumerSecretEncrypted = encryptedSecret(t, "sk_live_abc")
		})
		channelRepo := &fakeCardChannelRepo{byID: map[string]channelsDomain.SalesChannel{"chan-1": channelWithPaymentConn("chan-1", "conn-1")}}
		connRepo := &fakeCardConnRepo{byID: map[string]integrationsDomain.Connection{"conn-1": conn}}
		factory := &fakeCardPaymentClientFactory{}
		estimator := &fakeCheckoutEstimator{err: storefrontDomain.ErrCartNotFound}
		uc := appstorefront.NewCreatePaymentIntentUseCase(newMockGormDB(t, 1), channelRepo, connRepo, factory, testEncryptionSecret, estimator)

		_, err := uc.Execute(context.Background(), "chan-1", "cart-token", shipping, "")
		require.ErrorIs(t, err, storefrontDomain.ErrCartNotFound)
	})

	t.Run("creates a PaymentIntent for the estimated total and returns the publishable key", func(t *testing.T) {
		conn := stripeConnection("conn-1", func(c *integrationsDomain.Connection) {
			c.ConsumerSecretEncrypted = encryptedSecret(t, "sk_live_abc")
			c.ConsumerKey = "pk_live_xyz"
		})
		channelRepo := &fakeCardChannelRepo{byID: map[string]channelsDomain.SalesChannel{"chan-1": channelWithPaymentConn("chan-1", "conn-1")}}
		connRepo := &fakeCardConnRepo{byID: map[string]integrationsDomain.Connection{"conn-1": conn}}
		client := &fakeCardPaymentClient{createOut: integrationsDomain.CardPaymentIntent{ClientSecret: "pi_secret_1", ID: "pi_1"}}
		factory := &fakeCardPaymentClientFactory{client: client}
		estimator := &fakeCheckoutEstimator{out: &appstorefront.CheckoutEstimate{TotalMinorUnits: 4599, CurrencyCode: "EUR"}}
		uc := appstorefront.NewCreatePaymentIntentUseCase(newMockGormDB(t, 1), channelRepo, connRepo, factory, testEncryptionSecret, estimator)

		result, err := uc.Execute(context.Background(), "chan-1", "cart-token", shipping, "")
		require.NoError(t, err)
		require.Equal(t, "pi_secret_1", result.ClientSecret)
		require.Equal(t, "pk_live_xyz", result.PublishableKey)
		require.Equal(t, int64(4599), result.AmountMinorUnits)
		require.Equal(t, "EUR", result.CurrencyCode)
		require.Equal(t, int64(4599), client.gotAmount)
		require.Equal(t, "EUR", client.gotCurrency)
		require.Equal(t, "sk_live_abc", factory.gotKey)
		require.Equal(t, "cart-token", estimator.gotToken)
	})
}
