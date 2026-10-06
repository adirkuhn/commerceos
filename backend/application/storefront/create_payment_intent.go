//events:none writes nothing locally -- only asks the card provider for a payment intent; the order raises storefront.order.placed
package storefront

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/internal/crypto"
)

// PaymentIntentResult is what the storefront client needs to mount an
// embedded Stripe Elements card field and confirm the charge itself --
// PublishableKey is not sensitive (Stripe's own docs: safe client-side),
// unlike the connection's secret key, which never leaves the server.
type PaymentIntentResult struct {
	ClientSecret     string
	PublishableKey   string
	AmountMinorUnits int64
	CurrencyCode     string
}

// checkoutEstimator is *CheckoutCartUseCase.EstimateTotal narrowed to point
// of use (ISP) -- CreatePaymentIntentUseCase never duplicates
// subtotal/tax/shipping/discount math, it just asks for the same number
// Execute will (re)compute at actual checkout time.
type checkoutEstimator interface {
	EstimateTotal(ctx context.Context, token string, shipping CheckoutAddressInput, shippingMethodTitle string) (*CheckoutEstimate, error)
}

// CreatePaymentIntentUseCase starts a Stripe PaymentIntent for exactly what
// token's cart will cost at checkout, given shipping's address and (if
// already chosen) a shipping method -- the storefront calls this once the
// shopper has entered a shipping address (same trigger point
// getShippingOptions already uses), then mounts Stripe Elements against
// the returned ClientSecret.
type CreatePaymentIntentUseCase struct {
	db               *gorm.DB
	channelRepo      channelsDomain.SalesChannelRepository
	connRepo         integrationsDomain.ConnectionRepository
	stripeFactory    integrationsDomain.CardPaymentClientFactory
	encryptionSecret string
	estimator        checkoutEstimator
}

func NewCreatePaymentIntentUseCase(
	db *gorm.DB,
	channelRepo channelsDomain.SalesChannelRepository,
	connRepo integrationsDomain.ConnectionRepository,
	stripeFactory integrationsDomain.CardPaymentClientFactory,
	encryptionSecret string,
	estimator checkoutEstimator,
) *CreatePaymentIntentUseCase {
	return &CreatePaymentIntentUseCase{
		db: db, channelRepo: channelRepo, connRepo: connRepo,
		stripeFactory: stripeFactory, encryptionSecret: encryptionSecret, estimator: estimator,
	}
}

func (uc *CreatePaymentIntentUseCase) Execute(ctx context.Context, salesChannelID, cartToken string, shipping CheckoutAddressInput, shippingMethodTitle string) (*PaymentIntentResult, error) {
	conn, err := resolveStripeConnection(ctx, uc.db, uc.channelRepo, uc.connRepo, salesChannelID)
	if err != nil {
		return nil, fmt.Errorf("createPaymentIntent: %w", err)
	}
	if conn.ConsumerKey == "" {
		return nil, fmt.Errorf("createPaymentIntent: %w", storefrontDomain.ErrCardPaymentNotSupported)
	}

	estimate, err := uc.estimator.EstimateTotal(ctx, cartToken, shipping, shippingMethodTitle)
	if err != nil {
		return nil, fmt.Errorf("createPaymentIntent: %w", err)
	}

	apiKey, err := crypto.Decrypt(uc.encryptionSecret, conn.ConsumerSecretEncrypted)
	if err != nil {
		return nil, fmt.Errorf("createPaymentIntent: decrypt connection secret: %w", err)
	}
	client, err := uc.stripeFactory.NewCardPaymentClient(apiKey)
	if err != nil {
		return nil, fmt.Errorf("createPaymentIntent: %w", err)
	}
	intent, err := client.CreatePaymentIntent(ctx, estimate.TotalMinorUnits, estimate.CurrencyCode)
	if err != nil {
		return nil, fmt.Errorf("createPaymentIntent: %w", err)
	}

	return &PaymentIntentResult{
		ClientSecret: intent.ClientSecret, PublishableKey: conn.ConsumerKey,
		AmountMinorUnits: estimate.TotalMinorUnits, CurrencyCode: estimate.CurrencyCode,
	}, nil
}
