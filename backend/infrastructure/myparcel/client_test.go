package myparcel_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/infrastructure/myparcel"
)

func TestClient_CreateShipment_sendsAuthHeaderAndParsesResponse(t *testing.T) {
	var gotAuth, gotAccept string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAccept = r.Header.Get("Accept")
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"shipments":[{"id":123,"barcode":"3SMYPA1"}]}}`))
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

func TestClient_CreateShipment_insuredShipmentSendsInsuranceOption(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"shipments":[{"id":123,"barcode":"3SMYPA1"}]}}`))
	}))
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
	assert.Equal(t, float64(50000), insurance["amount"])
	assert.Equal(t, "EUR", insurance["currency"])
}

func TestClient_CreateShipment_noInsuranceOmitsInsuranceOption(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"shipments":[{"id":123,"barcode":"3SMYPA1"}]}}`))
	}))
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
	result, err := c.GetLabel(context.Background(), 123, "pdf")

	require.NoError(t, err)
	assert.Equal(t, "application/pdf", gotAccept)
	assert.Equal(t, "application/pdf", result.ContentType)
	assert.Equal(t, []byte("%PDF-fake"), result.Data)
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
	_, err := c.GetLabel(context.Background(), 123, "zpl")

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

func TestClient_RegisterWebhook_sendsCorrectHookType(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := myparcel.NewClient(myparcel.Config{APIKey: "k", BaseURL: srv.URL})
	err := c.RegisterWebhook(context.Background(), "https://example.com/webhooks/v1/tenant/myparcel?token=abc")

	require.NoError(t, err)
	hooks := gotBody["data"].(map[string]any)["hooks"].([]any)
	require.Len(t, hooks, 1)
	assert.Equal(t, "shipment_status_change", hooks[0].(map[string]any)["hook"])
	assert.Equal(t, "https://example.com/webhooks/v1/tenant/myparcel?token=abc", hooks[0].(map[string]any)["url"])
}
