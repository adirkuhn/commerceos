package postnl_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/infrastructure/postnl"
)

var testCreds = integrationsDomain.CarrierCredentials{
	Provider: integrationsDomain.ProviderPostNL, APIKey: "secret-key", CustomerCode: "DEVC", CustomerNumber: "11223344",
}

func newClient(t *testing.T, handler http.HandlerFunc) integrationsDomain.CarrierClient {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client, err := postnl.NewCarrierClientFactory(postnl.WithBaseURL(srv.URL)).NewClient(testCreds)
	require.NoError(t, err)
	return client
}

func sender() *integrationsDomain.CarrierAddress {
	return &integrationsDomain.CarrierAddress{Company: "Cantinho", Street: "Damrak", Number: "1", PostalCode: "1012AB", City: "Amsterdam", CC: "NL", Email: "shop@example.com"}
}

func shipmentRequest() integrationsDomain.CreateCarrierShipmentRequest {
	weight := 1200
	insured := 2550
	return integrationsDomain.CreateCarrierShipmentRequest{
		Recipient: integrationsDomain.CarrierAddress{
			Person: "Jane van Dijk", Street: "Waldorpstraat", Number: "3", NumberSuffix: "bis", PostalCode: "2521CA",
			City: "Den Haag", CC: "NL", Email: "jane@example.com", Phone: "+31612345678",
		},
		Sender:  sender(),
		Carrier: 1,
		Options: integrationsDomain.CarrierShipmentOptions{
			PackageType: 1, SignatureRequired: true, OnlyRecipient: true, InsuranceAmountMinorUnits: &insured, Currency: "EUR",
		},
		OrderReference: "order-1",
		WeightGrams:    &weight,
		LabelFormat:    "pdf",
	}
}

func labelconfirmOK(t *testing.T, gotBody *map[string]any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/shipment/delivery/v4/labelconfirm", r.URL.Path)
		assert.Equal(t, "secret-key", r.Header.Get("apikey"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		raw, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(raw, gotBody))
		_, _ = w.Write([]byte(`{"traceId":"t-1","items":[{"barcode":"3SDEVC123456789","labels":[{"label":"` +
			base64.StdEncoding.EncodeToString([]byte("%PDF-1.4 fake")) + `","outputType":"pdf","labelType":"Label"}]}],"warnings":[]}`))
	}
}

func TestCreateShipment_sendsLabelconfirmRequestBuiltFromTheShipment(t *testing.T) {
	var body map[string]any
	client := newClient(t, labelconfirmOK(t, &body))

	_, err := client.CreateShipment(context.Background(), shipmentRequest())

	require.NoError(t, err)
	assert.Equal(t, "parcel", body["shipmentType"])
	assert.Equal(t, map[string]any{"outputType": "pdf"}, body["labelSettings"])
	assert.Equal(t, map[string]any{
		"customerCode": "DEVC", "customerNumber": "11223344",
		"address": map[string]any{"companyName": "Cantinho", "street": "Damrak", "houseNumber": "1", "postalCode": "1012AB", "city": "Amsterdam", "countryIso": "NL"},
		"contact": map[string]any{"email": "shop@example.com"},
	}, body["sender"])
	assert.Equal(t, map[string]any{
		"type": "consumer",
		"address": map[string]any{
			"street": "Waldorpstraat", "houseNumber": "3", "houseNumberAddition": "bis", "postalCode": "2521CA", "city": "Den Haag", "countryIso": "NL",
		},
		"contact": map[string]any{"email": "jane@example.com", "firstName": "Jane", "lastName": "van Dijk", "mobileNumber": "+31612345678"},
	}, body["receiver"])
	assert.Equal(t, map[string]any{"deliveryConfirmation": "signature", "statedAddressOnly": true, "insuredValue": 25.5}, body["services"])
	items := body["items"].([]any)
	require.Len(t, items, 1)
	item := items[0].(map[string]any)
	assert.Equal(t, map[string]any{"weight": float64(1200)}, item["dimensions"])
	refs := item["customerReferences"].(map[string]any)
	assert.Len(t, refs["shipmentReference"], 32, "a unique 32-char reference, used later to look up the status")
	assert.Equal(t, "order-1", refs["shipmentRemark"])
}

func TestCreateShipment_returnsBarcodeReferenceAndDecodedLabel(t *testing.T) {
	var body map[string]any
	client := newClient(t, labelconfirmOK(t, &body))

	result, err := client.CreateShipment(context.Background(), shipmentRequest())

	require.NoError(t, err)
	assert.Equal(t, "3SDEVC123456789", result.Barcode)
	sentRef := body["items"].([]any)[0].(map[string]any)["customerReferences"].(map[string]any)["shipmentReference"]
	assert.Equal(t, sentRef, result.ExternalShipmentID)
	require.NotNil(t, result.Label)
	assert.Equal(t, []byte("%PDF-1.4 fake"), result.Label.Data)
	assert.Equal(t, "application/pdf", result.Label.ContentType)
}

