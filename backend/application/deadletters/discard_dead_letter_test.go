package deadletters_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/deadletters"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"github.com/adirkuhn/commerceos/domain/deadletter"
)

func TestDiscardDeadLetter_deletesExactlyOneAndAudits(t *testing.T) {
	other := sampleDeadLetter()
	other.Sequence = 8
	store, outbox := storeWith(sampleDeadLetter(), other), &eventrecordtest.Outbox{}
	uc := deadletters.NewDiscardDeadLetterUseCase(mockDB(t, 1), store, outbox)

	require.NoError(t, uc.Execute(superAdmin(), deadletter.StreamEvents, 7))

	assert.Equal(t, []uint64{7}, store.deleted)
	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "deadletters.dead_letter.discarded", e.Subject)
	assert.Equal(t, "e-1", e.Payload["entity_id"])
	assert.Equal(t, "t-1", e.Payload["tenant_id"])
	assert.Equal(t, "u-admin", e.Payload["actor"].(map[string]any)["user_id"])
	assert.Equal(t, map[string]any{
		"stream": "events", "sequence": float64(7), "subject": "oms.sales_order.status_changed",
		"consumer": "sales-order-status-changed-scheduler", "error": "woo: 403",
	}, e.Data())
}

func TestDiscardDeadLetter_deleteFailure(t *testing.T) {
	store, outbox := storeWith(sampleDeadLetter()), &eventrecordtest.Outbox{}
	store.deleteErr = errBoom
	uc := deadletters.NewDiscardDeadLetterUseCase(mockDB(t, 0), store, outbox)

	assert.ErrorIs(t, uc.Execute(superAdmin(), deadletter.StreamEvents, 7), errBoom)
	assert.Empty(t, outbox.Subjects())
}

func TestDiscardDeadLetter_withoutTenantRaisesNoEvent(t *testing.T) {
	dl := sampleDeadLetter()
	dl.TenantID = ""
	store, outbox := storeWith(dl), &eventrecordtest.Outbox{}
	uc := deadletters.NewDiscardDeadLetterUseCase(mockDB(t, 0), store, outbox)

	require.NoError(t, uc.Execute(superAdmin(), deadletter.StreamEvents, 7))

	assert.Equal(t, []uint64{7}, store.deleted)
	assert.Empty(t, outbox.Subjects())
}

func TestDiscardDeadLetter_notFound(t *testing.T) {
	store := storeWith()
	uc := deadletters.NewDiscardDeadLetterUseCase(mockDB(t, 0), store, &eventrecordtest.Outbox{})

	assert.ErrorIs(t, uc.Execute(superAdmin(), deadletter.StreamEvents, 7), deadletter.ErrNotFound)
	assert.Empty(t, store.deleted)
}

func TestDiscardDeadLetter_rejectsUnknownStream(t *testing.T) {
	uc := deadletters.NewDiscardDeadLetterUseCase(mockDB(t, 0), storeWith(), &eventrecordtest.Outbox{})

	assert.ErrorIs(t, uc.Execute(superAdmin(), "orders", 7), deadletter.ErrInvalidStream)
}

func TestDiscardDeadLetter_onlySuperAdmin(t *testing.T) {
	store := storeWith(sampleDeadLetter())
	uc := deadletters.NewDiscardDeadLetterUseCase(mockDB(t, 0), store, &eventrecordtest.Outbox{})

	assert.ErrorIs(t, uc.Execute(context.Background(), deadletter.StreamEvents, 7), deadletter.ErrForbidden)
	assert.Empty(t, store.deleted)
}
