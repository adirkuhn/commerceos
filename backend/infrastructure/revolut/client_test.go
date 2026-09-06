package revolut_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/infrastructure/revolut"
)

func TestClient_CreatePaymentLink_sendsAuthHeaderAndBody_usesCheckoutURLDirectly(t *testing.T) {
	var gotAuth string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"order-1","public_id":"pub-1","checkout_url":"https://checkout.revolut.com/pay/order-1"}`))
	}))
	defer srv.Close()

	c := revolut.NewClient(revolut.Config{APIKey: "secret-key", BaseURL: srv.URL})
	link, err := c.CreatePaymentLink(context.Background(), integrations.PaymentLinkRequest{
		AmountMinorUnits: 1999, CurrencyCode: "EUR", Description: "Order #WA-1", Reference: "WA-1",
	})

	require.NoError(t, err)
	assert.Equal(t, "Bearer secret-key", gotAuth)
	assert.Equal(t, float64(1999), gotBody["amount"])
	assert.Equal(t, "EUR", gotBody["currency"])
	assert.Equal(t, "https://checkout.revolut.com/pay/order-1", link.URL)
	assert.Equal(t, "order-1", link.ExternalID)
}

func TestClient_CreatePaymentLink_noCheckoutURL_fallsBackToPublicIDConstruction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"order-1","public_id":"pub-1"}`))
	}))
	defer srv.Close()

	c := revolut.NewClient(revolut.Config{APIKey: "secret-key", BaseURL: srv.URL})
	link, err := c.CreatePaymentLink(context.Background(), integrations.PaymentLinkRequest{AmountMinorUnits: 1000, CurrencyCode: "EUR"})

	require.NoError(t, err)
	assert.Equal(t, "https://checkout.revolut.com/payment-link/pub-1", link.URL)
}

func TestClient_CreatePaymentLink_nonSuccessStatus_returnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"invalid api key"}`))
	}))
	defer srv.Close()

	c := revolut.NewClient(revolut.Config{APIKey: "wrong-key", BaseURL: srv.URL})
	_, err := c.CreatePaymentLink(context.Background(), integrations.PaymentLinkRequest{AmountMinorUnits: 1000, CurrencyCode: "EUR"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}
