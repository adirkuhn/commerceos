package quotes_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genquotes "github.com/adirkuhn/commerceos/api/quotes"
	appquotes "github.com/adirkuhn/commerceos/application/quotes"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/domain/quotes"
	handler "github.com/adirkuhn/commerceos/infrastructure/handlers/quotes"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func init() { gin.SetMode(gin.TestMode) }

const (
	quoteID = "3f6d1c2a-0b7e-4c1d-9a2f-5e8b7c6d4a31"
	prodID  = "8a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d"
	skuID   = "5e1d2c3b-4a59-4f6e-8d7c-6b5a49382716"
)

var stamp = time.Date(2026, 10, 23, 10, 0, 0, 0, time.UTC)

func sample() *quotes.Quote {
	return &quotes.Quote{ID: quoteID, Number: 7, Name: "Restaurant Sabor", DiscountBasisPoints: 3000, CurrencyCode: "EUR",
		Lines:     []quotes.Line{{ProductID: prodID, SKUID: skuID, Quantity: 100, UnitPriceMinorUnits: 871}},
		CreatedAt: stamp, UpdatedAt: stamp}
}

type stubs struct {
	gotWarehouse string
	gotConvert   appquotes.ConvertQuoteInput
	stockOut     *appquotes.StockCheck
	convertOut   *appquotes.ConvertQuoteResult
	gotInput     quotes.Input
	gotID        string
	gotList      appquotes.ListQuotesInput
	out          *quotes.Quote
	listOut      *appquotes.ListQuotesResult
	err          error
	deleteCall   bool
}

func (s *stubs) Create(_ context.Context, in quotes.Input) (*quotes.Quote, error) {
	s.gotInput = in
	return s.out, s.err
}
func (s *stubs) Update(_ context.Context, id string, in quotes.Input) (*quotes.Quote, error) {
	s.gotID, s.gotInput = id, in
	return s.out, s.err
}
func (s *stubs) Get(_ context.Context, id string) (*quotes.Quote, error) {
	s.gotID = id
	return s.out, s.err
}
func (s *stubs) List(_ context.Context, in appquotes.ListQuotesInput) (*appquotes.ListQuotesResult, error) {
	s.gotList = in
	return s.listOut, s.err
}
func (s *stubs) Delete(_ context.Context, id string) error {
	s.gotID, s.deleteCall = id, true
	return s.err
}

// One adapter per use case port over the shared stubs.
type (
	stockStub   struct{ *stubs }
	convertStub struct{ *stubs }
	createStub  struct{ *stubs }
	updateStub  struct{ *stubs }
	getStub     struct{ *stubs }
	listStub    struct{ *stubs }
	deleteStub  struct{ *stubs }
)

func (s createStub) Execute(ctx context.Context, in quotes.Input) (*quotes.Quote, error) {
	return s.Create(ctx, in)
}
func (s updateStub) Execute(ctx context.Context, id string, in quotes.Input) (*quotes.Quote, error) {
	return s.Update(ctx, id, in)
}
func (s getStub) Execute(ctx context.Context, id string) (*quotes.Quote, error) {
	return s.Get(ctx, id)
}
func (s listStub) Execute(ctx context.Context, in appquotes.ListQuotesInput) (*appquotes.ListQuotesResult, error) {
	return s.List(ctx, in)
}
func (s stockStub) Execute(_ context.Context, id, warehouseID string) (*appquotes.StockCheck, error) {
	s.gotID, s.gotWarehouse = id, warehouseID
	return s.stockOut, s.err
}
func (s convertStub) Execute(_ context.Context, in appquotes.ConvertQuoteInput) (*appquotes.ConvertQuoteResult, error) {
	s.gotConvert = in
	return s.convertOut, s.err
}
func (s deleteStub) Execute(ctx context.Context, id string) error { return s.Delete(ctx, id) }

