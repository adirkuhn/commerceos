package integrations

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// NextStatusAfterTest maps a WooClient.TestConnection result to the
// connection's new status — extracted as a pure function so the decision
// itself is unit-tested without needing a real DB.
func NextStatusAfterTest(err error) integrationsDomain.Status {
	if err != nil {
		return integrationsDomain.StatusError
	}
	return integrationsDomain.StatusConnected
}

type TestConnectionUseCase struct {
	db               *gorm.DB
	connRepo         integrationsDomain.ConnectionRepository
	wooFactory       integrationsDomain.WooClientFactory
	encryptionSecret string
}

func NewTestConnectionUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	wooFactory integrationsDomain.WooClientFactory,
	encryptionSecret string,
) *TestConnectionUseCase {
	return &TestConnectionUseCase{
		db:               db,
		connRepo:         connRepo,
		wooFactory:       wooFactory,
		encryptionSecret: encryptionSecret,
	}
}

// Execute decrypts the stored credential, probes the real WooCommerce store,
// and persists the resulting status (connected/error) either way — the test
// outcome (testErr) must NOT be returned from the transaction closure below,
// since a non-nil error there rolls back the status Update too.
func (uc *TestConnectionUseCase) Execute(ctx context.Context, id string) (*integrationsDomain.Connection, error) {
	var conn integrationsDomain.Connection
	var testErr error

	txErr := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.connRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		conn = *existing

		secret, decryptErr := crypto.Decrypt(uc.encryptionSecret, conn.ConsumerSecretEncrypted)
		if decryptErr != nil {
			testErr = decryptErr
		} else {
			client, clientErr := uc.wooFactory.NewClient(conn.StoreURL.String(), conn.ConsumerKey, secret)
			if clientErr != nil {
				testErr = clientErr
			} else {
				testErr = client.TestConnection(ctx)
			}
		}

		conn.Status = NextStatusAfterTest(testErr)
		conn.UpdatedAt = time.Now()
		return uc.connRepo.Update(ctx, &conn)
	})
	if txErr != nil {
		return nil, fmt.Errorf("testConnection: %w", txErr)
	}
	if testErr != nil {
		return &conn, fmt.Errorf("testConnection: %w: %w", integrationsDomain.ErrConnectionTestFailed, testErr)
	}
	return &conn, nil
}
