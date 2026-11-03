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

type ConnectStripeInput struct {
	// SecretKey is Stripe's own "sk_..." secret API key -- no OAuth (Stripe
	// Connect exists but is out of scope; this is a direct-account
	// integration, same as Bol/WhatsApp's own connect flows).
	SecretKey string
	// PublishableKey ("pk_...") is not sensitive -- Stripe's own docs say
	// it's safe to expose client-side -- so it's stored plain in
	// Connection.ConsumerKey (unused for Stripe until now, see that
	// field's doc comment) rather than encrypted alongside SecretKey.
	// Optional: empty until the merchant sets it, which just means the
	// storefront's embedded Stripe Elements checkout isn't offered yet
	// (application/storefront.CreatePaymentIntentUseCase requires it) --
	// the existing hosted Checkout Session flow needs only SecretKey.
	PublishableKey string
}

// ConnectStripeResult carries the webhook ingest URL alongside the new
// Connection -- reversed from WhatsApp's ConnectWhatsAppResult (there we
// generate the secret and the merchant pastes our URL+secret into Evolution
// API; here Stripe generates ITS OWN signing secret when the merchant
// registers this URL in their own Dashboard, so no secret is generated or
// returned here -- see SetWebhookSecretUseCase's providedSecret param for
// where the merchant's own secret gets stored).
type ConnectStripeResult struct {
	Connection *integrationsDomain.Connection
	WebhookURL string
}

// ConnectStripeUseCase saves a Stripe secret key encrypted, same
// paste-a-key-no-OAuth pattern as ConnectBolUseCase/ConnectWhatsAppUseCase.
type ConnectStripeUseCase struct {
	db               *gorm.DB
	connRepo         integrationsDomain.ConnectionRepository
	outboxRepo       outbox.Repository
	encryptionSecret string
	publicAPIBaseURL string
}

func NewConnectStripeUseCase(db *gorm.DB, connRepo integrationsDomain.ConnectionRepository, outboxRepo outbox.Repository, encryptionSecret, publicAPIBaseURL string) *ConnectStripeUseCase {
	return &ConnectStripeUseCase{db: db, connRepo: connRepo, outboxRepo: outboxRepo, encryptionSecret: encryptionSecret, publicAPIBaseURL: publicAPIBaseURL}
}

func (uc *ConnectStripeUseCase) Execute(ctx context.Context, in ConnectStripeInput) (*ConnectStripeResult, error) {
	tenantID := tenant.IDFromContext(ctx)
	if tenantID == "" {
		return nil, fmt.Errorf("connectStripe: %w", integrationsDomain.ErrNoTenantContext)
	}

	encryptedSecret, err := crypto.Encrypt(uc.encryptionSecret, in.SecretKey)
	if err != nil {
		return nil, fmt.Errorf("connectStripe: %w", err)
	}

	var conn integrationsDomain.Connection
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		conn = integrationsDomain.Connection{
			TenantID: tenantID, Provider: integrationsDomain.ProviderStripe,
			ConsumerKey:             in.PublishableKey,
			ConsumerSecretEncrypted: encryptedSecret,
			Status:                  integrationsDomain.StatusConnected,
			CreatedAt:               now, UpdatedAt: now,
		}
		if _, err := integrationsDomain.SaveConnection(ctx, uc.connRepo, &conn); err != nil {
			return err
		}

		event := events.IntegrationConnected{
			Base:         events.Base{EventID: idgen.NewV4(), OccurredAt: now},
			TenantID:     tenantID,
			Provider:     string(integrationsDomain.ProviderStripe),
			ConnectionID: conn.ID,
		}
		payload, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("marshal event: %w", err)
		}
		return uc.outboxRepo.Enqueue(ctx, event.EventName(), event.EventID, payload, event.OccurredAt)
	})
	if err != nil {
		return nil, fmt.Errorf("connectStripe: %w", err)
	}
	return &ConnectStripeResult{
		Connection: &conn,
		WebhookURL: providerWebhookIngestURL(uc.publicAPIBaseURL, tenantID, integrationsDomain.ProviderStripe),
	}, nil
}
