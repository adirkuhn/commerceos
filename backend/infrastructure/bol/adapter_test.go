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

// TestAdapter_CreateOffer_mapsConditionCategoryToConditionName is a
// regression test for a real prod 400 from Bol: "condition.name: The
// supplied value is empty, but is required." Bol's Condition object
// requires name (enum NEW/AS_NEW/GOOD/REASONABLE/MODERATE); category is
// optional and derived from name if omitted. The adapter was putting
// BolOfferInput.ConditionCategory (always "NEW" today, from
// channelsDomain.BolConditionNew) into Condition.category instead of
// Condition.name, so name was always sent empty and Bol rejected every
// offer -- this is the third real bug behind "product not sent to Bol"
// after the async-response fix and the Resync-never-bootstraps-first-
// listing fix.
func TestAdapter_CreateOffer_mapsConditionCategoryToConditionName(t *testing.T) {
	var gotBody map[string]any
	pollCount := 0
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 299})
		case r.URL.Path == "/offers" && r.Method == http.MethodPost:
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{"processStatusId": "proc-1", "status": "PENDING"})
		case r.URL.Path == "/process-status/proc-1":
			pollCount++
			status := "PENDING"
			if pollCount >= 2 {
				status = "SUCCESS"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"processStatusId": "proc-1", "status": status, "entityId": "offer-1"})
		}
	})
	adapter := bol.NewAdapter(client)

	_, err := adapter.CreateOffer(t.Context(), integrationsDomain.BolOfferInput{
		EAN: "0630509892495", EconomicOperatorID: "op-1", ConditionCategory: "NEW", FulfilmentMethod: "FBR", DeliveryCode: "3-5d",
	})
	require.NoError(t, err)

	condition, _ := gotBody["condition"].(map[string]any)
	require.NotNil(t, condition, "request body must include a condition object")
	assert.Equal(t, "NEW", condition["name"], "Condition.name is required by Bol -- must not be left empty")

	fulfilment, _ := gotBody["fulfilment"].(map[string]any)
	require.NotNil(t, fulfilment, "request body must include a fulfilment object")
	assert.Equal(t, "3-5d", fulfilment["deliveryCode"], "deliveryCode is required by Bol for method FBR")
}
