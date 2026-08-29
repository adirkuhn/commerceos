//go:build integration

package repositories_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/domain/webhooks"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func saveWebhookEvent(t *testing.T, db *gorm.DB, ctx context.Context, provider, topic, eventID string, receivedAt time.Time) string {
	t.Helper()
	repo := repositories.NewWebhookEventRepo(db)
	var saved webhooks.WebhookEvent
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		saved = webhooks.WebhookEvent{
			TenantID: tenant.IDFromContext(ctx), Provider: provider, Topic: topic,
			EventID: eventID, Payload: []byte(`{}`), ReceivedAt: receivedAt,
		}
		return repo.Save(ctx, &saved)
	})
	require.NoError(t, err)
	return saved.ID
}

func TestWebhookEventRepo_Integration_List_filtersByProviderNewestFirst(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctxA := newTestTenant()
	repo := repositories.NewWebhookEventRepo(db)

	now := time.Now().UTC().Truncate(time.Second)
	saveWebhookEvent(t, db, ctxA, "myparcel", "shipment_status_change", "evt-1", now.Add(-2*time.Hour))
	saveWebhookEvent(t, db, ctxA, "woocommerce", "order.updated", "evt-2", now.Add(-1*time.Hour))
	saveWebhookEvent(t, db, ctxA, "myparcel", "shop_updated", "evt-3", now)

	var all []webhooks.WebhookEvent
	err := tenant.WithTenant(ctxA, db, func(ctx context.Context) error {
		var err error
		all, err = repo.List(ctx, "", 50)
		return err
	})
	require.NoError(t, err)
	require.Len(t, all, 3)
	require.Equal(t, []string{"evt-3", "evt-2", "evt-1"}, []string{all[0].EventID, all[1].EventID, all[2].EventID}, "newest first")

	var myParcelOnly []webhooks.WebhookEvent
	err = tenant.WithTenant(ctxA, db, func(ctx context.Context) error {
		var err error
		myParcelOnly, err = repo.List(ctx, "myparcel", 50)
		return err
	})
	require.NoError(t, err)
	require.Len(t, myParcelOnly, 2)
	for _, e := range myParcelOnly {
		require.Equal(t, "myparcel", e.Provider)
	}
}

func TestWebhookEventRepo_Integration_List_respectsLimit(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewWebhookEventRepo(db)

	now := time.Now().UTC()
	for i := 0; i < 5; i++ {
		saveWebhookEvent(t, db, ctx, "woocommerce", "order.updated", fmt.Sprintf("evt-%d", i), now.Add(time.Duration(i)*time.Minute))
	}

	var limited []webhooks.WebhookEvent
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		limited, err = repo.List(ctx, "", 2)
		return err
	})
	require.NoError(t, err)
	require.Len(t, limited, 2)
}

func TestWebhookEventRepo_Integration_Save_defaultsStatusToReceived(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewWebhookEventRepo(db)

	saveWebhookEvent(t, db, ctx, "woocommerce", "order.updated", "evt-default", time.Now())

	var all []webhooks.WebhookEvent
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		all, err = repo.List(ctx, "", 50)
		return err
	})
	require.NoError(t, err)
	require.Len(t, all, 1)
	require.Equal(t, webhooks.WebhookEventStatusReceived, all[0].Status)
	require.Empty(t, all[0].DispatchError)
}

func TestWebhookEventRepo_Integration_MarkDispatchResult_updatesStatusAndError(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewWebhookEventRepo(db)

	eventID := saveWebhookEvent(t, db, ctx, "myparcel", "shipment_status_change", "evt-mark", time.Now())

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return repo.MarkDispatchResult(ctx, eventID, webhooks.WebhookEventStatusFailed, "myparcel unreachable")
	})
	require.NoError(t, err)

	var all []webhooks.WebhookEvent
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		all, err = repo.List(ctx, "", 50)
		return err
	})
	require.NoError(t, err)
	require.Len(t, all, 1)
	require.Equal(t, webhooks.WebhookEventStatusFailed, all[0].Status)
	require.Equal(t, "myparcel unreachable", all[0].DispatchError)

	// Marking it processed on retry must overwrite the prior failure, not
	// stack another row.
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return repo.MarkDispatchResult(ctx, eventID, webhooks.WebhookEventStatusProcessed, "")
	})
	require.NoError(t, err)
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		all, err = repo.List(ctx, "", 50)
		return err
	})
	require.NoError(t, err)
	require.Len(t, all, 1)
	require.Equal(t, webhooks.WebhookEventStatusProcessed, all[0].Status)
	require.Empty(t, all[0].DispatchError)
}

// TestWebhookEventRepo_Integration_List_RLSHidesOtherTenantsEvents: RLS is
// never enforced against a superuser/BYPASSRLS role regardless of policy,
// so this only means something routed through commerceos_app -- the same
// non-superuser role backend/relay/scheduler actually connect as in
// production (see migration 020_app_role_and_force_rls.sql).
func TestWebhookEventRepo_Integration_List_RLSHidesOtherTenantsEvents(t *testing.T) {
	superDB, superDSN := newIntegrationDBAndDSN(t)
	db := newAppRoleDB(t, superDSN)
	_, ctxA := newTestTenant()
	_, ctxB := newTestTenant()
	repo := repositories.NewWebhookEventRepo(db)

	now := time.Now().UTC()
	saveWebhookEvent(t, superDB, ctxA, "myparcel", "shipment_status_change", "evt-a", now)
	saveWebhookEvent(t, superDB, ctxB, "myparcel", "shipment_status_change", "evt-b", now)

	var eventsForA []webhooks.WebhookEvent
	err := tenant.WithTenant(ctxA, db, func(ctx context.Context) error {
		var err error
		eventsForA, err = repo.List(ctx, "", 50)
		return err
	})
	require.NoError(t, err)
	require.Len(t, eventsForA, 1, "must not see the other tenant's event")
	require.Equal(t, "evt-a", eventsForA[0].EventID)
}
