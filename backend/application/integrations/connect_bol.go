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

type ConnectBolInput struct {
	ClientID           string
	ClientSecret       string
	EconomicOperatorID string
}

// ConnectBolUseCase is Bol's connect flow -- unlike WooCommerce there's no
// OAuth redirect/callback (Bol's API uses client_credentials, see
// infrastructure/bol), so this is the only connect path: paste the 3
// values from Bol's Retailer API portal, save encrypted.
type ConnectBolUseCase struct {
	db               *gorm.DB
	connRepo         integrationsDomain.ConnectionRepository
	outboxRepo       outbox.Repository
	encryptionSecret string
}

func NewConnectBolUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	outboxRepo outbox.Repository,
	encryptionSecret string,
) *ConnectBolUseCase {
	return &ConnectBolUseCase{db: db, connRepo: connRepo, outboxRepo: outboxRepo, encryptionSecret: encryptionSecret}
}

func (uc *ConnectBolUseCase) Execute(ctx context.Context, in ConnectBolInput) (*integrationsDomain.Connection, error) {
	tenantID := tenant.IDFromContext(ctx)
	if tenantID == "" {
		return nil, fmt.Errorf("connectBol: %w", integrationsDomain.ErrNoTenantContext)
	}

	encryptedSecret, err := crypto.Encrypt(uc.encryptionSecret, in.ClientSecret)
	if err != nil {
		return nil, fmt.Errorf("connectBol: %w", err)
	}

	economicOperatorID := in.EconomicOperatorID
	var conn integrationsDomain.Connection
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		conn = integrationsDomain.Connection{
			TenantID: tenantID, Provider: integrationsDomain.ProviderBol,
			ConsumerKey: in.ClientID, ConsumerSecretEncrypted: encryptedSecret,
			BolEconomicOperatorID: &economicOperatorID,
			Status:                integrationsDomain.StatusConnected,
			CreatedAt:             now, UpdatedAt: now,
		}
		if _, err := integrationsDomain.SaveConnection(ctx, uc.connRepo, &conn); err != nil {
			return err
		}

		event := events.IntegrationConnected{
			Base:         events.Base{EventID: idgen.NewV4(), OccurredAt: now},
			TenantID:     tenantID,
			Provider:     string(integrationsDomain.ProviderBol),
			ConnectionID: conn.ID,
		}
		payload, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("marshal event: %w", err)
		}
		return uc.outboxRepo.Enqueue(ctx, event.EventName(), event.EventID, payload, event.OccurredAt)
	})
	if err != nil {
		return nil, fmt.Errorf("connectBol: %w", err)
	}
	return &conn, nil
}
