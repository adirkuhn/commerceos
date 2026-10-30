package moneybird

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mb "github.com/adirkuhn/commerceos/domain/moneybird"
)

type recorded struct {
	method, path, query, auth string
	body                      map[string]any
}

// server answers each "METHOD /path" from routes and records every request.
func server(t *testing.T, routes map[string]func(w http.ResponseWriter)) (*httptest.Server, *[]recorded) {
	t.Helper()
	var calls []recorded
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		calls = append(calls, recorded{r.Method, r.URL.Path, r.URL.RawQuery, r.Header.Get("Authorization"), body})
		handler, ok := routes[r.Method+" "+r.URL.Path]
		if !ok {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		handler(w)
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

func reply(status int, body string) func(w http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}
}

func testClient(srv *httptest.Server, adminID string) mb.Client {
	return newClient(srv.URL, "tok-123", adminID, srv.Client(), newLimiter(100, time.Millisecond))
}

func TestClient_listAdministrations_usesAccountLevelPathAndBearerToken(t *testing.T) {
	srv, calls := server(t, map[string]func(http.ResponseWriter){
		"GET /administrations.json": reply(200, `[{"id":"123","name":"Cantinho","currency":"EUR"}]`),
	})

	got, err := testClient(srv, "").ListAdministrations(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []mb.Administration{{ID: "123", Name: "Cantinho", Currency: "EUR"}}, got)
	assert.Equal(t, "Bearer tok-123", (*calls)[0].auth)
}

func TestClient_listSalesTaxRates_onlyActiveSalesRates(t *testing.T) {
	srv, calls := server(t, map[string]func(http.ResponseWriter){
		"GET /adm/tax_rates.json": reply(200, `[{"id":"1","name":"21% btw","percentage":"21.0","active":true},{"id":"2","name":"old","percentage":"19.0","active":false}]`),
	})

	got, err := testClient(srv, "adm").ListSalesTaxRates(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []mb.TaxRate{{ID: "1", Name: "21% btw", Percentage: "21.0"}}, got)
	assert.Equal(t, "filter=tax_rate_type%3Asales_invoice", (*calls)[0].query)
}

func TestClient_listRevenueLedgerAccounts_onlyRevenue(t *testing.T) {
	srv, _ := server(t, map[string]func(http.ResponseWriter){
		"GET /adm/ledger_accounts.json": reply(200, `[{"id":"1","name":"Omzet","account_type":"revenue"},{"id":"2","name":"Kas","account_type":"current_assets"}]`),
	})

	got, err := testClient(srv, "adm").ListRevenueLedgerAccounts(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []mb.LedgerAccount{{ID: "1", Name: "Omzet", AccountType: "revenue"}}, got)
}

func TestClient_createInvoice_sendsExternalSalesInvoice(t *testing.T) {
	srv, calls := server(t, map[string]func(http.ResponseWriter){
		"POST /adm/external_sales_invoices.json": reply(201, `{"id":"inv-1","state":"open","total_unpaid":"12.11","total_price_incl_tax_base":"12.11"}`),
	})

	got, err := testClient(srv, "adm").CreateInvoice(context.Background(), mb.Invoice{
		ContactID: "c-1", Reference: "CO-1001", Date: "2026-10-29", DueDate: "2026-10-29", Currency: "EUR",
		PricesInclTax: true, Source: "CommerceOS", SourceURL: "https://x/sales-orders/1",
		Lines: []mb.DocumentLine{{Description: "Carne Seca ~380g", Amount: "1", PriceMinorUnits: 1140, TaxRateID: "tr-9", LedgerAccountID: "led-1"}},
	})

	require.NoError(t, err)
	assert.Equal(t, &mb.RemoteDocument{ID: "inv-1", State: "open", TotalUnpaid: "12.11", TotalInclTaxBase: "12.11"}, got)
	body := (*calls)[0].body["external_sales_invoice"].(map[string]any)
	assert.Equal(t, "c-1", body["contact_id"])
	assert.Equal(t, "CO-1001", body["reference"])
	assert.Equal(t, true, body["prices_are_incl_tax"])
	line := body["details_attributes"].([]any)[0].(map[string]any)
	assert.Equal(t, map[string]any{"description": "Carne Seca ~380g", "amount": "1", "price": "11.40", "tax_rate_id": "tr-9", "ledger_account_id": "led-1"}, line)
}

func TestClient_findInvoiceByReference_exactReferenceOnly(t *testing.T) {
	srv, calls := server(t, map[string]func(http.ResponseWriter){
		"GET /adm/external_sales_invoices.json": reply(200, `[{"id":"a","reference":"CO-10011"},{"id":"b","reference":"CO-1001","state":"paid"}]`),
	})

	got, err := testClient(srv, "adm").FindInvoiceByReference(context.Background(), "CO-1001")

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "b", got.ID)
	assert.Equal(t, "filter=reference%3ACO-1001", (*calls)[0].query)
}

