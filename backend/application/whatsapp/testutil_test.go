package whatsapp_test

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	appcustomers "github.com/adirkuhn/commerceos/application/customers"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	appwhatsapp "github.com/adirkuhn/commerceos/application/whatsapp"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	whatsappDomain "github.com/adirkuhn/commerceos/domain/whatsapp"
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
	byID    map[string]customersDomain.Customer
}

func (f *fakeCustomerRepo) FindByPhoneE164(_ context.Context, phone string) (*customersDomain.Customer, error) {
	if c, ok := f.byPhone[phone]; ok {
		return &c, nil
	}
	return nil, nil
}

func (f *fakeCustomerRepo) FindByID(_ context.Context, id string) (*customersDomain.Customer, error) {
	if f.byID != nil {
		if c, ok := f.byID[id]; ok {
			return &c, nil
		}
	}
	return nil, customersDomain.ErrCustomerNotFound
}

type fakeSalesOrderRepo struct {
	omsDomain.SalesOrderRepository
	byID map[string]omsDomain.SalesOrder
}

func (f *fakeSalesOrderRepo) FindByID(_ context.Context, id string) (*omsDomain.SalesOrder, error) {
	if o, ok := f.byID[id]; ok {
		return &o, nil
	}
	return nil, omsDomain.ErrSalesOrderNotFound
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
	info      *appwhatsapp.ProductInfo
	err       error
	byCode    map[string]appwhatsapp.ProductInfo
	errByCode map[string]error
}

func (f *fakeProductInfoLookup) Execute(_ context.Context, skuCode string) (*appwhatsapp.ProductInfo, error) {
	if f.byCode != nil {
		if err, ok := f.errByCode[skuCode]; ok {
			return nil, err
		}
		if info, ok := f.byCode[skuCode]; ok {
			return &info, nil
		}
	}
	return f.info, f.err
}

// ── shared fakes for order_flow_test.go ─────────────────────────────────────

type fakeConversationRepo struct {
	byPhone map[string]whatsappDomain.Conversation
	saved   []whatsappDomain.Conversation
}

func (f *fakeConversationRepo) FindByPhone(_ context.Context, phone string) (*whatsappDomain.Conversation, error) {
	if c, ok := f.byPhone[phone]; ok {
		return &c, nil
	}
	return nil, nil
}

func (f *fakeConversationRepo) Upsert(_ context.Context, c *whatsappDomain.Conversation) error {
	if f.byPhone == nil {
		f.byPhone = map[string]whatsappDomain.Conversation{}
	}
	f.byPhone[c.PhoneE164] = *c
	f.saved = append(f.saved, *c)
	return nil
}

type fakeSalesOrderCreator struct {
	in     appoms.CreateSalesOrderInput
	called bool
	out    *appoms.CreateSalesOrderResult
	err    error
}

func (f *fakeSalesOrderCreator) Execute(_ context.Context, in appoms.CreateSalesOrderInput) (*appoms.CreateSalesOrderResult, error) {
	f.in = in
	f.called = true
	if f.err != nil {
		return nil, f.err
	}
	if f.out != nil {
		return f.out, nil
	}
	return &appoms.CreateSalesOrderResult{SalesOrder: omsDomain.SalesOrder{ID: "order-1", Number: in.Number}}, nil
}

type fakeCustomerCreator struct {
	in     appcustomers.CreateCustomerInput
	called bool
	out    *customersDomain.Customer
	err    error
}

func (f *fakeCustomerCreator) Execute(_ context.Context, in appcustomers.CreateCustomerInput) (*customersDomain.Customer, error) {
	f.in = in
	f.called = true
	if f.err != nil {
		return nil, f.err
	}
	if f.out != nil {
		return f.out, nil
	}
	return &customersDomain.Customer{ID: "new-cust-1", FirstName: in.FirstName, LastName: in.LastName, Email: in.Email, PhoneE164: in.PhoneE164}, nil
}
