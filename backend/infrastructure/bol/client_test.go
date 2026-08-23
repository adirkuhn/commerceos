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

func TestClient_CreateOffer_sendsExpectedPayloadAndHeaders(t *testing.T) {
	var gotPath, gotMethod, gotAccept, gotContentType string
	var gotBody bol.CreateOfferRequest
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 299})
		case "/v11/offers":
			gotPath = r.URL.Path
			gotMethod = r.Method
			gotAccept = r.Header.Get("Accept")
			gotContentType = r.Header.Get("Content-Type")
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"offerId": "offer-1"})
		}
	})

	offerID, err := client.CreateOffer(t.Context(), bol.CreateOfferRequest{
		EAN: "0630509892495", EconomicOperatorID: "op-1",
		Pricing:    bol.Pricing{BundlePrices: []bol.BundlePrice{{Quantity: 1, UnitPrice: 19.99}}},
		Fulfilment: bol.Fulfilment{Method: "FBR"},
		Condition:  bol.Condition{Category: "NEW"},
		Stock:      &bol.Stock{Amount: 5, ManagedByRetailer: true},
	})
	require.NoError(t, err)
	assert.Equal(t, "offer-1", offerID)
	assert.Equal(t, "/v11/offers", gotPath)
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "application/vnd.retailer.v11+json", gotAccept)
	assert.Equal(t, "application/vnd.retailer.v11+json", gotContentType)
	assert.Equal(t, "0630509892495", gotBody.EAN)
	assert.Equal(t, "op-1", gotBody.EconomicOperatorID)
}

func TestClient_CreateOffer_returnsAPIErrorOnFailureResponse(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 299})
		case "/v11/offers":
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"violations": []map[string]any{{"property": "ean", "reason": "unknown EAN"}}})
		}
	})

	_, err := client.CreateOffer(t.Context(), bol.CreateOfferRequest{EAN: "unknown"})
	require.Error(t, err)
	assert.True(t, bol.IsUnknownEAN(err), "a 400 mentioning an unknown EAN must be classified as ErrUnknownEAN so PushProductToBolUseCase knows to submit product content first")
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
