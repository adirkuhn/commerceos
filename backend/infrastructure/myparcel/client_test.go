package myparcel_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/infrastructure/myparcel"
)

// shipmentCreateAndFetchServer builds an httptest.Server that handles both
// calls CreateShipment now makes: POST /shipments (real envelope
// data.ids[]{id,reference_identifier} -- barcode is NOT in this response,
// see client.go's package doc comment) and the follow-up
// GET /shipments/{id} (data.search_results.shipments[]) that fetches it.
func shipmentCreateAndFetchServer(t *testing.T, shipmentID int64, barcode string, captureBody *map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			require.NoError(t, json.NewDecoder(r.Body).Decode(captureBody))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":{"ids":[{"id":` + strconv.FormatInt(shipmentID, 10) + `,"reference_identifier":"order-abc"}]}}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"search_results":{"shipments":[{"id":` + strconv.FormatInt(shipmentID, 10) + `,"barcode":"` + barcode + `","status":1}]}}}`))
	}))
}

func TestClient_CreateShipment_sendsAuthHeaderAndParsesResponse(t *testing.T) {
	var gotAuth, gotAccept string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			gotAuth = r.Header.Get("Authorization")
			gotAccept = r.Header.Get("Accept")
			require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":{"ids":[{"id":123,"reference_identifier":"order-abc"}]}}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"search_results":{"shipments":[{"id":123,"barcode":"3SMYPA1","status":1}]}}}`))
	}))
	defer srv.Close()

	c := myparcel.NewClient(myparcel.Config{APIKey: "secret-key", BaseURL: srv.URL})
	result, err := c.CreateShipment(context.Background(), myparcel.ShipmentInput{
		Carrier:        1,
		Recipient:      myparcel.Recipient{Person: "Jane Doe", Street: "Damrak", Number: "1", City: "Amsterdam", CC: "NL"},
		Options:        myparcel.Options{PackageType: 1},
		OrderReference: "order-abc",
	})

	require.NoError(t, err)
	assert.Equal(t, int64(123), result.ShipmentID)
	assert.Equal(t, "3SMYPA1", result.Barcode)

	wantAuth := "bearer " + base64.StdEncoding.EncodeToString([]byte("secret-key"))
	assert.Equal(t, wantAuth, gotAuth)
	assert.Contains(t, gotAccept, "shipment+json")
	shipments := gotBody["data"].(map[string]any)["shipments"].([]any)
	require.Len(t, shipments, 1)
	assert.Equal(t, "order-abc", shipments[0].(map[string]any)["reference_identifier"])
}

// TestClient_CreateShipment_barcodeFetchFailureIsNonFatal is a regression
// test for the actual duplicate-shipment bug: if the id is confirmed but
// the follow-up barcode fetch fails, CreateShipment must still return
// success (with the real id, empty barcode) rather than an error -- an
// error here would make the caller believe creation failed and re-POST on
// retry, creating a second real MyParcel shipment.
func TestClient_CreateShipment_barcodeFetchFailureIsNonFatal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":{"ids":[{"id":123,"reference_identifier":"order-abc"}]}}`))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := myparcel.NewClient(myparcel.Config{APIKey: "k", BaseURL: srv.URL})
	result, err := c.CreateShipment(context.Background(), myparcel.ShipmentInput{
		Carrier: 1, Recipient: myparcel.Recipient{Person: "Jane Doe", Street: "Damrak", Number: "1", City: "Amsterdam", CC: "NL"},
		Options: myparcel.Options{PackageType: 1},
	})

	require.NoError(t, err)
	assert.Equal(t, int64(123), result.ShipmentID)
	assert.Equal(t, "", result.Barcode)
}

// TestClient_GetShipmentBarcode_fetchesFromShipmentsEndpoint proves the
// dedicated GET /shipments/{id} call this codebase's poll/manual-refresh
// paths use to backfill a barcode -- GetTrackTrace (GET /tracktraces/{id})
// never carries one.
func TestClient_GetShipmentBarcode_fetchesFromShipmentsEndpoint(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"search_results":{"shipments":[{"id":555,"barcode":"3SMYPA447656020","status":4}]}}}`))
	}))
	defer srv.Close()

	c := myparcel.NewClient(myparcel.Config{APIKey: "k", BaseURL: srv.URL})
	barcode, err := c.GetShipmentBarcode(context.Background(), 555)

	require.NoError(t, err)
	assert.Equal(t, "3SMYPA447656020", barcode)
	assert.Equal(t, "/shipments/555", gotPath)
}

func TestClient_GetShipmentBarcode_emptyResponse_returnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"search_results":{"shipments":[]}}}`))
	}))
	defer srv.Close()

	c := myparcel.NewClient(myparcel.Config{APIKey: "k", BaseURL: srv.URL})
	_, err := c.GetShipmentBarcode(context.Background(), 555)

	require.Error(t, err)
}

