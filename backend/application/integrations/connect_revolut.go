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

type ConnectRevolutInput struct {
	// APIKey is a Revolut Business Merchant API secret key -- no OAuth,
	// same paste-a-key pattern as ConnectStripeUseCase.
	APIKey string
}

// ConnectRevolutResult mirrors ConnectStripeResult -- see its doc comment.
type ConnectRevolutResult struct {
	Connection *integrationsDomain.Connection
	WebhookURL string
}

// ConnectRevolutUseCase saves a Revolut Merchant API key encrypted -- see
// ConnectStripeUseCase's doc comment, same shape/reasoning.
type ConnectRevolutUseCase struct {
	db               *gorm.DB
	connRepo         integrationsDomain.ConnectionRepository
	outboxRepo       outbox.Repository
	encryptionSecret string
	publicAPIBaseURL string
}

func NewConnectRevolutUseCase(db *gorm.DB, connRepo integrationsDomain.ConnectionRepository, outboxRepo outbox.Repository, encryptionSecret, publicAPIBaseURL string) *ConnectRevolutUseCase {
	return &ConnectRevolutUseCase{db: db, connRepo: connRepo, outboxRepo: outboxRepo, encryptionSecret: encryptionSecret, publicAPIBaseURL: publicAPIBaseURL}
}

func (uc *ConnectRevolutUseCase) Execute(ctx context.Context, in ConnectRevolutInput) (*ConnectRevolutResult, error) {
	tenantID := tenant.IDFromContext(ctx)
	if tenantID == "" {
		return nil, fmt.Errorf("connectRevolut: %w", integrationsDomain.ErrNoTenantContext)
	}

	encryptedSecret, err := crypto.Encrypt(uc.encryptionSecret, in.APIKey)
	if err != nil {
		return nil, fmt.Errorf("connectRevolut: %w", err)
	}

	var conn integrationsDomain.Connection
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		conn = integrationsDomain.Connection{
			TenantID: tenantID, Provider: integrationsDomain.ProviderRevolut,
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
			Provider:     string(integrationsDomain.ProviderRevolut),
			ConnectionID: conn.ID,
		}
		payload, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("marshal event: %w", err)
		}
		return uc.outboxRepo.Enqueue(ctx, event.EventName(), event.EventID, payload, event.OccurredAt)
	})
	if err != nil {
		return nil, fmt.Errorf("connectRevolut: %w", err)
	}
	return &ConnectRevolutResult{
		Connection: &conn,
		WebhookURL: providerWebhookIngestURL(uc.publicAPIBaseURL, tenantID, integrationsDomain.ProviderRevolut),
	}, nil
}
