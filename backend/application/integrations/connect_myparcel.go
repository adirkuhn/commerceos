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
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/idgen"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type ConnectMyParcelInput struct {
	APIKey string
}

// ConnectMyParcelUseCase is MyParcel's connect flow -- one credential (the
// API key), no OAuth redirect. Also creates a default
// ShippingConnectionSettings row in the same transaction, so
// ShippingCarrierOverride's FK (which references
// shipping_connection_settings, not integration_connections directly --
// see infrastructure/repositories/shipping_settings_repo.go's Upsert) is
// always satisfiable once a connection exists, rather than surfacing that
// as a confusing error the first time someone tries to set a per-carrier
// override.
type ConnectMyParcelUseCase struct {
	db               *gorm.DB
	connRepo         integrationsDomain.ConnectionRepository
	settingsRepo     shippingDomain.ShippingConnectionSettingsRepository
	outboxRepo       outbox.Repository
	encryptionSecret string
}

func NewConnectMyParcelUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	settingsRepo shippingDomain.ShippingConnectionSettingsRepository,
	outboxRepo outbox.Repository,
	encryptionSecret string,
) *ConnectMyParcelUseCase {
	return &ConnectMyParcelUseCase{
		db: db, connRepo: connRepo, settingsRepo: settingsRepo, outboxRepo: outboxRepo, encryptionSecret: encryptionSecret,
	}
}

func (uc *ConnectMyParcelUseCase) Execute(ctx context.Context, in ConnectMyParcelInput) (*integrationsDomain.Connection, error) {
	tenantID := tenant.IDFromContext(ctx)
	if tenantID == "" {
		return nil, fmt.Errorf("connectMyParcel: %w", integrationsDomain.ErrNoTenantContext)
	}

	encryptedKey, err := crypto.Encrypt(uc.encryptionSecret, in.APIKey)
	if err != nil {
		return nil, fmt.Errorf("connectMyParcel: %w", err)
	}

	var conn integrationsDomain.Connection
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		conn = integrationsDomain.Connection{
			TenantID: tenantID, Provider: integrationsDomain.ProviderMyParcel,
			ConsumerSecretEncrypted: encryptedKey,
			Status:                  integrationsDomain.StatusConnected,
			CreatedAt:               now, UpdatedAt: now,
		}
		if err := uc.connRepo.Create(ctx, &conn); err != nil {
			return err
		}

		if err := uc.settingsRepo.Upsert(ctx, &shippingDomain.ShippingConnectionSettings{
			ConnectionID: conn.ID, TenantID: tenantID,
			DefaultCarrier: shippingDomain.CarrierPostNL, DefaultPackageType: shippingDomain.PackageTypePackage,
			DefaultLabelFormat: shippingDomain.LabelFormatPDF,
			CreatedAt:          now, UpdatedAt: now,
		}); err != nil {
			return err
		}

		event := events.IntegrationConnected{
			Base:         events.Base{EventID: idgen.NewV4(), OccurredAt: now},
			TenantID:     tenantID,
			Provider:     string(integrationsDomain.ProviderMyParcel),
			ConnectionID: conn.ID,
		}
		payload, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("marshal event: %w", err)
		}
		return uc.outboxRepo.Enqueue(ctx, event.EventName(), event.EventID, payload, event.OccurredAt)
	})
	if err != nil {
		return nil, fmt.Errorf("connectMyParcel: %w", err)
	}
	return &conn, nil
}
