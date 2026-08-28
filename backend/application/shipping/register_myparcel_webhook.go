package shipping

import (
	"context"
	"fmt"
	"net/url"

	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// RegisterMyParcelWebhookUseCase registers this tenant's shipment-status
// callback URL with MyParcel. MyParcel signs nothing (no HMAC, see
// infrastructure/webhookproviders.MyParcelVerifier's doc comment) -- the
// tenant's own Connection.WebhookSecretEncrypted (set via
// application/integrations.SetWebhookSecretUseCase, same field the
// HMAC-based providers use, just compared differently) is embedded as a
// path segment in the callback URL instead (api/webhooks/openapi.yaml's
// receiveWebhookWithToken route), not a `?token=` query string -- less
// likely to leak into access logs/proxies/browser history. Requires that
// secret to already be set -- callers must run SetWebhookSecretUseCase
// first.
type RegisterMyParcelWebhookUseCase struct {
	db               *gorm.DB
	connRepo         integrationsDomain.ConnectionRepository
	myParcelFactory  integrationsDomain.MyParcelClientFactory
	encSecret        string
	publicAPIBaseURL string
}

func NewRegisterMyParcelWebhookUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	myParcelFactory integrationsDomain.MyParcelClientFactory,
	encryptionSecret string,
	publicAPIBaseURL string,
) *RegisterMyParcelWebhookUseCase {
	return &RegisterMyParcelWebhookUseCase{
		db: db, connRepo: connRepo, myParcelFactory: myParcelFactory,
		encSecret: encryptionSecret, publicAPIBaseURL: publicAPIBaseURL,
	}
}

func (uc *RegisterMyParcelWebhookUseCase) Execute(ctx context.Context, connectionID string) error {
	var conn integrationsDomain.Connection
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		c, err := uc.connRepo.FindByID(ctx, connectionID)
		if err != nil {
			return err
		}
		conn = *c
		return nil
	})
	if err != nil {
		return fmt.Errorf("registerMyParcelWebhook: %w", err)
	}

	if conn.WebhookSecretEncrypted == "" {
		return fmt.Errorf("registerMyParcelWebhook: %w", shippingDomain.ErrWebhookSecretNotSet)
	}
	webhookToken, err := crypto.Decrypt(uc.encSecret, conn.WebhookSecretEncrypted)
	if err != nil {
		return fmt.Errorf("registerMyParcelWebhook: decrypt webhook secret: %w", err)
	}
	apiKey, err := crypto.Decrypt(uc.encSecret, conn.ConsumerSecretEncrypted)
	if err != nil {
		return fmt.Errorf("registerMyParcelWebhook: decrypt api key: %w", err)
	}

	client, err := uc.myParcelFactory.NewClient(apiKey)
	if err != nil {
		return fmt.Errorf("registerMyParcelWebhook: build client: %w", err)
	}

	callbackURL := uc.publicAPIBaseURL + "/api/v1/webhooks/v1/" + url.PathEscape(conn.TenantID) + "/myparcel/" + url.PathEscape(webhookToken)
	if err := client.RegisterWebhook(ctx, callbackURL); err != nil {
		return fmt.Errorf("registerMyParcelWebhook: %w", err)
	}
	return nil
}
