package bol_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/infrastructure/bol"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

// TestAdapter_CreateOffer_mapsConditionCategory proves the adapter sends
// Condition.category (Bol's real v11 discriminated-union field for a NEW
// condition offer -- confirmed against the current, non-deprecated Offers
// API v11.5.0 spec) and that CreateOffer talks to POST /offers
// synchronously (201, offer id directly in the body -- no polling).
func TestAdapter_CreateOffer_mapsConditionCategory(t *testing.T) {
	var gotBody map[string]any
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 299})
		case r.URL.Path == "/offers" && r.Method == http.MethodPost:
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"offerId": "offer-1"})
		}
	})
	adapter := bol.NewAdapter(client)

	offerID, err := adapter.CreateOffer(t.Context(), integrationsDomain.BolOfferInput{
		EAN: "0630509892495", EconomicOperatorID: "op-1", ConditionCategory: "NEW", FulfilmentMethod: "FBR", DeliveryCode: "1-2d",
	})
	require.NoError(t, err)
	assert.Equal(t, "offer-1", offerID)

	condition, _ := gotBody["condition"].(map[string]any)
	require.NotNil(t, condition, "request body must include a condition object")
	assert.Equal(t, "NEW", condition["category"])

	fulfilment, _ := gotBody["fulfilment"].(map[string]any)
	require.NotNil(t, fulfilment, "request body must include a fulfilment object")
	assert.Equal(t, "BOL_DELIVERY_PROMISE", fulfilment["schedule"], "schedule is required by Bol for method FBR (confirmed via a real 400)")

	deliveryPromise, _ := fulfilment["deliveryPromise"].(map[string]any)
	require.NotNil(t, deliveryPromise, "deliveryPromise is required by Bol whenever schedule is BOL_DELIVERY_PROMISE (confirmed via a real 400)")
	assert.Equal(t, float64(1), deliveryPromise["minimumDaysToCustomer"])
	assert.Equal(t, float64(2), deliveryPromise["maximumDaysToCustomer"])
}

// TestAdapter_CreateOffer_deliveryPromiseFallsBackWhenDeliveryCodeUnparseable
// proves a stored delivery-code value that isn't a plain "N-Md" day range
// (e.g. leftover from Bol's old v10-based enum, or simply unset) still
// produces a valid deliveryPromise instead of an empty/invalid one that
// would 400 -- 3-5 days, matching DefaultBolDeliveryCode.
func TestAdapter_CreateOffer_deliveryPromiseFallsBackWhenDeliveryCodeUnparseable(t *testing.T) {
	var gotBody map[string]any
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 299})
		case r.URL.Path == "/offers" && r.Method == http.MethodPost:
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"offerId": "offer-1"})
		}
	})
	adapter := bol.NewAdapter(client)

	_, err := adapter.CreateOffer(t.Context(), integrationsDomain.BolOfferInput{
		EAN: "0630509892495", ConditionCategory: "NEW", FulfilmentMethod: "FBR", DeliveryCode: "VVB",
	})
	require.NoError(t, err)

	fulfilment := gotBody["fulfilment"].(map[string]any)
	deliveryPromise := fulfilment["deliveryPromise"].(map[string]any)
	assert.Equal(t, float64(3), deliveryPromise["minimumDaysToCustomer"])
	assert.Equal(t, float64(5), deliveryPromise["maximumDaysToCustomer"])
}

// TestAdapter_UpdateOffer_omitsCondition proves PATCH never sends a
// condition field -- Bol's real PatchOfferRequest schema has none (an
// offer's condition can't be changed after creation).
func TestAdapter_UpdateOffer_omitsCondition(t *testing.T) {
	var gotBody map[string]any
	var gotMethod string
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 299})
		case r.URL.Path == "/offers/offer-1":
			gotMethod = r.Method
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			w.WriteHeader(http.StatusNoContent)
		}
	})
	adapter := bol.NewAdapter(client)

	err := adapter.UpdateOffer(t.Context(), "offer-1", integrationsDomain.BolOfferInput{
		EAN: "0630509892495", EconomicOperatorID: "op-1", ConditionCategory: "NEW", FulfilmentMethod: "FBR",
	})
	require.NoError(t, err)
	assert.Equal(t, http.MethodPatch, gotMethod)
	_, hasCondition := gotBody["condition"]
	assert.False(t, hasCondition, "PATCH must never send condition -- Bol's schema has no such field")

	fulfilment, _ := gotBody["fulfilment"].(map[string]any)
	require.NotNil(t, fulfilment)
	assert.Equal(t, "BOL_DELIVERY_PROMISE", fulfilment["schedule"])
	require.NotNil(t, fulfilment["deliveryPromise"])
}

func TestAdapter_CreateAndUpdateOffer_sendEveryPriceLevelAsBundlePrices(t *testing.T) {
	var bodies []map[string]any
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 299})
		case r.URL.Path == "/offers":
			var b map[string]any
			_ = json.NewDecoder(r.Body).Decode(&b)
			bodies = append(bodies, b)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"offerId": "offer-1"})
		case r.URL.Path == "/offers/offer-1":
			var b map[string]any
			_ = json.NewDecoder(r.Body).Decode(&b)
			bodies = append(bodies, b)
			w.WriteHeader(http.StatusNoContent)
		}
	})
	adapter := bol.NewAdapter(client)
	in := integrationsDomain.BolOfferInput{
		EAN: "0630509892495", ConditionCategory: "NEW", FulfilmentMethod: "FBR",
		Prices: []integrationsDomain.BolPrice{{Quantity: 1, UnitPrice: 38.49}, {Quantity: 2, UnitPrice: 35.49}},
	}

	_, err := adapter.CreateOffer(t.Context(), in)
	require.NoError(t, err)
	require.NoError(t, adapter.UpdateOffer(t.Context(), "offer-1", in))

	want := []any{
		map[string]any{"quantity": float64(1), "unitPrice": 38.49},
		map[string]any{"quantity": float64(2), "unitPrice": 35.49},
	}
	require.Len(t, bodies, 2)
	for _, b := range bodies {
		pricing, _ := b["pricing"].(map[string]any)
		require.NotNil(t, pricing)
		assert.Equal(t, want, pricing["bundlePrices"])
	}
}
