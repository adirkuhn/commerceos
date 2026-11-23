package tools_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/assistant/tools"
	appquotes "github.com/adirkuhn/commerceos/application/quotes"
	"github.com/adirkuhn/commerceos/domain/quotes"
)

type stubGetQuote struct {
	out   *quotes.Quote
	err   error
	gotID string
}

func (s *stubGetQuote) Execute(_ context.Context, id string) (*quotes.Quote, error) {
	s.gotID = id
	return s.out, s.err
}

type stubListQuotes struct {
	out *appquotes.ListQuotesResult
	err error
	got appquotes.ListQuotesInput
}

func (s *stubListQuotes) Execute(_ context.Context, in appquotes.ListQuotesInput) (*appquotes.ListQuotesResult, error) {
	s.got = in
	return s.out, s.err
}

func TestGetQuoteTool(t *testing.T) {
	q := &quotes.Quote{ID: "q-1", Number: 3, Name: "Restaurant Sabor", CurrencyCode: "EUR", DiscountBasisPoints: 1000,
		Customer:  &quotes.CustomerRef{ID: "c-1", Name: "Sabor BV", Email: "inkoop@sabor.test"},
		Lines:     []quotes.Line{{ProductID: "p-1", SKUID: "s-1", Quantity: 10, UnitPriceMinorUnits: 900}},
		UpdatedAt: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)}
	get := &stubGetQuote{out: q}
	tool := tools.NewGetQuoteTool(get, &stubListQuotes{})
	assertStrictDefinition(t, tool, "get_quote")

	got := call(t, tool, `{"quote":"`+quoteUUID+`"}`)

	assert.Equal(t, quoteUUID, get.gotID)
	assert.EqualValues(t, 3, got["number"])
	assert.Equal(t, "/quotes/q-1", got["admin_path"])
	assert.Equal(t, "/customers/c-1", got["customer"].(map[string]any)["admin_path"])
	assert.Equal(t, "Sabor BV", got["customer"].(map[string]any)["name"])
	assert.EqualValues(t, q.Total(), got["total_minor_units"])
	assert.EqualValues(t, 1000, got["discount_basis_points"])
	lines := got["lines"].([]any)
	require.Len(t, lines, 1)
	assert.EqualValues(t, 900, lines[0].(map[string]any)["unit_price_minor_units"])
}

const quoteUUID = "3c59dc04-8c3e-4c2f-9a1b-6f0e2d1c7b55"

func TestGetQuoteTool_byReference(t *testing.T) {
	for _, ref := range []string{"Q-0012", "q12", "12", " #Q-12 "} {
		get := &stubGetQuote{out: &quotes.Quote{ID: "q-12", Number: 12}}
		list := &stubListQuotes{out: &appquotes.ListQuotesResult{Quotes: []quotes.Summary{{ID: "q-120", Number: 120}, {ID: "q-12", Number: 12}}}}

		got := call(t, tools.NewGetQuoteTool(get, list), `{"quote":"`+ref+`"}`)

		assert.Equal(t, "q-12", get.gotID, ref)
		assert.Equal(t, "12", list.got.Search, "searches by the bare number")
		assert.EqualValues(t, 12, got["number"])
		assert.Equal(t, "Q-0012", got["reference"])
	}
}

func TestGetQuoteTool_unknownReference(t *testing.T) {
	list := &stubListQuotes{out: &appquotes.ListQuotesResult{Quotes: []quotes.Summary{{ID: "q-120", Number: 120}}}}

	err := callErr(t, tools.NewGetQuoteTool(&stubGetQuote{}, list), `{"quote":"Q-0012"}`)

	assert.ErrorContains(t, err, "no quote Q-0012")
	assert.ErrorContains(t, err, "Q-0120")
}

func TestGetQuoteTool_errors(t *testing.T) {
	assert.ErrorIs(t, callErr(t, tools.NewGetQuoteTool(&stubGetQuote{}, &stubListQuotes{}), `{}`), tools.ErrInvalidArguments)
	assert.ErrorIs(t, callErr(t, tools.NewGetQuoteTool(&stubGetQuote{}, &stubListQuotes{}), `{"quote":"ana bakery"}`), tools.ErrInvalidArguments,
		"a name isn't a reference -- list_quotes searches names")
	assert.ErrorIs(t, callErr(t, tools.NewGetQuoteTool(&stubGetQuote{err: quotes.ErrQuoteNotFound}, &stubListQuotes{}), `{"quote":"`+quoteUUID+`"}`), quotes.ErrQuoteNotFound)
	assert.ErrorContains(t, callErr(t, tools.NewGetQuoteTool(&stubGetQuote{}, &stubListQuotes{err: errors.New("db down")}), `{"quote":"7"}`), "db down")
}

func TestListQuotesTool(t *testing.T) {
	list := &stubListQuotes{out: &appquotes.ListQuotesResult{Total: 1, Quotes: []quotes.Summary{
		{ID: "q-1", Number: 3, Name: "Sabor", CurrencyCode: "EUR", LineCount: 2, TotalMinorUnits: 9000, Customer: &quotes.CustomerRef{Name: "Sabor BV"}},
	}}}
	tool := tools.NewListQuotesTool(list)
	assertStrictDefinition(t, tool, "list_quotes")

	got := call(t, tool, `{"search":"sabor","limit":5}`)

	assert.Equal(t, appquotes.ListQuotesInput{Page: 1, PerPage: 5, Search: "sabor"}, list.got)
	assert.EqualValues(t, 1, got["total"])
	first := got["quotes"].([]any)[0].(map[string]any)
	assert.Equal(t, "Sabor BV", first["customer_name"])
	assert.Equal(t, "/quotes/q-1", first["admin_path"])
	assert.EqualValues(t, 9000, first["total_minor_units"])

	assert.ErrorContains(t, callErr(t, tools.NewListQuotesTool(&stubListQuotes{err: errors.New("db down")}), `{}`), "db down")
	assert.ErrorIs(t, callErr(t, tool, `{"limit":"x"}`), tools.ErrInvalidArguments)
}
