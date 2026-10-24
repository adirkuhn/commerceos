package quotes_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	appquotes "github.com/adirkuhn/commerceos/application/quotes"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/domain/quotes"
)

type stubStock struct {
	shortBySKU map[string]int64
	calls      []string
	err        error
}

func (s *stubStock) CheckAvailability(_ context.Context, skuID, warehouseID string, _ int64) (int64, error) {
	s.calls = append(s.calls, skuID+"@"+warehouseID)
	return s.shortBySKU[skuID], s.err
}

type stubCatalog struct {
	rows []apppms.PriceListRow
	got  apppms.ListPriceListInput
}

func (s *stubCatalog) Execute(_ context.Context, in apppms.ListPriceListInput) (*apppms.ListPriceListResult, error) {
	s.got = in
	return &apppms.ListPriceListResult{Rows: s.rows, PricesIncludeTax: true}, nil
}

type stubCreateOrder struct {
	got omsDomain.SalesOrder
	in  appoms.CreateSalesOrderInput
	out *appoms.CreateSalesOrderResult
	err error
}

func (s *stubCreateOrder) Execute(_ context.Context, in appoms.CreateSalesOrderInput) (*appoms.CreateSalesOrderResult, error) {
	s.in = in
	return s.out, s.err
}

func quoteForOrder() *quotes.Quote {
	return &quotes.Quote{ID: "q-1", TenantID: "t-1", Number: 7, Name: "Restaurant Sabor", CurrencyCode: "EUR",
		Lines: []quotes.Line{
			{ProductID: "p-1", SKUID: "s-1", Quantity: 10, UnitPriceMinorUnits: 1210},
			{ProductID: "p-2", SKUID: "s-2", Quantity: 0, UnitPriceMinorUnits: 500},
			{ProductID: "p-3", SKUID: "s-3", Quantity: 2, UnitPriceMinorUnits: 500},
		}}
}

func catalogRows() []apppms.PriceListRow {
	return []apppms.PriceListRow{
		{ProductID: "p-1", ProductName: "Feijão", SKUID: "s-1", SKUCode: "FEI-1", TaxRateBasisPoints: 2100},
		{ProductID: "p-3", ProductName: "Farofa", VariantName: "500g", SKUID: "s-3", SKUCode: "FAR-1", TaxRateBasisPoints: 900},
	}
}

func TestCheckQuoteStock_reportsEachShortLineInTheWarehouse(t *testing.T) {
	stock := &stubStock{shortBySKU: map[string]int64{"s-3": 2}}
	uc := appquotes.NewCheckQuoteStockUseCase(newMockGormDB(t), newStubRepo(quoteForOrder()), stock)

	got, err := uc.Execute(ctxFor(), "q-1", "wh-1")

	require.NoError(t, err)
	assert.Equal(t, []appquotes.StockCheckLine{
		{ProductID: "p-1", SKUID: "s-1", Quantity: 10, Available: 10},
		{ProductID: "p-3", SKUID: "s-3", Quantity: 2, Available: 0, Short: 2},
	}, got.Lines, "a zero-quantity line needs no stock")
	assert.True(t, got.HasShortfall())
	assert.Equal(t, []string{"s-1@wh-1", "s-3@wh-1"}, stock.calls)
}

func TestCheckQuoteStock_notFound(t *testing.T) {
	_, err := appquotes.NewCheckQuoteStockUseCase(newMockGormDB(t), newStubRepo(), &stubStock{}).Execute(ctxFor(), "missing", "wh-1")
	assert.ErrorIs(t, err, quotes.ErrQuoteNotFound)
}

func TestCheckQuoteStock_wrapsStockError(t *testing.T) {
	_, err := appquotes.NewCheckQuoteStockUseCase(newMockGormDB(t), newStubRepo(quoteForOrder()), &stubStock{err: errors.New("boom")}).
		Execute(ctxFor(), "q-1", "wh-1")
	assert.ErrorContains(t, err, "checkQuoteStock")
}

func convertDeps(t *testing.T, q *quotes.Quote) (appquotes.ConvertQuoteDeps, *stubCreateOrder, *eventrecordtest.Outbox) {
	orders := &stubCreateOrder{out: &appoms.CreateSalesOrderResult{SalesOrder: omsDomain.SalesOrder{ID: "so-1", Number: "B2B-1", NeedsStockReview: true}}}
	outbox := &eventrecordtest.Outbox{}
	return appquotes.ConvertQuoteDeps{
		Quotes: newStubRepo(q), Catalog: &stubCatalog{rows: catalogRows()}, Customers: newStubCustomers(),
		Orders: orders, Outbox: outbox, Now: func() time.Time { return time.Date(2026, 10, 24, 9, 0, 0, 0, time.UTC) },
	}, orders, outbox
}