func TestClient_findInvoiceByReference_noneIsNil(t *testing.T) {
	srv, _ := server(t, map[string]func(http.ResponseWriter){
		"GET /adm/external_sales_invoices.json": reply(200, `[]`),
	})
	got, err := testClient(srv, "adm").FindInvoiceByReference(context.Background(), "CO-1")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestClient_registerPayment(t *testing.T) {
	srv, calls := server(t, map[string]func(http.ResponseWriter){
		"POST /adm/external_sales_invoices/inv-1/payments.json": reply(201, `{}`),
	})

	err := testClient(srv, "adm").RegisterPayment(context.Background(), "inv-1", mb.Payment{Date: "2026-10-29", Price: "12.11", Method: "bank_transfer"})

	require.NoError(t, err)
	assert.Equal(t, map[string]any{"payment_date": "2026-10-29", "price": "12.11", "payment_method": "bank_transfer"}, (*calls)[0].body["payment"])
}

func TestClient_findContact_matchesEmailOrCompanyExactly(t *testing.T) {
	srv, _ := server(t, map[string]func(http.ResponseWriter){
		"GET /adm/contacts.json": reply(200, `[{"id":"1","company_name":"Ana Bakery","email":""},{"id":"2","company_name":"","email":"ana@example.nl"}]`),
	})

	got, err := testClient(srv, "adm").FindContact(context.Background(), "ana@example.nl")

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "2", got.ID)
}

func TestClient_findContact_fuzzyHitsAreNotAMatch(t *testing.T) {
	srv, _ := server(t, map[string]func(http.ResponseWriter){
		"GET /adm/contacts.json": reply(200, `[{"id":"1","company_name":"Ana Bakery B.V.","email":"info@anabakery.nl"}]`),
	})
	got, err := testClient(srv, "adm").FindContact(context.Background(), "Ana Bakery")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestClient_updateEstimate_replacesTheOldLines(t *testing.T) {
	srv, calls := server(t, map[string]func(http.ResponseWriter){
		"GET /adm/estimates/est-1.json":   reply(200, `{"id":"est-1","state":"draft","details":[{"id":"d1"},{"id":"d2"}]}`),
		"PATCH /adm/estimates/est-1.json": reply(200, `{"id":"est-1","state":"draft"}`),
	})

	_, err := testClient(srv, "adm").UpdateEstimate(context.Background(), "est-1", mb.Estimate{
		ContactID: "c-1", Reference: "Q-7", Date: "2026-10-29", Currency: "EUR", PricesInclTax: true,
		Lines: []mb.DocumentLine{{Description: "Farofa", Amount: "2", PriceMinorUnits: 349, TaxRateID: "tr-9", LedgerAccountID: "led-1"}},
	})

	require.NoError(t, err)
	details := (*calls)[1].body["estimate"].(map[string]any)["details_attributes"].([]any)
	require.Len(t, details, 3)
	assert.Equal(t, map[string]any{"id": "d1", "_destroy": true}, details[0])
	assert.Equal(t, map[string]any{"id": "d2", "_destroy": true}, details[1])
	assert.Equal(t, "Farofa", details[2].(map[string]any)["description"])
}

func TestClient_errors(t *testing.T) {
	cases := map[int]error{
		http.StatusTooManyRequests:     mb.ErrRateLimited,
		http.StatusUnauthorized:        mb.ErrTokenInvalid,
		http.StatusUnprocessableEntity: mb.ErrRequestFailed,
	}
	for status, want := range cases {
		srv, _ := server(t, map[string]func(http.ResponseWriter){
			"GET /administrations.json": reply(status, `{"error":"Tax rate is not valid"}`),
		})
		_, err := testClient(srv, "").ListAdministrations(context.Background())
		assert.True(t, errors.Is(err, want), "status %d: got %v", status, err)
	}
}

func TestClient_requestFailedCarriesMoneybirdsReason(t *testing.T) {
	srv, _ := server(t, map[string]func(http.ResponseWriter){
		"GET /administrations.json": reply(422, `{"error":{"tax_rate_id":["is not valid"]}}`),
	})
	_, err := testClient(srv, "").ListAdministrations(context.Background())
	assert.Contains(t, err.Error(), "is not valid")
}

func TestClient_deleteInvoiceAndEstimate(t *testing.T) {
	srv, calls := server(t, map[string]func(http.ResponseWriter){
		"DELETE /adm/external_sales_invoices/inv-1.json": reply(204, ``),
		"DELETE /adm/estimates/est-1.json":               reply(204, ``),
	})
	c := testClient(srv, "adm")

	require.NoError(t, c.DeleteInvoice(context.Background(), "inv-1"))
	require.NoError(t, c.DeleteEstimate(context.Background(), "est-1"))
	assert.Len(t, *calls, 2)
}

// Already deleted in Moneybird by hand: nothing left to do.
func TestClient_deleteSomethingAlreadyGoneIsNotAnError(t *testing.T) {
	srv, _ := server(t, map[string]func(http.ResponseWriter){
		"DELETE /adm/external_sales_invoices/inv-1.json": reply(404, `{"error":"record not found"}`),
	})
	require.NoError(t, testClient(srv, "adm").DeleteInvoice(context.Background(), "inv-1"))
}

func TestClient_deleteRefused(t *testing.T) {
	srv, _ := server(t, map[string]func(http.ResponseWriter){
		"DELETE /adm/external_sales_invoices/inv-1.json": reply(422, `{"error":"cannot delete"}`),
	})
	err := testClient(srv, "adm").DeleteInvoice(context.Background(), "inv-1")
	assert.True(t, errors.Is(err, mb.ErrRequestFailed))
}
