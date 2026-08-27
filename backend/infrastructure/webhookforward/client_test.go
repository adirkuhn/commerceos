package webhookforward_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/infrastructure/webhookforward"
)

func TestClient_Post_sendsBodyAndContentType_returnsStatus(t *testing.T) {
	var gotBody []byte
	var gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		gotContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusTeapot)
	}))
	defer srv.Close()

	c := webhookforward.NewClient()
	status, err := c.Post(context.Background(), srv.URL, "application/json", []byte(`{"id":1}`))

	require.NoError(t, err)
	assert.Equal(t, http.StatusTeapot, status)
	assert.Equal(t, []byte(`{"id":1}`), gotBody)
	assert.Equal(t, "application/json", gotContentType)
}

func TestClient_Post_unreachableTarget_returnsError(t *testing.T) {
	c := webhookforward.NewClient()
	_, err := c.Post(context.Background(), "http://127.0.0.1:1", "application/json", []byte(`{}`))

	require.Error(t, err)
}
