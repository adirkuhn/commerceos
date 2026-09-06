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
}

// ConnectStripeUseCase saves a Stripe secret key encrypted, same
// paste-a-key-no-OAuth pattern as ConnectBolUseCase/ConnectWhatsAppUseCase.
// No webhook secret generated -- unlike WhatsApp, nothing in this codebase
// listens for Stripe webhooks yet (payment-link reconciliation is a future
// piece, see domain/integrations.PaymentLink.ExternalID's doc comment).
type ConnectStripeUseCase struct {
	db               *gorm.DB
	connRepo         integrationsDomain.ConnectionRepository
	outboxRepo       outbox.Repository
	encryptionSecret string
}

func NewConnectStripeUseCase(db *gorm.DB, connRepo integrationsDomain.ConnectionRepository, outboxRepo outbox.Repository, encryptionSecret string) *ConnectStripeUseCase {
	return &ConnectStripeUseCase{db: db, connRepo: connRepo, outboxRepo: outboxRepo, encryptionSecret: encryptionSecret}
}

func (uc *ConnectStripeUseCase) Execute(ctx context.Context, in ConnectStripeInput) (*integrationsDomain.Connection, error) {
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
			ConsumerSecretEncrypted: encryptedSecret,
			Status:                  integrationsDomain.StatusConnected,
			CreatedAt:               now, UpdatedAt: now,
		}
		if err := uc.connRepo.Create(ctx, &conn); err != nil {
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
	return &conn, nil
}
