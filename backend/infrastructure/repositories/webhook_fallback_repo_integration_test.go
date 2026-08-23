//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/webhooks"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestWebhookFallbackRepo_Integration_SaveGetListUpdateStatus(t *testing.T) {
	db := newIntegrationDB(t)
	tenantID, ctx := newTestTenant()
	repo := repositories.NewWebhookFallbackRepo(db)

	var saved webhooks.FallbackEvent
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		saved = webhooks.FallbackEvent{
			TenantID: tenantID, Provider: "woocommerce", EventID: "abc123",
			Payload: []byte(`{"id":1}`), Headers: map[string]string{"X-Wc-Webhook-Topic": "order.updated"},
			Status: webhooks.FallbackStatusPendingRepublish, ReceivedAt: time.Now().UTC(),
		}
		return repo.Save(ctx, &saved)
	})
	require.NoError(t, err)
	require.NotEmpty(t, saved.ID)

	var fetched *webhooks.FallbackEvent
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		fetched, err = repo.Get(ctx, saved.ID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, "woocommerce", fetched.Provider)
	require.Equal(t, "order.updated", fetched.Headers["X-Wc-Webhook-Topic"])
	require.Equal(t, webhooks.FallbackStatusPendingRepublish, fetched.Status)

	// Save is idempotent on (tenant_id, provider, event_id) -- a redelivered
	// webhook that already failed to publish once must not error the second
	// time it lands here.
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		dup := webhooks.FallbackEvent{
			TenantID: tenantID, Provider: "woocommerce", EventID: "abc123",
			Payload: []byte(`{"id":1}`), Status: webhooks.FallbackStatusPendingRepublish, ReceivedAt: time.Now().UTC(),
		}
		return repo.Save(ctx, &dup)
	})
	require.NoError(t, err)

	var listed []webhooks.FallbackEvent
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		listed, err = repo.List(ctx, webhooks.FallbackStatusPendingRepublish)
		return err
	})
	require.NoError(t, err)
	require.Len(t, listed, 1)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return repo.UpdateStatus(ctx, saved.ID, webhooks.FallbackStatusAbandoned)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		listed, err := repo.List(ctx, webhooks.FallbackStatusPendingRepublish)
		require.NoError(t, err)
		require.Empty(t, listed)
		return nil
	})
	require.NoError(t, err)
}

func TestWebhookFallbackRepo_Integration_UpdateStatusNotFound(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewWebhookFallbackRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return repo.UpdateStatus(ctx, "00000000-0000-0000-0000-000000000000", webhooks.FallbackStatusAbandoned)
	})
	require.ErrorIs(t, err, webhooks.ErrFallbackEventNotFound)
}
