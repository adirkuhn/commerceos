package moneybird_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appmb "github.com/adirkuhn/commerceos/application/moneybird"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	mb "github.com/adirkuhn/commerceos/domain/moneybird"
	"github.com/adirkuhn/commerceos/domain/quotes"
)

type fakeQuotes struct{ q quotes.Quote }

func (f *fakeQuotes) FindByID(_ context.Context, id string) (*quotes.Quote, error) {
	if id != f.q.ID {
		return nil, quotes.ErrQuoteNotFound
	}
	q := f.q
	return &q, nil
}

type fakeCatalog struct {
	result apppms.ListPriceListResult
	asked  apppms.ListPriceListInput
}

func (f *fakeCatalog) Execute(_ context.Context, in apppms.ListPriceListInput) (*apppms.ListPriceListResult, error) {
	f.asked = in
	return &f.result, nil
}

func quoteFixture() (*fakeQuotes, *fakeCatalog) {
	q := quotes.Quote{
		ID: "q-1", Number: 7, Name: "Mercado Lusitano", CurrencyCode: "EUR",
		Customer:  &quotes.CustomerRef{ID: "cust-1", Name: "Mercado Lusitano B.V."},
		CreatedAt: time.Date(2026, 10, 27, 10, 0, 0, 0, time.UTC),
		Lines: []quotes.Line{
			{ProductID: "p-farofa", SKUID: "sku-farofa", Quantity: 24, UnitPriceMinorUnits: 299},
			{ProductID: "p-carne", SKUID: "sku-carne", Quantity: 0, UnitPriceMinorUnits: 2899}, // not ordered
			{ProductID: "p-guarana", SKUID: "sku-guarana", Quantity: 12, UnitPriceMinorUnits: 129},
		},
	}
	catalog := &fakeCatalog{result: apppms.ListPriceListResult{PricesIncludeTax: true, Rows: []apppms.PriceListRow{
		{ProductID: "p-farofa", ProductName: "Farofa Yoki", VariantName: "250g", SKUID: "sku-farofa", TaxRateBasisPoints: 900},
		{ProductID: "p-carne", ProductName: "Carne Seca", SKUID: "sku-carne", TaxRateBasisPoints: 900},
		{ProductID: "p-guarana", ProductName: "Guaraná Antarctica", SKUID: "sku-guarana", TaxRateBasisPoints: 900},
	}}}
	return &fakeQuotes{q: q}, catalog
}

type quoteHarness struct {
	client *fakeClient
	docs   *fakeDocuments
	outbox *eventrecordtest.Outbox
	uc     *appmb.SendQuoteUseCase
}

func newQuoteHarness(t *testing.T, q *fakeQuotes, catalog *fakeCatalog, docs ...mb.Document) *quoteHarness {
	h := &quoteHarness{client: &fakeClient{}, docs: newFakeDocuments(docs...), outbox: &eventrecordtest.Outbox{}}
	customers := &fakeCustomers{byID: map[string]customersDomain.Customer{
		"cust-1": {ID: "cust-1", CompanyName: "Mercado Lusitano B.V.", Email: "inkoop@mercado.nl", Billing: customersDomain.Address{Address1: "Kerkstraat 10", City: "Amsterdam", Country: "NL"}},
	}}
	h.uc = appmb.NewSendQuoteUseCase(mockDB(t, 8), appmb.SendQuoteDeps{
		Connections: connected(t, "tok"), Settings: newFakeSettings(configuredSettings()), Documents: h.docs, Contacts: &fakeContacts{},
		Quotes: q, Catalog: catalog, Customers: customers, Factory: &fakeFactory{client: h.client}, Outbox: h.outbox, Secret: testSecret,
	})
	return h
}

