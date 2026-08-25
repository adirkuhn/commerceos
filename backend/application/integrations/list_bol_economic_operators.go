package integrations

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ListBolEconomicOperatorsUseCase backs the "fetch from Bol" convenience
// button on the connection settings form -- calls Bol's real Economic
// Operator API v1 (a separate, differently-versioned API from the Retailer
// API this codebase otherwise talks to; see infrastructure/bol/client.go's
// ListEconomicOperators doc comment) so the merchant doesn't have to
// manually copy the id out of Bol's Partner Platform UI, the only other
// place it's visible. A 403 here (missing economic-operators.read/.manage
// OAuth scope on the connection's app registration) surfaces as a normal
// error -- the caller falls back to manual entry, which stays available
// regardless.
type ListBolEconomicOperatorsUseCase struct {
	db         *gorm.DB
	connRepo   integrationsDomain.ConnectionRepository
	bolFactory integrationsDomain.BolClientFactory
	encSecret  string
}

func NewListBolEconomicOperatorsUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	bolFactory integrationsDomain.BolClientFactory,
	encryptionSecret string,
) *ListBolEconomicOperatorsUseCase {
	return &ListBolEconomicOperatorsUseCase{db: db, connRepo: connRepo, bolFactory: bolFactory, encSecret: encryptionSecret}
}

func (uc *ListBolEconomicOperatorsUseCase) Execute(ctx context.Context, connectionID string) ([]integrationsDomain.BolEconomicOperator, error) {
	var conn integrationsDomain.Connection
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		c, err := uc.connRepo.FindByID(ctx, connectionID)
		if err != nil {
			return err
		}
		conn = *c
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listBolEconomicOperators: %w", err)
	}

	secret, err := crypto.Decrypt(uc.encSecret, conn.ConsumerSecretEncrypted)
	if err != nil {
		return nil, fmt.Errorf("listBolEconomicOperators: decrypt secret: %w", err)
	}
	client, err := uc.bolFactory.NewClient(conn.ConsumerKey, secret)
	if err != nil {
		return nil, fmt.Errorf("listBolEconomicOperators: build client: %w", err)
	}

	operators, err := client.ListEconomicOperators(ctx)
	if err != nil {
		return nil, fmt.Errorf("listBolEconomicOperators: %w: %w", integrationsDomain.ErrBolEconomicOperatorsRequestFailed, err)
	}
	return operators, nil
}
