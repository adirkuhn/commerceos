package integrations

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// UpdateWhatsAppCredentialsInput mirrors ConnectWhatsAppInput -- same three
// fields, but APIKey is optional here: empty means "keep the currently
// stored key," since the merchant is never shown the decrypted value to
// re-paste unchanged (see WhatsAppCredentialsModal's doc comment on the
// frontend). BaseURL/InstanceName are never secret, so they're always
// replaced with whatever's submitted, blank or not (Execute validates
// BaseURL the same way ConnectWhatsAppUseCase does).
type UpdateWhatsAppCredentialsInput struct {
	BaseURL      string
	InstanceName string
	APIKey       string
}

// UpdateWhatsAppCredentialsUseCase fixes what ConnectWhatsAppUseCase can't:
// rotating a wrong/expired API key (or base URL/instance name) on an
// existing connection in place. Disconnect+reconnect was the only prior
// option, and that creates a brand new Connection row -- silently
// resetting WhatsAppTestModeEnabled/WhatsAppTestPhoneE164/
// WhatsAppAIParsingEnabled to their zero values and rotating the webhook
// secret (breaking the already-configured evolution-go webhook) as a side
// effect of what should've been a one-field edit. This updates StoreURL/
// ConsumerKey/ConsumerSecretEncrypted only, leaving every other column
// (including WebhookSecretEncrypted) untouched.
type UpdateWhatsAppCredentialsUseCase struct {
	db               *gorm.DB
	connRepo         integrationsDomain.ConnectionRepository
	encryptionSecret string
	outbox           eventrecord.Enqueuer
}

func NewUpdateWhatsAppCredentialsUseCase(db *gorm.DB, connRepo integrationsDomain.ConnectionRepository, encryptionSecret string, outbox eventrecord.Enqueuer) *UpdateWhatsAppCredentialsUseCase {
	return &UpdateWhatsAppCredentialsUseCase{db: db, connRepo: connRepo, encryptionSecret: encryptionSecret, outbox: outbox}
}

func (uc *UpdateWhatsAppCredentialsUseCase) Execute(ctx context.Context, connectionID string, in UpdateWhatsAppCredentialsInput) (*integrationsDomain.Connection, error) {
	storeURL := integrationsDomain.StoreURL(in.BaseURL)
	if err := storeURL.Validate(); err != nil {
		return nil, fmt.Errorf("updateWhatsAppCredentials: %w", err)
	}

	var result *integrationsDomain.Connection
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.connRepo.FindByID(ctx, connectionID)
		if err != nil {
			return err
		}
		if existing.Provider != integrationsDomain.ProviderWhatsApp {
			return integrationsDomain.ErrNotWhatsAppConnection
		}
		before := *existing
		existing.StoreURL = storeURL
		existing.ConsumerKey = in.InstanceName
		if in.APIKey != "" {
			encrypted, err := crypto.Encrypt(uc.encryptionSecret, in.APIKey)
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
			Set("base_url", before.StoreURL.String(), existing.StoreURL.String()).
			Set("instance_name", before.ConsumerKey, existing.ConsumerKey)
		if in.APIKey != "" {
			changes = changes.Set("api_key", nil, secretChanged)
		}
		return recordConnectionUpdated(ctx, uc.outbox, *existing, changes)
	})
	if err != nil {
		if errors.Is(err, integrationsDomain.ErrNotWhatsAppConnection) || errors.Is(err, integrationsDomain.ErrConnectionNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("updateWhatsAppCredentials: %w", err)
	}
	return result, nil
}
