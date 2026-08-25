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
		ClientID:      "test-client-id",
		ClientSecret:  "test-client-secret",
		TokenURL:      srv.URL + "/token",
		BaseURL:       srv.URL,
		SharedBaseURL: srv.URL,
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
// hits Bol's real, CURRENT Offers API v11 (POST /offers,
// application/vnd.retailer.v11+json -- confirmed against the actual,
// currently-published Offers API v11.5.0 spec, found via Bol's public docs
// nav which lists "Offers API v11 (current)" vs "v10 (deprecated)") and
// that v11 is synchronous: 201 Created with the offer id directly in the
// body, no ProcessStatus/polling at all. The deprecated v10 endpoint this
// used to target accepted requests (202 + a real processStatusId) but its
// documented process-status polling 404'd in prod -- switching to the
// current, non-deprecated version sidesteps that entirely.
func TestClient_CreateOffer_sendsExpectedPayloadAndHeaders(t *testing.T) {
	var gotPath, gotMethod, gotAccept, gotContentType string
	var gotBody bol.CreateOfferRequest
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
			w.WriteHeader(http.StatusCreated)
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
	assert.Equal(t, "/offers", gotPath)
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "application/vnd.retailer.v11+json", gotAccept)
	assert.Equal(t, "application/vnd.retailer.v11+json", gotContentType)
	assert.Equal(t, "0630509892495", gotBody.EAN)
	assert.Equal(t, "op-1", gotBody.EconomicOperatorID)
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

	_, err := client.CreateOffer(t.Context(), bol.CreateOfferRequest{EAN: "unknown"})
	require.Error(t, err)
	assert.True(t, bol.IsUnknownEAN(err), "a 400 mentioning an unknown EAN must be classified as ErrUnknownEAN so PushProductToBolUseCase knows to submit product content first")
}

// TestClient_UpdateOffer_usesPatchNotPut proves UpdateOffer hits Bol's real
// confirmed method under v11 -- PATCH, not the earlier PUT guess -- and
// that it's synchronous (204 No Content, nothing to poll).
func TestClient_UpdateOffer_usesPatchNotPut(t *testing.T) {
	var gotPath, gotMethod, gotAccept string
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 299})
		case r.URL.Path == "/offers/offer-1":
			gotPath = r.URL.Path
			gotMethod = r.Method
			gotAccept = r.Header.Get("Accept")
			w.WriteHeader(http.StatusNoContent)
		}
	})

	err := client.UpdateOffer(t.Context(), "offer-1", bol.PatchOfferRequest{})
	require.NoError(t, err)
	assert.Equal(t, "/offers/offer-1", gotPath)
	assert.Equal(t, http.MethodPatch, gotMethod)
	assert.Equal(t, "application/vnd.retailer.v11+json", gotAccept)
}

// TestClient_FindOfferByEAN_matchesAnExistingOffer is a regression test
// for the "no link between our product and any Bol offer" gap: before
// this, PushProductToBolUseCase always assumed no offer exists yet
// whenever it had no local listing row, and would blindly POST a create
// -- which 409 Conflicts against an offer that already exists on Bol
// (created before this integration, or orphaned by a lost listing row).
// FindOfferByEAN lets the caller check first and link+update instead.
func TestClient_FindOfferByEAN_matchesAnExistingOffer(t *testing.T) {
	var gotPath, gotAccept string
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 299})
		case r.URL.Path == "/offers":
			gotPath = r.URL.Path + "?" + r.URL.RawQuery
			gotAccept = r.Header.Get("Accept")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"offers": []map[string]any{{"offerId": "offer-existing-1", "ean": "0630509892495"}},
			})
		}
	})

	offerID, found, err := client.FindOfferByEAN(t.Context(), "0630509892495")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "offer-existing-1", offerID)
	assert.Equal(t, "/offers?eans=0630509892495", gotPath)
	assert.Equal(t, "application/vnd.retailer.v11+json", gotAccept)
}

func TestClient_FindOfferByEAN_noMatchReturnsFalseNotError(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 299})
		case r.URL.Path == "/offers":
			_ = json.NewEncoder(w).Encode(map[string]any{"offers": []map[string]any{}})
		}
	})

	offerID, found, err := client.FindOfferByEAN(t.Context(), "0630509892495")
	require.NoError(t, err)
	assert.False(t, found)
	assert.Empty(t, offerID)
}

// TestClient_SubmitProductContent_pollsUploadReportUntilTerminal is also a
// path regression: prod hit a live 404 twice -- first because this used to
// call POST /product-content + GET /product-content/{id} (wrong resource
// name entirely), then after a first fix, POST /content/product (right
// resource, wrong pluralization -- taken from a stale third-party client's
// docs). Real paths, confirmed by downloading Bol's actual published specs
// directly (see client.go's SubmitProductContent doc comment): POST
// /content/products (plural) on the retailer API, GET
// /process-status/{id} on the separate shared API (SharedBaseURL, not
// BaseURL).
func TestClient_SubmitProductContent_pollsUploadReportUntilTerminal(t *testing.T) {
	pollCount := 0
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 299})
		case r.URL.Path == "/content/products" && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"processStatusId": "upload-1"})
		case strings.HasPrefix(r.URL.Path, "/process-status/upload-1"):
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

// TestClient_ListEconomicOperators_hitsRealPathAndHeader is a regression
// test proving this uses Bol's real, separate Economic Operator API v1
// (confirmed against the real downloaded spec: GET /economic-operators,
// Accept application/vnd.economic-operator.v1+json -- a different media
// type from every other endpoint in this client) rather than a guess.
func TestClient_ListEconomicOperators_hitsRealPathAndHeader(t *testing.T) {
	var gotPath, gotAccept string
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 299})
		case r.URL.Path == "/economic-operators":
			gotPath = r.URL.Path
			gotAccept = r.Header.Get("Accept")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"operators": []map[string]any{
					{"id": "3fa85f64-5717-4562-b3fc-2c963f66afa6", "name": "Cantinho Brasileiro", "status": "VALID"},
				},
			})
		}
	})

	operators, err := client.ListEconomicOperators(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "/economic-operators", gotPath)
	assert.Equal(t, "application/vnd.economic-operator.v1+json", gotAccept)
	require.Len(t, operators, 1)
	assert.Equal(t, "3fa85f64-5717-4562-b3fc-2c963f66afa6", operators[0].ID)
	assert.Equal(t, "Cantinho Brasileiro", operators[0].Name)
	assert.Equal(t, "VALID", operators[0].Status)
}