func TestCreateShipment_zplLabel(t *testing.T) {
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, map[string]any{"outputType": "zpl"}, body["labelSettings"])
		_, _ = w.Write([]byte(`{"items":[{"barcode":"3S1","labels":[{"label":"` + base64.StdEncoding.EncodeToString([]byte("^XA^XZ")) + `","outputType":"zpl"}]}]}`))
	})
	in := shipmentRequest()
	in.LabelFormat = "zpl"

	result, err := client.CreateShipment(context.Background(), in)

	require.NoError(t, err)
	assert.Equal(t, "application/vnd.shipment_label+zpl", result.Label.ContentType)
}

func TestCreateShipment_mailboxPackageIsLetterbox(t *testing.T) {
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "letterbox", body["shipmentType"])
		assert.Nil(t, body["services"], "no services requested")
		_, _ = w.Write([]byte(`{"items":[{"barcode":"3S1","labels":[{"label":"","outputType":"pdf"}]}]}`))
	})
	in := shipmentRequest()
	in.Options = integrationsDomain.CarrierShipmentOptions{PackageType: 2}

	_, err := client.CreateShipment(context.Background(), in)

	require.NoError(t, err)
}

func TestCreateShipment_companyRecipientIsBusiness(t *testing.T) {
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		receiver := body["receiver"].(map[string]any)
		assert.Equal(t, "business", receiver["type"])
		assert.Equal(t, "ACME BV", receiver["address"].(map[string]any)["companyName"])
		_, _ = w.Write([]byte(`{"items":[{"barcode":"3S1","labels":[]}]}`))
	})
	in := shipmentRequest()
	in.Recipient.Company = "ACME BV"

	_, err := client.CreateShipment(context.Background(), in)

	require.NoError(t, err)
}

func TestCreateShipment_withoutSenderAddressFailsBeforeCallingPostNL(t *testing.T) {
	client := newClient(t, func(http.ResponseWriter, *http.Request) { t.Fatal("PostNL must not be called") })
	in := shipmentRequest()
	in.Sender = nil

	_, err := client.CreateShipment(context.Background(), in)

	assert.ErrorIs(t, err, integrationsDomain.ErrCarrierSenderAddressMissing)
}

func TestCreateShipment_unsupportedPackageTypeFailsBeforeCallingPostNL(t *testing.T) {
	client := newClient(t, func(http.ResponseWriter, *http.Request) { t.Fatal("PostNL must not be called") })
	in := shipmentRequest()
	in.Options.PackageType = 5 // pallet

	_, err := client.CreateShipment(context.Background(), in)

	assert.ErrorIs(t, err, integrationsDomain.ErrCarrierOperationNotSupported)
}

func TestCreateShipment_businessErrorIncludesPostNLDetailAndTraceID(t *testing.T) {
	client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"type":"about:blank","title":"One or more business errors occurred","status":400,"errors":{"receiver.address.postalCode":["Invalid postal code"]},"traceId":"trace-9"}`))
	})

	_, err := client.CreateShipment(context.Background(), shipmentRequest())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 400")
	assert.Contains(t, err.Error(), "Invalid postal code")
	assert.Contains(t, err.Error(), "trace-9")
}

// Captured from the real sandbox on 2026-10-21 with a key whose account
// didn't match the customer code/number sent.
func TestCreateShipment_gatewayFaultIsReadable(t *testing.T) {
	client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"faults":[{"faultstring":"The user is not authorized for this operation based on the provided customerCode or customerNumber","detail":{"errorcode":"Validation Fault"}}]}`))
	})

	_, err := client.CreateShipment(context.Background(), shipmentRequest())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not authorized for this operation based on the provided customerCode or customerNumber")
}

func TestCreateShipment_unauthorized(t *testing.T) {
	client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"fault":{"faultstring":"Invalid ApiKey","detail":{"errorcode":"oauth.v2.InvalidApiKey"}}}`))
	})

	_, err := client.CreateShipment(context.Background(), shipmentRequest())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 401")
	assert.Contains(t, err.Error(), "Invalid ApiKey")
}

func TestNewClient_requiresCustomerCodeAndNumber(t *testing.T) {
	_, err := postnl.NewCarrierClientFactory().NewClient(integrationsDomain.CarrierCredentials{Provider: integrationsDomain.ProviderPostNL, APIKey: "k"})

	assert.Error(t, err)
}

func TestNotSupportedOperations(t *testing.T) {
	client := newClient(t, func(http.ResponseWriter, *http.Request) { t.Fatal("PostNL must not be called") })

	assert.ErrorIs(t, client.CancelShipment(context.Background(), "ref"), integrationsDomain.ErrCarrierOperationNotSupported)
	_, err := client.GetLabel(context.Background(), "ref", "pdf", "A4")
	assert.ErrorIs(t, err, integrationsDomain.ErrCarrierOperationNotSupported)
	_, err = client.GetDeliveryPriceEstimate(context.Background(), integrationsDomain.DeliveryPriceEstimateInput{})
	assert.ErrorIs(t, err, integrationsDomain.ErrCarrierOperationNotSupported)
}