func TestClient_CreateShipment_emptyIdsArrayOnSuccessReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"ids":[]}}`))
	}))
	defer srv.Close()

	c := myparcel.NewClient(myparcel.Config{APIKey: "k", BaseURL: srv.URL})
	_, err := c.CreateShipment(context.Background(), myparcel.ShipmentInput{
		Carrier: 1, Recipient: myparcel.Recipient{Person: "Jane Doe", Street: "Damrak", Number: "1", City: "Amsterdam", CC: "NL"},
		Options: myparcel.Options{PackageType: 1},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty response")
}

func TestClient_CreateShipment_insuredShipmentSendsInsuranceOption(t *testing.T) {
	var gotBody map[string]any
	srv := shipmentCreateAndFetchServer(t, 123, "3SMYPA1", &gotBody)
	defer srv.Close()

	c := myparcel.NewClient(myparcel.Config{APIKey: "secret-key", BaseURL: srv.URL})
	_, err := c.CreateShipment(context.Background(), myparcel.ShipmentInput{
		Carrier:   1,
		Recipient: myparcel.Recipient{Person: "Jane Doe", Street: "Damrak", Number: "1", City: "Amsterdam", CC: "NL"},
		Options: myparcel.Options{
			PackageType: 1, SignatureRequired: true, OnlyRecipient: true,
			InsuranceAmountMinorUnits: 50000, InsuranceCurrency: "USD",
		},
	})

	require.NoError(t, err)
	shipments := gotBody["data"].(map[string]any)["shipments"].([]any)
	options := shipments[0].(map[string]any)["options"].(map[string]any)
	insurance := options["insurance"].(map[string]any)
	assert.Equal(t, float64(50000), insurance["amount"])
	assert.Equal(t, "USD", insurance["currency"], "currency comes from the caller (tenant's Settings -> Localization -> Default Currency), not hardcoded")
}

// TestClient_CreateShipment_insuredShipmentWithNoCurrencyFallsBackToEUR
// covers a caller with no tenant currency to hand (e.g. Settings ->
// Localization -> Default Currency never configured -- TenantSettings' zero
// value) -- MyParcel requires an explicit currency whenever insured, so
// this can't just omit the field.
func TestClient_CreateShipment_insuredShipmentWithNoCurrencyFallsBackToEUR(t *testing.T) {
	var gotBody map[string]any
	srv := shipmentCreateAndFetchServer(t, 123, "3SMYPA1", &gotBody)
	defer srv.Close()

	c := myparcel.NewClient(myparcel.Config{APIKey: "secret-key", BaseURL: srv.URL})
	_, err := c.CreateShipment(context.Background(), myparcel.ShipmentInput{
		Carrier:   1,
		Recipient: myparcel.Recipient{Person: "Jane Doe", Street: "Damrak", Number: "1", City: "Amsterdam", CC: "NL"},
		Options:   myparcel.Options{PackageType: 1, SignatureRequired: true, OnlyRecipient: true, InsuranceAmountMinorUnits: 50000},
	})

	require.NoError(t, err)
	shipments := gotBody["data"].(map[string]any)["shipments"].([]any)
	options := shipments[0].(map[string]any)["options"].(map[string]any)
	insurance := options["insurance"].(map[string]any)
	assert.Equal(t, "EUR", insurance["currency"])
}

func TestClient_CreateShipment_noInsuranceOmitsInsuranceOption(t *testing.T) {
	var gotBody map[string]any
	srv := shipmentCreateAndFetchServer(t, 123, "3SMYPA1", &gotBody)
	defer srv.Close()

	c := myparcel.NewClient(myparcel.Config{APIKey: "secret-key", BaseURL: srv.URL})
	_, err := c.CreateShipment(context.Background(), myparcel.ShipmentInput{
		Carrier: 1, Recipient: myparcel.Recipient{Person: "Jane Doe", Street: "Damrak", Number: "1", City: "Amsterdam", CC: "NL"},
		Options: myparcel.Options{PackageType: 1},
	})

	require.NoError(t, err)
	shipments := gotBody["data"].(map[string]any)["shipments"].([]any)
	options := shipments[0].(map[string]any)["options"].(map[string]any)
	_, hasInsurance := options["insurance"]
	assert.False(t, hasInsurance)
}

// TestClient_CreateShipment_sendsPhysicalPropertiesWeight is a regression
// test: MyParcel requires shipments[].physical_properties.weight (grams)
// for non-EU and digital-stamp shipments (confirmed against
// 07.shipment-related-object-definitions.html) -- this field wasn't sent
// at all until this was added.
func TestClient_CreateShipment_sendsPhysicalPropertiesWeight(t *testing.T) {
	var gotBody map[string]any
	srv := shipmentCreateAndFetchServer(t, 123, "3SMYPA1", &gotBody)
	defer srv.Close()

	c := myparcel.NewClient(myparcel.Config{APIKey: "secret-key", BaseURL: srv.URL})
	weight := 1100
	_, err := c.CreateShipment(context.Background(), myparcel.ShipmentInput{
		Carrier: 1, Recipient: myparcel.Recipient{Person: "Jane Doe", Street: "Damrak", Number: "1", City: "Amsterdam", CC: "NL"},
		Options: myparcel.Options{PackageType: 1}, WeightGrams: &weight,
	})

	require.NoError(t, err)
	shipments := gotBody["data"].(map[string]any)["shipments"].([]any)
	physProps := shipments[0].(map[string]any)["physical_properties"].(map[string]any)
	assert.Equal(t, float64(1100), physProps["weight"])
}

func TestClient_CreateShipment_noWeightOmitsPhysicalProperties(t *testing.T) {
	var gotBody map[string]any
	srv := shipmentCreateAndFetchServer(t, 123, "3SMYPA1", &gotBody)
	defer srv.Close()

	c := myparcel.NewClient(myparcel.Config{APIKey: "secret-key", BaseURL: srv.URL})
	_, err := c.CreateShipment(context.Background(), myparcel.ShipmentInput{
		Carrier: 1, Recipient: myparcel.Recipient{Person: "Jane Doe", Street: "Damrak", Number: "1", City: "Amsterdam", CC: "NL"},
		Options: myparcel.Options{PackageType: 1},
	})

	require.NoError(t, err)
	shipments := gotBody["data"].(map[string]any)["shipments"].([]any)
	_, hasPhysProps := shipments[0].(map[string]any)["physical_properties"]
	assert.False(t, hasPhysProps)
}

// TestClient_CreateShipment_signatureAndOnlyRecipientAreSentAsNumbers is a
// regression test for a real MyParcel 422 ("Boolean value found, but a
// number is required... Does not have a value in the enumeration [0,1]")
// -- these two fields must be the JSON number 1/0, never true/false.
func TestClient_CreateShipment_signatureAndOnlyRecipientAreSentAsNumbers(t *testing.T) {
	var gotBody map[string]any
	srv := shipmentCreateAndFetchServer(t, 123, "3SMYPA1", &gotBody)
	defer srv.Close()

	c := myparcel.NewClient(myparcel.Config{APIKey: "secret-key", BaseURL: srv.URL})
	_, err := c.CreateShipment(context.Background(), myparcel.ShipmentInput{
		Carrier:   1,
		Recipient: myparcel.Recipient{Person: "Jane Doe", Street: "Damrak", Number: "1", City: "Amsterdam", CC: "NL"},
		Options:   myparcel.Options{PackageType: 1, SignatureRequired: true, OnlyRecipient: true},
	})

	require.NoError(t, err)
	shipments := gotBody["data"].(map[string]any)["shipments"].([]any)
	options := shipments[0].(map[string]any)["options"].(map[string]any)
	assert.IsType(t, float64(0), options["signature"], "must be a JSON number, not a boolean")
	assert.Equal(t, float64(1), options["signature"])
	assert.IsType(t, float64(0), options["only_recipient"], "must be a JSON number, not a boolean")
	assert.Equal(t, float64(1), options["only_recipient"])
}

func TestClient_CreateOrder_sendsExpectedRequestAndParsesResponse(t *testing.T) {
	var gotPath, gotContentType string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"orders":[{"uuid":"order-uuid-1","status":"open"}]}}`))
	}))
	defer srv.Close()

	c := myparcel.NewClient(myparcel.Config{APIKey: "secret-key", BaseURL: srv.URL})
	orderDate, _ := time.Parse(time.RFC3339, "2026-08-26T10:00:00Z")
	result, err := c.CreateOrder(context.Background(), myparcel.OrderInput{
		ExternalIdentifier: "order-abc", OrderDate: orderDate, AccountID: 111, ShopID: 222,
		Carrier: 1, PackageType: 1,
		Recipient:               myparcel.Recipient{Person: "Jane Doe", Street: "Damrak", Number: "1", City: "Amsterdam", CC: "NL"},
		PriceMinorUnits:         3300,
		VATMinorUnits:           700,
		PriceAfterVATMinorUnits: 4000,
		OrderLines: []myparcel.OrderLine{
			{Quantity: 2, SKU: "WIDGET-1", Name: "Widget", PriceMinorUnits: 2000, VATMinorUnits: 420, PriceAfterVATMinorUnits: 2420},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "order-uuid-1", result.UUID)
	assert.Equal(t, "open", result.Status)

	assert.Equal(t, "/fulfilment/orders", gotPath)
	assert.Contains(t, gotContentType, "application/json")
	orders := gotBody["data"].(map[string]any)["orders"].([]any)
	require.Len(t, orders, 1)
	order := orders[0].(map[string]any)
	assert.Equal(t, "order-abc", order["external_identifier"])
	assert.Equal(t, float64(111), order["account_id"])
	assert.Equal(t, float64(222), order["shop_id"])
	assert.Equal(t, float64(3300), order["price"])
	assert.Equal(t, float64(700), order["vat"])
	assert.Equal(t, float64(4000), order["price_after_vat"])
	shipment := order["shipment"].(map[string]any)
	assert.Equal(t, float64(1), shipment["carrier"])
	recipient := shipment["recipient"].(map[string]any)
	assert.Equal(t, "Jane Doe", recipient["person"])

	orderLines := order["order_lines"].([]any)
	require.Len(t, orderLines, 1)
	line := orderLines[0].(map[string]any)
	assert.Equal(t, float64(2), line["quantity"])
	assert.Equal(t, float64(2000), line["price"])
	assert.Equal(t, float64(420), line["vat"])
	assert.Equal(t, float64(2420), line["price_after_vat"])
	product := line["product"].(map[string]any)
	assert.Equal(t, "WIDGET-1", product["sku"])
	assert.Equal(t, "Widget", product["name"])
}

func TestClient_CreateOrder_nonOKStatus_returnsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":"invalid account_id"}`))
	}))
	defer srv.Close()

	c := myparcel.NewClient(myparcel.Config{APIKey: "secret-key", BaseURL: srv.URL})
	_, err := c.CreateOrder(context.Background(), myparcel.OrderInput{
		ExternalIdentifier: "order-abc", AccountID: 111, ShopID: 222, Carrier: 1,
		Recipient: myparcel.Recipient{Person: "Jane Doe", Street: "Damrak", Number: "1", City: "Amsterdam", CC: "NL"},
	})

	var apiErr *myparcel.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusUnprocessableEntity, apiErr.StatusCode)
}