func newRouter(t *testing.T, s *stubs) *gin.Engine {
	t.Helper()
	spec, err := genquotes.GetSpec()
	require.NoError(t, err)
	r := gin.New()
	g := r.Group("/api/v1")
	g.Use(middleware.JWT([]byte("s")))
	g.Use(ginmiddleware.OapiRequestValidatorWithOptions(spec, &ginmiddleware.Options{SilenceServersWarning: true}))
	genquotes.RegisterHandlers(g, handler.NewHandler(handler.UseCases{
		Create: createStub{s}, Update: updateStub{s}, Get: getStub{s}, List: listStub{s}, Delete: deleteStub{s},
		CheckStock: stockStub{s}, Convert: convertStub{s},
	}))
	return r
}

func token(t *testing.T) string {
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, middleware.Claims{UserID: "user-1", TenantID: "8d0c3f4e-1a2b-4c5d-9e8f-7a6b5c4d3e2f", Role: middleware.RoleCustomerUser,
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}).SignedString([]byte("s"))
	require.NoError(t, err)
	return s
}

func do(t *testing.T, r *gin.Engine, method, path, body string, withToken bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if withToken {
		req.Header.Set("Authorization", "Bearer "+token(t))
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

const validBody = `{"name":"Restaurant Sabor","discount_basis_points":3000,"currency_code":"EUR",
	"lines":[{"product_id":"` + prodID + `","sku_id":"` + skuID + `","quantity":100,"unit_price_minor_units":871}]}`

const sampleJSON = `{"id":"` + quoteID + `","number":7,"reference":"Q-0007","name":"Restaurant Sabor","customer":null,"currency_code":"EUR",
	"discount_basis_points":3000,"line_count":1,"total_minor_units":87100,
	"created_at":"2026-10-23T10:00:00Z","updated_at":"2026-10-23T10:00:00Z",
	"lines":[{"product_id":"` + prodID + `","sku_id":"` + skuID + `","quantity":100,"unit_price_minor_units":871,"line_total_minor_units":87100}]}`

func TestCreateQuote(t *testing.T) {
	s := &stubs{out: sample()}

	w := do(t, newRouter(t, s), http.MethodPost, "/api/v1/quotes", validBody, true)

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	assert.JSONEq(t, `{"data":`+sampleJSON+`}`, w.Body.String())
	assert.Equal(t, quotes.Input{Name: "Restaurant Sabor", DiscountBasisPoints: 3000, CurrencyCode: "EUR",
		Lines: []quotes.Line{{ProductID: prodID, SKUID: skuID, Quantity: 100, UnitPriceMinorUnits: 871}}}, s.gotInput)
}

func TestCreateQuote_unauthenticated(t *testing.T) {
	w := do(t, newRouter(t, &stubs{}), http.MethodPost, "/api/v1/quotes", validBody, false)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateQuote_contractRejectsBadBodies(t *testing.T) {
	for name, body := range map[string]string{
		"no name":            `{"discount_basis_points":0,"lines":[]}`,
		"discount over 100%": `{"name":"x","discount_basis_points":10001,"lines":[]}`,
		"negative qty":       `{"name":"x","discount_basis_points":0,"currency_code":"EUR","lines":[{"product_id":"` + prodID + `","sku_id":"` + skuID + `","quantity":-1,"unit_price_minor_units":1}]}`,
		"lowercase currency": `{"name":"x","discount_basis_points":0,"currency_code":"eur","lines":[]}`,
	} {
		t.Run(name, func(t *testing.T) {
			w := do(t, newRouter(t, &stubs{}), http.MethodPost, "/api/v1/quotes", body, true)
			assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
		})
	}
}

// The validator doesn't check format: uuid; binding the body does.
func TestCreateQuote_skuMustBeUUID(t *testing.T) {
	body := `{"name":"x","discount_basis_points":0,"currency_code":"EUR","lines":[{"product_id":"` + prodID + `","sku_id":"nope","quantity":1,"unit_price_minor_units":1}]}`
	s := &stubs{}

	w := do(t, newRouter(t, s), http.MethodPost, "/api/v1/quotes", body, true)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Empty(t, s.gotInput.Name, "the use case is never called")
}

func TestCreateQuote_domainValidationIs422WithCode(t *testing.T) {
	s := &stubs{err: quotes.ErrDuplicateLine}

	w := do(t, newRouter(t, s), http.MethodPost, "/api/v1/quotes", validBody, true)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), `"code":"QUOTE_DUPLICATE_LINE"`)
}

func TestGetQuote(t *testing.T) {
	s := &stubs{out: sample()}

	w := do(t, newRouter(t, s), http.MethodGet, "/api/v1/quotes/"+quoteID, "", true)

	require.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"data":`+sampleJSON+`}`, w.Body.String())
	assert.Equal(t, quoteID, s.gotID)
}

func TestGetQuote_notFound(t *testing.T) {
	w := do(t, newRouter(t, &stubs{err: quotes.ErrQuoteNotFound}), http.MethodGet, "/api/v1/quotes/"+quoteID, "", true)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), `"code":"QUOTE_NOT_FOUND"`)
}

func TestGetQuote_idMustBeUUID(t *testing.T) {
	w := do(t, newRouter(t, &stubs{}), http.MethodGet, "/api/v1/quotes/nope", "", true)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateQuote(t *testing.T) {
	s := &stubs{out: sample()}

	w := do(t, newRouter(t, s), http.MethodPut, "/api/v1/quotes/"+quoteID, validBody, true)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, quoteID, s.gotID)
	assert.Equal(t, "Restaurant Sabor", s.gotInput.Name)
}

func TestUpdateQuote_notFound(t *testing.T) {
	w := do(t, newRouter(t, &stubs{err: quotes.ErrQuoteNotFound}), http.MethodPut, "/api/v1/quotes/"+quoteID, validBody, true)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteQuote(t *testing.T) {
	s := &stubs{}

	w := do(t, newRouter(t, s), http.MethodDelete, "/api/v1/quotes/"+quoteID, "", true)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.True(t, s.deleteCall)
}

func TestDeleteQuote_notFound(t *testing.T) {
	w := do(t, newRouter(t, &stubs{err: quotes.ErrQuoteNotFound}), http.MethodDelete, "/api/v1/quotes/"+quoteID, "", true)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListQuotes(t *testing.T) {
	s := &stubs{listOut: &appquotes.ListQuotesResult{Total: 61, Quotes: []quotes.Summary{{
		ID: quoteID, Number: 7, Name: "Restaurant Sabor", CurrencyCode: "EUR", DiscountBasisPoints: 3000,
		Customer:  &quotes.CustomerRef{ID: "c-1", Name: "Ana Souza", Email: "ana@sabor.nl"},
		LineCount: 1, TotalMinorUnits: 87100, CreatedAt: stamp, UpdatedAt: stamp,
	}}}}

	w := do(t, newRouter(t, s), http.MethodGet, "/api/v1/quotes?page=2&per_page=25&search=sabor", "", true)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, appquotes.ListQuotesInput{Page: 2, PerPage: 25, Search: "sabor"}, s.gotList)
	assert.JSONEq(t, `{"data":[{"id":"`+quoteID+`","number":7,"reference":"Q-0007","name":"Restaurant Sabor",
		"customer":{"id":"c-1","name":"Ana Souza","email":"ana@sabor.nl"},"currency_code":"EUR",
		"discount_basis_points":3000,"line_count":1,"total_minor_units":87100,
		"created_at":"2026-10-23T10:00:00Z","updated_at":"2026-10-23T10:00:00Z"}],"meta":{"total":61}}`, w.Body.String())
}

func TestListQuotes_emptyIsList(t *testing.T) {
	s := &stubs{listOut: &appquotes.ListQuotesResult{}}

	w := do(t, newRouter(t, s), http.MethodGet, "/api/v1/quotes", "", true)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"data":[]`)
}

