package tools_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/assistant/tools"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

type stubListCustomers struct {
	out []customersDomain.Customer
	err error
}

func (s *stubListCustomers) Execute(context.Context) ([]customersDomain.Customer, error) {
	return s.out, s.err
}

type stubGetCustomer struct {
	out   *customersDomain.Customer
	err   error
	gotID string
}

func (s *stubGetCustomer) Execute(_ context.Context, id string) (*customersDomain.Customer, error) {
	s.gotID = id
	return s.out, s.err
}

type stubContactPersons struct {
	out []customersDomain.ContactPerson
	err error
}

func (s *stubContactPersons) Execute(context.Context, string) ([]customersDomain.ContactPerson, error) {
	return s.out, s.err
}

func customerFixtures() []customersDomain.Customer {
	return []customersDomain.Customer{
		{ID: "c-1", Type: customersDomain.TypeCompany, CompanyName: "Padaria São João", Email: "compras@saojoao.test", PhoneE164: "+31612345678", ClientNumber: "K12", VATNumber: "NL123B01"},
		{ID: "c-2", Type: customersDomain.TypePrivate, FirstName: "Ana", LastName: "Souza", Email: "ana@x.test"},
		{ID: "c-3", Type: customersDomain.TypePrivate, FirstName: "Bob", LastName: "Jansen", Email: "bob@y.test"},
	}
}

func TestSearchCustomersTool_matchesNameCompanyEmailPhoneAndIgnoresAccents(t *testing.T) {
	tool := tools.NewSearchCustomersTool(&stubListCustomers{out: customerFixtures()})
	assertStrictDefinition(t, tool, "search_customers")

	for query, want := range map[string]string{
		"sao joao": "c-1", "NL123B01": "c-1", "k12": "c-1", "0612345678": "c-1", "ana@x": "c-2", "jansen": "c-3",
	} {
		got := call(t, tool, `{"query":"`+query+`"}`)
		list := got["customers"].([]any)
		require.Len(t, list, 1, query)
		assert.Equal(t, want, list[0].(map[string]any)["id"], query)
	}
}

func TestSearchCustomersTool_limitAndErrors(t *testing.T) {
	tool := tools.NewSearchCustomersTool(&stubListCustomers{out: customerFixtures()})

	got := call(t, tool, `{"query":"test","limit":2}`)
	assert.Len(t, got["customers"], 2)
	assert.EqualValues(t, 3, got["total_matches"])

	assert.ErrorIs(t, callErr(t, tool, `{"query":" "}`), tools.ErrInvalidArguments)
	assert.ErrorContains(t, callErr(t, tools.NewSearchCustomersTool(&stubListCustomers{err: errors.New("db down")}), `{"query":"x"}`), "db down")
}

func TestGetCustomerTool(t *testing.T) {
	c := customerFixtures()[0]
	c.Billing = customersDomain.Address{Address1: "Rua 1", Postcode: "1011AA", City: "Amsterdam", Country: "NL"}
	c.CustomerNote = "Deliver before 9"
	c.KVKNumber = "12345678"
	c.DateCreated = time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	last := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	orders := &stubListOrders{out: &appoms.ListSalesOrdersResult{Total: 7, SalesOrders: []omsDomain.SalesOrder{{ID: "o-1", Number: "CO-40", OccurredAt: last}}}}
	persons := &stubContactPersons{out: []customersDomain.ContactPerson{{FirstName: "Rita", LastName: "Lima", Role: "Purchasing", Email: "rita@saojoao.test"}}}
	get := &stubGetCustomer{out: &c}
	tool := tools.NewGetCustomerTool(get, persons, orders)
	assertStrictDefinition(t, tool, "get_customer")

	got := call(t, tool, `{"customer_id":"c-1"}`)

	assert.Equal(t, "c-1", get.gotID)
	assert.Equal(t, "Padaria São João", got["name"])
	assert.Equal(t, "company", got["type"])
	assert.Equal(t, "NL123B01", got["vat_number"])
	assert.Equal(t, "12345678", got["kvk_number"])
	assert.Equal(t, "Rua 1, 1011AA Amsterdam, NL", got["billing_address"])
	assert.Equal(t, "Deliver before 9", got["note"])
	assert.EqualValues(t, 7, got["order_count"])
	assert.Equal(t, "CO-40", got["last_order"].(map[string]any)["number"])
	assert.Equal(t, "2026-11-20T00:00:00Z", got["last_order"].(map[string]any)["date"])
	assert.Equal(t, "Rita Lima", got["contact_persons"].([]any)[0].(map[string]any)["name"])
	require.NotNil(t, orders.got.CustomerID)
	assert.Equal(t, "c-1", *orders.got.CustomerID)
	assert.Equal(t, 1, orders.got.PerPage)
}

func TestGetCustomerTool_errors(t *testing.T) {
	c := customerFixtures()[0]
	ok := func() (*stubGetCustomer, *stubContactPersons, *stubListOrders) {
		return &stubGetCustomer{out: &c}, &stubContactPersons{}, &stubListOrders{out: &appoms.ListSalesOrdersResult{}}
	}
	g, p, o := ok()
	assert.ErrorIs(t, callErr(t, tools.NewGetCustomerTool(g, p, o), `{}`), tools.ErrInvalidArguments)

	g.err = customersDomain.ErrCustomerNotFound
	assert.ErrorIs(t, callErr(t, tools.NewGetCustomerTool(g, p, o), `{"customer_id":"x"}`), customersDomain.ErrCustomerNotFound)

	g, p, o = ok()
	p.err = errors.New("persons down")
	assert.ErrorContains(t, callErr(t, tools.NewGetCustomerTool(g, p, o), `{"customer_id":"c-1"}`), "persons down")

	g, p, o = ok()
	o.err = errors.New("orders down")
	assert.ErrorContains(t, callErr(t, tools.NewGetCustomerTool(g, p, o), `{"customer_id":"c-1"}`), "orders down")
}
