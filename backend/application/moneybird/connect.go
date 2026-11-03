package moneybird

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	mb "github.com/adirkuhn/commerceos/domain/moneybird"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/idgen"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type connectionStore interface {
	connectionLister
	Create(ctx context.Context, c *integrationsDomain.Connection) error
	Update(ctx context.Context, c *integrationsDomain.Connection) error
	FindDisconnected(ctx context.Context, provider integrationsDomain.Provider, storeURL integrationsDomain.StoreURL) (*integrationsDomain.Connection, error)
}

type ConnectInput struct {
	// APIToken is a Moneybird personal API token (Moneybird -> user
	// settings -> developers).
	APIToken string
}

// ConnectUseCase connects the shop's Moneybird account. The token is
// checked against Moneybird first; with a single administration it is
// chosen straight away. Connecting again replaces the token on the existing
// connection -- one Moneybird connection per shop.
type ConnectUseCase struct {
	db       *gorm.DB
	conns    connectionStore
	settings mb.SettingsRepository
	factory  mb.ClientFactory
	outbox   eventrecord.Enqueuer
	secret   string
}

func NewConnectUseCase(db *gorm.DB, conns connectionStore, settings mb.SettingsRepository, factory mb.ClientFactory, outbox eventrecord.Enqueuer, secret string) *ConnectUseCase {
	return &ConnectUseCase{db: db, conns: conns, settings: settings, factory: factory, outbox: outbox, secret: secret}
}

func (uc *ConnectUseCase) Execute(ctx context.Context, in ConnectInput) (*integrationsDomain.Connection, error) {
	tenantID := tenant.IDFromContext(ctx)
	if tenantID == "" {
		return nil, fmt.Errorf("connectMoneybird: %w", integrationsDomain.ErrNoTenantContext)
	}
	token := strings.TrimSpace(in.APIToken)
	if token == "" {
		return nil, fmt.Errorf("connectMoneybird: %w", mb.ErrTokenInvalid)
	}
	admins, err := uc.factory.New(token, "").ListAdministrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("connectMoneybird: %w", err)
	}
	encrypted, err := crypto.Encrypt(uc.secret, token)
	if err != nil {
		return nil, fmt.Errorf("connectMoneybird: %w", err)
	}

	var conn integrationsDomain.Connection
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		all, err := uc.conns.ListByTenant(ctx)
		if err != nil {
			return err
		}
		now := time.Now()
		if existing := findMoneybirdConnection(all); existing != nil {
			conn = *existing
			conn.ConsumerSecretEncrypted, conn.Status, conn.UpdatedAt = encrypted, integrationsDomain.StatusConnected, now
			if err := uc.conns.Update(ctx, &conn); err != nil {
				return err
			}
			return eventrecord.Record(ctx, uc.outbox, events.ConnectionUpdated, conn.ID,
				eventrecord.Data{"provider": string(integrationsDomain.ProviderMoneybird)},
				eventrecord.NewChanges().Set("api_key", "", "changed"))
		}
		conn = integrationsDomain.Connection{
			TenantID: tenantID, Provider: integrationsDomain.ProviderMoneybird, ConsumerSecretEncrypted: encrypted,
			Status: integrationsDomain.StatusConnected, CreatedAt: now, UpdatedAt: now,
		}
		reconnected, err := integrationsDomain.SaveConnection(ctx, uc.conns, &conn)
		if err != nil {
			return err
		}
		if reconnected {
			// the disconnected row kept its settings (COM-15)
			return uc.recordConnected(ctx, tenantID, conn.ID, now)
		}
		settings := mb.Settings{ConnectionID: conn.ID, TenantID: tenantID, CreatedAt: now, UpdatedAt: now}
		if len(admins) == 1 {
			settings.AdministrationID, settings.AdministrationName = admins[0].ID, admins[0].Name
		}
		if err := uc.settings.Upsert(ctx, &settings); err != nil {
			return err
		}
		return uc.recordConnected(ctx, tenantID, conn.ID, now)
	})
	if err != nil {
		return nil, fmt.Errorf("connectMoneybird: %w", err)
	}
	return &conn, nil
}

func (uc *ConnectUseCase) recordConnected(ctx context.Context, tenantID, connectionID string, now time.Time) error {
	event := events.IntegrationConnected{
		Base:     events.Base{EventID: idgen.NewV4(), OccurredAt: now},
		TenantID: tenantID, Provider: string(integrationsDomain.ProviderMoneybird), ConnectionID: connectionID,
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return uc.outbox.Enqueue(ctx, event.EventName(), event.EventID, payload, now)
}