func TestListQuotes_perPageCappedByContract(t *testing.T) {
	w := do(t, newRouter(t, &stubs{}), http.MethodGet, "/api/v1/quotes?per_page=1000", "", true)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateQuote_forACustomer(t *testing.T) {
	const customerID = "9d1e2f3a-4b5c-4d6e-8f7a-1b2c3d4e5f60"
	s := &stubs{out: sample()}
	body := `{"name":"x","customer_id":"` + customerID + `","discount_basis_points":0,"lines":[]}`

	w := do(t, newRouter(t, s), http.MethodPost, "/api/v1/quotes", body, true)

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	require.NotNil(t, s.gotInput.CustomerID)
	assert.Equal(t, customerID, *s.gotInput.CustomerID)
}

func TestCreateQuote_unknownCustomerIs422(t *testing.T) {
	s := &stubs{err: customersDomain.ErrCustomerNotFound}

	w := do(t, newRouter(t, s), http.MethodPost, "/api/v1/quotes", validBody, true)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), `"code":"CUSTOMER_NOT_FOUND"`)
}

const warehouseID = "6c5b4a39-2817-4f6e-9d8c-7b6a5f4e3d2c"

func TestCheckQuoteStock(t *testing.T) {
	s := &stubs{stockOut: &appquotes.StockCheck{Lines: []appquotes.StockCheckLine{
		{ProductID: prodID, SKUID: skuID, Quantity: 10, Available: 4, Short: 6},
	}}}

	w := do(t, newRouter(t, s), http.MethodGet, "/api/v1/quotes/"+quoteID+"/stock-check?warehouse_id="+warehouseID, "", true)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, warehouseID, s.gotWarehouse)
	assert.JSONEq(t, `{"data":{"has_shortfall":true,"lines":[{"product_id":"`+prodID+`","sku_id":"`+skuID+`","quantity":10,"available":4,"short":6}]}}`, w.Body.String())
}

