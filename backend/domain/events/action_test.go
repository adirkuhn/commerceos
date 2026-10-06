package events_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/events"
)

func TestEntityActionCarriesItsSubjectAndEnvelope(t *testing.T) {
	e := events.EntityAction{
		Base: events.Base{EventID: "ev-1"}, Type: events.PurchaseOrderCreated, TenantID: "t-1",
		Actor: events.Actor{Type: events.ActorUser, UserID: "u-1", Role: "customer_admin"}, EntityID: "po-1",
		Changes: map[string]events.FieldChange{"status": {Before: "draft", After: "sent"}},
		Data:    map[string]any{"source": "purchase_planner"},
	}

	assert.Equal(t, "procurement.purchase_order.created", e.EventName())
	raw, err := json.Marshal(e)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, "procurement.purchase_order.created", got["event"])
	assert.Equal(t, "ev-1", got["event_id"])
	assert.Equal(t, "po-1", got["entity_id"])
	assert.Equal(t, map[string]any{"type": "user", "user_id": "u-1", "role": "customer_admin"}, got["actor"])
	assert.Equal(t, map[string]any{"status": map[string]any{"before": "draft", "after": "sent"}}, got["changes"])
	assert.Equal(t, map[string]any{"source": "purchase_planner"}, got["data"])
}

func TestFindActionType(t *testing.T) {
	a, ok := events.FindActionType("procurement.purchase_order.created")
	require.True(t, ok)
	assert.Equal(t, events.PurchaseOrderCreated, a)
	_, ok = events.FindActionType("nope.nope.created")
	assert.False(t, ok)
}
