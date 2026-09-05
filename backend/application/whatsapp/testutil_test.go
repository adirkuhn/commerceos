package whatsapp_test

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	appwhatsapp "github.com/adirkuhn/commerceos/application/whatsapp"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

// newMockGormDB returns a *gorm.DB backed by sqlmock, wired to satisfy
// however many tenant.WithTenant Begin/set_config/Commit sequences the test
// needs -- same pattern as application/channels' own testutil_test.go.
func newMockGormDB(t *testing.T, n int) *gorm.DB {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { mockDB.Close() })

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: mockDB}), &gorm.Config{})
	require.NoError(t, err)

	mock.MatchExpectationsInOrder(false)
	for i := 0; i < n; i++ {
		mock.ExpectBegin()
		mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()
	}

	return gormDB
}

// ── shared fakes for handle_inbound_message_test.go ─────────────────────────

type fakeConnectionRepo struct {
	integrationsDomain.ConnectionRepository
	conns []integrationsDomain.Connection
}

func (f *fakeConnectionRepo) ListByTenant(_ context.Context) ([]integrationsDomain.Connection, error) {
	return f.conns, nil
}

type fakeCustomerRepo struct {
	customersDomain.CustomerRepository
	byPhone map[string]customersDomain.Customer
}

func (f *fakeCustomerRepo) FindByPhoneE164(_ context.Context, phone string) (*customersDomain.Customer, error) {
	if c, ok := f.byPhone[phone]; ok {
		return &c, nil
	}
	return nil, nil
}

type fakeEvolutionAPIClient struct {
	sentTo   string
	sentBody string
	err      error
}

func (f *fakeEvolutionAPIClient) SendText(_ context.Context, toPhone, body string) error {
	f.sentTo, f.sentBody = toPhone, body
	return f.err
}

type fakeEvolutionAPIClientFactory struct {
	client *fakeEvolutionAPIClient
}

func (f *fakeEvolutionAPIClientFactory) NewClient(_, _, _ string) (integrationsDomain.EvolutionAPIClient, error) {
	return f.client, nil
}

type fakeProductInfoLookup struct {
	info *appwhatsapp.ProductInfo
	err  error
}

func (f *fakeProductInfoLookup) Execute(_ context.Context, _ string) (*appwhatsapp.ProductInfo, error) {
	return f.info, f.err
}
