package eventrecord_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type outbox struct {
	subjects, ids []string
	payloads      [][]byte
	err           error
}

func (o *outbox) Enqueue(_ context.Context, subject, eventID string, payload []byte, _ time.Time) error {
	if o.err != nil {
		return o.err
	}
	o.subjects, o.ids, o.payloads = append(o.subjects, subject), append(o.ids, eventID), append(o.payloads, payload)
	return nil
}

func decode(t *testing.T, raw []byte) map[string]any {
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m))
	return m
}

func TestRecord_UserAction(t *testing.T) {
	ctx := tenant.WithUserID(tenant.WithRole(tenant.WithTenantID(context.Background(), "t-1"), tenant.RoleCustomerAdmin), "u-1")
	o := &outbox{}

	err := eventrecord.Record(ctx, o, events.PurchaseOrderCreated, "po-1",
		eventrecord.Data{"source": "manual"},
		eventrecord.NewChanges().Set("status", "draft", "sent").Set("notes", "x", "x"))

	require.NoError(t, err)
	require.Equal(t, []string{"procurement.purchase_order.created"}, o.subjects)
	got := decode(t, o.payloads[0])
	assert.Equal(t, o.ids[0], got["event_id"])
	assert.Equal(t, "t-1", got["tenant_id"])
	assert.Equal(t, "po-1", got["entity_id"])
	assert.Equal(t, map[string]any{"type": "user", "user_id": "u-1", "role": "customer_admin"}, got["actor"])
	assert.Equal(t, map[string]any{"source": "manual"}, got["data"])
	// Unchanged fields are left out.
	assert.Equal(t, map[string]any{"status": map[string]any{"before": "draft", "after": "sent"}}, got["changes"])
	assert.NotEmpty(t, got["occurred_at"])
}

func TestRecord_SystemActionWithoutUser(t *testing.T) {
	o := &outbox{}

	require.NoError(t, eventrecord.Record(tenant.WithTenantID(context.Background(), "t-1"), o, events.PurchaseOrderCreated, "po-1"))

	got := decode(t, o.payloads[0])
	assert.Equal(t, map[string]any{"type": "system"}, got["actor"])
	assert.NotContains(t, got, "changes")
	assert.NotContains(t, got, "data")
}

func TestRecord_OutboxError(t *testing.T) {
	boom := errors.New("boom")
	err := eventrecord.Record(context.Background(), &outbox{err: boom}, events.PurchaseOrderCreated, "po-1")
	assert.ErrorIs(t, err, boom)
}

func TestChanges_PointersCompareByValue(t *testing.T) {
	a, b, c := "x", "x", "y"
	ch := eventrecord.NewChanges().Set("same", &a, &b).Set("diff", &a, &c).Set("nil", (*string)(nil), &c)
	assert.Equal(t, []string{"diff", "nil"}, ch.Fields())
}