func TestClient_CreateShipment_nonOKStatus_returnsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"errors":[{"detail":"invalid postal_code"}]}`))
	}))
	defer srv.Close()

	c := myparcel.NewClient(myparcel.Config{APIKey: "k", BaseURL: srv.URL})
	_, err := c.CreateShipment(context.Background(), myparcel.ShipmentInput{})

	require.Error(t, err)
	var apiErr *myparcel.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusUnprocessableEntity, apiErr.StatusCode)
	assert.Contains(t, apiErr.Body, "invalid postal_code")
}

func TestClient_GetLabel_pdfFormat_requestsPDFAccept(t *testing.T) {
	var gotAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "application/pdf")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("%PDF-fake"))
	}))
	defer srv.Close()

	c := myparcel.NewClient(myparcel.Config{APIKey: "k", BaseURL: srv.URL})
	result, err := c.GetLabel(context.Background(), 123, "pdf", "")

	require.NoError(t, err)
	assert.Equal(t, "application/pdf", gotAccept)
	assert.Equal(t, "application/pdf", result.ContentType)
	assert.Equal(t, []byte("%PDF-fake"), result.Data)
}

// TestClient_GetLabel_sizeSendsFormatQueryParam is a regression test for the
// A6-print feature: size must reach MyParcel as the `format` query param on
// the shipment_labels GET request -- see the package doc comment's third
// flagged item (param name/values unconfirmed against a real sandbox).
func TestClient_GetLabel_sizeSendsFormatQueryParam(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/pdf")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("%PDF-fake"))
	}))
	defer srv.Close()

	c := myparcel.NewClient(myparcel.Config{APIKey: "k", BaseURL: srv.URL})
	_, err := c.GetLabel(context.Background(), 123, "pdf", "A6")

	require.NoError(t, err)
	assert.Equal(t, "format=A6", gotQuery)
}

func TestClient_GetLabel_noSizeOmitsQueryParam(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/pdf")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("%PDF-fake"))
	}))
	defer srv.Close()

	c := myparcel.NewClient(myparcel.Config{APIKey: "k", BaseURL: srv.URL})
	_, err := c.GetLabel(context.Background(), 123, "pdf", "")

	require.NoError(t, err)
	assert.Equal(t, "", gotQuery)
}

func TestClient_GetLabel_zplFormat_requestsZPLAccept(t *testing.T) {
	var gotAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccept = r.Header.Get("Accept")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("^XA^XZ"))
	}))
	defer srv.Close()

	c := myparcel.NewClient(myparcel.Config{APIKey: "k", BaseURL: srv.URL})
	_, err := c.GetLabel(context.Background(), 123, "zpl", "")

	require.NoError(t, err)
	assert.Equal(t, "application/vnd.shipment_label+zpl", gotAccept)
}

func TestClient_GetTrackTrace_parsesStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"tracktraces":[{"status":{"current":2,"main":"shipped"}}]}}`))
	}))
	defer srv.Close()

	c := myparcel.NewClient(myparcel.Config{APIKey: "k", BaseURL: srv.URL})
	result, err := c.GetTrackTrace(context.Background(), 123)

	require.NoError(t, err)
	assert.Equal(t, 2, result.StatusCode)
	assert.Equal(t, "shipped", result.StatusMain)
}

