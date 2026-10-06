package integrations

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type ConnectAIProviderInput struct {
	Provider integrationsDomain.AIProvider
	// APIKey is plaintext in, encrypted before it's ever written -- empty
	// is only valid for AIProviderOllama (see AIProvider.RequiresAPIKey).
	APIKey  string
	BaseURL string
	Model   string
	// FallbackModels is a comma-separated list of model IDs to retry, in
	// order, when Model comes back rate-limited -- see
	// domain/integrations.AIConnection.FallbackModels.
	FallbackModels string
}

// ConnectAIProviderUseCase upserts the tenant's single AIConnection --
// switching providers overwrites the existing row rather than adding a
// second one, per the "one active connection at a time" decision.
type ConnectAIProviderUseCase struct {
	db               *gorm.DB
	aiConnRepo       integrationsDomain.AIConnectionRepository
	encryptionSecret string
	outbox           eventrecord.Enqueuer
}

func NewConnectAIProviderUseCase(db *gorm.DB, aiConnRepo integrationsDomain.AIConnectionRepository, encryptionSecret string, outbox eventrecord.Enqueuer) *ConnectAIProviderUseCase {
	return &ConnectAIProviderUseCase{db: db, aiConnRepo: aiConnRepo, encryptionSecret: encryptionSecret, outbox: outbox}
}

func (uc *ConnectAIProviderUseCase) Execute(ctx context.Context, in ConnectAIProviderInput) (*integrationsDomain.AIConnection, error) {
	if !in.Provider.Valid() {
		return nil, fmt.Errorf("connectAIProvider: %w", integrationsDomain.ErrProviderNotSupported)
	}
	if in.Provider.RequiresAPIKey() && in.APIKey == "" {
		return nil, fmt.Errorf("connectAIProvider: %w", integrationsDomain.ErrAPIKeyRequired)
	}
	if in.Provider == integrationsDomain.AIProviderOllama && in.BaseURL == "" {
		return nil, fmt.Errorf("connectAIProvider: %w", integrationsDomain.ErrBaseURLRequired)
	}

	var apiKeyEncrypted string
	if in.APIKey != "" {
		encrypted, err := crypto.Encrypt(uc.encryptionSecret, in.APIKey)
		if err != nil {
			return nil, fmt.Errorf("connectAIProvider: %w", err)
		}
		apiKeyEncrypted = encrypted
	}

	tenantID := tenant.IDFromContext(ctx)
	if tenantID == "" {
		return nil, fmt.Errorf("connectAIProvider: %w", integrationsDomain.ErrNoTenantContext)
	}

	conn := &integrationsDomain.AIConnection{
		TenantID: tenantID, Provider: in.Provider,
		APIKeyEncrypted: apiKeyEncrypted, BaseURL: in.BaseURL, Model: in.Model, FallbackModels: in.FallbackModels,
		Status: integrationsDomain.StatusConnected, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.aiConnRepo.Upsert(ctx, conn); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.AIConnectionConnected, conn.ID, eventrecord.Data{
			"provider": string(conn.Provider), "model": conn.Model, "base_url": conn.BaseURL, "fallback_models": conn.FallbackModels,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("connectAIProvider: %w", err)
	}
	return conn, nil
}