func TestSendQuote_createsAnEstimateOfTheQuotedLines(t *testing.T) {
	q, catalog := quoteFixture()
	h := newQuoteHarness(t, q, catalog)

	doc, err := h.uc.Execute(tenantCtx(), "q-1")

	require.NoError(t, err)
	require.Len(t, h.client.createdEstimates, 1)
	est := h.client.createdEstimates[0]
	assert.Equal(t, "Q-0007", est.Reference)
	assert.Equal(t, "2026-10-27", est.Date)
	assert.True(t, est.PricesInclTax)
	assert.Equal(t, []mb.DocumentLine{
		{Description: "Farofa Yoki - 250g", Amount: "24", PriceMinorUnits: 299, TaxRateID: "tr-9", LedgerAccountID: "led-1"},
		{Description: "Guaraná Antarctica", Amount: "12", PriceMinorUnits: 129, TaxRateID: "tr-9", LedgerAccountID: "led-1"},
	}, est.Lines)
	assert.Equal(t, []string{"p-farofa", "p-guarana"}, catalog.asked.ProductIDs)
	require.Len(t, h.client.createdContacts, 1)
	assert.Equal(t, "inkoop@mercado.nl", h.client.createdContacts[0].Email)

	assert.Equal(t, mb.DocumentSynced, doc.Status)
	assert.Equal(t, "est-new", doc.MoneybirdID)
	assert.Equal(t, []string{"integrations.moneybird_document.synced"}, h.outbox.Subjects())
}

// The quote changed after sending: the draft estimate is updated, not duplicated.
func TestSendQuote_againUpdatesTheDraftEstimate(t *testing.T) {
	q, catalog := quoteFixture()
	sent := mb.Document{SourceType: mb.SourceQuote, SourceID: "q-1", MoneybirdID: "est-1", Status: mb.DocumentSynced}
	h := newQuoteHarness(t, q, catalog, sent)
	h.client.estimates = map[string]mb.RemoteDocument{"est-1": {ID: "est-1", State: "draft"}}

	doc, err := h.uc.Execute(tenantCtx(), "q-1")

	require.NoError(t, err)
	assert.Empty(t, h.client.createdEstimates)
	require.Len(t, h.client.updatedEstimates, 1)
	assert.Len(t, h.client.updatedEstimates[0].Lines, 2)
	assert.Equal(t, "est-1", doc.MoneybirdID)
}

// Sent to the customer from Moneybird: it can't be changed from here any more.
func TestSendQuote_sentEstimateIsLocked(t *testing.T) {
	q, catalog := quoteFixture()
	sent := mb.Document{SourceType: mb.SourceQuote, SourceID: "q-1", MoneybirdID: "est-1", Status: mb.DocumentSynced}
	h := newQuoteHarness(t, q, catalog, sent)
	h.client.estimates = map[string]mb.RemoteDocument{"est-1": {ID: "est-1", State: "open"}}

	_, err := h.uc.Execute(tenantCtx(), "q-1")

	assert.True(t, errors.Is(err, mb.ErrDocumentLocked))
	assert.Empty(t, h.client.updatedEstimates)
	assert.Equal(t, mb.DocumentSynced, h.docs.byKey["quote/q-1"].Status, "the estimate itself is fine -- not marked failed")
}

func TestSendQuote_noOrderedLines(t *testing.T) {
	q, catalog := quoteFixture()
	for i := range q.q.Lines {
		q.q.Lines[i].Quantity = 0
	}
	h := newQuoteHarness(t, q, catalog)

	_, err := h.uc.Execute(tenantCtx(), "q-1")

	assert.True(t, errors.Is(err, mb.ErrNothingToBook))
	assert.Empty(t, h.client.createdEstimates)
}

func TestSendQuote_unmappedRate(t *testing.T) {
	q, catalog := quoteFixture()
	catalog.result.Rows[0].TaxRateBasisPoints = 600
	h := newQuoteHarness(t, q, catalog)

	_, err := h.uc.Execute(tenantCtx(), "q-1")

	assert.True(t, errors.Is(err, mb.ErrTaxRateNotMapped))
	assert.Equal(t, mb.DocumentFailed, h.docs.last().Status)
	assert.Equal(t, []string{"integrations.moneybird_document.sync_failed"}, h.outbox.Subjects())
}

// A product deleted since the quote was made can't be priced into VAT.
func TestSendQuote_lineNoLongerInTheCatalog(t *testing.T) {
	q, catalog := quoteFixture()
	catalog.result.Rows = catalog.result.Rows[1:]
	h := newQuoteHarness(t, q, catalog)

	_, err := h.uc.Execute(tenantCtx(), "q-1")

	assert.True(t, errors.Is(err, quotes.ErrLineUnavailable))
}
