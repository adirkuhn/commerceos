package bol_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/infrastructure/bol"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *bol.Client) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client := bol.NewClient(bol.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		TokenURL:     srv.URL + "/token",
		BaseURL:      srv.URL,
	})
	return srv, client
}

func TestClient_GetAccessToken_sendsBasicAuthAndClientCredentialsGrant(t *testing.T) {
	var gotAuth, gotBody, gotAccept string
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/token" {
			return
		}
		gotAuth = r.Header.Get("Authorization")
		gotAccept = r.Header.Get("Accept")
		body := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "tok-123", "token_type": "Bearer", "expires_in": 299, "scope": "",
		})
	})

	tok, err := client.AccessToken(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "tok-123", tok)

	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("test-client-id:test-client-secret"))
	assert.Equal(t, wantAuth, gotAuth)
	assert.Equal(t, "application/json", gotAccept)
	assert.Contains(t, gotBody, "grant_type=client_credentials")
}

func TestClient_AccessToken_cachesUntilNearExpiry(t *testing.T) {
	calls := 0
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/token" {
			return
		}
		calls++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "tok", "token_type": "Bearer", "expires_in": 299,
		})
	})

	_, err := client.AccessToken(t.Context())
	require.NoError(t, err)
	_, err = client.AccessToken(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, calls, "second call within the token's lifetime must reuse the cached token, not refetch")
}

func TestClient_AccessToken_refetchesAfterExpiry(t *testing.T) {
	calls := 0
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/token" {
			return
		}
		calls++
		w.Header().Set("Content-Type", "application/json")
		// expires_in: 0 -- every subsequent call must be treated as
		// already-expired (this test doesn't want to actually sleep).
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "tok", "token_type": "Bearer", "expires_in": 0,
		})
	})

	_, err := client.AccessToken(t.Context())
	require.NoError(t, err)
	_, err = client.AccessToken(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 2, calls)
}

// TestClient_CreateOffer_sendsExpectedPayloadAndHeaders proves CreateOffer
// hits Bol's real confirmed path/method/version (POST /offers,
// application/vnd.retailer.v10+json -- verified against the actual
// Retailer API v10 OpenAPI spec, not a guess like the old /v11/offers path
// this replaces) and, since Bol's real response is 202 Accepted + a
// ProcessStatus (not an immediate offer id), polls until SUCCESS and
// returns the resulting entityId as the offer id.
func TestClient_CreateOffer_sendsExpectedPayloadAndHeaders(t *testing.T) {
	var gotPath, gotMethod, gotAccept, gotContentType string
	var gotBody bol.CreateOfferRequest
	pollCount := 0
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 299})
		case r.URL.Path == "/offers" && r.Method == http.MethodPost:
			gotPath = r.URL.Path
			gotMethod = r.Method
			gotAccept = r.Header.Get("Accept")
			gotContentType = r.Header.Get("Content-Type")
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{"processStatusId": "proc-1", "status": "PENDING"})
		case strings.HasPrefix(r.URL.Path, "/process-status/proc-1"):
			pollCount++
			status := "PENDING"
			if pollCount >= 2 {
				status = "SUCCESS"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"processStatusId": "proc-1", "status": status, "entityId": "offer-1"})
		}
	})

	offerID, err := client.CreateOffer(t.Context(), bol.CreateOfferRequest{
		EAN: "0630509892495", EconomicOperatorID: "op-1",
		Pricing:    bol.Pricing{BundlePrices: []bol.BundlePrice{{Quantity: 1, UnitPrice: 19.99}}},
		Fulfilment: bol.Fulfilment{Method: "FBR"},
		Condition:  bol.Condition{Category: "NEW"},
		Stock:      &bol.Stock{Amount: 5, ManagedByRetailer: true},
	}, bol.PollOptions{Interval: time.Millisecond, MaxAttempts: 5})
	require.NoError(t, err)
	assert.Equal(t, "offer-1", offerID)
	assert.Equal(t, "/offers", gotPath)
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "application/vnd.retailer.v10+json", gotAccept)
	assert.Equal(t, "application/vnd.retailer.v10+json", gotContentType)
	assert.Equal(t, "0630509892495", gotBody.EAN)
	assert.Equal(t, "op-1", gotBody.EconomicOperatorID)
	assert.GreaterOrEqual(t, pollCount, 2, "must actually poll, not treat the 202 as immediately final")
}

