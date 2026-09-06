package payments_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/payments"
)

func TestRevolutEventParser_orderCompleted_isHandled(t *testing.T) {
	p := payments.NewRevolutEventParser()
	body := []byte(`{"event": "ORDER_COMPLETED", "order_id": "9fc01989-3f61-4484-a5d9-ffe768531be9", "merchant_order_ext_ref": "WA-1A2B3C4D"}`)

	event, err := p.Parse(body)

	require.NoError(t, err)
	assert.True(t, event.Handled)
	assert.Equal(t, "WA-1A2B3C4D", event.OrderNumber)
	assert.Equal(t, "9fc01989-3f61-4484-a5d9-ffe768531be9", event.ExternalPaymentID)
}

func TestRevolutEventParser_otherEvents_areNotHandled(t *testing.T) {
	p := payments.NewRevolutEventParser()
	body := []byte(`{"event": "ORDER_AUTHORISED", "order_id": "abc"}`)

	event, err := p.Parse(body)

	require.NoError(t, err)
	assert.False(t, event.Handled)
}

func TestRevolutEventParser_malformedPayload_errors(t *testing.T) {
	p := payments.NewRevolutEventParser()

	_, err := p.Parse([]byte(`not json`))

	require.Error(t, err)
}