func TestConvertQuote_createsAProcessingOrderAtTheQuotedPrices(t *testing.T) {
	deps, orders, outbox := convertDeps(t, quoteForOrder())
	uc := appquotes.NewConvertQuoteToSalesOrderUseCase(newMockGormDBN(t, 2), deps)

	got, err := uc.Execute(ctxFor(), appquotes.ConvertQuoteInput{QuoteID: "q-1", WarehouseID: "wh-1", Number: "B2B-1"})

	require.NoError(t, err)
	assert.Equal(t, &appquotes.ConvertQuoteResult{SalesOrderID: "so-1", Number: "B2B-1", NeedsStockReview: true}, got)
	in := orders.in
	assert.Equal(t, omsDomain.SalesOrderStatusProcessing, in.Status)
	assert.Equal(t, omsDomain.SalesOrderSourceManual, in.Source)
	assert.Equal(t, "wh-1", in.WarehouseID)
	assert.Equal(t, "B2B-1", in.Number)
	assert.Equal(t, "EUR", in.CurrencyCode)
	require.NotNil(t, in.QuoteID)
	assert.Equal(t, "q-1", *in.QuoteID)
	assert.Nil(t, in.CustomerID)
	assert.Equal(t, "Quote Q-0007 — Restaurant Sabor", in.CustomerNote, "no customer: the quote name goes in the note")
	assert.Equal(t, []appoms.CreateSalesOrderLineItemInput{
		{SKUID: "s-1", Name: "Feijão", SKUCode: "FEI-1", Quantity: 10, PriceMinorUnits: 1210,
			SubtotalMinorUnits: 12100, SubtotalTaxMinorUnits: 2100, TotalMinorUnits: 12100, TotalTaxMinorUnits: 2100, TotalIncludesTax: true},
		{SKUID: "s-3", Name: "Farofa — 500g", SKUCode: "FAR-1", Quantity: 2, PriceMinorUnits: 500,
			SubtotalMinorUnits: 1000, SubtotalTaxMinorUnits: 83, TotalMinorUnits: 1000, TotalTaxMinorUnits: 83, TotalIncludesTax: true},
	}, in.LineItems, "zero-quantity lines are left out")
	assert.Equal(t, int64(13100), in.TotalMinorUnits)
	assert.Equal(t, int64(2183), in.TaxTotalMinorUnits)

	event, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "quotes.quote.converted", event.Subject)
	assert.Equal(t, map[string]any{"sales_order_id": "so-1", "number": float64(7), "needs_stock_review": true}, event.Data())
}

func TestConvertQuote_forACustomerCopiesItsAddresses(t *testing.T) {
	q := quoteForOrder()
	q.Customer = &quotes.CustomerRef{ID: "c-1"}
	deps, orders, _ := convertDeps(t, q)
	deps.Customers.(*stubCustomers).byID["c-1"].Billing = customersDomain.Address{FirstName: "Ana", City: "Utrecht"}
	deps.Customers.(*stubCustomers).byID["c-1"].Shipping = customersDomain.Address{City: "Amsterdam"}

	_, err := appquotes.NewConvertQuoteToSalesOrderUseCase(newMockGormDBN(t, 2), deps).
		Execute(ctxFor(), appquotes.ConvertQuoteInput{QuoteID: "q-1", WarehouseID: "wh-1"})

	require.NoError(t, err)
	require.NotNil(t, orders.in.CustomerID)
	assert.Equal(t, "c-1", *orders.in.CustomerID)
	assert.Equal(t, omsDomain.Address{FirstName: "Ana", City: "Utrecht"}, orders.in.Billing)
	assert.Equal(t, omsDomain.Address{City: "Amsterdam"}, orders.in.Shipping)
	assert.Equal(t, "Quote Q-0007", orders.in.CustomerNote)
}

func TestConvertQuote_aLineNoLongerInTheCatalogStopsIt(t *testing.T) {
	q := quoteForOrder()
	q.Lines = append(q.Lines, quotes.Line{ProductID: "p-gone", SKUID: "s-gone", Quantity: 1, UnitPriceMinorUnits: 100})
	deps, orders, _ := convertDeps(t, q)

	_, err := appquotes.NewConvertQuoteToSalesOrderUseCase(newMockGormDB(t), deps).
		Execute(ctxFor(), appquotes.ConvertQuoteInput{QuoteID: "q-1", WarehouseID: "wh-1"})

	assert.ErrorIs(t, err, quotes.ErrLineUnavailable)
	assert.Empty(t, orders.in.LineItems, "no order is created")
}

func TestConvertQuote_nothingToOrder(t *testing.T) {
	q := quoteForOrder()
	for i := range q.Lines {
		q.Lines[i].Quantity = 0
	}
	deps, _, _ := convertDeps(t, q)

	_, err := appquotes.NewConvertQuoteToSalesOrderUseCase(newMockGormDB(t), deps).
		Execute(ctxFor(), appquotes.ConvertQuoteInput{QuoteID: "q-1", WarehouseID: "wh-1"})

	assert.ErrorIs(t, err, quotes.ErrNothingToOrder)
}

func TestConvertQuote_orderErrorIsWrapped(t *testing.T) {
	deps, orders, outbox := convertDeps(t, quoteForOrder())
	orders.err = errors.New("boom")

	_, err := appquotes.NewConvertQuoteToSalesOrderUseCase(newMockGormDB(t), deps).
		Execute(ctxFor(), appquotes.ConvertQuoteInput{QuoteID: "q-1", WarehouseID: "wh-1"})

	assert.ErrorContains(t, err, "convertQuoteToSalesOrder")
	assert.Empty(t, outbox.Events)
}
