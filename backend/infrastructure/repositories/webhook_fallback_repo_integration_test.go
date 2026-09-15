//go:build integration

package repositories_test

import (
	"context"
	"fmt"
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
	var total int64
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		listed, total, err = repo.List(ctx, webhooks.FallbackStatusPendingRepublish, 20, 0)
		return err
	})
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.EqualValues(t, 1, total)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return repo.UpdateStatus(ctx, saved.ID, webhooks.FallbackStatusAbandoned)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		listed, total, err := repo.List(ctx, webhooks.FallbackStatusPendingRepublish, 20, 0)
		require.NoError(t, err)
		require.Empty(t, listed)
		require.EqualValues(t, 0, total)
		return nil
	})
	require.NoError(t, err)
}

// TestWebhookFallbackRepo_Integration_List_respectsLimitAndOffset pins the
// "Undelivered webhooks" admin list's pagination: total reflects every
// matching row regardless of page, while the returned slice is windowed.
func TestWebhookFallbackRepo_Integration_List_respectsLimitAndOffset(t *testing.T) {
	db := newIntegrationDB(t)
	tenantID, ctx := newTestTenant()
	repo := repositories.NewWebhookFallbackRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		for i := 0; i < 3; i++ {
			e := webhooks.FallbackEvent{
				TenantID: tenantID, Provider: "whatsapp", EventID: fmt.Sprintf("evt-%d", i),
				Payload: []byte(`{}`), Status: webhooks.FallbackStatusPendingRepublish,
				ReceivedAt: time.Now().UTC().Add(time.Duration(i) * time.Second),
			}
			if err := repo.Save(ctx, &e); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err)

	var page1, page2 []webhooks.FallbackEvent
	var total1, total2 int64
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		page1, total1, err = repo.List(ctx, webhooks.FallbackStatusPendingRepublish, 2, 0)
		if err != nil {
			return err
		}
		page2, total2, err = repo.List(ctx, webhooks.FallbackStatusPendingRepublish, 2, 2)
		return err
	})
	require.NoError(t, err)
	require.Len(t, page1, 2)
	require.Len(t, page2, 1)
	require.EqualValues(t, 3, total1)
	require.EqualValues(t, 3, total2)
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
