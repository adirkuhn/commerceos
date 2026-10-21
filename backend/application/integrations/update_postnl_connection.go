package integrations

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// UpdatePostNLConnectionInput replaces a PostNL connection's account and
// environment; an empty APIKey keeps the stored one.
type UpdatePostNLConnectionInput struct {
	APIKey         string
	CustomerCode   string
	CustomerNumber string
	Sandbox        bool
}

// UpdatePostNLConnectionUseCase changes a PostNL connection's credentials
// or switches it between sandbox and production (e.g. going live after
// PostNL validated the integration).
type UpdatePostNLConnectionUseCase struct {
	db               *gorm.DB
	connRepo         integrationsDomain.ConnectionRepository
	encryptionSecret string
	outbox           eventrecord.Enqueuer
}

func NewUpdatePostNLConnectionUseCase(db *gorm.DB, connRepo integrationsDomain.ConnectionRepository, encryptionSecret string, outbox eventrecord.Enqueuer) *UpdatePostNLConnectionUseCase {
	return &UpdatePostNLConnectionUseCase{db: db, connRepo: connRepo, encryptionSecret: encryptionSecret, outbox: outbox}
}

func (uc *UpdatePostNLConnectionUseCase) Execute(ctx context.Context, connectionID string, in UpdatePostNLConnectionInput) (*integrationsDomain.Connection, error) {
	code, number, err := postnlAccount(in.CustomerCode, in.CustomerNumber)
	if err != nil {
		return nil, fmt.Errorf("updatePostNLConnection: %w", err)
	}
	apiKey := strings.TrimSpace(in.APIKey)

	var result *integrationsDomain.Connection
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.connRepo.FindByID(ctx, connectionID)
		if err != nil {
			return err
		}
		if existing.Provider != integrationsDomain.ProviderPostNL {
			return integrationsDomain.ErrNotPostNLConnection
		}
		before := *existing
		existing.PostNLCustomerCode, existing.PostNLCustomerNumber, existing.PostNLSandbox = code, number, in.Sandbox
		if apiKey != "" {
			encrypted, err := crypto.Encrypt(uc.encryptionSecret, apiKey)
			if err != nil {
				return fmt.Errorf("encrypt: %w", err)
			}
			existing.ConsumerSecretEncrypted = encrypted
		}
		existing.UpdatedAt = time.Now()
		if err := uc.connRepo.Update(ctx, existing); err != nil {
			return err
		}
		result = existing
		changes := eventrecord.NewChanges().
			Set("customer_code", before.PostNLCustomerCode, existing.PostNLCustomerCode).
			Set("customer_number", before.PostNLCustomerNumber, existing.PostNLCustomerNumber).
			Set("sandbox", before.PostNLSandbox, existing.PostNLSandbox)
		if apiKey != "" {
			changes = changes.Set("api_key", nil, secretChanged)
		}
		return recordConnectionUpdated(ctx, uc.outbox, *existing, changes)
	})
	if err != nil {
		return nil, fmt.Errorf("updatePostNLConnection: %w", err)
	}
	return result, nil
}
