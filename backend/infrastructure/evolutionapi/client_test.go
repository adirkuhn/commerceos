package evolutionapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/infrastructure/evolutionapi"
)

func TestClient_SendText_sendsApiKeyHeaderAndBody(t *testing.T) {
	var gotAPIKey, gotPath string
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.Header.Get("apikey")
		gotPath = r.URL.Path
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := evolutionapi.NewClient(evolutionapi.Config{BaseURL: srv.URL, InstanceName: "tenant-1", APIKey: "secret-key"})
	err := c.SendText(context.Background(), "31612345678", "your order is on its way")

	require.NoError(t, err)
	assert.Equal(t, "secret-key", gotAPIKey)
	assert.Equal(t, "/send/text", gotPath)
	assert.Equal(t, "31612345678", gotBody["number"])
	assert.Equal(t, "your order is on its way", gotBody["text"])
}

func TestClient_SendText_nonSuccessStatus_returnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid apikey"}`))
	}))
	defer srv.Close()

	c := evolutionapi.NewClient(evolutionapi.Config{BaseURL: srv.URL, InstanceName: "tenant-1", APIKey: "wrong-key"})
	err := c.SendText(context.Background(), "31612345678", "hello")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}
