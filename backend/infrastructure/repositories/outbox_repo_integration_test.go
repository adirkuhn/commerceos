//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/domain/outbox"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestOutboxRepo_Integration_EnqueueAndFetchPending(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewOutboxRepository(db)
	eventID := uuid.NewString()

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return repo.Enqueue(ctx, "pms.product.changed", eventID, []byte(`{"a":1}`), time.Now())
	})
	require.NoError(t, err)

	pending, err := repo.FetchPending(context.Background(), 50)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, eventID, pending[0].EventID)
	require.Equal(t, outbox.StatusPending, pending[0].Status)
	require.Equal(t, 0, pending[0].RetryCount)
}

func TestOutboxRepo_Integration_MarkPublishedRemovesItFromPending(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewOutboxRepository(db)
	id := enqueueOne(t, ctx, db, repo, uuid.NewString())

	err := repo.MarkPublished(context.Background(), id)
	require.NoError(t, err)

	pending, err := repo.FetchPending(context.Background(), 50)
	require.NoError(t, err)
	require.Empty(t, pending)
}

// TestOutboxRepo_Integration_RecordPublishFailureRetriesUntilExhausted is
// the exact reliability gap being closed: a row that fails to publish must
// stay Pending (so the relay's next tick retries it) until it has genuinely
// exhausted its retry budget -- only then does it flip to Failed.
func TestOutboxRepo_Integration_RecordPublishFailureRetriesUntilExhausted(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewOutboxRepository(db)
	id := enqueueOne(t, ctx, db, repo, uuid.NewString())

	require.NoError(t, repo.RecordPublishFailure(context.Background(), id, 3))
	pending, err := repo.FetchPending(context.Background(), 50)
	require.NoError(t, err)
	require.Len(t, pending, 1, "1st failure of 3 -- still pending")
	require.Equal(t, 1, pending[0].RetryCount)

	require.NoError(t, repo.RecordPublishFailure(context.Background(), id, 3))
	pending, err = repo.FetchPending(context.Background(), 50)
	require.NoError(t, err)
	require.Len(t, pending, 1, "2nd failure of 3 -- still pending")
	require.Equal(t, 2, pending[0].RetryCount)

	require.NoError(t, repo.RecordPublishFailure(context.Background(), id, 3))
	pending, err = repo.FetchPending(context.Background(), 50)
	require.NoError(t, err)
	require.Empty(t, pending, "3rd failure of 3 -- budget exhausted, must no longer be picked up")
}

// TestOutboxRepo_Integration_ListFailedBySubjectPrefixAndResetToPending
// covers the backfill path used after the 2026-09-11 incident where
// "shipping.>" was missing from the JetStream stream's subjects: every
// ShipmentFulfillmentSignal outbox row failed to publish with "no stream
// matches subject" and, after exhausting its retry budget, landed in
// Failed. The relay only ever re-fetches Pending rows (see cmd/relay's
// FetchPending call), so once the stream is fixed these rows need an
// explicit requeue -- this is that mechanism (cmd/backfill-outbox-replay).
func TestOutboxRepo_Integration_ListFailedBySubjectPrefixAndResetToPending(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewOutboxRepository(db)

	shippingID := enqueueOneWithSubject(t, ctx, db, repo, "shipping.shipment.fulfillment_signal", uuid.NewString())
	require.NoError(t, repo.MarkFailed(context.Background(), shippingID))

	// A failed row on an unrelated subject must never be swept up by a
	// "shipping." prefix backfill -- ListFailedBySubjectPrefix is a prefix
	// match, not "every Failed row", so an inventory-sync incident and a
	// shipping incident can be replayed independently of each other.
	invID := enqueueOneWithSubject(t, ctx, db, repo, "inventory.stock.changed", uuid.NewString())
	require.NoError(t, repo.MarkFailed(context.Background(), invID))

	failed, err := repo.ListFailedBySubjectPrefix(context.Background(), "shipping.")
	require.NoError(t, err)
	require.Len(t, failed, 1)
	require.Equal(t, shippingID, failed[0].ID)
	require.Equal(t, outbox.StatusFailed, failed[0].Status)

	require.NoError(t, repo.ResetToPending(context.Background(), shippingID))

	pending, err := repo.FetchPending(context.Background(), 50)
	require.NoError(t, err)
	require.Len(t, pending, 1, "only the requeued shipping row -- the unrelated inventory row stays Failed")
	require.Equal(t, shippingID, pending[0].ID)
	require.Equal(t, 0, pending[0].RetryCount, "requeue must reset retry_count so a fresh publish attempt doesn't inherit the old exhausted budget")

	stillFailedShipping, err := repo.ListFailedBySubjectPrefix(context.Background(), "shipping.")
	require.NoError(t, err)
	require.Empty(t, stillFailedShipping, "requeued row must no longer show up as failed")

	stillFailedInventory, err := repo.ListFailedBySubjectPrefix(context.Background(), "inventory.")
	require.NoError(t, err)
	require.Len(t, stillFailedInventory, 1, "unrelated failed row must be untouched by a shipping-only requeue")
	require.Equal(t, invID, stillFailedInventory[0].ID)
}

func enqueueOne(t *testing.T, ctx context.Context, db *gorm.DB, repo *repositories.OutboxRepository, eventID string) uint {
	t.Helper()
	return enqueueOneWithSubject(t, ctx, db, repo, "pms.product.changed", eventID)
}

func enqueueOneWithSubject(t *testing.T, ctx context.Context, db *gorm.DB, repo *repositories.OutboxRepository, subject, eventID string) uint {
	t.Helper()
	var id uint
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return repo.Enqueue(ctx, subject, eventID, []byte(`{}`), time.Now())
	})
	require.NoError(t, err)
	pending, err := repo.FetchPending(context.Background(), 50)
	require.NoError(t, err)
	for _, m := range pending {
		if m.EventID == eventID {
			id = m.ID
		}
	}
	require.NotZero(t, id)
	return id
}
