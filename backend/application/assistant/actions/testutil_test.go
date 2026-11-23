package actions_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	appquotes "github.com/adirkuhn/commerceos/application/quotes"
	"github.com/adirkuhn/commerceos/domain/assistant"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/domain/quotes"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

var now = time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)

func clock() time.Time { return now }

func userCtx() context.Context {
	return tenant.WithUserID(tenant.WithTenantID(context.Background(), "t-1"), "u-1")
}

func ptr[T any](v T) *T { return &v }

type stubPrices struct {
	out *apppms.ListPriceListResult
	err error
	got apppms.ListPriceListInput
}

func (s *stubPrices) Execute(_ context.Context, in apppms.ListPriceListInput) (*apppms.ListPriceListResult, error) {
	s.got = in
	return s.out, s.err
}

type stubCustomers struct {
	byID map[string]*customersDomain.Customer
}

func (s *stubCustomers) Execute(_ context.Context, id string) (*customersDomain.Customer, error) {
	c, ok := s.byID[id]
	if !ok {
		return nil, customersDomain.ErrCustomerNotFound
	}
	return c, nil
}

type stubCreateQuote struct {
	got *quotes.Input
	err error
}

func (s *stubCreateQuote) Execute(_ context.Context, in quotes.Input) (*quotes.Quote, error) {
	s.got = &in
	if s.err != nil {
		return nil, s.err
	}
	return &quotes.Quote{ID: "q-1", Number: 12, Name: in.Name}, nil
}

type stubGetQuote struct {
	byID map[string]*quotes.Quote
}

func (s *stubGetQuote) Execute(_ context.Context, id string) (*quotes.Quote, error) {
	q, ok := s.byID[id]
	if !ok {
		return nil, quotes.ErrQuoteNotFound
	}
	return q, nil
}

type stubWarehouses struct {
	out []invDomain.Warehouse
	err error
}

func (s *stubWarehouses) Execute(context.Context) ([]invDomain.Warehouse, error) { return s.out, s.err }

type stubStock struct {
	out                 *appquotes.StockCheck
	err                 error
	gotQuote, gotWhouse string
}

func (s *stubStock) Execute(_ context.Context, quoteID, warehouseID string) (*appquotes.StockCheck, error) {
	s.gotQuote, s.gotWhouse = quoteID, warehouseID
	return s.out, s.err
}

type stubConvert struct {
	got *appquotes.ConvertQuoteInput
	err error
}

func (s *stubConvert) Execute(_ context.Context, in appquotes.ConvertQuoteInput) (*appquotes.ConvertQuoteResult, error) {
	s.got = &in
	if s.err != nil {
		return nil, s.err
	}
	return &appquotes.ConvertQuoteResult{SalesOrderID: "so-1", Number: "CO-31"}, nil
}

// assertStrictSchema checks a tool definition is a closed JSON-schema object
// naming its required fields.
func assertStrictSchema(t *testing.T, d assistant.ToolDefinition, required ...string) {
	t.Helper()
	assert.NotEmpty(t, d.Description)
	assert.Equal(t, "object", d.Parameters["type"])
	assert.Equal(t, false, d.Parameters["additionalProperties"])
	assert.ElementsMatch(t, required, d.Parameters["required"])
	_, err := json.Marshal(d.Parameters)
	require.NoError(t, err)
}

func decodeJSON(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m))
	return m
}
