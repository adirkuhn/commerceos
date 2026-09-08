package storefront

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// VerifyCardPaymentUseCase re-fetches a PaymentIntent from Stripe itself --
// CheckoutCartUseCase calls this rather than trusting a client-confirmed
// card payment, same "never trust client-supplied money facts" rule
// resolveShipping/resolveCouponDiscount already follow. Only Stripe
// connections support this (Revolut has no equivalent client-side-confirm
// SDK) -- any other provider, or no publishable key configured (see
// integrations.Connection.ConsumerKey's doc comment), returns
// ErrCardPaymentNotSupported.
type VerifyCardPaymentUseCase struct {
	db               *gorm.DB
	channelRepo      channelsDomain.SalesChannelRepository
	connRepo         integrationsDomain.ConnectionRepository
	stripeFactory    integrationsDomain.CardPaymentClientFactory
	encryptionSecret string
}

func NewVerifyCardPaymentUseCase(
	db *gorm.DB,
	channelRepo channelsDomain.SalesChannelRepository,
	connRepo integrationsDomain.ConnectionRepository,
	stripeFactory integrationsDomain.CardPaymentClientFactory,
	encryptionSecret string,
) *VerifyCardPaymentUseCase {
	return &VerifyCardPaymentUseCase{db: db, channelRepo: channelRepo, connRepo: connRepo, stripeFactory: stripeFactory, encryptionSecret: encryptionSecret}
}

func (uc *VerifyCardPaymentUseCase) Execute(ctx context.Context, salesChannelID, paymentIntentID string) (*integrationsDomain.CardPaymentStatus, error) {
	conn, err := uc.resolveStripeConnection(ctx, salesChannelID)
	if err != nil {
		return nil, fmt.Errorf("verifyCardPayment: %w", err)
	}

	apiKey, err := crypto.Decrypt(uc.encryptionSecret, conn.ConsumerSecretEncrypted)
	if err != nil {
		return nil, fmt.Errorf("verifyCardPayment: decrypt connection secret: %w", err)
	}
	client, err := uc.stripeFactory.NewCardPaymentClient(apiKey)
	if err != nil {
		return nil, fmt.Errorf("verifyCardPayment: %w", err)
	}
	status, err := client.RetrievePaymentIntent(ctx, paymentIntentID)
	if err != nil {
		return nil, fmt.Errorf("verifyCardPayment: %w", err)
	}
	return &status, nil
}

// resolveStripeConnection is shared by VerifyCardPaymentUseCase and
// CreatePaymentIntentUseCase -- both need "this channel's payment
// connection, and it must be Stripe" before doing anything else.
func resolveStripeConnection(ctx context.Context, db *gorm.DB, channelRepo channelsDomain.SalesChannelRepository, connRepo integrationsDomain.ConnectionRepository, salesChannelID string) (*integrationsDomain.Connection, error) {
	var conn *integrationsDomain.Connection
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		channel, err := channelRepo.FindByID(ctx, salesChannelID)
		if err != nil {
			return err
		}
		if channel.PaymentConnectionID == nil {
			return storefrontDomain.ErrCardPaymentNotSupported
		}
		c, err := connRepo.FindByID(ctx, *channel.PaymentConnectionID)
		if err != nil {
			return err
		}
		conn = c
		return nil
	})
	if err != nil {
		return nil, err
	}
	if conn.Provider != integrationsDomain.ProviderStripe {
		return nil, storefrontDomain.ErrCardPaymentNotSupported
	}
	return conn, nil
}

func (uc *VerifyCardPaymentUseCase) resolveStripeConnection(ctx context.Context, salesChannelID string) (*integrationsDomain.Connection, error) {
	return resolveStripeConnection(ctx, uc.db, uc.channelRepo, uc.connRepo, salesChannelID)
}
