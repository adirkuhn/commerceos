package whatsapp

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// GeneratePaymentLinkUseCase looks up the tenant's active WhatsApp
// SalesChannel and, if it has a payment method enabled (PaymentConnectionID
// set -- see SetSalesChannelPaymentMethodUseCase), generates a payment link
// through whichever provider that connection is for (Stripe or Revolut --
// the only two integrationsDomain.PaymentLinkClientFactory implementations
// today). ok=false (no error) covers every "nothing to generate" case: no
// WhatsApp channel, or one with no payment method enabled -- both are
// normal, expected states for a tenant that hasn't set payment up yet, not
// failures.
type GeneratePaymentLinkUseCase struct {
	db               *gorm.DB
	salesChannelRepo channelsDomain.SalesChannelRepository
	connRepo         integrationsDomain.ConnectionRepository
	stripeFactory    integrationsDomain.PaymentLinkClientFactory
	revolutFactory   integrationsDomain.PaymentLinkClientFactory
	encryptionSecret string
	publicBaseURL    string
}

func NewGeneratePaymentLinkUseCase(
	db *gorm.DB,
	salesChannelRepo channelsDomain.SalesChannelRepository,
	connRepo integrationsDomain.ConnectionRepository,
	stripeFactory integrationsDomain.PaymentLinkClientFactory,
	revolutFactory integrationsDomain.PaymentLinkClientFactory,
	encryptionSecret, publicBaseURL string,
) *GeneratePaymentLinkUseCase {
	return &GeneratePaymentLinkUseCase{
		db: db, salesChannelRepo: salesChannelRepo, connRepo: connRepo,
		stripeFactory: stripeFactory, revolutFactory: revolutFactory,
		encryptionSecret: encryptionSecret, publicBaseURL: publicBaseURL,
	}
}

func (uc *GeneratePaymentLinkUseCase) Execute(ctx context.Context, amountMinorUnits int64, currencyCode, description, reference string) (*integrationsDomain.PaymentLink, bool, error) {
	var conn *integrationsDomain.Connection
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		channels, err := uc.salesChannelRepo.ListByTenant(ctx)
		if err != nil {
			return err
		}
		var channel *channelsDomain.SalesChannel
		for i := range channels {
			if channels[i].Type == channelsDomain.ChannelTypeWhatsApp && channels[i].Active {
				channel = &channels[i]
				break
			}
		}
		if channel == nil || channel.PaymentConnectionID == nil {
			return nil
		}
		c, err := uc.connRepo.FindByID(ctx, *channel.PaymentConnectionID)
		if err != nil {
			return err
		}
		conn = c
		return nil
	})
	if err != nil {
		return nil, false, fmt.Errorf("generatePaymentLink: %w", err)
	}
	if conn == nil {
		return nil, false, nil
	}

	var factory integrationsDomain.PaymentLinkClientFactory
	switch conn.Provider {
	case integrationsDomain.ProviderStripe:
		factory = uc.stripeFactory
	case integrationsDomain.ProviderRevolut:
		factory = uc.revolutFactory
	default:
		// Set-time validation (SetSalesChannelPaymentMethodUseCase) already
		// rejects any non-payment provider -- reaching here would mean that
		// guard was bypassed some other way. Fail safe: no link, not a panic.
		return nil, false, nil
	}

	client, err := buildPaymentClient(factory, uc.encryptionSecret, *conn)
	if err != nil {
		return nil, false, fmt.Errorf("generatePaymentLink: %w", err)
	}

	link, err := client.CreatePaymentLink(ctx, integrationsDomain.PaymentLinkRequest{
		AmountMinorUnits: amountMinorUnits, CurrencyCode: currencyCode, Description: description, Reference: reference,
		SuccessURL: uc.publicBaseURL + "/payment-success", CancelURL: uc.publicBaseURL + "/payment-cancelled",
	})
	if err != nil {
		return nil, false, fmt.Errorf("generatePaymentLink: create link: %w", err)
	}
	return &link, true, nil
}
