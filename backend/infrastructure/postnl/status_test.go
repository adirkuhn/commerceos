package postnl_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func statusBody(code string) string {
	return fmt.Sprintf(`{"CurrentStatus":{"Shipment":{"MainBarcode":"3SDEVC288882229","Barcode":"3SDEVC288882229","Reference":"ref-1",
		"Status":{"TimeStamp":"08-11-2022 10:13:20","StatusCode":%q,"StatusDescription":"x","PhaseCode":"3","PhaseDescription":"Distribution"}}}}`, code)
}

func TestGetTrackTrace_looksUpByReferenceWithCustomerAccount(t *testing.T) {
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/shipment/v2/status/reference/ref-1", r.URL.Path)
		assert.Equal(t, "DEVC", r.URL.Query().Get("customerCode"))
		assert.Equal(t, "11223344", r.URL.Query().Get("customerNumber"))
		assert.Equal(t, "secret-key", r.Header.Get("apikey"))
		_, _ = w.Write([]byte(statusBody("7")))
	})

	tt, err := client.GetTrackTrace(context.Background(), "ref-1")

	require.NoError(t, err)
	assert.Equal(t, "distribution", tt.StatusMain)
	assert.Equal(t, 5, tt.StatusCode, "PostNL 7 is MyParcel's 5 (enroute - distribution)")
}

func TestGetTrackTrace_mapsPostNLCodesOntoTheSharedScale(t *testing.T) {
	cases := []struct {
		postnl string
		main   string
		code   int
	}{
		{"1", "registered", 2}, {"13", "registered", 2},
		{"2", "handed_to_carrier", 3}, {"3", "handed_to_carrier", 3},
		{"5", "sorting", 4}, {"21", "sorting", 4},
		{"7", "distribution", 5}, {"8", "distribution", 5},
		{"9", "customs", 6},
		{"11", "delivered", 7}, {"12", "delivered", 8}, {"22", "delivered", 9},
		{"99", "registered", 2},
	}
	for _, tc := range cases {
		t.Run("postnl "+tc.postnl, func(t *testing.T) {
			client := newClient(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(statusBody(tc.postnl))) })

			tt, err := client.GetTrackTrace(context.Background(), "ref-1")

			require.NoError(t, err)
			assert.Equal(t, tc.main, tt.StatusMain)
			assert.Equal(t, tc.code, tt.StatusCode)
		})
	}
}

func TestGetTrackTrace_notYetKnownToPostNLIsRegistered(t *testing.T) {
	client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"Warnings":[{"Code":"N01","Description":"Shipment not found"}]}`))
	})

	tt, err := client.GetTrackTrace(context.Background(), "ref-1")

	require.NoError(t, err)
	assert.Equal(t, "registered", tt.StatusMain)
	assert.Equal(t, 2, tt.StatusCode)
}

func TestGetShipmentBarcode_isTheMainBarcode(t *testing.T) {
	client := newClient(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(statusBody("2"))) })

	barcode, err := client.GetShipmentBarcode(context.Background(), "ref-1")

	require.NoError(t, err)
	assert.Equal(t, "3SDEVC288882229", barcode)
}

func TestGetTrackTrace_httpErrorPropagates(t *testing.T) {
	client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"Errors":[{"ErrorMsg":"boom"}]}`))
	})

	_, err := client.GetTrackTrace(context.Background(), "ref-1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 500")
}

// Captured from the real sandbox on 2026-10-21 right after creating a label.
func TestGetTrackTrace_realSandboxNoShipmentFoundYet(t *testing.T) {
	client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"CurrentStatus":{},"Warnings":[{"Message":"No shipment found","Code":"2"}]}`))
	})

	tt, err := client.GetTrackTrace(context.Background(), "ref-1")
	require.NoError(t, err)
	assert.Equal(t, "registered", tt.StatusMain)
	barcode, err := client.GetShipmentBarcode(context.Background(), "ref-1")
	require.NoError(t, err)
	assert.Empty(t, barcode)
}
