package apikeys

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"gorm.io/gorm"

	apikeysDomain "github.com/adirkuhn/commerceos/domain/apikeys"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// CreateAPIKeyResult carries the PLAINTEXT key back to the caller exactly
// once, at creation time -- only its hash persists (APIKey.KeyHash), so the
// merchant must paste it into the storefront's (e.g. the WooCommerce
// plugin's) settings right away.
type CreateAPIKeyResult struct {
	ID             string
	Name           string
	Description    string
	SalesChannelID *string
	CreatedAt      time.Time
	PlaintextKey   string
}

type CreateAPIKeyUseCase struct {
	db   *gorm.DB
	repo apikeysDomain.Repository
}

func NewCreateAPIKeyUseCase(db *gorm.DB, repo apikeysDomain.Repository) *CreateAPIKeyUseCase {
	return &CreateAPIKeyUseCase{db: db, repo: repo}
}

// Execute mints a new key. salesChannelID is nil for a tenant-wide key
// (the historical/default case, e.g. the WooCommerce plugin's) -- see
// apikeysDomain.APIKey.SalesChannelID.
func (uc *CreateAPIKeyUseCase) Execute(ctx context.Context, name, description string, salesChannelID *string) (*CreateAPIKeyResult, error) {
	if name == "" {
		return nil, fmt.Errorf("createApiKey: %w", apikeysDomain.ErrNameRequired)
	}

	plaintext, err := generateKey()
	if err != nil {
		return nil, fmt.Errorf("createApiKey: generate: %w", err)
	}

	k := &apikeysDomain.APIKey{Name: name, Description: description, KeyHash: hashKey(plaintext), SalesChannelID: salesChannelID}
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		k.TenantID = tenant.IDFromContext(ctx)
		return uc.repo.Create(ctx, k)
	})
	if err != nil {
		return nil, fmt.Errorf("createApiKey: %w", err)
	}

	return &CreateAPIKeyResult{
		ID: k.ID, Name: k.Name, Description: k.Description, SalesChannelID: k.SalesChannelID,
		CreatedAt: k.CreatedAt, PlaintextKey: plaintext,
	}, nil
}

// hashKey is the one hashing function both CreateAPIKeyUseCase (writes it)
// and ResolveAPIKeyUseCase (looks it up by it) use -- must stay identical
// between the two or every key silently stops working.
func hashKey(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

func generateKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