func TestClient_RegisterWebhook_registersEveryKnownHookType(t *testing.T) {
	var gotHooks []string
	var gotURLs []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		subs := body["data"].(map[string]any)["webhook_subscriptions"].([]any)
		require.Len(t, subs, 1)
		sub := subs[0].(map[string]any)
		gotHooks = append(gotHooks, sub["hook"].(string))
		gotURLs = append(gotURLs, sub["url"].(string))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := myparcel.NewClient(myparcel.Config{APIKey: "k", BaseURL: srv.URL})
	err := c.RegisterWebhook(context.Background(), "https://example.com/webhooks/v1/tenant/myparcel/abc")

	require.NoError(t, err)
	wantHooks := make([]string, len(integrations.AllMyParcelHookTypes))
	for i, h := range integrations.AllMyParcelHookTypes {
		wantHooks[i] = string(h)
	}
	assert.ElementsMatch(t, wantHooks, gotHooks)
	for _, u := range gotURLs {
		assert.Equal(t, "https://example.com/webhooks/v1/tenant/myparcel/abc", u)
	}
}

func TestClient_RegisterWebhook_oneHookFailingStillAttemptsRestAndReturnsError(t *testing.T) {
	var gotHooks []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		sub := body["data"].(map[string]any)["webhook_subscriptions"].([]any)[0].(map[string]any)
		hook := sub["hook"].(string)
		gotHooks = append(gotHooks, hook)
		if hook == "shop_updated" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := myparcel.NewClient(myparcel.Config{APIKey: "k", BaseURL: srv.URL})
	err := c.RegisterWebhook(context.Background(), "https://example.com/webhooks/v1/tenant/myparcel/abc")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "shop_updated")
	assert.Len(t, gotHooks, len(integrations.AllMyParcelHookTypes), "every hook type should still be attempted despite one failing")
}

