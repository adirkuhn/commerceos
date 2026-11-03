//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

func seedWooConnection(t *testing.T, db *gorm.DB, ctx context.Context, storeURL string) string {
	t.Helper()
	now := time.Now()
	conn := &integrationsDomain.Connection{
		Provider: integrationsDomain.ProviderWooCommerce, StoreURL: integrationsDomain.StoreURL(storeURL),
		ConsumerKey: "ck", ConsumerSecretEncrypted: "enc", Status: integrationsDomain.StatusConnected, CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		conn.TenantID = tenant.IDFromContext(ctx)
		return repositories.NewConnectionRepo(db).Create(ctx, conn)
	}))
	return conn.ID
}

func disconnect(t *testing.T, db *gorm.DB, ctx context.Context, id string) {
	t.Helper()
	repo := repositories.NewConnectionRepo(db)
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		c, err := repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		c.Disconnect(time.Now())
		return repo.Update(ctx, c)
	}))
}

// COM-15: disconnecting keeps the row (no cascade) but hides it from every
// ordinary lookup; FindDisconnected finds it again for a reconnect.
func TestConnectionRepo_Integration_DisconnectedIsHiddenButFoundForReconnect(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewConnectionRepo(db)
	id := seedWooConnection(t, db, ctx, "https://Shop.example/")
	other := seedWooConnection(t, db, ctx, "https://other.example")

	disconnect(t, db, ctx, id)

	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		_, err := repo.FindByID(ctx, id)
		assert.ErrorIs(t, err, integrationsDomain.ErrConnectionNotFound)

		all, err := repo.ListByTenant(ctx)
		require.NoError(t, err)
		require.Len(t, all, 1)
		assert.Equal(t, other, all[0].ID)

		found, err := repo.FindDisconnected(ctx, integrationsDomain.ProviderWooCommerce, "https://shop.example")
		require.NoError(t, err, "same store: case and trailing slash ignored")
		assert.Equal(t, id, found.ID)
		assert.Empty(t, found.ConsumerSecretEncrypted)

		_, err = repo.FindDisconnected(ctx, integrationsDomain.ProviderWooCommerce, "https://other.example")
		assert.ErrorIs(t, err, integrationsDomain.ErrConnectionNotFound, "a connected row is not a reconnect candidate")
		_, err = repo.FindDisconnected(ctx, integrationsDomain.ProviderBol, "https://shop.example")
		assert.ErrorIs(t, err, integrationsDomain.ErrConnectionNotFound)
		return nil
	}))
}

// The 2026-11-03 incident: deleting the connection cascaded into the sales
// channel, listings and schedules and nulled sales_orders.connection_id.
// Disconnect + reconnect must leave all of them in place, under the same id.
func TestConnectionRepo_Integration_DisconnectAndReconnectKeepEverythingReferencingIt(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewConnectionRepo(db)
	scheduleRepo := repositories.NewSyncScheduleRepo(db)
	id := seedWooConnection(t, db, ctx, "https://shop.example")

	var scheduleID string
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		s := &channelsDomain.SyncSchedule{
			TenantID: tenant.IDFromContext(ctx), ConnectionID: id, EntityType: channelsDomain.SyncEntityOrders,
			Direction: channelsDomain.SyncDirectionImport, Enabled: true, IntervalMinutes: 15, CreatedAt: now, UpdatedAt: now,
		}
		if err := scheduleRepo.Upsert(ctx, s); err != nil {
			return err
		}
		scheduleID = s.ID
		if err := scheduleRepo.TriggerNow(ctx, s.ID); err != nil {
			return err
		}
		tx, _ := txctx.FromContext(ctx)
		return tx.Exec(`INSERT INTO sales_channels (id, tenant_id, name, type, connection_id, active, created_at, updated_at)
			VALUES (gen_random_uuid(), ?, 'Woo', 'woocommerce', ?, true, now(), now())`, tenant.IDFromContext(ctx), id).Error
	}))

	disconnect(t, db, ctx, id)

	due := func() []channelsDomain.SyncSchedule {
		var out []channelsDomain.SyncSchedule
		require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			var err error
			out, err = scheduleRepo.ListDue(ctx, time.Now().Add(time.Minute))
			return err
		}))
		return out
	}
	assert.Empty(t, due(), "a disconnected connection's schedules never run")
	var channels int64
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		tx, _ := txctx.FromContext(ctx)
		return tx.Raw(`SELECT count(*) FROM sales_channels WHERE connection_id = ?`, id).Scan(&channels).Error
	}))
	assert.Equal(t, int64(1), channels, "no cascade")

	fresh := integrationsDomain.Connection{
		Provider: integrationsDomain.ProviderWooCommerce, StoreURL: "https://shop.example", ConsumerKey: "ck2",
		ConsumerSecretEncrypted: "enc2", Status: integrationsDomain.StatusConnected, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		fresh.TenantID = tenant.IDFromContext(ctx)
		reconnected, err := integrationsDomain.SaveConnection(ctx, repo, &fresh)
		assert.True(t, reconnected)
		return err
	}))

	assert.Equal(t, id, fresh.ID)
	require.Len(t, due(), 1)
	assert.Equal(t, scheduleID, due()[0].ID)
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		c, err := repo.FindByID(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, "enc2", c.ConsumerSecretEncrypted)
		assert.Equal(t, integrationsDomain.StatusConnected, c.Status)
		return nil
	}))
}
