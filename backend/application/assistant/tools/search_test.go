package tools_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/assistant/tools"
	appsearch "github.com/adirkuhn/commerceos/application/search"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
)

type stubSearch struct {
	out *appsearch.GlobalSearchResult
	err error
	got string
}

func (s *stubSearch) Execute(_ context.Context, q string) (*appsearch.GlobalSearchResult, error) {
	s.got = q
	return s.out, s.err
}

func TestSearchTool(t *testing.T) {
	s := &stubSearch{out: &appsearch.GlobalSearchResult{
		Products:    []pmsDomain.Product{{ID: "p 1/x", Name: "Doce de leite", Status: "active", Description: "long text"}},
		Customers:   []customersDomain.Customer{{ID: "c-1", FirstName: "Ana", LastName: "Souza", CompanyName: "Ana Bakery", Email: "ana@x.test", ClientNumber: "K12"}},
		Suppliers:   []suppliersDomain.Supplier{{ID: "s-1", Name: "Bauducco", Email: "orders@bauducco.example"}},
		SalesOrders: []omsDomain.SalesOrder{{ID: "o-1", Number: "CO-12", Status: omsDomain.SalesOrderStatusShipped, CurrencyCode: "EUR", TotalMinorUnits: 1299, Billing: omsDomain.Address{FirstName: "Ana", LastName: "Souza"}}},
	}}
	tool := tools.NewSearchTool(s)
	assertStrictDefinition(t, tool, "search")

	got := call(t, tool, `{"query":"ana"}`)

	assert.Equal(t, "ana", s.got)
	products := got["products"].([]any)
	require.Len(t, products, 1)
	assert.Equal(t, map[string]any{"id": "p 1/x", "name": "Doce de leite", "status": "active", "admin_path": "/products/p%201%2Fx"}, products[0],
		"the app builds the link, escaping the id")
	customers := got["customers"].([]any)
	assert.Equal(t, "Ana Bakery", customers[0].(map[string]any)["company_name"])
	assert.Equal(t, "Ana Souza", customers[0].(map[string]any)["name"])
	assert.Equal(t, "/customers/c-1", customers[0].(map[string]any)["admin_path"])
	assert.Equal(t, "/suppliers/s-1", got["suppliers"].([]any)[0].(map[string]any)["admin_path"])
	assert.Equal(t, "Bauducco", got["suppliers"].([]any)[0].(map[string]any)["name"])
	order := got["sales_orders"].([]any)[0].(map[string]any)
	assert.Equal(t, "CO-12", order["number"])
	assert.Equal(t, "/sales-orders/o-1", order["admin_path"])
	assert.EqualValues(t, 1299, order["total_minor_units"])
	assert.Equal(t, "Ana Souza", order["customer_name"])
}

func TestSearchTool_requiresQuery(t *testing.T) {
	err := callErr(t, tools.NewSearchTool(&stubSearch{}), `{"query":" "}`)
	assert.ErrorIs(t, err, tools.ErrInvalidArguments)

	err = callErr(t, tools.NewSearchTool(&stubSearch{}), `{"q":"x"}`)
	assert.ErrorIs(t, err, tools.ErrInvalidArguments, "unknown fields are rejected")
}

func TestSearchTool_useCaseError(t *testing.T) {
	err := callErr(t, tools.NewSearchTool(&stubSearch{err: errors.New("db down")}), `{"query":"x"}`)
	assert.ErrorContains(t, err, "db down")
}
