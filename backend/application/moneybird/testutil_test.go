package moneybird_test

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	mb "github.com/adirkuhn/commerceos/domain/moneybird"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

const testSecret = "0123456789abcdef0123456789abcdef"

func tenantCtx() context.Context { return tenant.WithTenantID(context.Background(), "tenant-1") }

// mockDB accepts up to n tenant.WithTenant transactions, in any order.
func mockDB(t *testing.T, n int) *gorm.DB {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { sqlDB.Close() })
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	require.NoError(t, err)
	mock.MatchExpectationsInOrder(false)
	for i := 0; i < n; i++ {
		mock.ExpectBegin()
		mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()
	}
	return db
}

// ── connections ──────────────────────────────────────────────────────────────

type fakeConnections struct {
	conns   []integrationsDomain.Connection
	created []integrationsDomain.Connection
	updated []integrationsDomain.Connection
}

func (f *fakeConnections) Create(_ context.Context, c *integrationsDomain.Connection) error {
	c.ID = "conn-new"
	f.created = append(f.created, *c)
	f.conns = append(f.conns, *c)
	return nil
}
func (f *fakeConnections) ListByTenant(_ context.Context) ([]integrationsDomain.Connection, error) {
	return f.conns, nil
}
func (f *fakeConnections) Update(_ context.Context, c *integrationsDomain.Connection) error {
	f.updated = append(f.updated, *c)
	return nil
}

func connected(t *testing.T, token string) *fakeConnections {
	t.Helper()
	enc, err := crypto.Encrypt(testSecret, token)
	require.NoError(t, err)
	return &fakeConnections{conns: []integrationsDomain.Connection{
		{ID: "conn-woo", TenantID: "tenant-1", Provider: integrationsDomain.ProviderWooCommerce},
		{ID: "conn-mb", TenantID: "tenant-1", Provider: integrationsDomain.ProviderMoneybird, ConsumerSecretEncrypted: enc, Status: integrationsDomain.StatusConnected},
	}}
}

// ── settings / documents / contacts ──────────────────────────────────────────

type fakeSettings struct {
	byConn  map[string]mb.Settings
	upserts []mb.Settings
}

func newFakeSettings(s ...mb.Settings) *fakeSettings {
	f := &fakeSettings{byConn: map[string]mb.Settings{}}
	for _, x := range s {
		f.byConn[x.ConnectionID] = x
	}
	return f
}
func (f *fakeSettings) Get(_ context.Context, id string) (*mb.Settings, error) {
	s, ok := f.byConn[id]
	if !ok {
		return nil, nil
	}
	return &s, nil
}
func (f *fakeSettings) Upsert(_ context.Context, s *mb.Settings) error {
	f.byConn[s.ConnectionID] = *s
	f.upserts = append(f.upserts, *s)
	return nil
}

type fakeDocuments struct {
	byKey map[string]mb.Document
	saved []mb.Document
}

func newFakeDocuments(docs ...mb.Document) *fakeDocuments {
	f := &fakeDocuments{byKey: map[string]mb.Document{}}
	for _, d := range docs {
		f.byKey[string(d.SourceType)+"/"+d.SourceID] = d
	}
	return f
}
func (f *fakeDocuments) Find(_ context.Context, t mb.SourceType, id string) (*mb.Document, error) {
	d, ok := f.byKey[string(t)+"/"+id]
	if !ok {
		return nil, nil
	}
	return &d, nil
}
func (f *fakeDocuments) Save(_ context.Context, d *mb.Document) error {
	f.byKey[string(d.SourceType)+"/"+d.SourceID] = *d
	f.saved = append(f.saved, *d)
	return nil
}
func (f *fakeDocuments) last() mb.Document { return f.saved[len(f.saved)-1] }

type fakeContacts struct{ byCustomer map[string]string }

func (f *fakeContacts) Find(_ context.Context, _, customerID string) (string, error) {
	return f.byCustomer[customerID], nil
}
func (f *fakeContacts) Save(_ context.Context, _, _, customerID, id string) error {
	if f.byCustomer == nil {
		f.byCustomer = map[string]string{}
	}
	f.byCustomer[customerID] = id
	return nil
}

// ── Moneybird client ─────────────────────────────────────────────────────────

