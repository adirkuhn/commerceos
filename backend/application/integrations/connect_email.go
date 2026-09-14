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

// ConnectEmailInput maps directly onto domain/integrations.ProviderEmail's
// field mapping (see that constant's doc comment) -- Host/Port/From are
// required, Username/Password optional (a relay that needs no auth is
// valid; infrastructure/messaging/smtp.Client already sends unauthenticated
// when Username is blank).
type ConnectEmailInput struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

// ConnectEmailUseCase saves a tenant's own outbound SMTP relay, same
// paste-credentials-no-OAuth pattern as ConnectBolUseCase/ConnectStripeUseCase
// -- no webhook step (email has no inbound webhook), so this returns just
// the Connection, same simpler shape as ConnectBolUseCase.
type ConnectEmailUseCase struct {
	db               *gorm.DB
	connRepo         integrationsDomain.ConnectionRepository
	outboxRepo       outbox.Repository
	encryptionSecret string
}

func NewConnectEmailUseCase(db *gorm.DB, connRepo integrationsDomain.ConnectionRepository, outboxRepo outbox.Repository, encryptionSecret string) *ConnectEmailUseCase {
	return &ConnectEmailUseCase{db: db, connRepo: connRepo, outboxRepo: outboxRepo, encryptionSecret: encryptionSecret}
}

func (uc *ConnectEmailUseCase) Execute(ctx context.Context, in ConnectEmailInput) (*integrationsDomain.Connection, error) {
	if in.Host == "" {
		return nil, fmt.Errorf("connectEmail: %w", integrationsDomain.ErrEmailHostRequired)
	}
	if in.Port == "" {
		return nil, fmt.Errorf("connectEmail: %w", integrationsDomain.ErrEmailPortRequired)
	}
	if in.From == "" {
		return nil, fmt.Errorf("connectEmail: %w", integrationsDomain.ErrEmailFromRequired)
	}

	tenantID := tenant.IDFromContext(ctx)
	if tenantID == "" {
		return nil, fmt.Errorf("connectEmail: %w", integrationsDomain.ErrNoTenantContext)
	}

	encryptedPassword, err := crypto.Encrypt(uc.encryptionSecret, in.Password)
	if err != nil {
		return nil, fmt.Errorf("connectEmail: %w", err)
	}

	var conn integrationsDomain.Connection
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		port, from := in.Port, in.From
		conn = integrationsDomain.Connection{
			TenantID: tenantID, Provider: integrationsDomain.ProviderEmail,
			StoreURL:                integrationsDomain.StoreURL(in.Host),
			ConsumerKey:             in.Username,
			ConsumerSecretEncrypted: encryptedPassword,
			SMTPPort:                &port,
			EmailFrom:               &from,
			Status:                  integrationsDomain.StatusConnected,
			CreatedAt:               now, UpdatedAt: now,
		}
		if err := uc.connRepo.Create(ctx, &conn); err != nil {
			return err
		}

		event := events.IntegrationConnected{
			Base:         events.Base{EventID: idgen.NewV4(), OccurredAt: now},
			TenantID:     tenantID,
			Provider:     string(integrationsDomain.ProviderEmail),
			ConnectionID: conn.ID,
		}
		payload, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("marshal event: %w", err)
		}
		return uc.outboxRepo.Enqueue(ctx, event.EventName(), event.EventID, payload, event.OccurredAt)
	})
	if err != nil {
		return nil, fmt.Errorf("connectEmail: %w", err)
	}
	return &conn, nil
}
