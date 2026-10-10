//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestNotificationRepo_Integration_RoundTripsParams(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewNotificationRepo(db)

	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		if err := repo.Create(ctx, &notificationsDomain.Notification{TenantID: tenant.IDFromContext(ctx), Type: notificationsDomain.TypeStockCloseToExpiry,
			Params: map[string]any{"count": 4, "window_days": 30}, Title: "Stock close to expiry", CreatedAt: time.Now()}); err != nil {
			return err
		}
		return repo.Create(ctx, &notificationsDomain.Notification{TenantID: tenant.IDFromContext(ctx), Type: "legacy", Title: "Old", CreatedAt: time.Now().Add(-time.Hour)})
	}))

	var list []notificationsDomain.Notification
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		list, err = repo.List(ctx, 10)
		return err
	}))

	require.Len(t, list, 2)
	assert.Equal(t, map[string]any{"count": float64(4), "window_days": float64(30)}, list[0].Params)
	assert.Equal(t, map[string]any{}, list[1].Params, "a row without params reads as empty, not nil")
}
