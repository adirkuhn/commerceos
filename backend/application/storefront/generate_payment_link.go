package storefront

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/events"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// buildPaymentClient mirrors application/whatsapp's unexported helper of
// the same name -- decrypts conn's secret and builds a live client through
// factory. Duplicated rather than imported for the same
// no-cross-application-package-coupling reason as this package's own doc
// comment explains.
func buildPaymentClient(factory integrationsDomain.PaymentLinkClientFactory, encryptionSecret string, conn integrationsDomain.Connection) (integrationsDomain.PaymentLinkClient, error) {
	apiKey, err := crypto.Decrypt(encryptionSecret, conn.ConsumerSecretEncrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt connection secret: %w", err)
	}
	return factory.NewClient(apiKey)
}

// GeneratePaymentLinkUseCase generates a checkout link for one specific
// sales channel -- unlike application/whatsapp's use case of the same name
// (which scans for "the" WhatsApp channel), this one is always called with
// the exact channel a cart belongs to, since a tenant can have more than
// one internal channel each with its own payment method (see
// SalesChannel.PaymentConnectionID's doc comment, generic on purpose).
// ok=false (no error) covers every "nothing to generate" case: the channel
// has no payment method enabled -- a normal state for a tenant that hasn't
// set one up yet, not a failure.
type GeneratePaymentLinkUseCase struct {
	db               *gorm.DB
	channelRepo      channelsDomain.SalesChannelRepository
	connRepo         integrationsDomain.ConnectionRepository
	stripeFactory    integrationsDomain.PaymentLinkClientFactory
	revolutFactory   integrationsDomain.PaymentLinkClientFactory
	encryptionSecret string
	publicBaseURL    string
	outbox           eventrecord.Enqueuer
}

func NewGeneratePaymentLinkUseCase(
	db *gorm.DB,
	channelRepo channelsDomain.SalesChannelRepository,
	connRepo integrationsDomain.ConnectionRepository,
	stripeFactory integrationsDomain.PaymentLinkClientFactory,
	revolutFactory integrationsDomain.PaymentLinkClientFactory,
	encryptionSecret, publicBaseURL string,
	outbox eventrecord.Enqueuer,
) *GeneratePaymentLinkUseCase {
	return &GeneratePaymentLinkUseCase{
		db: db, channelRepo: channelRepo, connRepo: connRepo,
		stripeFactory: stripeFactory, revolutFactory: revolutFactory,
		encryptionSecret: encryptionSecret, publicBaseURL: publicBaseURL, outbox: outbox,
	}
}

func (uc *GeneratePaymentLinkUseCase) Execute(ctx context.Context, salesChannelID string, amountMinorUnits int64, currencyCode, description, reference string) (*integrationsDomain.PaymentLink, bool, error) {
	var conn *integrationsDomain.Connection
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		channel, err := uc.channelRepo.FindByID(ctx, salesChannelID)
		if err != nil {
			return err
		}
		if channel.PaymentConnectionID == nil {
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
	// The link now exists at the provider -- record it in its own
	// transaction (there's no CommerceOS write to share one with).
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return eventrecord.Record(ctx, uc.outbox, events.PaymentLinkCreated, link.ExternalID, eventrecord.Data{
			"sales_channel_id": salesChannelID, "provider": string(conn.Provider), "reference": reference,
			"amount_minor_units": amountMinorUnits, "currency_code": currencyCode,
		})
	})
	if err != nil {
		return nil, false, fmt.Errorf("generatePaymentLink: %w", err)
	}
	return &link, true, nil
}
