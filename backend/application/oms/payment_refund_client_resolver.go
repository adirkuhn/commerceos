package oms

import (
	"context"
	"fmt"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// PaymentRefundClientResolver looks up the tenant's own Stripe/Revolut
// Connection for a given provider name and builds a live client scoped to
// its decrypted API key -- same Connection+factory+decrypt pattern
// application/whatsapp.GeneratePaymentLinkUseCase's buildPaymentClient
// already uses, reimplemented here (not imported) since application/oms
// importing application/whatsapp would be a cross-application-package
// coupling this codebase avoids (see application/payments.
// WebhookDispatchHandler's identical reasoning for its own small
// package-local SalesOrderUpdater interface). Picking has no SalesChannel
// to resolve a connection through the way WhatsApp checkout does -- this
// resolves directly by provider name instead, which is all
// CompletePickingUseCase's refund replay has (SalesOrder.PaymentProvider).
type PaymentRefundClientResolver struct {
	db               *gorm.DB
	connRepo         integrationsDomain.ConnectionRepository
	stripeFactory    integrationsDomain.PaymentLinkClientFactory
	revolutFactory   integrationsDomain.PaymentLinkClientFactory
	encryptionSecret string
}

func NewPaymentRefundClientResolver(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	stripeFactory integrationsDomain.PaymentLinkClientFactory,
	revolutFactory integrationsDomain.PaymentLinkClientFactory,
	encryptionSecret string,
) *PaymentRefundClientResolver {
	return &PaymentRefundClientResolver{
		db: db, connRepo: connRepo, stripeFactory: stripeFactory, revolutFactory: revolutFactory, encryptionSecret: encryptionSecret,
	}
}

// Resolve returns ok=false (no error) when the tenant has no connection for
// provider, or provider isn't a recognized payment provider -- both normal,
// expected states (see CompletePickingUseCase's refund replay, which treats
// either the same as an API-call failure: mark the line item refunded, flag
// SalesOrder.NeedsManualRefundReview, never block completion).
func (r *PaymentRefundClientResolver) Resolve(ctx context.Context, provider string) (integrationsDomain.PaymentLinkClient, bool, error) {
	var conns []integrationsDomain.Connection
	err := tenant.WithTenant(ctx, r.db, func(ctx context.Context) error {
		var err error
		conns, err = r.connRepo.ListByTenant(ctx)
		return err
	})
	if err != nil {
		return nil, false, fmt.Errorf("paymentRefundClientResolver: %w", err)
	}

	var conn *integrationsDomain.Connection
	for i := range conns {
		if string(conns[i].Provider) == provider {
			conn = &conns[i]
			break
		}
	}
	if conn == nil {
		return nil, false, nil
	}

	var factory integrationsDomain.PaymentLinkClientFactory
	switch conn.Provider {
	case integrationsDomain.ProviderStripe:
		factory = r.stripeFactory
	case integrationsDomain.ProviderRevolut:
		factory = r.revolutFactory
	default:
		return nil, false, nil
	}

	apiKey, err := crypto.Decrypt(r.encryptionSecret, conn.ConsumerSecretEncrypted)
	if err != nil {
		return nil, false, fmt.Errorf("paymentRefundClientResolver: decrypt connection secret: %w", err)
	}
	client, err := factory.NewClient(apiKey)
	if err != nil {
		return nil, false, fmt.Errorf("paymentRefundClientResolver: build client: %w", err)
	}
	return client, true, nil
}
