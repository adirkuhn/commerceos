package integrations

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/domain/outbox"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/state"
)

type CompleteWooConnectInput struct {
	State          string
	ConsumerKey    string
	ConsumerSecret string
}

// CompleteWooConnectUseCase is invoked by the public, unauthenticated
// WooCommerce callback handler — there is no JWT on this request, so the
// tenant is derived entirely from the signed state token minted by
// StartWooConnectUseCase, not from request context/claims.
type CompleteWooConnectUseCase struct {
	db               *gorm.DB
	connRepo         integrationsDomain.ConnectionRepository
	outboxRepo       outbox.Repository
	stateSecret      []byte
	encryptionSecret string
}

func NewCompleteWooConnectUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	outboxRepo outbox.Repository,
	stateSecret []byte,
	encryptionSecret string,
) *CompleteWooConnectUseCase {
	return &CompleteWooConnectUseCase{
		db:               db,
		connRepo:         connRepo,
		outboxRepo:       outboxRepo,
		stateSecret:      stateSecret,
		encryptionSecret: encryptionSecret,
	}
}

func (uc *CompleteWooConnectUseCase) Execute(ctx context.Context, in CompleteWooConnectInput) (*integrationsDomain.Connection, error) {
	parsedState, err := state.ParseConnectState(uc.stateSecret, in.State, time.Now())
	if err != nil {
		return nil, fmt.Errorf("completeWooConnect: %w: %w", integrationsDomain.ErrStateInvalid, err)
	}

	encryptedSecret, err := crypto.Encrypt(uc.encryptionSecret, in.ConsumerSecret)
	if err != nil {
		return nil, fmt.Errorf("completeWooConnect: %w", err)
	}

	conn, err := createWooConnection(ctx, uc.db, uc.connRepo, uc.outboxRepo,
		parsedState.TenantID, parsedState.StoreURL, in.ConsumerKey, encryptedSecret)
	if err != nil {
		return nil, fmt.Errorf("completeWooConnect: %w", err)
	}
	return conn, nil
}
