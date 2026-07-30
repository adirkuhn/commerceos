package integrations

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/domain/outbox"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type ManualConnectWooCommerceInput struct {
	StoreURL       string
	ConsumerKey    string
	ConsumerSecret string
}

// ManualConnectWooCommerceUseCase lets a tenant paste an already-generated
// WooCommerce REST API key/secret (Woo admin > Settings > Advanced > REST
// API) instead of going through the wc-auth redirect — needed when the
// merchant's store can't reach this API's callback_url (e.g. a real
// production store and a dev-only backend). Runs on an authenticated route,
// so the tenant comes from request context, not a signed state token.
type ManualConnectWooCommerceUseCase struct {
	db               *gorm.DB
	connRepo         integrationsDomain.ConnectionRepository
	outboxRepo       outbox.Repository
	encryptionSecret string
}

func NewManualConnectWooCommerceUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	outboxRepo outbox.Repository,
	encryptionSecret string,
) *ManualConnectWooCommerceUseCase {
	return &ManualConnectWooCommerceUseCase{
		db:               db,
		connRepo:         connRepo,
		outboxRepo:       outboxRepo,
		encryptionSecret: encryptionSecret,
	}
}

func (uc *ManualConnectWooCommerceUseCase) Execute(ctx context.Context, in ManualConnectWooCommerceInput) (*integrationsDomain.Connection, error) {
	storeURL := integrationsDomain.StoreURL(in.StoreURL)
	if err := storeURL.Validate(); err != nil {
		return nil, fmt.Errorf("manualConnectWooCommerce: %w", err)
	}

	tenantID := tenant.IDFromContext(ctx)
	if tenantID == "" {
		return nil, fmt.Errorf("manualConnectWooCommerce: %w", integrationsDomain.ErrNoTenantContext)
	}

	encryptedSecret, err := crypto.Encrypt(uc.encryptionSecret, in.ConsumerSecret)
	if err != nil {
		return nil, fmt.Errorf("manualConnectWooCommerce: %w", err)
	}

	conn, err := createWooConnection(ctx, uc.db, uc.connRepo, uc.outboxRepo,
		tenantID, in.StoreURL, in.ConsumerKey, encryptedSecret)
	if err != nil {
		return nil, fmt.Errorf("manualConnectWooCommerce: %w", err)
	}
	return conn, nil
}
