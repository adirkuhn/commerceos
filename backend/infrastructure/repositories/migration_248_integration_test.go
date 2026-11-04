//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// cascadingToConnections lists the foreign keys that would delete their
// rows along with a connection.
func cascadingToConnections(t *testing.T, db *gorm.DB) []string {
	t.Helper()
	names := []string{}
	require.NoError(t, db.Raw(`
		SELECT conname FROM pg_constraint
		WHERE confrelid = 'integration_connections'::regclass AND confdeltype = 'c'
		ORDER BY conname`).Scan(&names).Error)
	return names
}

// The 2026-11-03 incident: one deleted connection row took 664 product
// links, the sales channel and its schedules with it. Nothing cascades now.
func TestMigration248_Integration_noForeignKeyCascadesFromAConnection(t *testing.T) {
	db := newIntegrationDB(t)
	assert.Empty(t, cascadingToConnections(t, db))
}

func TestMigration248_Integration_aConnectionWithLinksCannotBeDeleted(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	id := seedWooConnection(t, db, ctx, "https://shop.example")
	scheduleRepo := repositories.NewSyncScheduleRepo(db)
	now := time.Now()
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return scheduleRepo.Upsert(ctx, &channelsDomain.SyncSchedule{
			TenantID: tenant.IDFromContext(ctx), ConnectionID: id,
			EntityType: channelsDomain.SyncEntityOrders, Direction: channelsDomain.SyncDirectionImport,
			Enabled: true, IntervalMinutes: 5, CreatedAt: now, UpdatedAt: now,
		})
	}))

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		tx, _ := txctx.FromContext(ctx)
		return tx.Exec(`DELETE FROM integration_connections WHERE id = ?`, id).Error
	})

	require.Error(t, err, "the delete must fail, not wipe what hangs off the row")
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		schedules, err := scheduleRepo.ListByConnection(ctx, id)
		require.NoError(t, err)
		assert.Len(t, schedules, 1)
		return nil
	}))
}

func TestMigration248_Integration_goingDownRestoresTheCascades(t *testing.T) {
	db := newIntegrationDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)

	require.NoError(t, goose.DownTo(sqlDB, migrationsDir, 247))
	assert.Len(t, cascadingToConnections(t, db), 8)

	require.NoError(t, goose.UpTo(sqlDB, migrationsDir, 248))
	assert.Empty(t, cascadingToConnections(t, db))
}
