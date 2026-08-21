//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// TestScheduledJobRepo_Integration_SeededRowsRoundTripAndRunNowWorks proves
// migration 085's seed rows exist and the repo's ListAll/FindByKey/
// RecordRunResult/TriggerNow round-trip against real Postgres -- no
// tenant_id/RLS on this table (see the migration's doc comment), so this
// runs under a plain SuperAdmin-wrapped transaction, same as every other
// cross-tenant repo call in this codebase.
func TestScheduledJobRepo_Integration_SeededRowsRoundTripAndRunNowWorks(t *testing.T) {
	db := newIntegrationDB(t)
	adminCtx := tenant.WithSuperAdmin(context.Background())
	repo := repositories.NewScheduledJobRepo(db)

	err := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		jobs, err := repo.ListAll(ctx)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(jobs), 3, "migration 085 seeds 3 rows")

		job, err := repo.FindByKey(ctx, "expired_batches")
		require.NoError(t, err)
		require.True(t, job.Enabled)
		require.Equal(t, 1, job.IntervalMinutes)

		ranAt := time.Now().Truncate(time.Second)
		nextRunAt := ranAt.Add(time.Minute)
		require.NoError(t, repo.RecordRunResult(ctx, "expired_batches", ranAt, "success", "", nextRunAt))

		updated, err := repo.FindByKey(ctx, "expired_batches")
		require.NoError(t, err)
		require.Equal(t, "success", updated.LastRunStatus)
		require.NotNil(t, updated.LastRunAt)
		require.WithinDuration(t, ranAt, *updated.LastRunAt, time.Second)

		require.NoError(t, repo.TriggerNow(ctx, "expired_batches"))
		triggered, err := repo.FindByKey(ctx, "expired_batches")
		require.NoError(t, err)
		require.WithinDuration(t, time.Now(), triggered.NextRunAt, 5*time.Second)

		due, err := repo.ListDue(ctx, time.Now().Add(time.Hour))
		require.NoError(t, err)
		require.NotEmpty(t, due, "next_run_at was just pulled to now, well within an hour from now")

		return nil
	})
	require.NoError(t, err)
}
