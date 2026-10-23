package quotes_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appquotes "github.com/adirkuhn/commerceos/application/quotes"
	"github.com/adirkuhn/commerceos/domain/quotes"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func ctxFor() context.Context {
	return tenant.WithUserID(tenant.WithTenantID(context.Background(), "t-1"), "user-1")
}

func input() quotes.Input {
	return quotes.Input{Name: "Restaurant Sabor", DiscountBasisPoints: 3000, CurrencyCode: "EUR",
		Lines: []quotes.Line{{ProductID: "p-1", SKUID: "s-1", Quantity: 100, UnitPriceMinorUnits: 871}}}
}

func saved() *quotes.Quote {
	created := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	return &quotes.Quote{ID: "q-1", TenantID: "t-1", Number: 3, Name: "Old name", DiscountBasisPoints: 1000, CurrencyCode: "EUR",
		Lines:     []quotes.Line{{ProductID: "p-1", SKUID: "s-1", Quantity: 10, UnitPriceMinorUnits: 900}},
		CreatedBy: "user-0", CreatedAt: created, UpdatedAt: created}
}

func TestCreateQuote_storesNumberedQuoteAndRecordsEvent(t *testing.T) {
	repo, outbox := newStubRepo(), &eventrecordtest.Outbox{}

	q, err := appquotes.NewCreateQuoteUseCase(newMockGormDB(t), repo, outbox).Execute(ctxFor(), input())

	require.NoError(t, err)
	assert.Equal(t, "q-new", q.ID)
	assert.Equal(t, int64(12), q.Number)
	assert.Equal(t, "t-1", q.TenantID)
	assert.Equal(t, "user-1", q.CreatedBy)
	require.Len(t, repo.created, 1)
	event, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "quotes.quote.created", event.Subject)
	assert.Equal(t, map[string]any{"number": float64(12), "name": "Restaurant Sabor", "line_count": float64(1), "total_minor_units": float64(87100), "currency_code": "EUR"}, event.Data())
}

func TestCreateQuote_invalidInputStoresNothing(t *testing.T) {
	repo, outbox := newStubRepo(), &eventrecordtest.Outbox{}
	in := input()
	in.Name = ""

	_, err := appquotes.NewCreateQuoteUseCase(nil, repo, outbox).Execute(ctxFor(), in)

	assert.ErrorIs(t, err, quotes.ErrNameRequired)
	assert.Empty(t, repo.created)
	assert.Empty(t, outbox.Events)
}

func TestCreateQuote_wrapsRepoError(t *testing.T) {
	repo := newStubRepo()
	repo.err = errors.New("boom")

	_, err := appquotes.NewCreateQuoteUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{}).Execute(ctxFor(), input())

	assert.ErrorContains(t, err, "createQuote")
}

func TestUpdateQuote_savesOverAndRecordsChanges(t *testing.T) {
	repo, outbox := newStubRepo(saved()), &eventrecordtest.Outbox{}

	q, err := appquotes.NewUpdateQuoteUseCase(newMockGormDB(t), repo, outbox).Execute(ctxFor(), "q-1", input())

	require.NoError(t, err)
	assert.Equal(t, int64(3), q.Number)
	assert.Equal(t, "Restaurant Sabor", q.Name)
	assert.Equal(t, "user-0", q.CreatedBy, "the author stays")
	require.Len(t, repo.updated, 1)
	event, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "quotes.quote.updated", event.Subject)
	assert.Equal(t, map[string][2]any{
		"name":                  {"Old name", "Restaurant Sabor"},
		"discount_basis_points": {float64(1000), float64(3000)},
		"total_minor_units":     {float64(9000), float64(87100)},
	}, event.Changes())
}

func TestUpdateQuote_notFound(t *testing.T) {
	_, err := appquotes.NewUpdateQuoteUseCase(newMockGormDB(t), newStubRepo(), &eventrecordtest.Outbox{}).Execute(ctxFor(), "missing", input())

	assert.ErrorIs(t, err, quotes.ErrQuoteNotFound)
}

func TestUpdateQuote_invalidInput(t *testing.T) {
	repo := newStubRepo(saved())
	in := input()
	in.DiscountBasisPoints = 20000

	_, err := appquotes.NewUpdateQuoteUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{}).Execute(ctxFor(), "q-1", in)

	assert.ErrorIs(t, err, quotes.ErrInvalidDiscount)
	assert.Empty(t, repo.updated)
}

func TestDeleteQuote_deletesAndRecordsEvent(t *testing.T) {
	repo, outbox := newStubRepo(saved()), &eventrecordtest.Outbox{}

	err := appquotes.NewDeleteQuoteUseCase(newMockGormDB(t), repo, outbox).Execute(ctxFor(), "q-1")

	require.NoError(t, err)
	assert.Equal(t, []string{"q-1"}, repo.deleted)
	event, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "quotes.quote.deleted", event.Subject)
	assert.Equal(t, map[string]any{"number": float64(3), "name": "Old name"}, event.Data())
}

func TestDeleteQuote_notFound(t *testing.T) {
	err := appquotes.NewDeleteQuoteUseCase(newMockGormDB(t), newStubRepo(), &eventrecordtest.Outbox{}).Execute(ctxFor(), "missing")

	assert.ErrorIs(t, err, quotes.ErrQuoteNotFound)
}

func TestGetQuote(t *testing.T) {
	q, err := appquotes.NewGetQuoteUseCase(newMockGormDB(t), newStubRepo(saved())).Execute(ctxFor(), "q-1")

	require.NoError(t, err)
	assert.Equal(t, "Old name", q.Name)
}

func TestGetQuote_notFound(t *testing.T) {
	_, err := appquotes.NewGetQuoteUseCase(newMockGormDB(t), newStubRepo()).Execute(ctxFor(), "missing")

	assert.ErrorIs(t, err, quotes.ErrQuoteNotFound)
}

func TestListQuotes_pagesWithDefaultsAndCap(t *testing.T) {
	repo := newStubRepo()
	repo.listOut, repo.listTotal = []quotes.Summary{{ID: "q-1"}}, 61
	uc := appquotes.NewListQuotesUseCase(newMockGormDB(t), repo)

	out, err := uc.Execute(ctxFor(), appquotes.ListQuotesInput{Page: 3, PerPage: 500, Search: " sabor "})

	require.NoError(t, err)
	assert.Equal(t, int64(61), out.Total)
	assert.Len(t, out.Quotes, 1)
	assert.Equal(t, quotes.ListFilter{Search: "sabor", Limit: 200, Offset: 400}, repo.gotFilter)
}

func TestListQuotes_defaultPage(t *testing.T) {
	repo := newStubRepo()

	_, err := appquotes.NewListQuotesUseCase(newMockGormDB(t), repo).Execute(ctxFor(), appquotes.ListQuotesInput{})

	require.NoError(t, err)
	assert.Equal(t, quotes.ListFilter{Limit: 50, Offset: 0}, repo.gotFilter)
}