func TestClient_CreateWebhookSubscription_sendsGivenHookAndParsesResponse(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"webhook_subscriptions":[{"id":42,"hook":"order_status_change","url":"https://example.com/hook"}]}}`))
	}))
	defer srv.Close()

	c := myparcel.NewClient(myparcel.Config{APIKey: "k", BaseURL: srv.URL})
	sub, err := c.CreateWebhookSubscription(context.Background(), "order_status_change", "https://example.com/hook")

	require.NoError(t, err)
	hooks := gotBody["data"].(map[string]any)["webhook_subscriptions"].([]any)
	assert.Equal(t, "order_status_change", hooks[0].(map[string]any)["hook"])
	assert.Equal(t, myparcel.WebhookSubscription{ID: "42", Hook: "order_status_change", URL: "https://example.com/hook"}, sub)
}

func TestClient_ListWebhookSubscriptions_parsesEveryHookType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/webhook_subscriptions", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"webhook_subscriptions":[
			{"id":1,"hook":"shipment_status_change","url":"https://a.example.com"},
			{"id":2,"hook":"shipment_label_created","url":"https://b.example.com"}
		]}}`))
	}))
	defer srv.Close()

	c := myparcel.NewClient(myparcel.Config{APIKey: "k", BaseURL: srv.URL})
	subs, err := c.ListWebhookSubscriptions(context.Background())

	require.NoError(t, err)
	require.Len(t, subs, 2)
	assert.Equal(t, myparcel.WebhookSubscription{ID: "1", Hook: "shipment_status_change", URL: "https://a.example.com"}, subs[0])
	assert.Equal(t, myparcel.WebhookSubscription{ID: "2", Hook: "shipment_label_created", URL: "https://b.example.com"}, subs[1])
}