type fakeClient struct {
	administrations []mb.Administration
	taxRates        []mb.TaxRate
	ledgers         []mb.LedgerAccount
	err             error

	contacts         map[string]mb.Contact // by id
	createdContacts  []mb.Contact
	existingInvoice  *mb.RemoteDocument
	onFindReference  func()
	invoices         map[string]mb.RemoteDocument
	createdInvoices  []mb.Invoice
	createInvoiceErr error
	payments         []mb.Payment
	paymentErr       error
	estimates        map[string]mb.RemoteDocument
	createdEstimates []mb.Estimate
	updatedEstimates []mb.Estimate
}

func (c *fakeClient) ListAdministrations(context.Context) ([]mb.Administration, error) {
	return c.administrations, c.err
}
func (c *fakeClient) ListSalesTaxRates(context.Context) ([]mb.TaxRate, error) { return c.taxRates, c.err }
func (c *fakeClient) ListRevenueLedgerAccounts(context.Context) ([]mb.LedgerAccount, error) {
	return c.ledgers, c.err
}
func (c *fakeClient) GetContact(_ context.Context, id string) (*mb.Contact, error) {
	ct, ok := c.contacts[id]
	if !ok {
		return nil, mb.ErrRequestFailed
	}
	return &ct, nil
}
func (c *fakeClient) FindContact(_ context.Context, q string) (*mb.Contact, error) {
	for _, ct := range c.contacts {
		if ct.Email == q || ct.CompanyName == q {
			return &ct, nil
		}
	}
	return nil, nil
}
func (c *fakeClient) CreateContact(_ context.Context, ct mb.Contact) (*mb.Contact, error) {
	ct.ID = "mb-contact-new"
	c.createdContacts = append(c.createdContacts, ct)
	if c.contacts == nil {
		c.contacts = map[string]mb.Contact{}
	}
	c.contacts[ct.ID] = ct
	return &ct, nil
}
func (c *fakeClient) FindInvoiceByReference(context.Context, string) (*mb.RemoteDocument, error) {
	if c.onFindReference != nil {
		c.onFindReference()
	}
	return c.existingInvoice, nil
}
func (c *fakeClient) GetInvoice(_ context.Context, id string) (*mb.RemoteDocument, error) {
	d, ok := c.invoices[id]
	if !ok {
		return nil, mb.ErrRequestFailed
	}
	return &d, nil
}
func (c *fakeClient) CreateInvoice(_ context.Context, inv mb.Invoice) (*mb.RemoteDocument, error) {
	if c.createInvoiceErr != nil {
		return nil, c.createInvoiceErr
	}
	c.createdInvoices = append(c.createdInvoices, inv)
	return &mb.RemoteDocument{ID: "inv-new", State: "open", TotalUnpaid: "12.11", TotalInclTaxBase: "12.11"}, nil
}
func (c *fakeClient) RegisterPayment(_ context.Context, _ string, p mb.Payment) error {
	if c.paymentErr != nil {
		return c.paymentErr
	}
	c.payments = append(c.payments, p)
	return nil
}
func (c *fakeClient) GetEstimate(_ context.Context, id string) (*mb.RemoteDocument, error) {
	d, ok := c.estimates[id]
	if !ok {
		return nil, mb.ErrRequestFailed
	}
	return &d, nil
}
func (c *fakeClient) CreateEstimate(_ context.Context, e mb.Estimate) (*mb.RemoteDocument, error) {
	c.createdEstimates = append(c.createdEstimates, e)
	return &mb.RemoteDocument{ID: "est-new", State: "draft"}, nil
}
func (c *fakeClient) UpdateEstimate(_ context.Context, id string, e mb.Estimate) (*mb.RemoteDocument, error) {
	c.updatedEstimates = append(c.updatedEstimates, e)
	return &mb.RemoteDocument{ID: id, State: "draft"}, nil
}

type fakeFactory struct {
	client          *fakeClient
	tokens, adminIDs []string
}

func (f *fakeFactory) New(token, adminID string) mb.Client {
	f.tokens = append(f.tokens, token)
	f.adminIDs = append(f.adminIDs, adminID)
	return f.client
}

func configuredSettings() mb.Settings {
	return mb.Settings{
		ConnectionID: "conn-mb", TenantID: "tenant-1", AdministrationID: "adm-1", AdministrationName: "Cantinho",
		DefaultLedgerAccountID: "led-1",
		TaxRates: []mb.TaxRateMapping{
			{RateBasisPoints: 2100, MoneybirdTaxRateID: "tr-21"}, {RateBasisPoints: 900, MoneybirdTaxRateID: "tr-9"},
		},
		PaymentMethods: []mb.PaymentMethodMapping{{Method: "bacs", MoneybirdPaymentMethod: "bank_transfer"}},
	}
}