func TestCheckQuoteStock_needsAWarehouse(t *testing.T) {
	w := do(t, newRouter(t, &stubs{}), http.MethodGet, "/api/v1/quotes/"+quoteID+"/stock-check", "", true)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCheckQuoteStock_notFound(t *testing.T) {
	w := do(t, newRouter(t, &stubs{err: quotes.ErrQuoteNotFound}), http.MethodGet, "/api/v1/quotes/"+quoteID+"/stock-check?warehouse_id="+warehouseID, "", true)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestConvertQuote(t *testing.T) {
	s := &stubs{convertOut: &appquotes.ConvertQuoteResult{SalesOrderID: "so-1", Number: "B2B-1", NeedsStockReview: true}}

	w := do(t, newRouter(t, s), http.MethodPost, "/api/v1/quotes/"+quoteID+"/sales-orders", `{"warehouse_id":"`+warehouseID+`","number":"B2B-1"}`, true)

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	assert.Equal(t, appquotes.ConvertQuoteInput{QuoteID: quoteID, WarehouseID: warehouseID, Number: "B2B-1"}, s.gotConvert)
	assert.JSONEq(t, `{"data":{"sales_order_id":"so-1","number":"B2B-1","needs_stock_review":true}}`, w.Body.String())
}

func TestConvertQuote_contract(t *testing.T) {
	r := newRouter(t, &stubs{})
	assert.Equal(t, http.StatusBadRequest, do(t, r, http.MethodPost, "/api/v1/quotes/"+quoteID+"/sales-orders", `{}`, true).Code, "warehouse required")
	assert.Equal(t, http.StatusUnauthorized, do(t, r, http.MethodPost, "/api/v1/quotes/"+quoteID+"/sales-orders", `{"warehouse_id":"`+warehouseID+`"}`, false).Code)
}

func TestConvertQuote_domainErrors(t *testing.T) {
	for err, status := range map[error]int{
		quotes.ErrQuoteNotFound:   http.StatusNotFound,
		quotes.ErrLineUnavailable: http.StatusUnprocessableEntity,
		quotes.ErrNothingToOrder:  http.StatusUnprocessableEntity,
	} {
		w := do(t, newRouter(t, &stubs{err: err}), http.MethodPost, "/api/v1/quotes/"+quoteID+"/sales-orders", `{"warehouse_id":"`+warehouseID+`"}`, true)
		assert.Equal(t, status, w.Code, err.Error())
	}
}
