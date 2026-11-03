package deadletter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/deadletter"
)

func TestParseStream(t *testing.T) {
	for _, s := range []string{"events", "webhooks"} {
		got, err := deadletter.ParseStream(s)
		require.NoError(t, err)
		assert.Equal(t, deadletter.Stream(s), got)
	}

	_, err := deadletter.ParseStream("orders")
	assert.ErrorIs(t, err, deadletter.ErrInvalidStream)
}

func TestIdentify_domainEvent(t *testing.T) {
	eventID, tenantID := deadletter.Identify([]byte(`{"event_id":"e-1","tenant_id":"t-1","sales_order_id":"so-1"}`))

	assert.Equal(t, "e-1", eventID)
	assert.Equal(t, "t-1", tenantID)
}

// A webhook dead letter is a webhooks.Envelope, marshalled without json tags.
func TestIdentify_webhookEnvelope(t *testing.T) {
	eventID, tenantID := deadletter.Identify([]byte(`{"EventID":"sha-1","TenantID":"t-2","Provider":"woocommerce"}`))

	assert.Equal(t, "sha-1", eventID)
	assert.Equal(t, "t-2", tenantID)
}

func TestIdentify_notJSON(t *testing.T) {
	eventID, tenantID := deadletter.Identify([]byte(`not json`))

	assert.Empty(t, eventID)
	assert.Empty(t, tenantID)
}

func TestDeadLetter_ConsumerKnown(t *testing.T) {
	assert.True(t, deadletter.DeadLetter{Consumer: "c"}.ConsumerKnown())
	assert.False(t, deadletter.DeadLetter{}.ConsumerKnown(), "dead-lettered before failure headers existed")
}

func TestFilter_Matches(t *testing.T) {
	dl := deadletter.DeadLetter{Subject: "oms.sales_order.status_changed", Consumer: "c1", TenantID: "t-1"}

	assert.True(t, deadletter.Filter{}.Matches(dl), "empty filter matches all")
	assert.True(t, deadletter.Filter{Subject: "oms.sales_order.status_changed", Consumer: "c1", TenantID: "t-1"}.Matches(dl))
	assert.False(t, deadletter.Filter{Subject: "pms.product.changed"}.Matches(dl))
	assert.False(t, deadletter.Filter{Consumer: "c2"}.Matches(dl))
	assert.False(t, deadletter.Filter{TenantID: "t-2"}.Matches(dl))
}

func TestFilter_LimitOrDefault(t *testing.T) {
	assert.Equal(t, deadletter.DefaultLimit, deadletter.Filter{}.LimitOrDefault())
	assert.Equal(t, 10, deadletter.Filter{Limit: 10}.LimitOrDefault())
	assert.Equal(t, deadletter.MaxLimit, deadletter.Filter{Limit: deadletter.MaxLimit + 1}.LimitOrDefault())
}