func TestClient_CreateOffer_returnsAPIErrorOnFailureResponse(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 299})
		case r.URL.Path == "/offers" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"violations": []map[string]any{{"property": "ean", "reason": "unknown EAN"}}})
		}
	})

	_, err := client.CreateOffer(t.Context(), bol.CreateOfferRequest{EAN: "unknown"}, bol.PollOptions{})
	require.Error(t, err)
	assert.True(t, bol.IsUnknownEAN(err), "a 400 mentioning an unknown EAN must be classified as ErrUnknownEAN so PushProductToBolUseCase knows to submit product content first")
}

// TestClient_CreateOffer_returnsErrorWhenProcessingFails proves a
// FAILURE/TIMEOUT terminal status (the offer request was accepted but Bol
// rejected it during processing -- a different failure mode than a 4xx on
// the initial POST) surfaces as a real Go error instead of silently
// returning an empty offer id, which is exactly how this bug hid in prod
// for as long as it did.
func TestClient_CreateOffer_returnsErrorWhenProcessingFails(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 299})
		case r.URL.Path == "/offers" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{"processStatusId": "proc-2", "status": "PENDING"})
		case strings.HasPrefix(r.URL.Path, "/process-status/proc-2"):
			_ = json.NewEncoder(w).Encode(map[string]any{"processStatusId": "proc-2", "status": "FAILURE", "errorMessage": "invalid category"})
		}
	})

	_, err := client.CreateOffer(t.Context(), bol.CreateOfferRequest{EAN: "0630509892495"}, bol.PollOptions{Interval: time.Millisecond, MaxAttempts: 5})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid category")
}

// TestClient_UpdateOffer_usesPutNotPatch proves UpdateOffer hits Bol's real
// confirmed method (PUT, not the old PATCH guess) and path (no /v11
// prefix), and also polls the resulting ProcessStatus to a terminal state.
func TestClient_UpdateOffer_usesPutNotPatch(t *testing.T) {
	var gotPath, gotMethod string
	pollCount := 0
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 299})
		case r.URL.Path == "/offers/offer-1":
			gotPath = r.URL.Path
			gotMethod = r.Method
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{"processStatusId": "proc-3", "status": "PENDING"})
		case strings.HasPrefix(r.URL.Path, "/process-status/proc-3"):
			pollCount++
			status := "PENDING"
			if pollCount >= 2 {
				status = "SUCCESS"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"processStatusId": "proc-3", "status": status, "entityId": "offer-1"})
		}
	})

	err := client.UpdateOffer(t.Context(), "offer-1", bol.CreateOfferRequest{EAN: "0630509892495"}, bol.PollOptions{Interval: time.Millisecond, MaxAttempts: 5})
	require.NoError(t, err)
	assert.Equal(t, "/offers/offer-1", gotPath)
	assert.Equal(t, http.MethodPut, gotMethod)
	assert.GreaterOrEqual(t, pollCount, 2)
}

func TestClient_SubmitProductContent_pollsUploadReportUntilTerminal(t *testing.T) {
	pollCount := 0
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 299})
		case r.URL.Path == "/product-content" && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"processStatusId": "upload-1"})
		case strings.HasPrefix(r.URL.Path, "/product-content/upload-1"):
			pollCount++
			status := "PENDING"
			if pollCount >= 2 {
				status = "SUCCESS"
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"processStatusId": "upload-1", "status": status})
		}
	})

	err := client.SubmitProductContent(t.Context(), bol.ProductContentRequest{
		Language: "nl",
		Attributes: []bol.ContentAttribute{
			{ID: "EAN", Values: []bol.AttributeValue{{Value: "0630509892495"}}},
		},
	}, bol.PollOptions{Interval: time.Millisecond, MaxAttempts: 5})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, pollCount, 2)
}

func TestClient_GetCommission_parsesResponse(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 299})
		case strings.HasPrefix(r.URL.Path, "/commission/"):
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ean": "0630509892495", "fixedAmount": 0.30, "percentage": 12.5,
			})
		}
	})

	commission, err := client.GetCommission(t.Context(), "0630509892495", 19.99)
	require.NoError(t, err)
	assert.Equal(t, "0630509892495", commission.EAN)
	assert.InDelta(t, 0.30, commission.FixedAmount, 0.001)
	assert.InDelta(t, 12.5, commission.Percentage, 0.001)
}