func TestClient_DeleteWebhookSubscription_deletesByID(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := myparcel.NewClient(myparcel.Config{APIKey: "k", BaseURL: srv.URL})
	err := c.DeleteWebhookSubscription(context.Background(), "42")

	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, gotMethod)
	assert.Equal(t, "/webhook_subscriptions/42", gotPath)
}

func TestClient_DeleteShipment_deletesByID(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := myparcel.NewClient(myparcel.Config{APIKey: "k", BaseURL: srv.URL})
	err := c.DeleteShipment(context.Background(), 12345)

	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, gotMethod)
	assert.Equal(t, "/shipments/12345", gotPath)
}

func TestClient_DeleteShipment_afterLabelCreated_returnsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"errors":[{"message":"Shipment cannot be deleted"}]}`))
	}))
	defer srv.Close()

	c := myparcel.NewClient(myparcel.Config{APIKey: "k", BaseURL: srv.URL})
	err := c.DeleteShipment(context.Background(), 12345)

	require.Error(t, err)
	var apiErr *myparcel.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusUnprocessableEntity, apiErr.StatusCode)
}

func TestClient_GetDeliveryOptions_sendsExpectedQueryAndAcceptHeader(t *testing.T) {
	var gotPath, gotQuery, gotAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery, gotAccept = r.URL.Path, r.URL.RawQuery, r.Header.Get("Accept")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"delivery":[{"date":"2026-09-02","time":[
			{"price":{"currency":"EUR","amount":1000},"price_comment":"standard"},
			{"price":{"currency":"EUR","amount":1500},"price_comment":"morning"}
		]}],"pickup":[]}}`))
	}))
	defer srv.Close()

	c := myparcel.NewClient(myparcel.Config{APIKey: "k", BaseURL: srv.URL})
	options, err := c.GetDeliveryOptions(context.Background(), myparcel.DeliveryOptionsInput{
		CC: "NL", PostalCode: "1234 AB", Number: "31", Carrier: 1,
	})

	require.NoError(t, err)
	assert.Equal(t, "/delivery_options", gotPath)
	assert.Equal(t, "application/json;charset=utf-8;version=2.0", gotAccept)
	q, err := url.ParseQuery(gotQuery)
	require.NoError(t, err)
	assert.Equal(t, "NL", q.Get("cc"))
	assert.Equal(t, "1234 AB", q.Get("postal_code"))
	assert.Equal(t, "31", q.Get("number"))
	assert.Equal(t, "1", q.Get("carrier"))
	assert.Equal(t, "myparcel", q.Get("platform"))
	require.Len(t, options, 2)
	assert.Equal(t, myparcel.DeliveryPriceOption{PriceComment: "standard", AmountMinorUnits: 1000, Currency: "EUR"}, options[0])
	assert.Equal(t, myparcel.DeliveryPriceOption{PriceComment: "morning", AmountMinorUnits: 1500, Currency: "EUR"}, options[1])
}

func TestClient_GetDeliveryOptions_nonOKStatus_returnsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"errors":[{"message":"cc must be NL or BE"}]}`))
	}))
	defer srv.Close()

	c := myparcel.NewClient(myparcel.Config{APIKey: "k", BaseURL: srv.URL})
	_, err := c.GetDeliveryOptions(context.Background(), myparcel.DeliveryOptionsInput{
		CC: "NL", PostalCode: "1234 AB", Number: "31", Carrier: 1,
	})

	require.Error(t, err)
	var apiErr *myparcel.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusUnprocessableEntity, apiErr.StatusCode)
}
