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

func enqueueOne(t *testing.T, ctx context.Context, db *gorm.DB, repo *repositories.OutboxRepository, eventID string) uint {
	t.Helper()
	var id uint
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return repo.Enqueue(ctx, "pms.product.changed", eventID, []byte(`{}`), time.Now())
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
