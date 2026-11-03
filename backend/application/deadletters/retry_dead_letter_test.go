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

func TestRetryDeadLetter_republishesRemovesAndAudits(t *testing.T) {
	dl := sampleDeadLetter()
	store, pub, outbox := storeWith(dl), &stubPublisher{}, &eventrecordtest.Outbox{}
	uc := deadletters.NewRetryDeadLetterUseCase(mockDB(t, 1), store, pub, outbox)

	require.NoError(t, uc.Execute(superAdmin(), deadletter.StreamEvents, 7))

	assert.Equal(t, []deadletter.DeadLetter{dl}, pub.published)
	assert.Equal(t, []uint64{7}, store.deleted)
	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "deadletters.dead_letter.retried", e.Subject)
	assert.Equal(t, "e-1", e.Payload["entity_id"])
	assert.Equal(t, "t-1", e.Payload["tenant_id"], "audited in the dead letter's own tenant")
	assert.Equal(t, "u-admin", e.Payload["actor"].(map[string]any)["user_id"])
	assert.Equal(t, map[string]any{
		"stream": "events", "sequence": float64(7), "subject": "oms.sales_order.status_changed",
		"consumer": "sales-order-status-changed-scheduler",
	}, e.Data())
}

// A failed republish must leave the dead letter where it is (COM-14).
func TestRetryDeadLetter_publishFailureKeepsTheDeadLetter(t *testing.T) {
	store, pub, outbox := storeWith(sampleDeadLetter()), &stubPublisher{err: errBoom}, &eventrecordtest.Outbox{}
	uc := deadletters.NewRetryDeadLetterUseCase(mockDB(t, 0), store, pub, outbox)

	err := uc.Execute(superAdmin(), deadletter.StreamEvents, 7)

	assert.ErrorIs(t, err, deadletter.ErrRetryFailed)
	assert.ErrorIs(t, err, errBoom)
	assert.Empty(t, store.deleted)
	assert.Empty(t, outbox.Subjects())
}

func TestRetryDeadLetter_deleteFailureIsReported(t *testing.T) {
	store, outbox := storeWith(sampleDeadLetter()), &eventrecordtest.Outbox{}
	store.deleteErr = errBoom
	uc := deadletters.NewRetryDeadLetterUseCase(mockDB(t, 0), store, &stubPublisher{}, outbox)

	err := uc.Execute(superAdmin(), deadletter.StreamEvents, 7)

	assert.ErrorIs(t, err, errBoom)
	assert.Empty(t, outbox.Subjects())
}

// A dead letter with no tenant (e.g. a platform-wide event) is retried
// without an audit event: there is no tenant to raise it in.
func TestRetryDeadLetter_withoutTenantRaisesNoEvent(t *testing.T) {
	dl := sampleDeadLetter()
	dl.TenantID = ""
	store, pub, outbox := storeWith(dl), &stubPublisher{}, &eventrecordtest.Outbox{}
	uc := deadletters.NewRetryDeadLetterUseCase(mockDB(t, 0), store, pub, outbox)

	require.NoError(t, uc.Execute(superAdmin(), deadletter.StreamEvents, 7))

	assert.Len(t, pub.published, 1)
	assert.Equal(t, []uint64{7}, store.deleted)
	assert.Empty(t, outbox.Subjects())
}

func TestRetryDeadLetter_notFound(t *testing.T) {
	pub := &stubPublisher{}
	uc := deadletters.NewRetryDeadLetterUseCase(mockDB(t, 0), storeWith(), pub, &eventrecordtest.Outbox{})

	err := uc.Execute(superAdmin(), deadletter.StreamEvents, 99)

	assert.ErrorIs(t, err, deadletter.ErrNotFound)
	assert.Empty(t, pub.published)
}

func TestRetryDeadLetter_rejectsUnknownStream(t *testing.T) {
	uc := deadletters.NewRetryDeadLetterUseCase(mockDB(t, 0), storeWith(), &stubPublisher{}, &eventrecordtest.Outbox{})

	assert.ErrorIs(t, uc.Execute(superAdmin(), "orders", 7), deadletter.ErrInvalidStream)
}

func TestRetryDeadLetter_onlySuperAdmin(t *testing.T) {
	pub := &stubPublisher{}
	uc := deadletters.NewRetryDeadLetterUseCase(mockDB(t, 0), storeWith(sampleDeadLetter()), pub, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), deadletter.StreamEvents, 7)

	assert.ErrorIs(t, err, deadletter.ErrForbidden)
	assert.Empty(t, pub.published)
}
