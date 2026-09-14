package messagecenter_test

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/domain/messaging"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/crypto"
)

const testEncryptionSecret = "test-encryption-secret"

func strPtr(s string) *string { return &s }

func encryptedSecret(t *testing.T, plaintext string) string {
	t.Helper()
	enc, err := crypto.Encrypt(testEncryptionSecret, plaintext)
	require.NoError(t, err)
	return enc
}

func whatsappConnection(t *testing.T) integrationsDomain.Connection {
	return integrationsDomain.Connection{
		ID: "conn-1", Provider: integrationsDomain.ProviderWhatsApp,
		StoreURL: "https://evolution.example.com", ConsumerKey: "tenant-1-instance",
		ConsumerSecretEncrypted: encryptedSecret(t, "api-key-plain"),
	}
}

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

type stubMessageTemplateRepo struct {
	created   []mcDomain.MessageTemplate
	updated   []mcDomain.MessageTemplate
	deletedID string
	byID      map[string]mcDomain.MessageTemplate
	all       []mcDomain.MessageTemplate
	findErr   error
}

func (s *stubMessageTemplateRepo) Create(_ context.Context, t *mcDomain.MessageTemplate) error {
	t.ID = "template-1"
	s.created = append(s.created, *t)
	return nil
}

func (s *stubMessageTemplateRepo) FindByID(_ context.Context, id string) (*mcDomain.MessageTemplate, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	tmpl, ok := s.byID[id]
	if !ok {
		return nil, mcDomain.ErrTemplateNotFound
	}
	return &tmpl, nil
}

func (s *stubMessageTemplateRepo) ListByTenant(_ context.Context) ([]mcDomain.MessageTemplate, error) {
	return s.all, nil
}

func (s *stubMessageTemplateRepo) Update(_ context.Context, t *mcDomain.MessageTemplate) error {
	s.updated = append(s.updated, *t)
	return nil
}

func (s *stubMessageTemplateRepo) Delete(_ context.Context, id string) error {
	s.deletedID = id
	return nil
}

type stubAutomationRuleRepo struct {
	created   []mcDomain.AutomationRule
	updated   []mcDomain.AutomationRule
	deletedID string
	byID      map[string]mcDomain.AutomationRule
	all       []mcDomain.AutomationRule
	bySubject []mcDomain.AutomationRule
}

func (s *stubAutomationRuleRepo) Create(_ context.Context, r *mcDomain.AutomationRule) error {
	r.ID = "rule-1"
	s.created = append(s.created, *r)
	return nil
}

func (s *stubAutomationRuleRepo) FindByID(_ context.Context, id string) (*mcDomain.AutomationRule, error) {
	rule, ok := s.byID[id]
	if !ok {
		return nil, mcDomain.ErrRuleNotFound
	}
	return &rule, nil
}

func (s *stubAutomationRuleRepo) ListByTenant(_ context.Context) ([]mcDomain.AutomationRule, error) {
	return s.all, nil
}

func (s *stubAutomationRuleRepo) ListActiveBySubject(_ context.Context, _ string) ([]mcDomain.AutomationRule, error) {
	return s.bySubject, nil
}

func (s *stubAutomationRuleRepo) Update(_ context.Context, r *mcDomain.AutomationRule) error {
	s.updated = append(s.updated, *r)
	return nil
}

func (s *stubAutomationRuleRepo) Delete(_ context.Context, id string) error {
	s.deletedID = id
	return nil
}

type fakeSalesOrderRepo struct {
	byID map[string]omsDomain.SalesOrder
}

func (f *fakeSalesOrderRepo) FindByID(_ context.Context, id string) (*omsDomain.SalesOrder, error) {
	if o, ok := f.byID[id]; ok {
		return &o, nil
	}
	return nil, omsDomain.ErrSalesOrderNotFound
}

type fakeCustomerRepo struct {
	byID map[string]customersDomain.Customer
}

func (f *fakeCustomerRepo) FindByID(_ context.Context, id string) (*customersDomain.Customer, error) {
	if c, ok := f.byID[id]; ok {
		return &c, nil
	}
	return nil, customersDomain.ErrCustomerNotFound
}

type fakeConnectionRepo struct {
	conns []integrationsDomain.Connection
}

func (f *fakeConnectionRepo) ListByTenant(_ context.Context) ([]integrationsDomain.Connection, error) {
	return f.conns, nil
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

type fakeEmailClient struct {
	sent []messaging.Message
	err  error
}

func (f *fakeEmailClient) Send(_ context.Context, msg messaging.Message) error {
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, msg)
	return nil
}
