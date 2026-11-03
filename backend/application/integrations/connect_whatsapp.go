package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/domain/events"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/domain/outbox"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/idgen"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type ConnectWhatsAppInput struct {
	BaseURL      string
	InstanceName string
	APIKey       string
}

// ConnectWhatsAppResult carries the plaintext webhook URL back to the
// caller exactly once, at connect time -- same convention as
// SetWebhookSecretResult's PlaintextSecret: only WebhookSecretEncrypted
// persists, so the merchant must paste this URL into Evolution API's own
// webhook settings right away.
type ConnectWhatsAppResult struct {
	Connection integrationsDomain.Connection
	WebhookURL string
}

// ConnectWhatsAppUseCase is WhatsApp's connect flow -- like Bol, no OAuth
// redirect (Evolution API is a self-hosted gateway, not an OAuth provider):
// paste the instance's base URL/name/API key, save encrypted. Unlike
// Woo/MyParcel's connect+SetWebhookSecretUseCase+CreateProviderWebhookUseCase
// three-step flow, this generates the webhook secret and returns the
// ready-to-paste delivery URL in the same call -- Evolution API has no
// confirmed API to register a webhook remotely (see
// infrastructure/evolutionapi's package doc comment), so there is no
// provider-side "create" step to defer to; the merchant pastes the URL into
// Evolution API's own dashboard manually, same as MyParcel's callback-URL
// setup.
type ConnectWhatsAppUseCase struct {
	db               *gorm.DB
	connRepo         integrationsDomain.ConnectionRepository
	outboxRepo       outbox.Repository
	encryptionSecret string
	publicAPIBaseURL string
}

func NewConnectWhatsAppUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	outboxRepo outbox.Repository,
	encryptionSecret, publicAPIBaseURL string,
) *ConnectWhatsAppUseCase {
	return &ConnectWhatsAppUseCase{
		db: db, connRepo: connRepo, outboxRepo: outboxRepo,
		encryptionSecret: encryptionSecret, publicAPIBaseURL: publicAPIBaseURL,
	}
}

func (uc *ConnectWhatsAppUseCase) Execute(ctx context.Context, in ConnectWhatsAppInput) (*ConnectWhatsAppResult, error) {
	tenantID := tenant.IDFromContext(ctx)
	if tenantID == "" {
		return nil, fmt.Errorf("connectWhatsapp: %w", integrationsDomain.ErrNoTenantContext)
	}

	storeURL := integrationsDomain.StoreURL(in.BaseURL)
	if err := storeURL.Validate(); err != nil {
		return nil, fmt.Errorf("connectWhatsapp: %w", err)
	}

	encryptedAPIKey, err := crypto.Encrypt(uc.encryptionSecret, in.APIKey)
	if err != nil {
		return nil, fmt.Errorf("connectWhatsapp: %w", err)
	}
	webhookSecret, err := generateSecret()
	if err != nil {
		return nil, fmt.Errorf("connectWhatsapp: generate webhook secret: %w", err)
	}
	encryptedWebhookSecret, err := crypto.Encrypt(uc.encryptionSecret, webhookSecret)
	if err != nil {
		return nil, fmt.Errorf("connectWhatsapp: %w", err)
	}

	var conn integrationsDomain.Connection
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		conn = integrationsDomain.Connection{
			TenantID: tenantID, Provider: integrationsDomain.ProviderWhatsApp,
			StoreURL: storeURL, ConsumerKey: in.InstanceName, ConsumerSecretEncrypted: encryptedAPIKey,
			WebhookSecretEncrypted: encryptedWebhookSecret,
			Status:                 integrationsDomain.StatusConnected,
			CreatedAt:              now, UpdatedAt: now,
		}
		if _, err := integrationsDomain.SaveConnection(ctx, uc.connRepo, &conn); err != nil {
			return err
		}

		event := events.IntegrationConnected{
			Base:         events.Base{EventID: idgen.NewV4(), OccurredAt: now},
			TenantID:     tenantID,
			Provider:     string(integrationsDomain.ProviderWhatsApp),
			ConnectionID: conn.ID,
		}
		payload, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("marshal event: %w", err)
		}
		return uc.outboxRepo.Enqueue(ctx, event.EventName(), event.EventID, payload, event.OccurredAt)
	})
	if err != nil {
		return nil, fmt.Errorf("connectWhatsapp: %w", err)
	}

	webhookURL := providerWebhookIngestURLWithToken(uc.publicAPIBaseURL, tenantID, integrationsDomain.ProviderWhatsApp, webhookSecret)
	return &ConnectWhatsAppResult{Connection: conn, WebhookURL: webhookURL}, nil
}
