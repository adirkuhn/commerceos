package stripe_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/integrations"
	commercestripe "github.com/adirkuhn/commerceos/infrastructure/stripe"
)

func TestClient_CreatePaymentLink_sendsExpectedFormParamsAndParsesResponse(t *testing.T) {
	var gotAuth string
	var gotForm url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		require.NoError(t, r.ParseForm())
		gotForm = r.Form
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"cs_test_1","url":"https://checkout.stripe.com/pay/cs_test_1"}`))
	}))
	defer srv.Close()

	c := commercestripe.NewClient(commercestripe.Config{APIKey: "sk_test_secret", BaseURL: srv.URL})
	link, err := c.CreatePaymentLink(context.Background(), integrations.PaymentLinkRequest{
		AmountMinorUnits: 1999, CurrencyCode: "EUR", Description: "Order #WA-1", Reference: "WA-1",
		SuccessURL: "https://example.com/success", CancelURL: "https://example.com/cancel",
	})

	require.NoError(t, err)
	assert.Equal(t, "Bearer sk_test_secret", gotAuth)
	assert.Equal(t, "payment", gotForm.Get("mode"))
	assert.Equal(t, "https://example.com/success", gotForm.Get("success_url"))
	assert.Equal(t, "https://example.com/cancel", gotForm.Get("cancel_url"))
	assert.Equal(t, "WA-1", gotForm.Get("client_reference_id"))
	assert.Equal(t, "eur", gotForm.Get("line_items[0][price_data][currency]"), "currency must be lowercased for Stripe's API")
	assert.Equal(t, "1999", gotForm.Get("line_items[0][price_data][unit_amount]"))
	assert.Equal(t, "Order #WA-1", gotForm.Get("line_items[0][price_data][product_data][name]"))

	assert.Equal(t, "https://checkout.stripe.com/pay/cs_test_1", link.URL)
	assert.Equal(t, "cs_test_1", link.ExternalID)
}

func TestClient_CreatePaymentLink_nonSuccessStatus_returnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid api key","type":"invalid_request_error"}}`))
	}))
	defer srv.Close()

	c := commercestripe.NewClient(commercestripe.Config{APIKey: "sk_test_wrong", BaseURL: srv.URL})
	_, err := c.CreatePaymentLink(context.Background(), integrations.PaymentLinkRequest{
		AmountMinorUnits: 1000, CurrencyCode: "EUR", SuccessURL: "https://example.com/success", CancelURL: "https://example.com/cancel",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid api key")
}
