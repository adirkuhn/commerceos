package integrations

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// SetWebhookSecretResult carries the PLAINTEXT secret back to the caller
// exactly once, at generation time -- it's never stored or retrievable in
// plaintext again (only WebhookSecretEncrypted persists), so the merchant
// must paste it into the provider's webhook admin UI right away.
type SetWebhookSecretResult struct {
	ConnectionID    string
	PlaintextSecret string
}

// SetWebhookSecretUseCase (re)generates the per-connection secret
// application/webhooks.ReceiveWebhookUseCase verifies inbound webhook
// signatures against. Re-running it rotates the secret -- the old value
// stops verifying immediately, so the merchant must update the provider's
// webhook config with the new one.
//
// providedSecret in Execute lets a caller supply the value instead of
// generating one -- the reverse of MyParcel/WhatsApp's flow (we generate,
// merchant pastes it into the provider), Stripe/Revolut generate their own
// signing secret when the merchant registers a webhook endpoint in their
// dashboard, so the merchant pastes THEIR value in here instead. Empty
// string keeps the original generate-a-random-value behavior.
type SetWebhookSecretUseCase struct {
	db               *gorm.DB
	connRepo         integrationsDomain.ConnectionRepository
	encryptionSecret string
}

func NewSetWebhookSecretUseCase(db *gorm.DB, connRepo integrationsDomain.ConnectionRepository, encryptionSecret string) *SetWebhookSecretUseCase {
	return &SetWebhookSecretUseCase{db: db, connRepo: connRepo, encryptionSecret: encryptionSecret}
}

func (uc *SetWebhookSecretUseCase) Execute(ctx context.Context, connectionID string, providedSecret string) (*SetWebhookSecretResult, error) {
	plaintext := providedSecret
	if plaintext == "" {
		generated, err := generateSecret()
		if err != nil {
			return nil, fmt.Errorf("setWebhookSecret: generate: %w", err)
		}
		plaintext = generated
	}

	var result SetWebhookSecretResult
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.connRepo.FindByID(ctx, connectionID)
		if err != nil {
			return err
		}
		encrypted, err := crypto.Encrypt(uc.encryptionSecret, plaintext)
		if err != nil {
			return fmt.Errorf("encrypt: %w", err)
		}
		existing.WebhookSecretEncrypted = encrypted
		existing.UpdatedAt = time.Now()
		if err := uc.connRepo.Update(ctx, existing); err != nil {
			return err
		}
		result = SetWebhookSecretResult{ConnectionID: existing.ID, PlaintextSecret: plaintext}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("setWebhookSecret: %w", err)
	}
	return &result, nil
}

func generateSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
