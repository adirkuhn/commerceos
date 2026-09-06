package payments_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/payments"
)

func TestStripeEventParser_checkoutSessionCompletedAndPaid_isHandled(t *testing.T) {
	p := payments.NewStripeEventParser()
	body := []byte(`{
		"type": "checkout.session.completed",
		"data": {"object": {"id": "cs_test_123", "client_reference_id": "WA-1A2B3C4D", "payment_status": "paid"}}
	}`)

	event, err := p.Parse(body)

	require.NoError(t, err)
	assert.True(t, event.Handled)
	assert.Equal(t, "WA-1A2B3C4D", event.OrderNumber)
	assert.Equal(t, "cs_test_123", event.ExternalPaymentID)
}

func TestStripeEventParser_checkoutSessionCompletedButUnpaid_isNotHandled(t *testing.T) {
	p := payments.NewStripeEventParser()
	body := []byte(`{
		"type": "checkout.session.completed",
		"data": {"object": {"id": "cs_test_123", "client_reference_id": "WA-1A2B3C4D", "payment_status": "unpaid"}}
	}`)

	event, err := p.Parse(body)

	require.NoError(t, err)
	assert.False(t, event.Handled)
}

func TestStripeEventParser_otherEventTypes_areNotHandled(t *testing.T) {
	p := payments.NewStripeEventParser()
	body := []byte(`{"type": "payment_intent.succeeded", "data": {"object": {}}}`)

	event, err := p.Parse(body)

	require.NoError(t, err)
	assert.False(t, event.Handled)
}

func TestStripeEventParser_malformedPayload_errors(t *testing.T) {
	p := payments.NewStripeEventParser()

	_, err := p.Parse([]byte(`not json`))

	require.Error(t, err)
}
