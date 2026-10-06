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
	outbox           eventrecord.Enqueuer
}

func NewTestConnectionUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	wooFactory integrationsDomain.WooClientFactory,
	encryptionSecret string,
	outbox eventrecord.Enqueuer,
) *TestConnectionUseCase {
	return &TestConnectionUseCase{
		db:               db,
		connRepo:         connRepo,
		wooFactory:       wooFactory,
		encryptionSecret: encryptionSecret,
		outbox:           outbox,
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

		before := conn.Status
		conn.Status = NextStatusAfterTest(testErr)
		conn.UpdatedAt = time.Now()
		if err := uc.connRepo.Update(ctx, &conn); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.ConnectionTested, conn.ID,
			eventrecord.Data{"provider": string(conn.Provider), "succeeded": testErr == nil},
			eventrecord.NewChanges().Set("status", string(before), string(conn.Status)))
	})
	if txErr != nil {
		return nil, fmt.Errorf("testConnection: %w", txErr)
	}
	if testErr != nil {
		return &conn, fmt.Errorf("testConnection: %w: %w", integrationsDomain.ErrConnectionTestFailed, testErr)
	}
	return &conn, nil
}
