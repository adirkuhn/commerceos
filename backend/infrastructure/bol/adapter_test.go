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
		EAN: "0630509892495", EconomicOperatorID: "op-1", ConditionCategory: "NEW", FulfilmentMethod: "FBR",
	})
	require.NoError(t, err)
	assert.Equal(t, "offer-1", offerID)

	condition, _ := gotBody["condition"].(map[string]any)
	require.NotNil(t, condition, "request body must include a condition object")
	assert.Equal(t, "NEW", condition["category"])

	fulfilment, _ := gotBody["fulfilment"].(map[string]any)
	require.NotNil(t, fulfilment, "request body must include a fulfilment object")
	assert.Equal(t, "BOL_DELIVERY_PROMISE", fulfilment["schedule"], "schedule is required by Bol for method FBR (confirmed via a real 400)")
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
}
